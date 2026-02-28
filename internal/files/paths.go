package files

import (
	"fmt"
	"path/filepath"
)

// ResolveAgainst resolves path against baseDir when it is not absolute.
func ResolveAgainst(path string, baseDir string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(baseDir, path))
}

// AbsClean resolves path to an absolute cleaned path.
func AbsClean(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}
	return filepath.Clean(abs), nil
}
