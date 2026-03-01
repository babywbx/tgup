package app

import (
	"context"

	"github.com/babywbx/tgup/internal/xerrors"
)

// MCPServe executes MCP server flow (implemented in later milestones).
func MCPServe(ctx context.Context) error {
	_ = ctx
	return xerrors.Wrap(xerrors.CodeMCP, "mcp serve not implemented yet", nil)
}

// MCPSchema executes MCP schema output flow (implemented in later milestones).
func MCPSchema(ctx context.Context) error {
	_ = ctx
	return xerrors.Wrap(xerrors.CodeMCP, "mcp schema not implemented yet", nil)
}
