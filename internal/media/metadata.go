package media

import "context"

// MetadataProber provides media metadata lookup.
type MetadataProber interface {
	ProbeVideo(ctx context.Context, path string) (VideoMetadata, error)
}

// VideoMetadata is app-owned normalized video metadata.
type VideoMetadata struct {
	DurationSeconds float64
	Width           int
	Height          int
}
