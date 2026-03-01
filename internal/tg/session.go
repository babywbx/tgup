package tg

import "context"

// SessionStore persists Telegram session blobs.
type SessionStore interface {
	Load(ctx context.Context) ([]byte, error)
	Save(ctx context.Context, data []byte) error
}
