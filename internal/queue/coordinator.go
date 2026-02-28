package queue

import "context"

// Coordinator defines cross-process run ordering behavior.
type Coordinator interface {
	RunID() string
	WaitUntilTurn(ctx context.Context, onWait func(ahead int)) error
	Heartbeat(ctx context.Context) error
	Finish(ctx context.Context, status string) error
	Cancel(ctx context.Context) error
	Close() error
}
