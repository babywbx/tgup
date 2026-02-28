package files

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestResolveAgainst(t *testing.T) {
	t.Parallel()

	base := filepath.Clean("/tmp/base")
	if got := ResolveAgainst("a/b", base); got != filepath.Clean("/tmp/base/a/b") {
		t.Fatalf("unexpected resolved path: %q", got)
	}
	if got := ResolveAgainst("/var/x", base); got != filepath.Clean("/var/x") {
		t.Fatalf("unexpected absolute path: %q", got)
	}
}

func TestEnsureWithinRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	child := filepath.Join(root, "a", "b.txt")
	if err := os.MkdirAll(filepath.Dir(child), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(child, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := EnsureWithinRoot(root, child); err != nil {
		t.Fatalf("expected child path to be allowed, got %v", err)
	}

	outside := filepath.Join(root, "..", "outside.txt")
	err := EnsureWithinRoot(root, outside)
	if !errors.Is(err, ErrPathEscapesRoot) {
		t.Fatalf("expected ErrPathEscapesRoot, got %v", err)
	}
}

func TestEnsureWithinRootSymlinkEscape(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior is environment-dependent on windows")
	}

	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}

	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	err := EnsureWithinRoot(root, link)
	if !errors.Is(err, ErrPathEscapesRoot) {
		t.Fatalf("expected symlink escape to be blocked, got %v", err)
	}
}

func TestDeriveForcePath(t *testing.T) {
	t.Parallel()

	got := DeriveForcePath("/tmp/state.sqlite", 1234, time.Unix(1700000000, 0))
	want := filepath.Clean("/tmp/state.sqlite.force.1234.1700000000")
	if got != want {
		t.Fatalf("unexpected force path: got %q want %q", got, want)
	}
}
