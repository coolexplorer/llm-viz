package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// handleSSE streams usage events to the client for the given session.
// Clients connect with: GET /api/sse?session_id=xxx
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	events := s.broadcaster.Subscribe(sessionID)
	defer s.broadcaster.Unsubscribe(sessionID)

	s.logger.Info("SSE client connected", "session_id", sessionID)
	defer s.logger.Info("SSE client disconnected", "session_id", sessionID)

	// Send an initial ping to confirm the connection is alive.
	fmt.Fprintf(w, "event: ping\ndata: {\"session_id\": %q}\n\n", sessionID)
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-events:
			if !ok {
				// Channel closed by Unsubscribe.
				return
			}
			data, err := json.Marshal(event)
			if err != nil {
				s.logger.Error("SSE marshal error", "error", err)
				continue
			}
			fmt.Fprintf(w, "event: usage\ndata: %s\n\n", data)
			flusher.Flush()

		case <-ticker.C:
			// Send a heartbeat to keep the connection alive through proxies.
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}
