package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/service"
)

// handleUsageHistory handles GET /api/usage/history
func (s *Server) handleUsageHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		http.Error(w, "provider parameter required", http.StatusBadRequest)
		return
	}

	apiKey := r.URL.Query().Get("api_key")
	if apiKey == "" {
		http.Error(w, "api_key parameter required", http.StatusBadRequest)
		return
	}

	// Parse date range (default to last 30 days)
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	if startStr := r.URL.Query().Get("start_date"); startStr != "" {
		if parsed, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = parsed
		}
	}

	if endStr := r.URL.Query().Get("end_date"); endStr != "" {
		if parsed, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = parsed
		}
	}

	// Create usage history service
	usageService := service.NewUsageHistoryService()

	var result *domain.UsageHistoryResponse
	var err error

	switch provider {
	case "openai":
		result, err = usageService.GetOpenAIUsageHistory(r.Context(), apiKey, startDate, endDate)
	case "anthropic":
		result, err = usageService.GetAnthropicUsageHistory(r.Context(), apiKey, startDate, endDate)
	default:
		http.Error(w, "unsupported provider", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
