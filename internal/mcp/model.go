package mcp

import "time"

// Job is an app-owned MCP job model.
type Job struct {
	ID        string
	Command   string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Event is an app-owned MCP event model.
type Event struct {
	Seq       int64
	JobID     string
	Type      string
	Payload   []byte
	CreatedAt time.Time
}
