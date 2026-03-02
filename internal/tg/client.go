package tg

import "context"

// Client groups app-owned Telegram capabilities.
type Client interface {
	Transport
	Close(ctx context.Context) error
}
