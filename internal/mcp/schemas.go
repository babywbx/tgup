package mcp

import "encoding/json"

// ToolSchema is an MCP tool description with JSON Schema.
type ToolSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

// SchemaDocument is the schema payload.
type SchemaDocument struct {
	Version string       `json:"version"`
	Tools   []ToolSchema `json:"tools"`
}

// Schema returns the complete tool schema document.
func Schema() SchemaDocument {
	return SchemaDocument{
		Version: "2026-02-25",
		Tools: []ToolSchema{
			{
				Name:        "tgup.health",
				Description: "Check server health and available tools",
				InputSchema: emptySchema(),
			},
			{
				Name:        "tgup.dry_run",
				Description: "Preview scan and plan results without uploading",
				InputSchema: dryRunSchema(),
			},
			{
				Name:        "tgup.run.start",
				Description: "Start an asynchronous upload job",
				InputSchema: runStartSchema(),
			},
			{
				Name:        "tgup.run.sync",
				Description: "Run a synchronous upload and wait for completion",
				InputSchema: runSyncSchema(),
			},
			{
				Name:        "tgup.run.status",
				Description: "Get the status and result of a job",
				InputSchema: jobIDSchema(),
			},
			{
				Name:        "tgup.run.cancel",
				Description: "Cancel a running or queued job",
				InputSchema: jobIDSchema(),
			},
			{
				Name:        "tgup.run.events",
				Description: "Retrieve events for a job with pagination",
				InputSchema: runEventsSchema(),
			},
			{
				Name:        "tgup.schema.get",
				Description: "Get the full tool contract schema",
				InputSchema: emptySchema(),
			},
		},
	}
}

func emptySchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func runSpecProperties() map[string]any {
	return map[string]any{
		"src":            map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "source paths"},
		"recursive":      map[string]any{"type": "boolean", "description": "scan recursively"},
		"followSymlinks": map[string]any{"type": "boolean", "description": "follow symlinks"},
		"includeExt":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "include extensions"},
		"excludeExt":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "exclude extensions"},
		"order":          map[string]any{"type": "string", "enum": jsonArray("name", "mtime", "size", "random"), "description": "sort order"},
		"reverse":        map[string]any{"type": "boolean", "description": "reverse sort"},
		"albumMax":       map[string]any{"type": "integer", "minimum": 1, "maximum": 10, "description": "max items per album"},
		"target":         map[string]any{"type": "string", "description": "telegram target"},
		"caption":        map[string]any{"type": "string", "description": "default caption"},
		"parseMode":      map[string]any{"type": "string", "enum": jsonArray("plain", "md"), "description": "caption parse mode"},
		"concurrency":    map[string]any{"type": "integer", "minimum": 1, "description": "album concurrency"},
		"resume":         map[string]any{"type": "boolean", "description": "enable resume"},
		"strictMetadata": map[string]any{"type": "boolean", "description": "strict metadata checks"},
		"imageMode":      map[string]any{"type": "string", "enum": jsonArray("auto", "photo", "document"), "description": "image upload mode"},
		"videoThumbnail": map[string]any{"type": "string", "description": "video thumbnail: auto, off, or path"},
		"duplicate":      map[string]any{"type": "string", "enum": jsonArray("skip", "ask", "upload"), "description": "duplicate policy"},
		"configPath":     map[string]any{"type": "string", "description": "path to config file"},
	}
}

func dryRunSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"runSpec": map[string]any{
				"type":       "object",
				"properties": runSpecProperties(),
			},
			"showFiles": map[string]any{"type": "boolean", "description": "include file paths in preview"},
		},
		"required": jsonArray("runSpec"),
	}
}

func runStartSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"runSpec": map[string]any{
				"type":       "object",
				"properties": runSpecProperties(),
			},
		},
		"required": jsonArray("runSpec"),
	}
}

func runSyncSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"runSpec": map[string]any{
				"type":       "object",
				"properties": runSpecProperties(),
			},
			"timeoutSec": map[string]any{"type": "integer", "minimum": 1, "maximum": 3600, "description": "timeout in seconds"},
		},
		"required": jsonArray("runSpec"),
	}
}

func jobIDSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"jobId": map[string]any{"type": "string", "description": "job identifier"},
		},
		"required": jsonArray("jobId"),
	}
}

func runEventsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"jobId":    map[string]any{"type": "string", "description": "job identifier"},
			"sinceSeq": map[string]any{"type": "integer", "minimum": 0, "description": "return events after this sequence"},
			"limit":    map[string]any{"type": "integer", "minimum": 1, "maximum": 1000, "description": "max events to return"},
		},
		"required": jsonArray("jobId"),
	}
}

func jsonArray(values ...string) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

// SchemaJSON returns the schema as pretty JSON bytes.
func SchemaJSON() ([]byte, error) {
	return json.MarshalIndent(Schema(), "", "  ")
}
