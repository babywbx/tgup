package mcp

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"

	"github.com/babywbx/tgup/internal/files"
)

var errUnauthorized = errors.New("unauthorized")

// ValidateBearerToken checks a bearer token using constant-time compare.
func ValidateBearerToken(provided string, expected string) error {
	if strings.TrimSpace(expected) == "" {
		return nil
	}
	provided = strings.TrimSpace(provided)
	if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
		return errUnauthorized
	}
	return nil
}

// ValidatePathInRoot checks that path is within allowRoot.
func ValidatePathInRoot(allowRoot string, path string) error {
	if err := files.EnsureWithinRoot(allowRoot, path); err != nil {
		return fmt.Errorf("validate path root: %w", err)
	}
	return nil
}
