package mcp

// ToolSchema is a minimal MCP tool description model.
type ToolSchema struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SchemaDocument is the bootstrap schema payload.
type SchemaDocument struct {
	Version string       `json:"version"`
	Tools   []ToolSchema `json:"tools"`
}

// Schema returns the current bootstrap schema.
func Schema() SchemaDocument {
	return SchemaDocument{
		Version: "0.1.0",
		Tools: []ToolSchema{
			{Name: "dry_run", Description: "preview scan/plan result"},
		},
	}
}
