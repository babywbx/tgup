package mcp

import "fmt"

// Config holds MCP server options.
type Config struct {
	Enabled   bool
	Host      string
	Port      int
	AuthToken string
	AllowRoot string
}

// Validate checks minimal MCP config invariants.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("mcp port must be in range 1..65535")
	}
	if c.AllowRoot == "" {
		return fmt.Errorf("mcp allow_root is required when mcp is enabled")
	}
	return nil
}
