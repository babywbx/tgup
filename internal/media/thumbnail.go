package media

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Thumbnailer extracts thumbnails for videos.
type Thumbnailer interface {
	ExtractVideoThumbnail(ctx context.Context, videoPath string) (thumbnailPath string, cleanup func(), err error)
}

// FFMpegThumbnailer extracts JPEG thumbnails via ffmpeg CLI.
type FFMpegThumbnailer struct{}

// ExtractVideoThumbnail generates a 320px-wide JPEG thumbnail.
// Tries 1s seek first, falls back to 0s for very short videos.
func (FFMpegThumbnailer) ExtractVideoThumbnail(ctx context.Context, videoPath string) (string, func(), error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return "", nil, fmt.Errorf("ffmpeg_missing: ffmpeg not found in PATH")
	}

	tmp, err := os.CreateTemp("", "tgup-thumb-*.jpg")
	if err != nil {
		return "", nil, fmt.Errorf("ffmpeg_error: create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()

	cleanupFn := func() { _ = os.Remove(tmpPath) }

	// Try at 1 second, then 0 seconds.
	for _, seek := range []string{"00:00:01.000", "00:00:00.000"} {
		args := []string{
			"-y", "-loglevel", "error",
			"-ss", seek,
			"-i", videoPath,
			"-frames:v", "1",
			"-vf", "scale='min(320,iw)':-1",
			tmpPath,
		}

		cmd := exec.CommandContext(ctx, "ffmpeg", args...)
		if _, runErr := cmd.CombinedOutput(); runErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				cleanupFn()
				return "", nil, ctxErr
			}
			continue
		}

		info, statErr := os.Stat(tmpPath)
		if statErr == nil && info.Size() > 0 {
			return tmpPath, cleanupFn, nil
		}
	}

	cleanupFn()
	return "", nil, fmt.Errorf("ffmpeg_empty_thumbnail: could not extract thumbnail from %s", videoPath)
}

// CheckFFMpeg returns nil if ffmpeg is available on PATH.
func CheckFFMpeg() error {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found in PATH")
	}
	return nil
}

// NoopThumbnailer returns no thumbnail (used when video_thumbnail="off").
type NoopThumbnailer struct{}

// ExtractVideoThumbnail always returns empty (thumbnails disabled).
func (NoopThumbnailer) ExtractVideoThumbnail(_ context.Context, _ string) (string, func(), error) {
	return "", func() {}, nil
}

// ValidateVideoThumbnailPolicy checks if the policy string is valid.
func ValidateVideoThumbnailPolicy(policy string) error {
	p := strings.TrimSpace(strings.ToLower(policy))
	if p == "auto" || p == "off" {
		return nil
	}
	// Also accept file paths.
	if p != "" {
		return nil
	}
	return fmt.Errorf("invalid video_thumbnail policy: %q", policy)
}
