package config

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type lookupEnvFn func(string) (string, bool)

// LoadEnv reads TGUP_* environment overrides into an Overlay.
func LoadEnv() (Overlay, error) {
	return loadEnv(os.LookupEnv)
}

func loadEnv(lookup lookupEnvFn) (Overlay, error) {
	if err := validateNoConflictingKeys(lookup, "TGUP_SESSION", "TGUP_SESSION_PATH"); err != nil {
		return Overlay{}, err
	}
	if err := validateNoConflictingKeys(lookup, "TGUP_STATE", "TGUP_STATE_PATH"); err != nil {
		return Overlay{}, err
	}

	var ov Overlay
	var err error

	ov.Telegram.APIID, err = envIntAny(lookup, "TGUP_API_ID")
	if err != nil {
		return Overlay{}, err
	}
	ov.Telegram.APIHash = envStringAny(lookup, "TGUP_API_HASH")
	ov.Telegram.SessionPath = envStringAny(lookup, "TGUP_SESSION", "TGUP_SESSION_PATH")

	ov.Paths.StatePath = envStringAny(lookup, "TGUP_STATE", "TGUP_STATE_PATH")
	ov.Paths.ArtifactsDir = envStringAny(lookup, "TGUP_ARTIFACTS_DIR")

	ov.Scan.Src = envCSVAny(lookup, "TGUP_SRC")
	ov.Scan.IncludeExt = envCSVAny(lookup, "TGUP_INCLUDE_EXT")
	ov.Scan.ExcludeExt = envCSVAny(lookup, "TGUP_EXCLUDE_EXT")
	ov.Scan.Recursive, err = envBoolAny(lookup, "TGUP_RECURSIVE")
	if err != nil {
		return Overlay{}, err
	}
	ov.Scan.FollowSymlinks, err = envBoolAny(lookup, "TGUP_FOLLOW_SYMLINKS")
	if err != nil {
		return Overlay{}, err
	}

	ov.Plan.Order = envStringAny(lookup, "TGUP_ORDER")
	ov.Plan.Reverse, err = envBoolAny(lookup, "TGUP_REVERSE")
	if err != nil {
		return Overlay{}, err
	}
	ov.Plan.AlbumMax, err = envIntAny(lookup, "TGUP_ALBUM_MAX")
	if err != nil {
		return Overlay{}, err
	}

	ov.Upload.Target = envStringAny(lookup, "TGUP_TARGET")
	ov.Upload.Caption = envStringAny(lookup, "TGUP_CAPTION")
	ov.Upload.ParseMode = envStringAny(lookup, "TGUP_PARSE_MODE")
	ov.Upload.ConcurrencyAlbum, err = envIntAny(lookup, "TGUP_CONCURRENCY_ALBUM")
	if err != nil {
		return Overlay{}, err
	}
	ov.Upload.Threads, err = envIntAny(lookup, "TGUP_THREADS")
	if err != nil {
		return Overlay{}, err
	}
	ov.Upload.PoolSize, err = envIntAny(lookup, "TGUP_POOL_SIZE")
	if err != nil {
		return Overlay{}, err
	}
	ov.Upload.Resume, err = envBoolAny(lookup, "TGUP_RESUME")
	if err != nil {
		return Overlay{}, err
	}
	ov.Upload.StrictMetadata, err = envBoolAny(lookup, "TGUP_STRICT_METADATA")
	if err != nil {
		return Overlay{}, err
	}
	ov.Upload.ImageMode = envStringAny(lookup, "TGUP_IMAGE_MODE")
	ov.Upload.VideoThumbnail = envStringAny(lookup, "TGUP_VIDEO_THUMBNAIL")
	ov.Upload.Duplicate = envStringAny(lookup, "TGUP_DUPLICATE")

	ov.Maintenance.Enabled, err = envBoolAny(lookup, "TGUP_MAINTENANCE_ENABLED")
	if err != nil {
		return Overlay{}, err
	}
	ov.Maintenance.IntervalHours, err = envFloatAny(lookup, "TGUP_MAINTENANCE_INTERVAL_HOURS")
	if err != nil {
		return Overlay{}, err
	}
	ov.Maintenance.RetentionSentDays, err = envIntAny(lookup, "TGUP_MAINTENANCE_RETENTION_SENT_DAYS")
	if err != nil {
		return Overlay{}, err
	}
	ov.Maintenance.RetentionFailedDays, err = envIntAny(lookup, "TGUP_MAINTENANCE_RETENTION_FAILED_DAYS")
	if err != nil {
		return Overlay{}, err
	}
	ov.Maintenance.RetentionQueueDays, err = envIntAny(lookup, "TGUP_MAINTENANCE_RETENTION_QUEUE_DAYS")
	if err != nil {
		return Overlay{}, err
	}
	ov.Maintenance.MaxDBMB, err = envIntAny(lookup, "TGUP_MAINTENANCE_MAX_DB_MB")
	if err != nil {
		return Overlay{}, err
	}
	ov.Maintenance.MaxUploadRows, err = envIntAny(lookup, "TGUP_MAINTENANCE_MAX_UPLOAD_ROWS")
	if err != nil {
		return Overlay{}, err
	}
	ov.Maintenance.FirstRunPreview, err = envBoolAny(lookup, "TGUP_MAINTENANCE_FIRST_RUN_PREVIEW")
	if err != nil {
		return Overlay{}, err
	}
	ov.Maintenance.VacuumCooldownHours, err = envFloatAny(lookup, "TGUP_MAINTENANCE_VACUUM_COOLDOWN_HOURS")
	if err != nil {
		return Overlay{}, err
	}
	ov.Maintenance.VacuumMinReclaimMB, err = envIntAny(lookup, "TGUP_MAINTENANCE_VACUUM_MIN_RECLAIM_MB")
	if err != nil {
		return Overlay{}, err
	}

	ov.MCP.Enabled, err = envBoolAny(lookup, "TGUP_MCP_ENABLED")
	if err != nil {
		return Overlay{}, err
	}
	ov.MCP.Host = envStringAny(lookup, "TGUP_MCP_HOST")
	ov.MCP.Port, err = envIntAny(lookup, "TGUP_MCP_PORT")
	if err != nil {
		return Overlay{}, err
	}
	ov.MCP.Token = envStringAny(lookup, "TGUP_MCP_TOKEN")
	ov.MCP.AllowRoots = envCSVAny(lookup, "TGUP_MCP_ALLOW_ROOTS")
	ov.MCP.ControlDB = envStringAny(lookup, "TGUP_MCP_CONTROL_DB")
	ov.MCP.EventRetentionHours, err = envFloatAny(lookup, "TGUP_MCP_EVENT_RETENTION_HOURS")
	if err != nil {
		return Overlay{}, err
	}
	ov.MCP.MaxConcurrentJobs, err = envIntAny(lookup, "TGUP_MCP_MAX_CONCURRENT_JOBS")
	if err != nil {
		return Overlay{}, err
	}
	ov.MCP.EnableSSE, err = envBoolAny(lookup, "TGUP_MCP_ENABLE_SSE")
	if err != nil {
		return Overlay{}, err
	}
	ov.MCP.AllowedOrigins = envCSVAny(lookup, "TGUP_MCP_ALLOWED_ORIGINS")

	return ov, nil
}

