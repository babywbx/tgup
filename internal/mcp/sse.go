package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

const (
	sseMaxConnections = 50
	sseLifetime       = 1 * time.Hour
	sseKeepalive      = 15 * time.Second
	sseBacklogLimit   = 1000
)

// SSEHandler manages Server-Sent Events connections.
type SSEHandler struct {
	events     EventStore
	authToken  string
	connCount  atomic.Int32
}

// NewSSEHandler creates an SSE handler.
func NewSSEHandler(events EventStore, authToken string) *SSEHandler {
	return &SSEHandler{
		events:    events,
		authToken: authToken,
	}
}

// ServeHTTP handles SSE GET requests.
func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check connection limit.
	if h.connCount.Load() >= sseMaxConnections {
		http.Error(w, "too many connections", http.StatusServiceUnavailable)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	h.connCount.Add(1)
	defer h.connCount.Add(-1)

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Parse Last-Event-ID for replay.
	var lastSeq int64
	if id := r.Header.Get("Last-Event-ID"); id != "" {
		lastSeq, _ = strconv.ParseInt(id, 10, 64)
	}

	// Subscribe to all events.
	sub := h.events.Register("")
	defer sub.Close()

	// Replay backlog if requested.
	if lastSeq > 0 {
		h.replayBacklog(r.Context(), w, flusher, lastSeq)
	}

	// Lifetime timer.
	ctx, cancel := context.WithTimeout(r.Context(), sseLifetime)
	defer cancel()

	keepalive := time.NewTicker(sseKeepalive)
	defer keepalive.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-sub.C():
			h.writeEvent(w, flusher, event)
		case <-keepalive.C:
			fmt.Fprintf(w, ":keepalive\n\n")
			flusher.Flush()
		}
	}
}

func (h *SSEHandler) replayBacklog(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, sinceSeq int64) {
	// Replay in chunks.
	seq := sinceSeq
	for {
		events, hasMore, err := h.events.List(ctx, "", seq, sseBacklogLimit)
		if err != nil || len(events) == 0 {
			break
		}
		for _, e := range events {
			h.writeEvent(w, flusher, e)
			seq = e.Seq
		}
		if !hasMore {
			break
		}
	}
}

func (h *SSEHandler) writeEvent(w http.ResponseWriter, flusher http.Flusher, event Event) {
	data, err := json.Marshal(EventEnvelope{
		Seq:     event.Seq,
		JobID:   event.JobID,
		Type:    event.Type,
		Payload: event.Payload,
		Ts:      event.CreatedAt,
	})
	if err != nil {
		return
	}

	fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.Seq, event.Type, data)
	flusher.Flush()
}
