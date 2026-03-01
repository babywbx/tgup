package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/babywbx/tgup/internal/config"
	"github.com/babywbx/tgup/internal/mcp"
)

// MCPServeOptions holds parameters for mcp serve.
type MCPServeOptions struct {
	ConfigPath string
	CLI        config.Overlay
}

// MCPServe starts the MCP HTTP server.
func MCPServe(ctx context.Context, opts MCPServeOptions) error {
	cfg, err := config.Resolve(opts.ConfigPath, opts.CLI)
	if err != nil {
		return fmt.Errorf("resolve config: %w", err)
	}

	if !cfg.MCP.Enabled {
		return fmt.Errorf("MCP is not enabled in configuration (set mcp.enabled = true)")
	}

	mcpCfg := mcp.Config{
		Enabled:             cfg.MCP.Enabled,
		Host:                cfg.MCP.Host,
		Port:                cfg.MCP.Port,
		AuthToken:           cfg.MCP.Token,
		AllowRoots:          cfg.MCP.AllowRoots,
		ControlDB:           cfg.MCP.ControlDB,
		EventRetentionHours: cfg.MCP.EventRetentionHours,
		MaxConcurrentJobs:   cfg.MCP.MaxConcurrentJobs,
		EnableSSE:           cfg.MCP.EnableSSE,
		AllowedOrigins:      cfg.MCP.AllowedOrigins,
	}

	// Ensure control DB directory exists.
	dbPath := mcpCfg.ControlDB
	if dbPath == "" {
		dbPath = filepath.Join("data", "mcp.sqlite")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("create control db dir: %w", err)
	}

	// Open event store.
	events, err := mcp.OpenEventStore(dbPath)
	if err != nil {
		return fmt.Errorf("open event store: %w", err)
	}
	defer events.Close()

	// Open job manager.
	jobs, err := mcp.OpenJobManager(dbPath, events, mcp.JobManagerConfig{
		MaxConcurrent: mcpCfg.MaxConcurrentJobs,
	})
	if err != nil {
		return fmt.Errorf("open job manager: %w", err)
	}
	defer jobs.Close()

	// Create bridge and server.
	bridge := mcp.NewBridge(mcpCfg.AllowRoots)
	server := mcp.NewMCPServer(events, jobs, bridge, mcpCfg)

	// Create SSE handler.
	var sse *mcp.SSEHandler
	if mcpCfg.EnableSSE {
		sse = mcp.NewSSEHandler(events, mcpCfg.AuthToken)
	}

	// Create HTTP handler and server.
	handler := mcp.NewHandler(mcpCfg, server, sse)
	httpServer, err := mcp.NewServer(mcpCfg, handler)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	// Periodic event cleanup.
	cleanupCtx, cleanupCancel := context.WithCancel(ctx)
	defer cleanupCancel()
	go func() {
		retention := time.Duration(mcpCfg.EventRetentionHours * float64(time.Hour))
		if retention <= 0 {
			retention = 72 * time.Hour
		}
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-cleanupCtx.Done():
				return
			case <-ticker.C:
				events.CleanupWithRetention(cleanupCtx, retention)
			}
		}
	}()

	// Signal handling for graceful shutdown.
	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start server in goroutine.
	errCh := make(chan error, 1)
	go func() {
		fmt.Fprintf(os.Stdout, "mcp server listening on %s\n", httpServer.Addr())
		errCh <- httpServer.Start()
	}()

	select {
	case <-sigCtx.Done():
		fmt.Fprintln(os.Stdout, "shutting down mcp server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
