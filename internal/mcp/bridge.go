package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/babywbx/tgup/internal/config"
	"github.com/babywbx/tgup/internal/plan"
	"github.com/babywbx/tgup/internal/scan"
)

// Bridge connects MCP tool handlers to the core tgup pipeline.
type Bridge struct {
	AllowRoots []string
}

// NewBridge creates a service bridge.
func NewBridge(allowRoots []string) *Bridge {
	return &Bridge{AllowRoots: allowRoots}
}

// DryRun executes a scan+plan preview without uploading.
func (b *Bridge) DryRun(ctx context.Context, input DryRunInput) (DryRunOutput, error) {
	spec := input.RunSpec

	// Build config overlay from RunSpec.
	overlay := specToOverlay(spec)

	cfg, err := config.Resolve(spec.ConfigPath, overlay)
	if err != nil {
		return DryRunOutput{}, fmt.Errorf("resolve config: %w", err)
	}

	// Validate RESOLVED paths (after config merging) against allow roots.
	if err := b.validatePaths(cfg.Scan.Src); err != nil {
		return DryRunOutput{}, err
	}

	items, err := scan.Discover(scan.Options{
		Src:            cfg.Scan.Src,
		Recursive:      cfg.Scan.Recursive,
		FollowSymlinks: cfg.Scan.FollowSymlinks,
		IncludeExt:     cfg.Scan.IncludeExt,
		ExcludeExt:     cfg.Scan.ExcludeExt,
	})
	if err != nil {
		return DryRunOutput{}, fmt.Errorf("scan: %w", err)
	}

	pl := plan.Build(items, plan.Options{
		Order:    cfg.Plan.Order,
		Reverse:  cfg.Plan.Reverse,
		AlbumMax: cfg.Plan.AlbumMax,
	})

	// Build preview.
	maxAlbums := 20
	maxItems := 0
	if input.ShowFiles {
		maxItems = 10
	}

	preview := make([]AlbumPreview, 0, min(maxAlbums, len(pl.Albums)))
	for i, album := range pl.Albums {
		if i >= maxAlbums {
			break
		}
		ap := AlbumPreview{
			Label: album.Label,
			Count: len(album.Items),
		}
		if maxItems > 0 {
			for j, item := range album.Items {
				if j >= maxItems {
					break
				}
				ap.Items = append(ap.Items, item.Path)
			}
		}
		preview = append(preview, ap)
	}

	return DryRunOutput{
		Sources:  cfg.Scan.Src,
		Items:    len(items),
		Albums:   len(pl.Albums),
		Order:    cfg.Plan.Order,
		Reverse:  cfg.Plan.Reverse,
		AlbumMax: cfg.Plan.AlbumMax,
		Preview:  preview,
	}, nil
}

// RunJob executes a full upload pipeline, emitting events.
// This is called by the job manager runner.
func (b *Bridge) RunJob(ctx context.Context, spec RunSpec, emit func(Event)) (*RunResult, error) {
	overlay := specToOverlay(spec)

	cfg, err := config.Resolve(spec.ConfigPath, overlay)
	if err != nil {
		return nil, fmt.Errorf("resolve config: %w", err)
	}

	// Validate RESOLVED paths against allow roots.
	if err := b.validatePaths(cfg.Scan.Src); err != nil {
		return nil, err
	}

	// Emit config resolved event.
	emitJSON(emit, "config.resolved", map[string]any{
		"sources": cfg.Scan.Src,
		"target":  cfg.Upload.Target,
	})

	// Scan.
	items, err := scan.Discover(scan.Options{
		Src:            cfg.Scan.Src,
		Recursive:      cfg.Scan.Recursive,
		FollowSymlinks: cfg.Scan.FollowSymlinks,
		IncludeExt:     cfg.Scan.IncludeExt,
		ExcludeExt:     cfg.Scan.ExcludeExt,
	})
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}

	emitJSON(emit, "scan.done", map[string]any{
		"items": len(items),
	})

	if len(items) == 0 {
		return &RunResult{Total: 0}, nil
	}

	// Plan.
	pl := plan.Build(items, plan.Options{
		Order:    cfg.Plan.Order,
		Reverse:  cfg.Plan.Reverse,
		AlbumMax: cfg.Plan.AlbumMax,
	})

	emitJSON(emit, "plan.done", map[string]any{
		"albums": len(pl.Albums),
		"items":  len(items),
	})

	// The full upload pipeline requires Telegram connection and state store.
	// For MCP, we delegate to app.RunUpload via a wrapper.
	// Since we can't easily wire the full pipeline here without circular deps,
	// we return the plan result and let the server layer handle the upload.
	// This keeps the bridge focused on scan+plan orchestration.

	// For now, mark as not implemented for the upload portion.
	// The dry-run path is fully functional.
	return nil, fmt.Errorf("full upload via MCP not yet wired (scan+plan complete: %d albums, %d items)",
		len(pl.Albums), len(items))
}

func (b *Bridge) validatePaths(paths []string) error {
	for _, p := range paths {
		allowed := false
		for _, root := range b.AllowRoots {
			if err := ValidatePathInRoot(root, p); err == nil {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("path %q not within allowed roots", p)
		}
	}
	return nil
}

// specToOverlay converts a RunSpec to a config.Overlay.
func specToOverlay(spec RunSpec) config.Overlay {
	var ov config.Overlay

	if len(spec.Src) > 0 {
		src := make([]string, len(spec.Src))
		copy(src, spec.Src)
		ov.Scan.Src = &src
	}
	if spec.Recursive != nil {
		ov.Scan.Recursive = spec.Recursive
	}
	if spec.FollowSymlinks != nil {
		ov.Scan.FollowSymlinks = spec.FollowSymlinks
	}
	if len(spec.IncludeExt) > 0 {
		ext := make([]string, len(spec.IncludeExt))
		copy(ext, spec.IncludeExt)
		ov.Scan.IncludeExt = &ext
	}
	if len(spec.ExcludeExt) > 0 {
		ext := make([]string, len(spec.ExcludeExt))
		copy(ext, spec.ExcludeExt)
		ov.Scan.ExcludeExt = &ext
	}
	if spec.Order != "" {
		ov.Plan.Order = &spec.Order
	}
	if spec.Reverse != nil {
		ov.Plan.Reverse = spec.Reverse
	}
	if spec.AlbumMax != nil {
		ov.Plan.AlbumMax = spec.AlbumMax
	}
	if spec.Target != "" {
		ov.Upload.Target = &spec.Target
	}
	if spec.Caption != "" {
		ov.Upload.Caption = &spec.Caption
	}
	if spec.ParseMode != "" {
		ov.Upload.ParseMode = &spec.ParseMode
	}
	if spec.Concurrency != nil {
		ov.Upload.ConcurrencyAlbum = spec.Concurrency
	}
	if spec.Resume != nil {
		ov.Upload.Resume = spec.Resume
	}
	if spec.StrictMetadata != nil {
		ov.Upload.StrictMetadata = spec.StrictMetadata
	}
	if spec.ImageMode != "" {
		ov.Upload.ImageMode = &spec.ImageMode
	}
	if spec.VideoThumbnail != "" {
		ov.Upload.VideoThumbnail = &spec.VideoThumbnail
	}
	if spec.Duplicate != "" {
		ov.Upload.Duplicate = &spec.Duplicate
	}

	return ov
}

func emitJSON(emit func(Event), eventType string, data any) {
	payload, _ := json.Marshal(data)
	emit(Event{
		Type:    eventType,
		Payload: json.RawMessage(payload),
	})
}
