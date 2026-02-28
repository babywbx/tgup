package config

// Default returns baseline configuration values before file/env/CLI overlays.
func Default() Config {
	return Config{
		Telegram: TelegramConfig{
			SessionPath: "./secrets/session.session",
		},
		Paths: PathsConfig{
			StatePath:    "./data/state.sqlite",
			ArtifactsDir: "./data/runs",
		},
		Scan: ScanConfig{
			Src:            []string{},
			Recursive:      true,
			FollowSymlinks: false,
		},
		Plan: PlanConfig{
			Order:    "mtime",
			AlbumMax: 10,
		},
		Upload: UploadConfig{
			Target:           "me",
			Caption:          "",
			ParseMode:        "plain",
			ConcurrencyAlbum: 5,
			Resume:           true,
			StrictMetadata:   false,
			ImageMode:        "auto",
			VideoThumbnail:   "auto",
			Duplicate:        "ask",
		},
		Maintenance: MaintenanceConfig{
			Enabled:             true,
			IntervalHours:       6,
			RetentionSentDays:   90,
			RetentionFailedDays: 30,
			RetentionQueueDays:  7,
			MaxDBMB:             256,
			MaxUploadRows:       300000,
			FirstRunPreview:     true,
			VacuumCooldownHours: 24,
			VacuumMinReclaimMB:  32,
		},
		MCP: MCPConfig{
			Enabled:             false,
			Host:                "127.0.0.1",
			Port:                8765,
			Token:               "",
			AllowRoots:          []string{"./media", "./data", "./secrets"},
			ControlDB:           "./data/mcp.sqlite",
			EventRetentionHours: 72,
			MaxConcurrentJobs:   4,
			EnableSSE:           true,
			AllowedOrigins:      []string{},
		},
	}
}
