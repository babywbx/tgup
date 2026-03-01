package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Server is the MCP HTTP server.
type Server struct {
	httpServer *http.Server
}

// NewServer builds an MCP HTTP server.
func NewServer(cfg Config, handler http.Handler) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	return &Server{
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 0, // Disabled for SSE long-lived connections.
			IdleTimeout:  120 * time.Second,
		},
	}, nil
}

// Start starts serving HTTP.
func (s *Server) Start() error {
	if s == nil || s.httpServer == nil {
		return fmt.Errorf("mcp server not initialized")
	}
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

// Addr returns the configured listen address.
func (s *Server) Addr() string {
	if s == nil || s.httpServer == nil {
		return ""
	}
	return s.httpServer.Addr
}
