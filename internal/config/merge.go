package config

// Overlay contains optional fields used to override config values.
// Nil fields mean "not set" and therefore do not override previous values.
type Overlay struct {
	Telegram    TelegramOverlay    `json:"telegram,omitempty" toml:"telegram"`
	Paths       PathsOverlay       `json:"paths,omitempty" toml:"paths"`
	Scan        ScanOverlay        `json:"scan,omitempty" toml:"scan"`
	Plan        PlanOverlay        `json:"plan,omitempty" toml:"plan"`
	Upload      UploadOverlay      `json:"upload,omitempty" toml:"upload"`
	Maintenance MaintenanceOverlay `json:"maintenance,omitempty" toml:"maintenance"`
	MCP         MCPOverlay         `json:"mcp,omitempty" toml:"mcp"`
}

type TelegramOverlay struct {
	APIID             *int    `json:"api_id,omitempty" toml:"api_id"`
	APIHash           *string `json:"api_hash,omitempty" toml:"api_hash"`
	SessionPath       *string `json:"session,omitempty" toml:"session"`
	SessionPathLegacy *string `json:"session_path,omitempty" toml:"session_path"`
}

type PathsOverlay struct {
	StatePath       *string `json:"state,omitempty" toml:"state"`
	StatePathLegacy *string `json:"state_path,omitempty" toml:"state_path"`
	ArtifactsDir    *string `json:"artifacts_dir,omitempty" toml:"artifacts_dir"`
}

type ScanOverlay struct {
	Src            *[]string `json:"src,omitempty" toml:"src"`
	Recursive      *bool     `json:"recursive,omitempty" toml:"recursive"`
	FollowSymlinks *bool     `json:"follow_symlinks,omitempty" toml:"follow_symlinks"`
	IncludeExt     *[]string `json:"include_ext,omitempty" toml:"include_ext"`
	ExcludeExt     *[]string `json:"exclude_ext,omitempty" toml:"exclude_ext"`
}

type PlanOverlay struct {
	Order    *string `json:"order,omitempty" toml:"order"`
	Reverse  *bool   `json:"reverse,omitempty" toml:"reverse"`
	AlbumMax *int    `json:"album_max,omitempty" toml:"album_max"`
}

type UploadOverlay struct {
	Target           *string `json:"target,omitempty" toml:"target"`
	Caption          *string `json:"caption,omitempty" toml:"caption"`
	ParseMode        *string `json:"parse_mode,omitempty" toml:"parse_mode"`
	ConcurrencyAlbum *int    `json:"concurrency_album,omitempty" toml:"concurrency_album"`
	Resume           *bool   `json:"resume,omitempty" toml:"resume"`
	StrictMetadata   *bool   `json:"strict_metadata,omitempty" toml:"strict_metadata"`
	ImageMode        *string `json:"image_mode,omitempty" toml:"image_mode"`
	VideoThumbnail   *string `json:"video_thumbnail,omitempty" toml:"video_thumbnail"`
	Duplicate        *string `json:"duplicate,omitempty" toml:"duplicate"`
	// state keeps compatibility with the legacy [upload].state key.
	StatePathCompat *string `json:"state,omitempty" toml:"state"`
}

type MaintenanceOverlay struct {
	Enabled             *bool    `json:"enabled,omitempty" toml:"enabled"`
	IntervalHours       *float64 `json:"interval_hours,omitempty" toml:"interval_hours"`
	RetentionSentDays   *int     `json:"retention_sent_days,omitempty" toml:"retention_sent_days"`
	RetentionFailedDays *int     `json:"retention_failed_days,omitempty" toml:"retention_failed_days"`
	RetentionQueueDays  *int     `json:"retention_queue_days,omitempty" toml:"retention_queue_days"`
	MaxDBMB             *int     `json:"max_db_mb,omitempty" toml:"max_db_mb"`
	MaxUploadRows       *int     `json:"max_upload_rows,omitempty" toml:"max_upload_rows"`
	FirstRunPreview     *bool    `json:"first_run_preview,omitempty" toml:"first_run_preview"`
	VacuumCooldownHours *float64 `json:"vacuum_cooldown_hours,omitempty" toml:"vacuum_cooldown_hours"`
	VacuumMinReclaimMB  *int     `json:"vacuum_min_reclaim_mb,omitempty" toml:"vacuum_min_reclaim_mb"`
}

type MCPOverlay struct {
	Enabled             *bool     `json:"enabled,omitempty" toml:"enabled"`
	Host                *string   `json:"host,omitempty" toml:"host"`
	Port                *int      `json:"port,omitempty" toml:"port"`
	Token               *string   `json:"token,omitempty" toml:"token"`
	AllowRoots          *[]string `json:"allow_roots,omitempty" toml:"allow_roots"`
	ControlDB           *string   `json:"control_db,omitempty" toml:"control_db"`
	EventRetentionHours *float64  `json:"event_retention_hours,omitempty" toml:"event_retention_hours"`
	MaxConcurrentJobs   *int      `json:"max_concurrent_jobs,omitempty" toml:"max_concurrent_jobs"`
	EnableSSE           *bool     `json:"enable_sse,omitempty" toml:"enable_sse"`
	AllowedOrigins      *[]string `json:"allowed_origins,omitempty" toml:"allowed_origins"`
}

