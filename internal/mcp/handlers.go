package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// NewHandler registers the full MCP HTTP mux with auth, JSON-RPC, SSE, and health.
func NewHandler(cfg Config, server *MCPServer, sse *SSEHandler) http.Handler {
	mux := http.NewServeMux()

	// Health check (no auth).
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	// Schema endpoint.
	mux.HandleFunc("/schema", withAuth(cfg.AuthToken, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(Schema()); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}))

	// MCP endpoint - multiplexed by method and Accept header.
	// OPTIONS is handled before auth for CORS preflight.
	mux.HandleFunc("/mcp", withSecurityHeaders(func(w http.ResponseWriter, r *http.Request) {
		// CORS preflight must not require auth.
		if r.Method == http.MethodOptions {
			handleCORS(w, r, cfg)
			return
		}

		// All other methods require auth.
		if cfg.AuthToken != "" {
			provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if err := ValidateBearerToken(provided, cfg.AuthToken); err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		// Set CORS headers on actual responses too.
		if origin := r.Header.Get("Origin"); origin != "" {
			if len(cfg.AllowedOrigins) == 0 || isOriginAllowed(origin, cfg.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
		}

		switch r.Method {
		case http.MethodPost:
			handleMCPPost(w, r, server, cfg)
		case http.MethodGet:
			if cfg.EnableSSE && sse != nil {
				if accept := r.Header.Get("Accept"); strings.Contains(accept, "text/event-stream") {
					sse.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "use POST for JSON-RPC or GET with Accept: text/event-stream for SSE", http.StatusBadRequest)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	return mux
}

func handleMCPPost(w http.ResponseWriter, r *http.Request, server *MCPServer, cfg Config) {
	// Validate origin if configured.
	if len(cfg.AllowedOrigins) > 0 {
		origin := r.Header.Get("Origin")
		if origin != "" && !isOriginAllowed(origin, cfg.AllowedOrigins) {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
	}

	// Track session.
	sessionID := r.Header.Get("MCP-Session-Id")
	if sessionID != "" {
		server.TrackSession(sessionID)
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	resp := server.HandleJSONRPC(r.Context(), body)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func handleCORS(w http.ResponseWriter, r *http.Request, cfg Config) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if len(cfg.AllowedOrigins) > 0 && !isOriginAllowed(origin, cfg.AllowedOrigins) {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, MCP-Session-Id, Last-Event-ID")
	w.Header().Set("Access-Control-Max-Age", "86400")
	w.WriteHeader(http.StatusNoContent)
}

func withAuth(token string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token != "" {
			provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if err := ValidateBearerToken(provided, token); err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

func withSecurityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store")
		next(w, r)
	}
}

func isOriginAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == "*" || strings.EqualFold(a, origin) {
			return true
		}
	}
	return false
}
