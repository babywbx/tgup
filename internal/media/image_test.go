package media

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func TestNeedsCompress(t *testing.T) {
	tests := []struct {
		w, h int
		size int64
		want bool
	}{
		{4032, 3024, 5_000_000, false}, // phone photo, small file
		{6240, 4160, 8_000_000, true},  // DSLR: w+h=10400 > 10000
		{5000, 5001, 1_000_000, true},  // w+h=10001 > 10000
		{5000, 5000, 1_000_000, false}, // w+h=10000, exactly at limit
		{3000, 2000, 11_000_000, true}, // small dims, big file
		{1920, 1080, 500_000, false},   // normal screenshot
	}
	for _, tt := range tests {
		got := needsCompress(tt.w, tt.h, tt.size, detectSumWH)
		if got != tt.want {
			t.Errorf("needsCompress(%d, %d, %d, %d) = %v, want %v", tt.w, tt.h, tt.size, detectSumWH, got, tt.want)
		}
	}
}

func TestFitSumWH(t *testing.T) {
	tests := []struct {
		w, h, maxSum int
		wantW, wantH int
	}{
		{6240, 4160, 10000, 6000, 4000},   // DSLR photo
		{5000, 5000, 10000, 5000, 5000},   // exactly at limit
		{4032, 3024, 10000, 4032, 3024},   // within limit, no change
		{10000, 10000, 10000, 5000, 5000}, // extreme case
		{20000, 1, 10000, 9999, 0},        // extreme aspect — clamped to 1
	}
	for _, tt := range tests {
		gw, gh := fitSumWH(tt.w, tt.h, tt.maxSum)
		if gw+gh > tt.maxSum {
			t.Errorf("fitSumWH(%d, %d, %d) = (%d, %d), sum %d > %d",
				tt.w, tt.h, tt.maxSum, gw, gh, gw+gh, tt.maxSum)
		}
		// Allow ±1 for rounding.
		if abs(gw-tt.wantW) > 1 || abs(gh-tt.wantH) > 1 {
			t.Errorf("fitSumWH(%d, %d, %d) = (%d, %d), want ~(%d, %d)",
				tt.w, tt.h, tt.maxSum, gw, gh, tt.wantW, tt.wantH)
		}
	}
}

func TestFitLongEdge(t *testing.T) {
	tests := []struct {
		w, h, maxEdge int
		wantW, wantH  int
	}{
		{6240, 4160, 2560, 2560, 1706},
		{4160, 6240, 2560, 1706, 2560},
		{2560, 1440, 2560, 2560, 1440}, // already within limit
		{1920, 1080, 2560, 1920, 1080}, // already within limit
	}
	for _, tt := range tests {
		gw, gh := fitLongEdge(tt.w, tt.h, tt.maxEdge)
		if gw > tt.maxEdge || gh > tt.maxEdge {
			t.Errorf("fitLongEdge(%d, %d, %d) = (%d, %d), exceeds max",
				tt.w, tt.h, tt.maxEdge, gw, gh)
		}
		if abs(gw-tt.wantW) > 1 || abs(gh-tt.wantH) > 1 {
			t.Errorf("fitLongEdge(%d, %d, %d) = (%d, %d), want ~(%d, %d)",
				tt.w, tt.h, tt.maxEdge, gw, gh, tt.wantW, tt.wantH)
		}
	}
}

func TestAspectRatio(t *testing.T) {
	tests := []struct {
		w, h int
		want float64
	}{
		{1920, 1080, 1.777},
		{1080, 1920, 1.777},
		{5000, 5000, 1.0},
		{20000, 1000, 20.0},
	}
	for _, tt := range tests {
		got := aspectRatio(tt.w, tt.h)
		if abs64(got-tt.want) > 0.01 {
			t.Errorf("aspectRatio(%d, %d) = %.3f, want ~%.3f", tt.w, tt.h, got, tt.want)
		}
	}
}

func TestCompressPhoto_OversizedPNG(t *testing.T) {
	// Create a 6000×5000 PNG (w+h=11000 > 10000).
	img := image.NewRGBA(image.Rect(0, 0, 6000, 5000))
	for y := 0; y < 5000; y++ {
		for x := 0; x < 6000; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}

	tmp, err := os.CreateTemp("", "tgup-test-*.png")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	if err := png.Encode(tmp, img); err != nil {
		tmp.Close()
		t.Fatal(err)
	}
	tmp.Close()

	compressed, cleanup, err := CompressPhoto(tmp.Name())
	if err != nil {
		t.Fatalf("CompressPhoto: %v", err)
	}
	if compressed == "" {
		t.Fatal("expected compression, got empty path")
	}
	defer cleanup()

	// Verify output dimensions.
	outF, err := os.Open(compressed)
	if err != nil {
		t.Fatal(err)
	}
	outCfg, _, err := image.DecodeConfig(outF)
	outF.Close()
	if err != nil {
		t.Fatal(err)
	}

	if outCfg.Width+outCfg.Height > targetSumWH {
		t.Errorf("output w+h = %d, exceeds %d", outCfg.Width+outCfg.Height, targetSumWH)
	}

	// Verify output file size.
	info, err := os.Stat(compressed)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() > maxPhotoBytes {
		t.Errorf("output size = %d, exceeds %d", info.Size(), maxPhotoBytes)
	}

	t.Logf("input: 6000×5000 PNG → output: %d×%d JPEG, %d bytes",
		outCfg.Width, outCfg.Height, info.Size())
}

func TestCompressPhoto_WithinLimits(t *testing.T) {
	// 200×100 PNG, well within limits — should not compress.
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	tmp, err := os.CreateTemp("", "tgup-test-small-*.png")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	if err := png.Encode(tmp, img); err != nil {
		tmp.Close()
		t.Fatal(err)
	}
	tmp.Close()

	compressed, _, err := CompressPhoto(tmp.Name())
	if err != nil {
		t.Fatalf("CompressPhoto: %v", err)
	}
	if compressed != "" {
		t.Errorf("expected no compression for small image, got %s", compressed)
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func abs64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
