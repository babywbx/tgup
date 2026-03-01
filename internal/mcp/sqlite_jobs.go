package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// JobRunner executes a job. It should emit events via the emit callback and
// return the result when done.
type JobRunner func(ctx context.Context, spec RunSpec, emit func(Event)) (*RunResult, error)

// JobManager manages MCP job lifecycle with SQLite persistence and
// semaphore-based concurrency control.
type JobManager struct {
	db       *sql.DB
	events   EventStore
	sem      chan struct{}
	mu       sync.RWMutex
	cancels  map[string]context.CancelFunc
	waiters  map[string][]chan struct{}
	maxQueue int
}

// JobManagerConfig configures the job manager.
type JobManagerConfig struct {
	MaxConcurrent int
	MaxQueue      int
}

// OpenJobManager opens a SQLite-backed job manager.
func OpenJobManager(dbPath string, events EventStore, cfg JobManagerConfig) (*JobManager, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(wal)&_pragma=busy_timeout(60000)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open job manager: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := migrateJobStore(db); err != nil {
		db.Close()
		return nil, err
	}

	maxConcurrent := cfg.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 4
	}
	maxQueue := cfg.MaxQueue
	if maxQueue <= 0 {
		maxQueue = 100
	}

	return &JobManager{
		db:       db,
		events:   events,
		sem:      make(chan struct{}, maxConcurrent),
		cancels:  make(map[string]context.CancelFunc),
		waiters:  make(map[string][]chan struct{}),
		maxQueue: maxQueue,
	}, nil
}

