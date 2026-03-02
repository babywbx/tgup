package config

import (
	"errors"
	"fmt"
	"strings"
)

var (
	errInvalidOrder      = errors.New("must be one of: name, mtime, size, random")
	errInvalidParseMode  = errors.New("must be one of: plain, md")
	errInvalidImageMode  = errors.New("must be one of: auto, photo, document, compress")
	errInvalidDuplicate  = errors.New("must be one of: skip, ask, upload")
	errInvalidThumbnail  = errors.New("must be auto, off, or a file path")
	errInvalidPort       = errors.New("must be in range 1..65535 when MCP is enabled")
	errInvalidAlbumMax   = errors.New("must be > 0")
	errInvalidConcurrent = errors.New("must be > 0")
	errMissingSrc        = errors.New("must contain at least one source path")
	errMissingAPIID      = errors.New("must be set (non-zero)")
	errMissingAPIHash    = errors.New("must be set (non-empty)")
)

// ValidationError describes a single invalid field.
type ValidationError struct {
	Field string
	Err   error
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s %s", e.Field, e.Err)
}

// ValidateTelegram checks only Telegram auth configuration fields.
func ValidateTelegram(cfg Config) error {
	errs := make([]error, 0, 2)
	if cfg.Telegram.APIID == 0 {
		errs = append(errs, ValidationError{Field: "telegram.api_id", Err: errMissingAPIID})
	}
	if cfg.Telegram.APIHash == "" {
		errs = append(errs, ValidationError{Field: "telegram.api_hash", Err: errMissingAPIHash})
	}
	return errors.Join(errs...)
}

// Validate checks basic configuration invariants.
func Validate(cfg Config) error {
	errs := make([]error, 0)
	if cfg.Telegram.APIID == 0 {
		errs = append(errs, ValidationError{Field: "telegram.api_id", Err: errMissingAPIID})
	}
	if cfg.Telegram.APIHash == "" {
		errs = append(errs, ValidationError{Field: "telegram.api_hash", Err: errMissingAPIHash})
	}
	if len(cfg.Scan.Src) == 0 {
		errs = append(errs, ValidationError{Field: "scan.src", Err: errMissingSrc})
	}
	if !oneOf(strings.ToLower(cfg.Plan.Order), "name", "mtime", "size", "random") {
		errs = append(errs, ValidationError{Field: "plan.order", Err: errInvalidOrder})
	}
	if cfg.Plan.AlbumMax <= 0 {
		errs = append(errs, ValidationError{Field: "plan.album_max", Err: errInvalidAlbumMax})
	}
	if cfg.Upload.ConcurrencyAlbum <= 0 {
		errs = append(errs, ValidationError{Field: "upload.concurrency_album", Err: errInvalidConcurrent})
	}
	if !oneOf(strings.ToLower(cfg.Upload.ParseMode), "plain", "md") {
		errs = append(errs, ValidationError{Field: "upload.parse_mode", Err: errInvalidParseMode})
	}
	if !oneOf(strings.ToLower(cfg.Upload.ImageMode), "auto", "photo", "document", "compress") {
		errs = append(errs, ValidationError{Field: "upload.image_mode", Err: errInvalidImageMode})
	}
	if strings.TrimSpace(cfg.Upload.VideoThumbnail) == "" {
		errs = append(errs, ValidationError{Field: "upload.video_thumbnail", Err: errInvalidThumbnail})
	}
	if !oneOf(strings.ToLower(cfg.Upload.Duplicate), "skip", "ask", "upload") {
		errs = append(errs, ValidationError{Field: "upload.duplicate", Err: errInvalidDuplicate})
	}
	if cfg.MCP.Enabled && (cfg.MCP.Port <= 0 || cfg.MCP.Port > 65535) {
		errs = append(errs, ValidationError{Field: "mcp.port", Err: errInvalidPort})
	}
	return errors.Join(errs...)
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func normalizeExtensions(exts []string) []string {
	if exts == nil {
		return nil
	}
	out := make([]string, 0, len(exts))
	for _, ext := range exts {
		ext = strings.TrimSpace(strings.ToLower(ext))
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		out = append(out, ext)
	}
	return out
}
