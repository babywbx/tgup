package mcp

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateBearerToken(t *testing.T) {
	t.Parallel()

	if err := ValidateBearerToken("abc", "abc"); err != nil {
		t.Fatalf("expected token match, got %v", err)
	}
	if err := ValidateBearerToken("", ""); err != nil {
		t.Fatalf("expected empty expected token to allow, got %v", err)
	}
	err := ValidateBearerToken("x", "y")
	if !errors.Is(err, errUnauthorized) {
		t.Fatalf("expected errUnauthorized, got %v", err)
	}
}

func TestValidatePathInRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	inside := filepath.Join(root, "a.txt")
	if err := os.WriteFile(inside, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := ValidatePathInRoot(root, inside); err != nil {
		t.Fatalf("expected inside path allowed, got %v", err)
	}

	outside := filepath.Join(root, "..", "b.txt")
	if err := ValidatePathInRoot(root, outside); err == nil {
		t.Fatal("expected outside path to fail")
	}
}
