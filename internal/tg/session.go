package tg

// SessionStore persists Telegram session blobs.
type SessionStore interface {
	Load() ([]byte, error)
	Save(data []byte) error
}
