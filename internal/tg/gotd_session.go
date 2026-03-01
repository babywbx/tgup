package tg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gotd/td/session"
)

// FileSessionStore persists Telegram session data to a file on disk.
// Implements both our SessionStore interface and gotd's session.Storage.
type FileSessionStore struct {
	Path string
}

// Load reads session data from disk (our interface).
func (s *FileSessionStore) Load() ([]byte, error) {
	data, err := os.ReadFile(s.Path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

// Save writes session data to disk (our interface).
func (s *FileSessionStore) Save(data []byte) error {
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}
	return os.WriteFile(s.Path, data, 0o600)
}

// LoadSession implements gotd session.Storage.
func (s *FileSessionStore) LoadSession(_ context.Context) ([]byte, error) {
	data, err := s.Load()
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, session.ErrNotFound
	}
	return data, nil
}

// StoreSession implements gotd session.Storage.
func (s *FileSessionStore) StoreSession(_ context.Context, data []byte) error {
	return s.Save(data)
}
