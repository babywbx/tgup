package mcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

// NewHandler registers a minimal MCP HTTP mux with optional auth.
func NewHandler(cfg Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/schema", func(w http.ResponseWriter, r *http.Request) {
		if cfg.AuthToken != "" {
			token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if err := ValidateBearerToken(token, cfg.AuthToken); err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(Schema()); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	})
	return mux
}
