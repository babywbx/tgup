package files

import (
	"fmt"
	"path/filepath"
	"time"
)

// DeriveForcePath returns an isolated path using ".force.<pid>.<ts>" suffix.
func DeriveForcePath(basePath string, pid int, now time.Time) string {
	basePath = filepath.Clean(basePath)
	return fmt.Sprintf("%s.force.%d.%d", basePath, pid, now.Unix())
}
