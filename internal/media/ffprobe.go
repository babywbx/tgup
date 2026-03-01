package media

import (
	"context"
	"fmt"
)

// FFProbeMetadataProber is a placeholder ffprobe-backed prober.
// Real external-tool integration will be implemented in a later milestone.
type FFProbeMetadataProber struct{}

// ProbeVideo currently returns an explicit not-implemented error.
func (FFProbeMetadataProber) ProbeVideo(ctx context.Context, path string) (VideoMetadata, error) {
	_ = ctx
	_ = path
	return VideoMetadata{}, fmt.Errorf("ffprobe metadata probe is not implemented yet")
}
