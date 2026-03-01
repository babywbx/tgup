package upload

import (
	"github.com/babywbx/tgup/internal/media"
	"github.com/babywbx/tgup/internal/plan"
	"github.com/babywbx/tgup/internal/state"
	"github.com/babywbx/tgup/internal/tg"
)

// AlbumFile is one file within an upload album, bridging scan and upload.
type AlbumFile struct {
	Path      string
	Size      int64
	MTimeNS   int64
	Kind      media.Kind
	Metadata  *media.VideoMetadata // populated during precheck
	Thumbnail string               // populated during thumbnail extraction
}

// Input is the full upload pipeline input.
type Input struct {
	Plan        []plan.Album
	Transport   tg.Transport
	Store       state.Store
	Prober      media.MetadataProber
	Thumbnailer media.Thumbnailer
	Config      Config
	OnProgress  func(Snapshot)
	OnEvent     func(Event)
}

// Config holds upload-specific settings.
type Config struct {
	Target         string
	Caption        string
	ParseMode      string
	Concurrency    int
	StrictMetadata bool
	ImageMode      string
	VideoThumbnail string
	Resume         bool
	Duplicate      DuplicatePolicy
}

// Snapshot reports progress at a point in time.
type Snapshot struct {
	SentBytes    int64
	TotalBytes   int64
	SentFiles    int
	TotalFiles   int
	SentAlbums   int
	TotalAlbums  int
	FailedAlbums int
	CurrentLabel string
}

// Event is emitted during upload for MCP/logging.
type Event struct {
	Type    string
	Album   string
	Files   int
	Error   string
	Details map[string]interface{}
}

// ResumeKeyFromFile creates a state.ResumeKey from an AlbumFile.
func ResumeKeyFromFile(f AlbumFile) state.ResumeKey {
	return state.ResumeKey{
		Path:    f.Path,
		Size:    f.Size,
		MTimeNS: f.MTimeNS,
	}
}
