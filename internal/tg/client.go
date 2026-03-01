package tg

import "context"

// Client groups app-owned Telegram capabilities.
type Client interface {
	AuthService
	Transport
	Close(ctx context.Context) error
}
