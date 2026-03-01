package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/babywbx/tgup/internal/xerrors"
	_ "modernc.org/sqlite"
)

const sqliteDriverName = "sqlite"

// SQLiteStore implements Store with SQLite persistence.
type SQLiteStore struct {
	db *sql.DB
}

// OpenSQLite opens or creates a state.sqlite-compatible store.
func OpenSQLite(path string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, xerrors.Wrap(xerrors.CodeState, "create state directory", err)
	}

	db, err := sql.Open(sqliteDriverName, path)
	if err != nil {
		return nil, xerrors.Wrap(xerrors.CodeState, "open sqlite", err)
	}
	db.SetMaxOpenConns(1)

	store := &SQLiteStore{db: db}
	if err := store.initSchema(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) initSchema(ctx context.Context) error {
	stmts := []string{
		createUploadsTableSQL,
		createRunQueueTableSQL,
		createRunQueueStatusIndexSQL,
		createRunQueueHeartbeatIndexSQL,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return xerrors.Wrap(xerrors.CodeState, "init schema", err)
		}
	}
	return nil
}

// MarkSent marks one item as sent.
func (s *SQLiteStore) MarkSent(ctx context.Context, in MarkSentInput) error {
	now := time.Now().Unix()
	rawIDs, err := json.Marshal(in.MessageIDs)
	if err != nil {
		return xerrors.Wrap(xerrors.CodeState, "marshal message ids", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO uploads(path, size, mtime_ns, status, target, error_reason, message_ids, album_group_id, created_at, updated_at)
		VALUES(?, ?, ?, 'sent', ?, '', ?, ?, ?, ?)
		ON CONFLICT(path, size, mtime_ns) DO UPDATE SET
			status='sent',
			target=excluded.target,
			error_reason='',
			message_ids=excluded.message_ids,
			album_group_id=excluded.album_group_id,
			updated_at=excluded.updated_at
	`,
		in.Key.Path,
		in.Key.Size,
		in.Key.MTimeNS,
		in.Target,
		string(rawIDs),
		in.AlbumGroupID,
		now,
		now,
	)
	if err != nil {
		return xerrors.Wrap(xerrors.CodeState, "mark sent", err)
	}
	return nil
}

// MarkFailed marks one item as failed.
func (s *SQLiteStore) MarkFailed(ctx context.Context, in MarkFailedInput) error {
	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO uploads(path, size, mtime_ns, status, target, error_reason, message_ids, album_group_id, created_at, updated_at)
		VALUES(?, ?, ?, 'failed', ?, ?, '[]', '', ?, ?)
		ON CONFLICT(path, size, mtime_ns) DO UPDATE SET
			status='failed',
			target=excluded.target,
			error_reason=excluded.error_reason,
			message_ids='[]',
			album_group_id='',
			updated_at=excluded.updated_at
	`,
		in.Key.Path,
		in.Key.Size,
		in.Key.MTimeNS,
		in.Target,
		in.ErrorReason,
		now,
		now,
	)
	if err != nil {
		return xerrors.Wrap(xerrors.CodeState, "mark failed", err)
	}
	return nil
}

// IsDone checks whether an item is already marked sent.
func (s *SQLiteStore) IsDone(ctx context.Context, item ResumeKey) (bool, error) {
	var status string
	err := s.db.QueryRowContext(ctx,
		`SELECT status FROM uploads WHERE path=? AND size=? AND mtime_ns=?`,
		item.Path,
		item.Size,
		item.MTimeNS,
	).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, xerrors.Wrap(xerrors.CodeState, "check done", err)
	}
	return status == "sent", nil
}

// ListPending returns input keys that are not marked sent.
func (s *SQLiteStore) ListPending(ctx context.Context, items []ResumeKey) ([]ResumeKey, error) {
	out := make([]ResumeKey, 0, len(items))
	for _, item := range items {
		done, err := s.IsDone(ctx, item)
		if err != nil {
			return nil, err
		}
		if !done {
			out = append(out, item)
		}
	}
	return out, nil
}

