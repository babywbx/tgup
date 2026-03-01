package tg

import "context"

// ProgressFunc reports sent bytes progress.
type ProgressFunc func(sentBytes int64, totalBytes int64)

// Transport defines app-owned upload transport behavior.
type Transport interface {
	ResolveTarget(ctx context.Context, target string) (ResolvedTarget, error)
	SendSingle(ctx context.Context, req SendSingleRequest) (SendResult, error)
	SendAlbum(ctx context.Context, req SendAlbumRequest) (SendResult, error)
}

// VideoMeta carries video dimensions and duration for upload attributes.
type VideoMeta struct {
	Duration float64
	Width    int
	Height   int
}

// SendSingleRequest describes one media send.
type SendSingleRequest struct {
	Target            ResolvedTarget
	Path              string
	Caption           string
	ParseMode         string
	ForceDocument     bool
	SupportsStreaming bool
	ThumbnailPath     string
	Video             *VideoMeta
	Progress          ProgressFunc
}

// SendAlbumRequest describes grouped media send.
type SendAlbumRequest struct {
	Target    ResolvedTarget
	Items     []AlbumMedia
	ParseMode string
	Progress  ProgressFunc
}

// AlbumMedia is one media item in an album request.
type AlbumMedia struct {
	Path              string
	Caption           string
	ForceDocument     bool
	SupportsStreaming bool
	ThumbnailPath     string
	Video             *VideoMeta
}

// SendResult is the normalized transport response.
type SendResult struct {
	MessageIDs []int
	GroupID    string
	Messages   []SentMessage
}
