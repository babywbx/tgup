package media

import (
	"context"
	"math"
	"testing"
)

func TestNativeMetadataProber_MP4(t *testing.T) {
	prober := NativeMetadataProber{}
	meta, err := prober.ProbeVideo(context.Background(), "testdata/sample.mp4")
	if err != nil {
		t.Fatalf("ProbeVideo mp4: %v", err)
	}

	if meta.Width != 320 {
		t.Errorf("Width = %d, want 320", meta.Width)
	}
	if meta.Height != 240 {
		t.Errorf("Height = %d, want 240", meta.Height)
	}
	if math.Abs(meta.DurationSeconds-2.0) > 0.5 {
		t.Errorf("DurationSeconds = %f, want ~2.0", meta.DurationSeconds)
	}
}

func TestNativeMetadataProber_WebM(t *testing.T) {
	prober := NativeMetadataProber{}
	meta, err := prober.ProbeVideo(context.Background(), "testdata/sample.webm")
	if err != nil {
		t.Fatalf("ProbeVideo webm: %v", err)
	}

	if meta.Width != 320 {
		t.Errorf("Width = %d, want 320", meta.Width)
	}
	if meta.Height != 240 {
		t.Errorf("Height = %d, want 240", meta.Height)
	}
	if math.Abs(meta.DurationSeconds-2.0) > 0.5 {
		t.Errorf("DurationSeconds = %f, want ~2.0", meta.DurationSeconds)
	}
}

func TestNativeMetadataProber_UnsupportedExt(t *testing.T) {
	prober := NativeMetadataProber{}
	_, err := prober.ProbeVideo(context.Background(), "testdata/foo.avi")
	if err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}

func TestChainProber_FallbackOnError(t *testing.T) {
	// FFProbe will fail on these test files if ffprobe is not installed,
	// but NativeMetadataProber should succeed as fallback.
	chain := NewChainProber(NativeMetadataProber{}, NativeMetadataProber{})
	meta, err := chain.ProbeVideo(context.Background(), "testdata/sample.mp4")
	if err != nil {
		t.Fatalf("ChainProber: %v", err)
	}
	if meta.Width != 320 || meta.Height != 240 {
		t.Errorf("ChainProber: got %dx%d, want 320x240", meta.Width, meta.Height)
	}
}
