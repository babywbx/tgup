package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/babywbx/tgup/internal/app"
	"github.com/babywbx/tgup/internal/config"
	"github.com/babywbx/tgup/internal/mcp"
)

func runMCP(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "missing mcp subcommand (serve|schema)")
		return 2
	}

	switch args[0] {
	case "serve":
		return runMCPServe(args[1:], stdout, stderr)
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

func runMCPServe(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("mcp serve", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var configPath string
	fs.StringVar(&configPath, "config", "", "path to config file")

	token := &stringValue{}
	host := &stringValue{}
	port := &intValue{}
	controlDB := &stringValue{}
	enableSSE := &boolValue{}
	maxJobs := &intValue{}

	fs.Var(token, "token", "bearer auth token")
	fs.Var(host, "host", "listen host")
	fs.Var(port, "port", "listen port")
	fs.Var(controlDB, "control-db", "control database path")
	fs.Var(enableSSE, "enable-sse", "enable SSE streaming")
	fs.Var(maxJobs, "max-concurrent-jobs", "max concurrent jobs")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	cli := config.Overlay{}
	// Force MCP enabled.
	enabled := true
	cli.MCP.Enabled = &enabled

	if token.set {
		cli.MCP.Token = token.ptr()
	}
	if host.set {
		cli.MCP.Host = host.ptr()
	}
	if port.set {
		cli.MCP.Port = port.ptr()
	}
	if controlDB.set {
		cli.MCP.ControlDB = controlDB.ptr()
	}
	if enableSSE.set {
		cli.MCP.EnableSSE = enableSSE.ptr()
	}
	if maxJobs.set {
		cli.MCP.MaxConcurrentJobs = maxJobs.ptr()
	}

	_ = stdout // MCPServe writes to os.Stdout directly.

	if err := app.MCPServe(context.Background(), app.MCPServeOptions{
		ConfigPath: configPath,
		CLI:        cli,
	}); err != nil {
		fmt.Fprintf(stderr, "mcp serve: %v\n", err)
		return 1
	}
	return 0
}
