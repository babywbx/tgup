package media

import "context"

// ChainProber tries the primary prober first, falling back to the secondary.
type ChainProber struct {
	primary  MetadataProber
	fallback MetadataProber
}

// NewChainProber returns a prober that tries primary first, then fallback.
func NewChainProber(primary, fallback MetadataProber) ChainProber {
	return ChainProber{primary: primary, fallback: fallback}
}

// ProbeVideo tries the primary prober; on failure, tries the fallback.
func (c ChainProber) ProbeVideo(ctx context.Context, path string) (VideoMetadata, error) {
	meta, err := c.primary.ProbeVideo(ctx, path)
	if err == nil {
		return meta, nil
	}
	return c.fallback.ProbeVideo(ctx, path)
}
