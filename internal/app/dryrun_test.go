package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wbx/tgup/internal/config"
)

func TestExecuteDryRunAndWrite(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "media", "a.jpg"))
	mustWriteFile(t, filepath.Join(root, "media", "b.mp4"))
	mustWriteFile(t, filepath.Join(root, "media", "ignored.txt"))

	cfgPath := filepath.Join(root, "tgup.toml")
	cfgBody := `
[telegram]
api_id = 12345
api_hash = "abc123"

[scan]
src = ["media"]
recursive = true
include_ext = ["jpg", "mp4"]

[plan]
order = "name"
album_max = 2
`
	if err := os.WriteFile(cfgPath, []byte(cfgBody), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	result, err := ExecuteDryRun(cfgPath, config.Overlay{})
	if err != nil {
		t.Fatalf("ExecuteDryRun() error = %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if len(result.Plan.Albums) != 1 {
		t.Fatalf("expected 1 album, got %d", len(result.Plan.Albums))
	}

	var buf bytes.Buffer
	if err := WriteDryRun(&buf, result, RenderOptions{MaxAlbums: 5, MaxItemsPerAlbum: 2}); err != nil {
		t.Fatalf("WriteDryRun() error = %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		"dry-run summary",
		"items: 2",
		"albums: 1",
		"preview",
		"a.jpg",
		"b.mp4",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestExecuteDryRunAppliesCLIOverlay(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "base", "a.jpg"))
	mustWriteFile(t, filepath.Join(root, "override", "b.jpg"))

	cfgPath := filepath.Join(root, "tgup.toml")
	cfgBody := `
[telegram]
api_id = 12345
api_hash = "abc123"

[scan]
src = ["base"]
recursive = true
`
	if err := os.WriteFile(cfgPath, []byte(cfgBody), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cliSrc := []string{filepath.Join(root, "override")}
	result, err := ExecuteDryRun(cfgPath, config.Overlay{
		Scan: config.ScanOverlay{Src: &cliSrc},
	})
	if err != nil {
		t.Fatalf("ExecuteDryRun() error = %v", err)
	}
	if len(result.Items) != 1 || !strings.HasSuffix(result.Items[0].Path, "b.jpg") {
		t.Fatalf("expected CLI src override to select b.jpg, got %#v", result.Items)
	}
}

func TestBuildDryRunSummary(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "media", "a.jpg"))
	mustWriteFile(t, filepath.Join(root, "media", "b.jpg"))
	mustWriteFile(t, filepath.Join(root, "media", "c.jpg"))

	apiID := 12345
	result, err := ExecuteDryRun("", config.Overlay{
		Telegram: config.TelegramOverlay{
			APIID:   &apiID,
			APIHash: strPtr("abc123"),
		},
		Scan: config.ScanOverlay{
			Src:       &[]string{filepath.Join(root, "media")},
			Recursive: boolPtr(true),
		},
		Plan: config.PlanOverlay{
			Order:    strPtr("name"),
			AlbumMax: intPtr(2),
		},
	})
	if err != nil {
		t.Fatalf("ExecuteDryRun() error = %v", err)
	}

	summary := BuildDryRunSummary(result, RenderOptions{
		MaxAlbums:        1,
		MaxItemsPerAlbum: 1,
	})
	if summary.Items != 3 || summary.Albums != 2 {
		t.Fatalf("unexpected summary counts: %#v", summary)
	}
	if len(summary.Preview) != 1 || len(summary.Preview[0].Items) != 1 {
		t.Fatalf("unexpected preview: %#v", summary.Preview)
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func boolPtr(v bool) *bool { return &v }

func intPtr(v int) *int { return &v }

func strPtr(v string) *string { return &v }
