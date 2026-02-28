package config

// Config stores the resolved runtime configuration used by the CLI.
type Config struct {
	Telegram    TelegramConfig
	Paths       PathsConfig
	Scan        ScanConfig
	Plan        PlanConfig
	Upload      UploadConfig
	Maintenance MaintenanceConfig
	MCP         MCPConfig
}

type TelegramConfig struct {
	APIID       int
	APIHash     string
	SessionPath string
}

type PathsConfig struct {
	StatePath    string
	ArtifactsDir string
}

type ScanConfig struct {
	Src            []string
	Recursive      bool
	FollowSymlinks bool
	IncludeExt     []string
	ExcludeExt     []string
}

type PlanConfig struct {
	Order    string
	Reverse  bool
	AlbumMax int
}

type UploadConfig struct {
	Target           string
	Caption          string
	ParseMode        string
	ConcurrencyAlbum int
	Resume           bool
	StrictMetadata   bool
	ImageMode        string
	VideoThumbnail   string
	Duplicate        string
}

type MaintenanceConfig struct {
	Enabled bool
}

type MCPConfig struct {
	Enabled bool
	Host    string
	Port    int
}
