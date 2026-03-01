package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/babywbx/tgup/internal/config"
)

func TestWriteDryRunGolden(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "media", "a.jpg"))
	mustWriteFile(t, filepath.Join(root, "media", "b.mp4"))

	src := []string{filepath.Join(root, "media")}
	apiID := 12345
	result, err := ExecuteDryRun("", config.Overlay{
		Telegram: config.TelegramOverlay{
			APIID:   &apiID,
			APIHash: strPtr("abc123"),
		},
		Scan: config.ScanOverlay{
			Src:       &src,
			Recursive: boolPtr(true),
		},
		Plan: config.PlanOverlay{
			Order:    strPtr("name"),
			AlbumMax: intPtr(10),
		},
	})
	if err != nil {
		t.Fatalf("ExecuteDryRun() error = %v", err)
	}

	var buf bytes.Buffer
	if err := WriteDryRun(&buf, result, RenderOptions{MaxAlbums: 5, MaxItemsPerAlbum: 3}); err != nil {
		t.Fatalf("WriteDryRun() error = %v", err)
	}

	got := strings.ReplaceAll(buf.String(), filepath.ToSlash(root), "<ROOT>")
	got = strings.ReplaceAll(got, root, "<ROOT>")

	want := mustReadGolden(t, "dryrun/basic.txt")
	if got != want {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func mustReadGolden(t *testing.T, rel string) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	path := filepath.Join(cwd, "..", "..", "testdata", "golden", rel)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %q: %v", path, err)
	}
	return string(raw)
}
