package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/babywbx/tgup/internal/app"
	"github.com/babywbx/tgup/internal/mcp"
)

func runMCP(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "missing mcp subcommand (serve|schema)")
		return 2
	}

	switch args[0] {
	case "serve":
		if err := app.MCPServe(context.Background()); err != nil {
			fmt.Fprintf(stderr, "mcp serve failed: %v\n", err)
			return 1
		}
		return 0
	case "schema":
		if len(args) > 1 {
			fmt.Fprintf(stderr, "unexpected mcp schema args: %v\n", args[1:])
			return 2
		}
		if err := json.NewEncoder(stdout).Encode(mcp.Schema()); err != nil {
			fmt.Fprintf(stderr, "mcp schema failed: %v\n", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "unknown mcp subcommand: %s\n", args[0])
		return 2
	}
}
