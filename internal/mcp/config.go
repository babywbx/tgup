package mcp

import "fmt"

// Config holds MCP server options.
type Config struct {
	Enabled             bool
	Host                string
	Port                int
	AuthToken           string
	AllowRoots          []string
	ControlDB           string
	EventRetentionHours float64
	MaxConcurrentJobs   int
	EnableSSE           bool
	AllowedOrigins      []string
}

// Validate checks minimal MCP config invariants.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("mcp port must be in range 1..65535")
	}
	if len(c.AllowRoots) == 0 {
		return fmt.Errorf("mcp allow_roots is required when mcp is enabled")
	}
	return nil
}

// FromAppConfig converts the app's MCPConfig to the mcp package Config.
func FromAppConfig(app struct {
	Enabled             bool
	Host                string
	Port                int
	Token               string
	AllowRoots          []string
	ControlDB           string
	EventRetentionHours float64
	MaxConcurrentJobs   int
	EnableSSE           bool
	AllowedOrigins      []string
}) Config {
	return Config{
		Enabled:             app.Enabled,
		Host:                app.Host,
		Port:                app.Port,
		AuthToken:           app.Token,
		AllowRoots:          app.AllowRoots,
		ControlDB:           app.ControlDB,
		EventRetentionHours: app.EventRetentionHours,
		MaxConcurrentJobs:   app.MaxConcurrentJobs,
		EnableSSE:           app.EnableSSE,
		AllowedOrigins:      app.AllowedOrigins,
	}
}