func migrateJobStore(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS mcp_jobs (
			id          TEXT PRIMARY KEY,
			status      TEXT NOT NULL DEFAULT 'queued',
			config_path TEXT NOT NULL DEFAULT '',
			run_spec    BLOB,
			created_at  TEXT NOT NULL,
			started_at  TEXT,
			finished_at TEXT,
			result      BLOB,
			error       TEXT NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_jobs_status ON mcp_jobs(status);
	`)
	if err != nil {
		return fmt.Errorf("migrate job store: %w", err)
	}
	return nil
}

// Start creates a new job and launches its runner in a background goroutine.
func (m *JobManager) Start(ctx context.Context, spec RunSpec, runner JobRunner) (Job, error) {
	// Check queue depth.
	count, err := m.countNonTerminal(ctx)
	if err != nil {
		return Job{}, err
	}
	if count >= m.maxQueue {
		return Job{}, fmt.Errorf("queue full: %d/%d jobs", count, m.maxQueue)
	}

	job := Job{
		ID:         uuid.New().String(),
		Status:     JobQueued,
		ConfigPath: spec.ConfigPath,
		RunSpec:    &spec,
		CreatedAt:  time.Now().UTC(),
	}

	specBytes, err := json.Marshal(spec)
	if err != nil {
		return Job{}, fmt.Errorf("marshal run spec: %w", err)
	}

	_, err = m.db.ExecContext(ctx,
		"INSERT INTO mcp_jobs (id, status, config_path, run_spec, created_at) VALUES (?, ?, ?, ?, ?)",
		job.ID, job.Status, job.ConfigPath, specBytes, job.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Job{}, fmt.Errorf("insert job: %w", err)
	}

	// Emit queued event.
	m.emitEvent(ctx, job.ID, "job.queued", nil, false)

	// Launch runner goroutine.
	runCtx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	m.cancels[job.ID] = cancel
	m.mu.Unlock()

	go m.run(runCtx, job.ID, spec, runner)

	return job, nil
}

func (m *JobManager) run(ctx context.Context, jobID string, spec RunSpec, runner JobRunner) {
	defer func() {
		m.mu.Lock()
		delete(m.cancels, jobID)
		// Notify waiters.
		for _, ch := range m.waiters[jobID] {
			close(ch)
		}
		delete(m.waiters, jobID)
		m.mu.Unlock()
	}()

	// Acquire semaphore.
	select {
	case m.sem <- struct{}{}:
	case <-ctx.Done():
		m.finishJob(jobID, JobCancelled, nil, "cancelled before start")
		return
	}
	defer func() { <-m.sem }()

	// Mark running.
	now := time.Now().UTC()
	m.db.Exec(
		"UPDATE mcp_jobs SET status = ?, started_at = ? WHERE id = ?",
		JobRunning, now.Format(time.RFC3339Nano), jobID)
	m.emitEvent(context.Background(), jobID, "job.running", nil, false)

	// Execute runner with panic recovery.
	result, err := m.safeRun(ctx, jobID, spec, runner)

	if ctx.Err() != nil {
		m.finishJob(jobID, JobCancelled, result, "cancelled")
		return
	}
	if err != nil {
		m.finishJob(jobID, JobFailed, result, err.Error())
		return
	}
	m.finishJob(jobID, JobDone, result, "")
}

func (m *JobManager) safeRun(ctx context.Context, jobID string, spec RunSpec, runner JobRunner) (result *RunResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("job runner panic: %v", r)
		}
	}()

	emit := func(e Event) {
		e.JobID = jobID
		m.events.Append(context.Background(), e)
	}

	return runner(ctx, spec, emit)
}

func (m *JobManager) finishJob(jobID string, status string, result *RunResult, errMsg string) {
	now := time.Now().UTC()
	var resultBytes []byte
	if result != nil {
		resultBytes, _ = json.Marshal(result)
	}

	m.db.Exec(
		"UPDATE mcp_jobs SET status = ?, finished_at = ?, result = ?, error = ? WHERE id = ?",
		status, now.Format(time.RFC3339Nano), resultBytes, errMsg, jobID)

	payload, _ := json.Marshal(map[string]any{
		"status": status,
		"error":  errMsg,
	})
	m.emitEvent(context.Background(), jobID, "job."+status, payload, true)
}

// Cancel requests cancellation of a job.
func (m *JobManager) Cancel(ctx context.Context, jobID string) (Job, error) {
	m.mu.RLock()
	cancel, ok := m.cancels[jobID]
	m.mu.RUnlock()

	if ok {
		cancel()
	} else {
		// Maybe it's still queued — mark as cancelled directly.
		res, err := m.db.ExecContext(ctx,
			"UPDATE mcp_jobs SET status = ?, finished_at = ? WHERE id = ? AND status = ?",
			JobCancelled, time.Now().UTC().Format(time.RFC3339Nano), jobID, JobQueued)
		if err != nil {
			return Job{}, fmt.Errorf("cancel job: %w", err)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			m.emitEvent(ctx, jobID, "job.cancelled", nil, true)
		}
	}

	return m.Get(ctx, jobID)
}

// Get returns a job by ID.
func (m *JobManager) Get(ctx context.Context, jobID string) (Job, error) {
	var job Job
	var createdAt, startedAt, finishedAt sql.NullString
	var specBytes, resultBytes []byte

	err := m.db.QueryRowContext(ctx,
		"SELECT id, status, config_path, run_spec, created_at, started_at, finished_at, result, error FROM mcp_jobs WHERE id = ?",
		jobID).Scan(&job.ID, &job.Status, &job.ConfigPath, &specBytes, &createdAt, &startedAt, &finishedAt, &resultBytes, &job.Error)
	if err == sql.ErrNoRows {
		return Job{}, fmt.Errorf("job not found: %s", jobID)
	}
	if err != nil {
		return Job{}, fmt.Errorf("get job: %w", err)
	}

	if createdAt.Valid {
		job.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt.String)
	}
	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339Nano, startedAt.String)
		job.StartedAt = &t
	}
	if finishedAt.Valid {
		t, _ := time.Parse(time.RFC3339Nano, finishedAt.String)
		job.FinishedAt = &t
	}
	if specBytes != nil {
		var spec RunSpec
		if json.Unmarshal(specBytes, &spec) == nil {
			job.RunSpec = &spec
		}
	}
	if resultBytes != nil {
		var result RunResult
		if json.Unmarshal(resultBytes, &result) == nil {
			job.Result = &result
		}
	}

	return job, nil
}

// Wait blocks until a job reaches a terminal state.
func (m *JobManager) Wait(ctx context.Context, jobID string) (Job, error) {
	// Register waiter BEFORE checking terminal state to avoid lost wakeup.
	ch := make(chan struct{})
	m.mu.Lock()
	m.waiters[jobID] = append(m.waiters[jobID], ch)
	m.mu.Unlock()

	// Check if already terminal (after registering waiter).
	job, err := m.Get(ctx, jobID)
	if err != nil {
		return Job{}, err
	}
	if job.IsTerminal() {
		return job, nil
	}

	select {
	case <-ch:
		return m.Get(ctx, jobID)
	case <-ctx.Done():
		return Job{}, ctx.Err()
	}
}

func (m *JobManager) countNonTerminal(ctx context.Context) (int, error) {
	var count int
	err := m.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM mcp_jobs WHERE status IN (?, ?)",
		JobQueued, JobRunning).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count jobs: %w", err)
	}
	return count, nil
}

func (m *JobManager) emitEvent(ctx context.Context, jobID string, eventType string, payload json.RawMessage, terminal bool) {
	m.events.Append(ctx, Event{
		JobID:    jobID,
		Type:     eventType,
		Payload:  payload,
		Terminal: terminal,
	})
}

// Close closes the underlying database.
func (m *JobManager) Close() error {
	return m.db.Close()
}
