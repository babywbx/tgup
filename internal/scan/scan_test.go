package scan

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func TestDiscoverRespectsRecursiveAndFilters(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "a.jpg"))
	mustWriteFile(t, filepath.Join(root, "b.mp4"))
	mustWriteFile(t, filepath.Join(root, "c.txt"))
	mustWriteFile(t, filepath.Join(root, "sub", "d.png"))
	mustWriteFile(t, filepath.Join(root, "sub", "e.mov"))

	items, err := Discover(Options{
		Src:       []string{root},
		Recursive: false,
	})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if got := collectBaseNames(items); !slices.Equal(got, []string{"a.jpg", "b.mp4"}) {
		t.Fatalf("unexpected non-recursive result: %#v", got)
	}

	items, err = Discover(Options{
		Src:        []string{root},
		Recursive:  true,
		IncludeExt: []string{"jpg", "png", "mov"},
		ExcludeExt: []string{".mov"},
	})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if got := collectBaseNames(items); !slices.Equal(got, []string{"a.jpg", "d.png"}) {
		t.Fatalf("unexpected recursive/filter result: %#v", got)
	}
}

func TestDiscoverFollowSymlinkAndDedup(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior is environment-dependent on windows")
	}

	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	mustWriteFile(t, filepath.Join(realDir, "x.jpg"))

	linkDir := filepath.Join(root, "linkdir")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatalf("create symlink dir: %v", err)
	}

	items, err := Discover(Options{
		Src:            []string{realDir, linkDir},
		Recursive:      true,
		FollowSymlinks: false,
	})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one item with symlink disabled, got %d", len(items))
	}

	items, err = Discover(Options{
		Src:            []string{realDir, linkDir},
		Recursive:      true,
		FollowSymlinks: true,
	})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected deduped item with symlink enabled, got %d", len(items))
	}
}

func TestDiscoverParentDirAndKind(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "album-1", "cover.jpg"))
	mustWriteFile(t, filepath.Join(root, "album-1", "clip.mp4"))

	items, err := Discover(Options{
		Src:       []string{root},
		Recursive: true,
	})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	for _, item := range items {
		if item.ParentDir != "album-1" {
			t.Fatalf("expected parent dir album-1, got %q", item.ParentDir)
		}
		if filepath.Clean(item.SrcRoot) != filepath.Clean(root) {
			t.Fatalf("expected src root %q, got %q", root, item.SrcRoot)
		}
	}
	if items[0].Kind == "" || items[1].Kind == "" {
		t.Fatalf("expected kinds to be detected: %#v", items)
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

func collectBaseNames(items []Item) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, filepath.Base(item.Path))
	}
	return out
}
