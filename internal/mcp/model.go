package mcp

import (
	"encoding/json"
	"time"
)

// Job statuses.
const (
	JobQueued    = "queued"
	JobRunning   = "running"
	JobDone      = "done"
	JobFailed    = "failed"
	JobCancelled = "cancelled"
)

// Job is an MCP job model.
type Job struct {
	ID         string     `json:"id"`
	Status     string     `json:"status"`
	ConfigPath string     `json:"configPath,omitempty"`
	RunSpec    *RunSpec   `json:"runSpec,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	Result     *RunResult `json:"result,omitempty"`
	Error      string     `json:"error,omitempty"`
}

// IsTerminal returns true if the job is in a final state.
func (j Job) IsTerminal() bool {
	return j.Status == JobDone || j.Status == JobFailed || j.Status == JobCancelled
}

// Event is an MCP event model.
type Event struct {
	Seq       int64           `json:"seq"`
	JobID     string          `json:"jobId"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Terminal  bool            `json:"terminal,omitempty"`
	CreatedAt time.Time       `json:"ts"`
}

// RunSpec describes a complete upload run specification from MCP clients.
type RunSpec struct {
	Src            []string `json:"src,omitempty"`
	Recursive      *bool    `json:"recursive,omitempty"`
	FollowSymlinks *bool    `json:"followSymlinks,omitempty"`
	IncludeExt     []string `json:"includeExt,omitempty"`
	ExcludeExt     []string `json:"excludeExt,omitempty"`
	Order          string   `json:"order,omitempty"`
	Reverse        *bool    `json:"reverse,omitempty"`
	AlbumMax       *int     `json:"albumMax,omitempty"`
	Target         string   `json:"target,omitempty"`
	Caption        string   `json:"caption,omitempty"`
	ParseMode      string   `json:"parseMode,omitempty"`
	Concurrency    *int     `json:"concurrency,omitempty"`
	Resume         *bool    `json:"resume,omitempty"`
	StrictMetadata *bool    `json:"strictMetadata,omitempty"`
	ImageMode      string   `json:"imageMode,omitempty"`
	VideoThumbnail string   `json:"videoThumbnail,omitempty"`
	Duplicate      string   `json:"duplicate,omitempty"`
	ConfigPath     string   `json:"configPath,omitempty"`
}

// RunResult captures upload outcome.
type RunResult struct {
	Sent     int  `json:"sent"`
	Failed   int  `json:"failed"`
	Skipped  int  `json:"skipped"`
	Total    int  `json:"total"`
	Canceled bool `json:"canceled,omitempty"`
}

// AlbumPreview is a plan album preview.
type AlbumPreview struct {
	Label string   `json:"label"`
	Count int      `json:"count"`
	Items []string `json:"items,omitempty"`
}

// --- Input types ---

// DryRunInput is the input for tgup.dry_run.
type DryRunInput struct {
	RunSpec   RunSpec `json:"runSpec"`
	ShowFiles bool    `json:"showFiles,omitempty"`
}

// RunStartInput is the input for tgup.run.start.
type RunStartInput struct {
	RunSpec RunSpec `json:"runSpec"`
}

// RunSyncInput is the input for tgup.run.sync.
type RunSyncInput struct {
	RunSpec    RunSpec `json:"runSpec"`
	TimeoutSec int     `json:"timeoutSec,omitempty"`
}

// RunStatusInput is the input for tgup.run.status.
type RunStatusInput struct {
	JobID string `json:"jobId"`
}

// RunCancelInput is the input for tgup.run.cancel.
type RunCancelInput struct {
	JobID string `json:"jobId"`
}

// RunEventsInput is the input for tgup.run.events.
type RunEventsInput struct {
	JobID    string `json:"jobId"`
	SinceSeq int64  `json:"sinceSeq,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// --- Output types ---

// HealthOutput is returned by tgup.health.
type HealthOutput struct {
	Status  string   `json:"status"`
	Version string   `json:"version"`
	Tools   []string `json:"tools"`
}

// DryRunOutput is returned by tgup.dry_run.
type DryRunOutput struct {
	Sources  []string       `json:"sources"`
	Items    int            `json:"items"`
	Albums   int            `json:"albums"`
	Order    string         `json:"order"`
	Reverse  bool           `json:"reverse"`
	AlbumMax int            `json:"albumMax"`
	Preview  []AlbumPreview `json:"preview"`
}

// RunStartOutput is returned by tgup.run.start.
type RunStartOutput struct {
	JobID  string `json:"jobId"`
	Status string `json:"status"`
}

// RunSyncOutput is returned by tgup.run.sync.
type RunSyncOutput struct {
	JobID  string     `json:"jobId"`
	Status string     `json:"status"`
	Result *RunResult `json:"result,omitempty"`
	Error  string     `json:"error,omitempty"`
}

// RunStatusOutput is returned by tgup.run.status.
type RunStatusOutput struct {
	JobID      string     `json:"jobId"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"createdAt"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	Result     *RunResult `json:"result,omitempty"`
	Error      string     `json:"error,omitempty"`
}

// RunCancelOutput is returned by tgup.run.cancel.
type RunCancelOutput struct {
	JobID  string `json:"jobId"`
	Status string `json:"status"`
}

// RunEventsOutput is returned by tgup.run.events.
type RunEventsOutput struct {
	Events  []EventEnvelope `json:"events"`
	HasMore bool            `json:"hasMore"`
}

// EventEnvelope wraps a single event for API responses.
type EventEnvelope struct {
	Seq     int64           `json:"seq"`
	JobID   string          `json:"jobId"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Ts      time.Time       `json:"ts"`
}

// SchemaGetOutput wraps the schema document.
type SchemaGetOutput struct {
	Version string       `json:"version"`
	Tools   []ToolSchema `json:"tools"`
}
