package files

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// ErrPathEscapesRoot indicates that a target path is outside the allowed root.
var ErrPathEscapesRoot = errors.New("path escapes allowed root")

// EnsureWithinRoot validates that target path is within root after normalization.
func EnsureWithinRoot(root string, target string) error {
	if strings.TrimSpace(root) == "" {
		return fmt.Errorf("root path is required")
	}
	rootAbs, err := AbsClean(root)
	if err != nil {
		return err
	}
	targetAbs, err := AbsClean(target)
	if err != nil {
		return err
	}

	rootEval := rootAbs
	if resolved, err := filepath.EvalSymlinks(rootAbs); err == nil {
		rootEval = filepath.Clean(resolved)
	}
	targetEval := targetAbs
	if resolved, err := filepath.EvalSymlinks(targetAbs); err == nil {
		targetEval = filepath.Clean(resolved)
	}

	rel, err := filepath.Rel(rootEval, targetEval)
	if err != nil {
		return fmt.Errorf("compare paths: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ErrPathEscapesRoot
	}
	return nil
}
