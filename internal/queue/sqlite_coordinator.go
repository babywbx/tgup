package queue

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const sqliteDriverName = "sqlite"

// SQLiteOptions controls queue timing behavior.
type SQLiteOptions struct {
	HeartbeatTTL time.Duration
	PollInterval time.Duration
}

// SQLiteCoordinator coordinates run turn-taking via SQLite.
type SQLiteCoordinator struct {
	db           *sql.DB
	runID        string
	heartbeatTTL time.Duration
	pollInterval time.Duration
	createdAt    int64
}

// OpenSQLite opens queue coordinator over the given sqlite path.
func OpenSQLite(path string, runID string, opts SQLiteOptions) (*SQLiteCoordinator, error) {
	if runID == "" {
		return nil, fmt.Errorf("run id is required")
	}
	if opts.HeartbeatTTL <= 0 {
		opts.HeartbeatTTL = 30 * time.Second
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = 250 * time.Millisecond
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create queue directory: %w", err)
	}
	db, err := sql.Open(sqliteDriverName, path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	c := &SQLiteCoordinator{
		db:           db,
		runID:        runID,
		heartbeatTTL: opts.HeartbeatTTL,
		pollInterval: opts.PollInterval,
	}
	if err := c.initSchema(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return c, nil
}

func (c *SQLiteCoordinator) initSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS run_queue (
			run_id TEXT PRIMARY KEY,
			status TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			heartbeat_at INTEGER NOT NULL,
			finished_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_run_queue_status_created ON run_queue(status, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_run_queue_heartbeat ON run_queue(status, heartbeat_at)`,
	}
	for _, stmt := range stmts {
		if _, err := c.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("init queue schema: %w", err)
		}
	}
	return nil
}

// RunID returns current run identifier.
func (c *SQLiteCoordinator) RunID() string { return c.runID }

// WaitUntilTurn waits until this run is at queue head.
func (c *SQLiteCoordinator) WaitUntilTurn(ctx context.Context, onWait func(ahead int)) error {
	now := time.Now().Unix()
	if c.createdAt == 0 {
		c.createdAt = now
		if _, err := c.db.ExecContext(ctx, `
			INSERT INTO run_queue(run_id, status, created_at, heartbeat_at, finished_at)
			VALUES(?, 'waiting', ?, ?, 0)
			ON CONFLICT(run_id) DO UPDATE SET
				status='waiting',
				heartbeat_at=excluded.heartbeat_at,
				finished_at=0
		`, c.runID, c.createdAt, now); err != nil {
			return fmt.Errorf("register run queue row: %w", err)
		}
	}

	for {
		if err := c.markStale(ctx); err != nil {
			return err
		}

		ahead, err := c.countAhead(ctx)
		if err != nil {
			return err
		}
		if ahead == 0 {
			if _, err := c.db.ExecContext(ctx,
				`UPDATE run_queue SET status='running', heartbeat_at=? WHERE run_id=?`,
				time.Now().Unix(),
				c.runID,
			); err != nil {
				return fmt.Errorf("mark running: %w", err)
			}
			return nil
		}

		if onWait != nil {
			onWait(ahead)
		}
		timer := time.NewTimer(c.pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// Heartbeat updates liveness for current run.
func (c *SQLiteCoordinator) Heartbeat(ctx context.Context) error {
	_, err := c.db.ExecContext(ctx,
		`UPDATE run_queue SET heartbeat_at=? WHERE run_id=?`,
		time.Now().Unix(),
		c.runID,
	)
	if err != nil {
		return fmt.Errorf("heartbeat: %w", err)
	}
	return nil
}

// Finish marks run completed with status.
func (c *SQLiteCoordinator) Finish(ctx context.Context, status string) error {
	if status == "" {
		status = "finished"
	}
	now := time.Now().Unix()
	_, err := c.db.ExecContext(ctx,
		`UPDATE run_queue SET status=?, heartbeat_at=?, finished_at=? WHERE run_id=?`,
		status,
		now,
		now,
		c.runID,
	)
	if err != nil {
		return fmt.Errorf("finish: %w", err)
	}
	return nil
}

// Cancel marks run canceled.
func (c *SQLiteCoordinator) Cancel(ctx context.Context) error {
	return c.Finish(ctx, "canceled")
}

// Close closes underlying database.
func (c *SQLiteCoordinator) Close() error {
	if c == nil || c.db == nil {
		return nil
	}
	return c.db.Close()
}

func (c *SQLiteCoordinator) countAhead(ctx context.Context) (int, error) {
	var ahead int
	err := c.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM run_queue
		WHERE status IN ('waiting', 'running')
		  AND (
				created_at < ?
				OR (created_at = ? AND run_id < ?)
		  )
	`,
		c.createdAt,
		c.createdAt,
		c.runID,
	).Scan(&ahead)
	if err != nil {
		return 0, fmt.Errorf("count ahead: %w", err)
	}
	return ahead, nil
}

func (c *SQLiteCoordinator) markStale(ctx context.Context) error {
	staleBefore := time.Now().Add(-c.heartbeatTTL).Unix()
	_, err := c.db.ExecContext(ctx, `
		UPDATE run_queue
		SET status='stale', finished_at=?
		WHERE status IN ('waiting', 'running')
		  AND heartbeat_at < ?
		  AND run_id <> ?
	`, time.Now().Unix(), staleBefore, c.runID)
	if err != nil {
		return fmt.Errorf("mark stale: %w", err)
	}
	return nil
}
