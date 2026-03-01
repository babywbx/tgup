package mcp

import "context"

// EventStore persists and streams job events.
type EventStore interface {
	Append(ctx context.Context, event Event) (Event, error)
	List(ctx context.Context, jobID string, sinceSeq int64, limit int) (events []Event, hasMore bool, err error)
	Register(jobID string) Subscription
	Cleanup(ctx context.Context) error
	Close() error
}

// Subscription receives event stream updates.
type Subscription interface {
	C() <-chan Event
	Close()
}
