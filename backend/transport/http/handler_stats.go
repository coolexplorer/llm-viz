package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/port"
)

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID != "" {
		s.handleSessionStats(w, r, sessionID)
		return
	}

	s.handleProviderStats(w, r)
}

func (s *Server) handleSessionStats(w http.ResponseWriter, r *http.Request, sessionID string) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	history, err := s.tracker.GetSessionHistory(r.Context(), sessionID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"session_id": sessionID,
		"records":    history,
		"count":      len(history),
	})
}

func (s *Server) handleProviderStats(w http.ResponseWriter, r *http.Request) {
	filter := port.TimeRangeFilter{}

	if startStr := r.URL.Query().Get("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.Start = t
		}
	}
	if endStr := r.URL.Query().Get("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.End = t
		}
	}
	if providerStr := r.URL.Query().Get("provider"); providerStr != "" {
		filter.Provider = domain.ProviderID(providerStr)
	}

	stats, err := s.tracker.GetProviderStats(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"providers": stats,
	})
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	models, err := s.pricing.ListModels(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"models":             models,
		"available_providers": s.tracker.ListAvailableProviders(),
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"status":    "ok",
		"providers": s.tracker.ListAvailableProviders(),
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
