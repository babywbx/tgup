package media

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gomp4 "github.com/abema/go-mp4"
	"github.com/remko/go-mkvparse"
)

// NativeMetadataProber extracts video metadata using pure Go parsers.
// Supports MP4/MOV (via go-mp4) and MKV/WebM (via go-mkvparse).
type NativeMetadataProber struct{}

// ProbeVideo extracts duration, width, and height from a video file.
func (NativeMetadataProber) ProbeVideo(_ context.Context, path string) (VideoMetadata, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mp4", ".mov":
		return probeMP4(path)
	case ".mkv", ".webm":
		return probeMKV(path)
	default:
		return VideoMetadata{}, fmt.Errorf("native_probe: unsupported format %q", ext)
	}
}

// probeMP4 extracts metadata from MP4/MOV files via moov atom.
func probeMP4(path string) (VideoMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return VideoMetadata{}, fmt.Errorf("native_probe: %w", err)
	}
	defer f.Close()

	info, err := gomp4.Probe(f)
	if err != nil {
		return VideoMetadata{}, fmt.Errorf("native_probe: mp4 probe: %w", err)
	}

	var meta VideoMetadata

	// Duration from mvhd (movie header).
	if info.Timescale > 0 && info.Duration > 0 {
		meta.DurationSeconds = float64(info.Duration) / float64(info.Timescale)
	}

	// Width/height from tkhd (track header) of the first video track.
	if _, err := f.Seek(0, 0); err != nil {
		return VideoMetadata{}, fmt.Errorf("native_probe: seek: %w", err)
	}

	boxes, err := gomp4.ExtractBoxWithPayload(f, nil, gomp4.BoxPath{
		gomp4.BoxTypeMoov(),
		gomp4.BoxTypeTrak(),
		gomp4.BoxTypeTkhd(),
	})
	if err != nil {
		// Return what we have if tkhd extraction fails.
		return meta, nil
	}

	for _, box := range boxes {
		tkhd, ok := box.Payload.(*gomp4.Tkhd)
		if !ok {
			continue
		}
		w := int(tkhd.GetWidthInt())
		h := int(tkhd.GetHeightInt())
		if w > 0 && h > 0 {
			meta.Width = w
			meta.Height = h
			break
		}
	}

	return meta, nil
}

// probeMKV extracts metadata from MKV/WebM files via EBML elements.
func probeMKV(path string) (VideoMetadata, error) {
	h := &mkvHandler{}
	if err := mkvparse.ParsePath(path, h); err != nil {
		return VideoMetadata{}, fmt.Errorf("native_probe: mkv parse: %w", err)
	}
	return h.metadata(), nil
}

// mkvHandler collects video metadata from EBML elements.
type mkvHandler struct {
	mkvparse.DefaultHandler
	timecodeScale int64   // nanoseconds per tick, default 1_000_000
	duration      float64 // in timecodeScale units
	pixelWidth    int
	pixelHeight   int
	inVideo       bool
}

func (h *mkvHandler) HandleMasterBegin(id mkvparse.ElementID, _ mkvparse.ElementInfo) (bool, error) {
	switch id {
	case mkvparse.SegmentElement, mkvparse.InfoElement, mkvparse.TracksElement, mkvparse.TrackEntryElement:
		return true, nil
	case mkvparse.VideoElement:
		h.inVideo = true
		return true, nil
	}
	// Skip everything else (clusters, cues, etc.) for speed.
	return false, nil
}

func (h *mkvHandler) HandleMasterEnd(id mkvparse.ElementID, _ mkvparse.ElementInfo) error {
	if id == mkvparse.VideoElement {
		h.inVideo = false
	}
	return nil
}

func (h *mkvHandler) HandleInteger(id mkvparse.ElementID, value int64, _ mkvparse.ElementInfo) error {
	switch {
	case id == mkvparse.TimecodeScaleElement:
		h.timecodeScale = value
	case id == mkvparse.PixelWidthElement && h.inVideo && h.pixelWidth == 0:
		h.pixelWidth = int(value)
	case id == mkvparse.PixelHeightElement && h.inVideo && h.pixelHeight == 0:
		h.pixelHeight = int(value)
	}
	return nil
}

func (h *mkvHandler) HandleFloat(id mkvparse.ElementID, value float64, _ mkvparse.ElementInfo) error {
	if id == mkvparse.DurationElement {
		h.duration = value
	}
	return nil
}

func (h *mkvHandler) metadata() VideoMetadata {
	scale := h.timecodeScale
	if scale <= 0 {
		scale = 1_000_000 // MKV default: 1ms
	}
	var dur float64
	if h.duration > 0 {
		dur = h.duration * float64(scale) / 1e9
	}
	return VideoMetadata{
		DurationSeconds: dur,
		Width:           h.pixelWidth,
		Height:          h.pixelHeight,
	}
}
