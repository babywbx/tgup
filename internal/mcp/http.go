package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Server is a minimal HTTP holder for MCP handlers.
type Server struct {
	httpServer *http.Server
}

// NewServer builds an MCP HTTP server shell.
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
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
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
