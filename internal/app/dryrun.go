package app

import (
	"fmt"
	"io"

	"github.com/babywbx/tgup/internal/artifacts"
	"github.com/babywbx/tgup/internal/config"
	"github.com/babywbx/tgup/internal/plan"
	"github.com/babywbx/tgup/internal/scan"
)

// DryRunResult captures deterministic dry-run output data.
type DryRunResult struct {
	Config config.Config
	Items  []scan.Item
	Plan   plan.Plan
}

// ExecuteDryRun resolves configuration and builds a scan/plan result.
func ExecuteDryRun(configPath string, cliOverlay config.Overlay) (DryRunResult, error) {
	cfg, err := config.Resolve(configPath, cliOverlay)
	if err != nil {
		return DryRunResult{}, err
	}

	items, err := scan.Discover(scan.Options{
		Src:            cfg.Scan.Src,
		Recursive:      cfg.Scan.Recursive,
		FollowSymlinks: cfg.Scan.FollowSymlinks,
		IncludeExt:     cfg.Scan.IncludeExt,
		ExcludeExt:     cfg.Scan.ExcludeExt,
	})
	if err != nil {
		return DryRunResult{}, fmt.Errorf("scan: %w", err)
	}

	pl := plan.Build(items, plan.Options{
		Order:    cfg.Plan.Order,
		Reverse:  cfg.Plan.Reverse,
		AlbumMax: cfg.Plan.AlbumMax,
	})

	return DryRunResult{
		Config: cfg,
		Items:  items,
		Plan:   pl,
	}, nil
}

// RenderOptions controls textual preview length.
type RenderOptions struct {
	MaxAlbums        int
	MaxItemsPerAlbum int
}

// WriteDryRun writes a stable text summary for dry-run output.
func WriteDryRun(w io.Writer, result DryRunResult, opts RenderOptions) error {
	maxAlbums := opts.MaxAlbums
	if maxAlbums <= 0 {
		maxAlbums = 5
	}
	maxItems := opts.MaxItemsPerAlbum
	if maxItems <= 0 {
		maxItems = 3
	}

	if _, err := fmt.Fprintf(w,
		"dry-run summary\nsources: %d\nitems: %d\nalbums: %d\norder: %s\nreverse: %t\nalbum_max: %d\n",
		len(result.Config.Scan.Src),
		len(result.Items),
		len(result.Plan.Albums),
		result.Config.Plan.Order,
		result.Config.Plan.Reverse,
		result.Config.Plan.AlbumMax,
	); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "\npreview"); err != nil {
		return err
	}
	for i, album := range result.Plan.Albums {
		if i >= maxAlbums {
			if _, err := fmt.Fprintf(w, "... (%d more albums)\n", len(result.Plan.Albums)-i); err != nil {
				return err
			}
			break
		}
		if _, err := fmt.Fprintf(w, "%d. %s [%d]\n", i+1, album.Label, len(album.Items)); err != nil {
			return err
		}
		for j, item := range album.Items {
			if j >= maxItems {
				if _, err := fmt.Fprintf(w, "   ... (%d more items)\n", len(album.Items)-j); err != nil {
					return err
				}
				break
			}
			if _, err := fmt.Fprintf(w, "   - %s\n", item.Path); err != nil {
				return err
			}
		}
	}
	return nil
}

// BuildDryRunSummary creates a serializable report model.
func BuildDryRunSummary(result DryRunResult, opts RenderOptions) artifacts.DryRunSummary {
	maxAlbums := opts.MaxAlbums
	if maxAlbums <= 0 {
		maxAlbums = 5
	}
	maxItems := opts.MaxItemsPerAlbum
	if maxItems <= 0 {
		maxItems = 3
	}

	preview := make([]artifacts.AlbumPreview, 0, minInt(maxAlbums, len(result.Plan.Albums)))
	for i, album := range result.Plan.Albums {
		if i >= maxAlbums {
			break
		}
		items := make([]string, 0, minInt(maxItems, len(album.Items)))
		for j, item := range album.Items {
			if j >= maxItems {
				break
			}
			items = append(items, item.Path)
		}
		preview = append(preview, artifacts.AlbumPreview{
			Label: album.Label,
			Count: len(album.Items),
			Items: items,
		})
	}

	sources := make([]string, len(result.Config.Scan.Src))
	copy(sources, result.Config.Scan.Src)

	return artifacts.DryRunSummary{
		Sources:  sources,
		Items:    len(result.Items),
		Albums:   len(result.Plan.Albums),
		Order:    result.Config.Plan.Order,
		Reverse:  result.Config.Plan.Reverse,
		AlbumMax: result.Config.Plan.AlbumMax,
		Preview:  preview,
	}
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
