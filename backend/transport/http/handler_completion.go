package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/service"
)

// completionRequest is the JSON body for POST /api/complete.
type completionRequest struct {
	Provider   string           `json:"provider"`
	Model      string           `json:"model"`
	Messages   []domain.Message `json:"messages"`
	MaxTokens  int              `json:"max_tokens"`
	SessionID  string           `json:"session_id"`
	ProjectTag string           `json:"project_tag,omitempty"`
	APIKey     string           `json:"api_key,omitempty"` // hashed for storage, not stored raw
}

// completionResponse is the JSON response from POST /api/complete.
type completionResponse struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Provider  string                 `json:"provider"`
	Model     string                 `json:"model"`
	Usage     domain.NormalizedUsage `json:"usage"`
	SessionID string                 `json:"session_id"`
}

func (s *Server) handleCompletion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req completionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Provider == "" {
		http.Error(w, "provider is required", http.StatusBadRequest)
		return
	}
	if req.Model == "" {
		http.Error(w, "model is required", http.StatusBadRequest)
		return
	}
	if len(req.Messages) == 0 {
		http.Error(w, "messages is required", http.StatusBadRequest)
		return
	}

	// Resolve API key: if it's a UUID (key_id), decrypt from KeyManager
	apiKey := req.APIKey
	if apiKey != "" && !isRawAPIKey(apiKey) {
		// Assume it's a key_id - decrypt from storage
		decrypted, err := s.keyManager.GetDecryptedKey(r.Context(), req.APIKey)
		if err != nil {
			http.Error(w, "invalid or expired API key", http.StatusUnauthorized)
			return
		}
		apiKey = decrypted
	}

	trackReq := service.TrackRequest{
		Provider:   domain.ProviderID(req.Provider),
		Model:      req.Model,
		Messages:   req.Messages,
		MaxTokens:  req.MaxTokens,
		SessionID:  req.SessionID,
		APIKey:     apiKey,
		ProjectTag: req.ProjectTag,
	}

	usage, err := s.tracker.TrackCompletion(r.Context(), trackReq)
	if err != nil {
		writeError(w, err)
		return
	}

	resp := completionResponse{
		ID:        usage.ID,
		Provider:  string(usage.Provider),
		Model:     usage.Model,
		Usage:     *usage,
		SessionID: usage.SessionID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// writeError maps domain errors to HTTP status codes.
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrInvalidAPIKey):
		status = http.StatusUnauthorized
	case errors.Is(err, domain.ErrRateLimited):
		status = http.StatusTooManyRequests
	case errors.Is(err, domain.ErrUnknownProvider):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrModelNotFound):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrContextExceeded):
		status = http.StatusUnprocessableEntity
	case errors.Is(err, domain.ErrProviderUnavailable):
		status = http.StatusServiceUnavailable
	}
	http.Error(w, err.Error(), status)
}

// isRawAPIKey returns true if the string looks like a raw API key (not a UUID).
func isRawAPIKey(key string) bool {
	// Raw API keys typically start with "sk-" (OpenAI, Anthropic, etc.)
	// UUIDs are 36 chars with hyphens: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
	return strings.HasPrefix(key, "sk-") || strings.HasPrefix(key, "api-")
}
