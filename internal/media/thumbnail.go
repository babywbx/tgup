package media

import "context"

// Thumbnailer extracts thumbnails for videos.
type Thumbnailer interface {
	ExtractVideoThumbnail(ctx context.Context, videoPath string) (thumbnailPath string, cleanup func(), err error)
}
