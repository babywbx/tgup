package state

import "time"

// ResumeKey identifies a media item in resume state.
// Identity semantics: absolute resolved path + size + mtime_ns.
type ResumeKey struct {
	Path    string
	Size    int64
	MTimeNS int64
}

// UploadRow represents one persisted upload state row.
type UploadRow struct {
	ResumeKey
	Status        string
	Target        string
	ErrorReason   string
	MessageIDs    []int
	AlbumGroupID  string
	CreatedAtUnix int64
	UpdatedAtUnix int64
}

// MaintenanceConfig controls cleanup behavior.
type MaintenanceConfig struct {
	Enabled      bool
	MaxAge       time.Duration
	KeepFailed   bool
	PreviewLimit int
}

// CleanupReport summarizes state cleanup effects.
type CleanupReport struct {
	Preview          bool
	DeletedUploads   int
	DeletedQueueRows int
	Vacuumed         bool
}

// MarkSentInput marks an item as sent.
type MarkSentInput struct {
	Key          ResumeKey
	Target       string
	MessageIDs   []int
	AlbumGroupID string
}

// MarkFailedInput marks an item as failed.
type MarkFailedInput struct {
	Key         ResumeKey
	Target      string
	ErrorReason string
}
