package progress

import "fmt"

// Snapshot is one progress point.
type Snapshot struct {
	SentBytes  int64
	TotalBytes int64
}

// RenderLine renders a one-line progress string.
func RenderLine(s Snapshot) string {
	if s.TotalBytes <= 0 {
		return fmt.Sprintf("sent=%d", s.SentBytes)
	}
	pct := float64(s.SentBytes) / float64(s.TotalBytes) * 100
	return fmt.Sprintf("sent=%d/%d (%.1f%%)", s.SentBytes, s.TotalBytes, pct)
}
