package state

const (
	createUploadsTableSQL = `CREATE TABLE IF NOT EXISTS uploads (
		path TEXT NOT NULL,
		size INTEGER NOT NULL,
		mtime_ns INTEGER NOT NULL,
		status TEXT NOT NULL,
		target TEXT NOT NULL DEFAULT '',
		error_reason TEXT NOT NULL DEFAULT '',
		message_ids TEXT NOT NULL DEFAULT '[]',
		album_group_id TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		PRIMARY KEY(path, size, mtime_ns)
	)`
	createRunQueueTableSQL = `CREATE TABLE IF NOT EXISTS run_queue (
		run_id TEXT PRIMARY KEY,
		status TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		heartbeat_at INTEGER NOT NULL,
		finished_at INTEGER NOT NULL DEFAULT 0
	)`
	createRunQueueStatusIndexSQL    = `CREATE INDEX IF NOT EXISTS idx_run_queue_status_created ON run_queue(status, created_at)`
	createRunQueueHeartbeatIndexSQL = `CREATE INDEX IF NOT EXISTS idx_run_queue_heartbeat ON run_queue(status, heartbeat_at)`
)
