package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteEventStore implements EventStore backed by SQLite.
type SQLiteEventStore struct {
	db *sql.DB

	subMu   sync.RWMutex
	subs    map[string][]*sqliteSub // keyed by jobID
	globalS []*sqliteSub            // "" key = all jobs
}

type sqliteSub struct {
	jobID string
	ch    chan Event
	done  chan struct{}
	once  sync.Once
}

func (s *sqliteSub) C() <-chan Event { return s.ch }

func (s *sqliteSub) Close() {
	s.once.Do(func() { close(s.done) })
}

// OpenEventStore opens a SQLite event store at path.
func OpenEventStore(path string) (*SQLiteEventStore, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(wal)&_pragma=busy_timeout(60000)&_pragma=foreign_keys(on)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open event store: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := migrateEventStore(db); err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteEventStore{
		db:   db,
		subs: make(map[string][]*sqliteSub),
	}, nil
}

func migrateEventStore(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS mcp_events (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id    TEXT    NOT NULL,
			seq       INTEGER NOT NULL,
			type      TEXT    NOT NULL,
			payload   BLOB,
			terminal  BOOLEAN NOT NULL DEFAULT 0,
			ts        TEXT    NOT NULL,
			UNIQUE(job_id, seq)
		);
		CREATE INDEX IF NOT EXISTS idx_events_job_seq ON mcp_events(job_id, seq);

		CREATE TABLE IF NOT EXISTS mcp_sessions (
			id         TEXT PRIMARY KEY,
			created_at TEXT NOT NULL,
			last_seen  TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("migrate event store: %w", err)
	}
	return nil
}

// Append inserts a new event atomically and notifies subscribers.
// Seq is assigned via a single INSERT with subquery to avoid races.
func (s *SQLiteEventStore) Append(ctx context.Context, event Event) (Event, error) {
	now := time.Now().UTC()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}

	payloadBytes, err := marshalPayload(event.Payload)
	if err != nil {
		return Event{}, err
	}

	// Atomic seq assignment: COALESCE(MAX(seq),0)+1 in a single INSERT.
	var id int64
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO mcp_events (job_id, seq, type, payload, terminal, ts)
		 VALUES (?, COALESCE((SELECT MAX(seq) FROM mcp_events WHERE job_id = ?), 0) + 1, ?, ?, ?, ?)
		 RETURNING seq`,
		event.JobID, event.JobID, event.Type, payloadBytes, event.Terminal,
		event.CreatedAt.Format(time.RFC3339Nano)).Scan(&id)
	if err != nil {
		return Event{}, fmt.Errorf("event store: insert: %w", err)
	}
	event.Seq = id

	// Notify subscribers.
	s.notify(event)

	return event, nil
}

func marshalPayload(raw json.RawMessage) ([]byte, error) {
	if raw == nil {
		return nil, nil
	}
	return []byte(raw), nil
}

// List returns events after sinceSeq, up to limit.
// If jobID is empty, returns events across all jobs (ordered by id).
func (s *SQLiteEventStore) List(ctx context.Context, jobID string, sinceSeq int64, limit int) ([]Event, bool, error) {
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}

	var query string
	var args []any
	if jobID == "" {
		// Global: use auto-increment id as ordering key for SSE replay.
		query = "SELECT id, job_id, type, payload, terminal, ts FROM mcp_events WHERE id > ? ORDER BY id ASC LIMIT ?"
		args = []any{sinceSeq, limit + 1}
	} else {
		query = "SELECT seq, job_id, type, payload, terminal, ts FROM mcp_events WHERE job_id = ? AND seq > ? ORDER BY seq ASC LIMIT ?"
		args = []any{jobID, sinceSeq, limit + 1}
	}

	// Fetch limit+1 to detect hasMore.
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("event store: list: %w", err)
	}
	defer rows.Close()

	events := make([]Event, 0, limit)
	for rows.Next() {
		var e Event
		var ts string
		var payload []byte
		if err := rows.Scan(&e.Seq, &e.JobID, &e.Type, &payload, &e.Terminal, &ts); err != nil {
			return nil, false, fmt.Errorf("event store: scan: %w", err)
		}
		if payload != nil {
			e.Payload = json.RawMessage(payload)
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339Nano, ts)
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("event store: rows: %w", err)
	}

	hasMore := len(events) > limit
	if hasMore {
		events = events[:limit]
	}
	return events, hasMore, nil
}

// Register creates a subscription for events on the given jobID.
// Use "" for all jobs.
func (s *SQLiteEventStore) Register(jobID string) Subscription {
	sub := &sqliteSub{
		jobID: jobID,
		ch:    make(chan Event, 64),
		done:  make(chan struct{}),
	}

	s.subMu.Lock()
	if jobID == "" {
		s.globalS = append(s.globalS, sub)
	} else {
		s.subs[jobID] = append(s.subs[jobID], sub)
	}
	s.subMu.Unlock()

	// Auto-cleanup when done.
	go func() {
		<-sub.done
		s.removeSub(sub)
	}()

	return sub
}

func (s *SQLiteEventStore) removeSub(sub *sqliteSub) {
	s.subMu.Lock()
	defer s.subMu.Unlock()

	if sub.jobID == "" {
		s.globalS = removeSqliteSub(s.globalS, sub)
	} else {
		list := s.subs[sub.jobID]
		list = removeSqliteSub(list, sub)
		if len(list) == 0 {
			delete(s.subs, sub.jobID)
		} else {
			s.subs[sub.jobID] = list
		}
	}
}

func removeSqliteSub(list []*sqliteSub, target *sqliteSub) []*sqliteSub {
	for i, s := range list {
		if s == target {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func (s *SQLiteEventStore) notify(event Event) {
	s.subMu.RLock()
	defer s.subMu.RUnlock()

	// Notify job-specific subscribers.
	for _, sub := range s.subs[event.JobID] {
		select {
		case sub.ch <- event:
		default:
			// Drop if channel full.
		}
	}

	// Notify global subscribers.
	for _, sub := range s.globalS {
		select {
		case sub.ch <- event:
		default:
		}
	}
}

// Cleanup deletes events older than maxAge.
func (s *SQLiteEventStore) Cleanup(ctx context.Context) error {
	cutoff := time.Now().UTC().Add(-72 * time.Hour)
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM mcp_events WHERE ts < ?", cutoff.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("event store: cleanup: %w", err)
	}
	return nil
}

// CleanupWithRetention deletes events older than the given duration.
func (s *SQLiteEventStore) CleanupWithRetention(ctx context.Context, retention time.Duration) error {
	cutoff := time.Now().UTC().Add(-retention)
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM mcp_events WHERE ts < ?", cutoff.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("event store: cleanup: %w", err)
	}
	return nil
}

// TouchSession updates a session's last_seen timestamp, creating it if needed.
func (s *SQLiteEventStore) TouchSession(ctx context.Context, sessionID string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO mcp_sessions (id, created_at, last_seen) VALUES (?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET last_seen = excluded.last_seen`,
		sessionID, now, now)
	if err != nil {
		return fmt.Errorf("event store: touch session: %w", err)
	}
	return nil
}

// CleanupSessions removes sessions older than maxAge.
func (s *SQLiteEventStore) CleanupSessions(ctx context.Context, maxAge time.Duration) error {
	cutoff := time.Now().UTC().Add(-maxAge).Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM mcp_sessions WHERE last_seen < ?", cutoff)
	if err != nil {
		return fmt.Errorf("event store: cleanup sessions: %w", err)
	}
	return nil
}

// Close closes the underlying database.
func (s *SQLiteEventStore) Close() error {
	return s.db.Close()
}