func envStringAny(lookup lookupEnvFn, keys ...string) *string {
	for _, key := range keys {
		if v, ok := lookup(key); ok {
			value := strings.TrimSpace(v)
			return &value
		}
	}
	return nil
}

func envCSVAny(lookup lookupEnvFn, keys ...string) *[]string {
	for _, key := range keys {
		if v, ok := lookup(key); ok {
			out := splitCSV(v)
			return &out
		}
	}
	return nil
}

func envBoolAny(lookup lookupEnvFn, keys ...string) (*bool, error) {
	for _, key := range keys {
		if v, ok := lookup(key); ok {
			parsed, err := parseBoolLike(v)
			if err != nil {
				return nil, fmt.Errorf("%s: invalid bool %q", key, v)
			}
			return &parsed, nil
		}
	}
	return nil, nil
}

func envIntAny(lookup lookupEnvFn, keys ...string) (*int, error) {
	for _, key := range keys {
		if v, ok := lookup(key); ok {
			parsed, err := strconv.Atoi(strings.TrimSpace(v))
			if err != nil {
				return nil, fmt.Errorf("%s: invalid int %q", key, v)
			}
			return &parsed, nil
		}
	}
	return nil, nil
}

func envFloatAny(lookup lookupEnvFn, keys ...string) (*float64, error) {
	for _, key := range keys {
		if v, ok := lookup(key); ok {
			parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
			if err != nil {
				return nil, fmt.Errorf("%s: invalid float %q", key, v)
			}
			return &parsed, nil
		}
	}
	return nil, nil
}

func parseBoolLike(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	}
	return false, fmt.Errorf("invalid bool value: %q", value)
}

func validateNoConflictingKeys(lookup lookupEnvFn, keyA string, keyB string) error {
	a, aOK := lookup(keyA)
	b, bOK := lookup(keyB)
	if !aOK || !bOK {
		return nil
	}
	if strings.TrimSpace(a) == strings.TrimSpace(b) {
		return nil
	}
	keys := []string{keyA, keyB}
	sort.Strings(keys)
	return fmt.Errorf("conflicting env values: %s and %s", keys[0], keys[1])
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
