package upload

import (
	"context"
	"fmt"

	"github.com/babywbx/tgup/internal/media"
)

// PrecheckResult holds precheck outcomes for one album.
type PrecheckResult struct {
	Violations []FileViolation
	Warnings   []string
}

// FileViolation pairs a file path with a violation reason.
type FileViolation struct {
	Path   string
	Reason string
}

// HasViolations returns true if any file failed validation.
func (r PrecheckResult) HasViolations() bool {
	return len(r.Violations) > 0
}

// PrecheckAlbum probes metadata for each video in the album.
// Returns violations (invalid metadata) and warnings (probe issues).
func PrecheckAlbum(ctx context.Context, prober media.MetadataProber, files []AlbumFile) PrecheckResult {
	var result PrecheckResult
	if prober == nil {
		for _, f := range files {
			if f.Kind == media.KindVideo {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: metadata prober unavailable", f.Path))
			}
		}
		return result
	}

	for i := range files {
		f := &files[i]
		if f.Kind != media.KindVideo {
			continue
		}

		meta, err := prober.ProbeVideo(ctx, f.Path)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", f.Path, err))
			continue
		}

		if reason := media.ViolationReason(meta); reason != "" {
			result.Violations = append(result.Violations, FileViolation{
				Path:   f.Path,
				Reason: reason,
			})
		}

		f.Metadata = &meta
	}
	return result
}

// ValidateDuplicatePolicy validates duplicate policy values.
func ValidateDuplicatePolicy(policy string) error {
	switch DuplicatePolicy(policy) {
	case DuplicateSkip, DuplicateAsk, DuplicateUpload:
		return nil
	default:
		return fmt.Errorf("invalid duplicate policy: %s", policy)
	}
}