// GetUploadRow returns persisted upload row for one key.
func (s *SQLiteStore) GetUploadRow(ctx context.Context, item ResumeKey) (*UploadRow, error) {
	var row UploadRow
	var messageIDsRaw string
	err := s.db.QueryRowContext(ctx, `
		SELECT path, size, mtime_ns, status, target, error_reason, message_ids, album_group_id, created_at, updated_at
		FROM uploads
		WHERE path=? AND size=? AND mtime_ns=?
	`,
		item.Path,
		item.Size,
		item.MTimeNS,
	).Scan(
		&row.Path,
		&row.Size,
		&row.MTimeNS,
		&row.Status,
		&row.Target,
		&row.ErrorReason,
		&messageIDsRaw,
		&row.AlbumGroupID,
		&row.CreatedAtUnix,
		&row.UpdatedAtUnix,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, xerrors.Wrap(xerrors.CodeState, "get upload row", err)
	}
	if err := json.Unmarshal([]byte(messageIDsRaw), &row.MessageIDs); err != nil {
		return nil, xerrors.Wrap(xerrors.CodeState, "parse message ids", err)
	}
	return &row, nil
}

// ApplyMaintenance applies cleanup rules, with preview support.
func (s *SQLiteStore) ApplyMaintenance(ctx context.Context, cfg MaintenanceConfig, force bool) (CleanupReport, error) {
	report := CleanupReport{Preview: !force}
	if !cfg.Enabled {
		return report, nil
	}
	if force && cfg.MaxAge <= 0 {
		return CleanupReport{}, xerrors.Wrap(xerrors.CodeState, "maintenance: max_age must be positive for force mode", nil)
	}

	cutoffUnix := int64(0)
	if cfg.MaxAge > 0 {
		cutoffUnix = time.Now().Add(-cfg.MaxAge).Unix()
	}

	uploadWhere, uploadArgs := buildUploadCleanupWhere(cfg, cutoffUnix)
	queueWhere, queueArgs := buildQueueCleanupWhere(cutoffUnix)

	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM uploads WHERE "+uploadWhere, uploadArgs...).Scan(&report.DeletedUploads); err != nil {
		return CleanupReport{}, xerrors.Wrap(xerrors.CodeState, "maintenance preview uploads", err)
	}
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM run_queue WHERE "+queueWhere, queueArgs...).Scan(&report.DeletedQueueRows); err != nil {
		return CleanupReport{}, xerrors.Wrap(xerrors.CodeState, "maintenance preview queue", err)
	}

	if !force {
		return report, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CleanupReport{}, xerrors.Wrap(xerrors.CodeState, "begin maintenance tx", err)
	}
	defer tx.Rollback()

	uploadRes, err := tx.ExecContext(ctx, "DELETE FROM uploads WHERE "+uploadWhere, uploadArgs...)
	if err != nil {
		return CleanupReport{}, xerrors.Wrap(xerrors.CodeState, "delete uploads", err)
	}
	queueRes, err := tx.ExecContext(ctx, "DELETE FROM run_queue WHERE "+queueWhere, queueArgs...)
	if err != nil {
		return CleanupReport{}, xerrors.Wrap(xerrors.CodeState, "delete run_queue", err)
	}
	if err := tx.Commit(); err != nil {
		return CleanupReport{}, xerrors.Wrap(xerrors.CodeState, "commit maintenance tx", err)
	}

	if n, raErr := uploadRes.RowsAffected(); raErr == nil {
		report.DeletedUploads = int(n)
	}
	if n, raErr := queueRes.RowsAffected(); raErr == nil {
		report.DeletedQueueRows = int(n)
	}

	// VACUUM is best-effort; cannot run inside a transaction.
	if _, vacuumErr := s.db.ExecContext(ctx, "VACUUM"); vacuumErr == nil {
		report.Vacuumed = true
	}
	report.Preview = false
	return report, nil
}

// Close closes underlying database.
func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