// Merge applies overlays from left to right.
func Merge(base Config, overlays ...Overlay) Config {
	out := base
	for _, ov := range overlays {
		applyTelegram(&out.Telegram, ov.Telegram)
		applyPaths(&out.Paths, ov.Paths)
		applyScan(&out.Scan, ov.Scan)
		applyPlan(&out.Plan, ov.Plan)
		applyUpload(&out.Upload, ov.Upload)
		if ov.Upload.StatePathCompat != nil && ov.Paths.StatePath == nil && ov.Paths.StatePathLegacy == nil {
			out.Paths.StatePath = *ov.Upload.StatePathCompat
		}
		applyMaintenance(&out.Maintenance, ov.Maintenance)
		applyMCP(&out.MCP, ov.MCP)
	}
	return out
}

func applyTelegram(dst *TelegramConfig, src TelegramOverlay) {
	if src.APIID != nil {
		dst.APIID = *src.APIID
	}
	if src.APIHash != nil {
		dst.APIHash = *src.APIHash
	}
	if src.SessionPath != nil {
		dst.SessionPath = *src.SessionPath
	} else if src.SessionPathLegacy != nil {
		dst.SessionPath = *src.SessionPathLegacy
	}
}

func applyPaths(dst *PathsConfig, src PathsOverlay) {
	if src.StatePath != nil {
		dst.StatePath = *src.StatePath
	} else if src.StatePathLegacy != nil {
		dst.StatePath = *src.StatePathLegacy
	}
	if src.ArtifactsDir != nil {
		dst.ArtifactsDir = *src.ArtifactsDir
	}
}

func applyScan(dst *ScanConfig, src ScanOverlay) {
	if src.Src != nil {
		dst.Src = cloneStrings(*src.Src)
	}
	if src.Recursive != nil {
		dst.Recursive = *src.Recursive
	}
	if src.FollowSymlinks != nil {
		dst.FollowSymlinks = *src.FollowSymlinks
	}
	if src.IncludeExt != nil {
		dst.IncludeExt = normalizeExtensions(*src.IncludeExt)
	}
	if src.ExcludeExt != nil {
		dst.ExcludeExt = normalizeExtensions(*src.ExcludeExt)
	}
}

func applyPlan(dst *PlanConfig, src PlanOverlay) {
	if src.Order != nil {
		dst.Order = *src.Order
	}
	if src.Reverse != nil {
		dst.Reverse = *src.Reverse
	}
	if src.AlbumMax != nil {
		dst.AlbumMax = *src.AlbumMax
	}
}

func applyUpload(dst *UploadConfig, src UploadOverlay) {
	if src.Target != nil {
		dst.Target = *src.Target
	}
	if src.Caption != nil {
		dst.Caption = *src.Caption
	}
	if src.ParseMode != nil {
		dst.ParseMode = *src.ParseMode
	}
	if src.ConcurrencyAlbum != nil {
		dst.ConcurrencyAlbum = *src.ConcurrencyAlbum
	}
	if src.Resume != nil {
		dst.Resume = *src.Resume
	}
	if src.StrictMetadata != nil {
		dst.StrictMetadata = *src.StrictMetadata
	}
	if src.ImageMode != nil {
		dst.ImageMode = *src.ImageMode
	}
	if src.VideoThumbnail != nil {
		dst.VideoThumbnail = *src.VideoThumbnail
	}
	if src.Duplicate != nil {
		dst.Duplicate = *src.Duplicate
	}
}

func applyMaintenance(dst *MaintenanceConfig, src MaintenanceOverlay) {
	if src.Enabled != nil {
		dst.Enabled = *src.Enabled
	}
	if src.IntervalHours != nil {
		dst.IntervalHours = *src.IntervalHours
	}
	if src.RetentionSentDays != nil {
		dst.RetentionSentDays = *src.RetentionSentDays
	}
	if src.RetentionFailedDays != nil {
		dst.RetentionFailedDays = *src.RetentionFailedDays
	}
	if src.RetentionQueueDays != nil {
		dst.RetentionQueueDays = *src.RetentionQueueDays
	}
	if src.MaxDBMB != nil {
		dst.MaxDBMB = *src.MaxDBMB
	}
	if src.MaxUploadRows != nil {
		dst.MaxUploadRows = *src.MaxUploadRows
	}
	if src.FirstRunPreview != nil {
		dst.FirstRunPreview = *src.FirstRunPreview
	}
	if src.VacuumCooldownHours != nil {
		dst.VacuumCooldownHours = *src.VacuumCooldownHours
	}
	if src.VacuumMinReclaimMB != nil {
		dst.VacuumMinReclaimMB = *src.VacuumMinReclaimMB
	}
}

func applyMCP(dst *MCPConfig, src MCPOverlay) {
	if src.Enabled != nil {
		dst.Enabled = *src.Enabled
	}
	if src.Host != nil {
		dst.Host = *src.Host
	}
	if src.Port != nil {
		dst.Port = *src.Port
	}
	if src.Token != nil {
		dst.Token = *src.Token
	}
	if src.AllowRoots != nil {
		dst.AllowRoots = cloneStrings(*src.AllowRoots)
	}
	if src.ControlDB != nil {
		dst.ControlDB = *src.ControlDB
	}
	if src.EventRetentionHours != nil {
		dst.EventRetentionHours = *src.EventRetentionHours
	}
	if src.MaxConcurrentJobs != nil {
		dst.MaxConcurrentJobs = *src.MaxConcurrentJobs
	}
	if src.EnableSSE != nil {
		dst.EnableSSE = *src.EnableSSE
	}
	if src.AllowedOrigins != nil {
		dst.AllowedOrigins = cloneStrings(*src.AllowedOrigins)
	}
}

func cloneStrings(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
