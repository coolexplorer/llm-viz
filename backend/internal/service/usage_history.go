package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
)

// UsageHistoryService fetches historical usage data from providers
type UsageHistoryService struct {
	httpClient *http.Client
}

// NewUsageHistoryService creates a new usage history service
func NewUsageHistoryService() *UsageHistoryService {
	return &UsageHistoryService{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// OpenAIUsageResponse represents the OpenAI /v1/usage API response
type OpenAIUsageResponse struct {
	Object string `json:"object"`
	Data   []struct {
		AggregationTimestamp int64   `json:"aggregation_timestamp"`
		NumModelRequests     int     `json:"n_requests"`
		Operation            string  `json:"operation"`
		SnapshotID           string  `json:"snapshot_id"`
		NumTokens            int64   `json:"n_context_tokens_total"`
		NumGeneratedTokens   int64   `json:"n_generated_tokens_total"`
		Cost                 float64 `json:"cost,omitempty"`
	} `json:"data"`
}

// GetOpenAIUsageHistory fetches usage history from OpenAI
func (s *UsageHistoryService) GetOpenAIUsageHistory(
	ctx context.Context,
	apiKey string,
	startDate, endDate time.Time,
) (*domain.UsageHistoryResponse, error) {
	// Format dates as Unix timestamps
	startTime := startDate.Unix()
	endTime := endDate.Unix()

	url := fmt.Sprintf(
		"https://api.openai.com/v1/usage?start_time=%d&end_time=%d",
		startTime, endTime,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var usageResp OpenAIUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usageResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Convert to domain model
	dataPoints := make([]domain.UsageHistoryPoint, 0, len(usageResp.Data))
	var totalCost float64

	for _, d := range usageResp.Data {
		point := domain.UsageHistoryPoint{
			Date:         time.Unix(d.AggregationTimestamp, 0),
			InputTokens:  d.NumTokens,
			OutputTokens: d.NumGeneratedTokens,
			TotalTokens:  d.NumTokens + d.NumGeneratedTokens,
			CostUSD:      d.Cost,
			RequestCount: d.NumModelRequests,
		}
		dataPoints = append(dataPoints, point)
		totalCost += d.Cost
	}

	return &domain.UsageHistoryResponse{
		Provider:   domain.ProviderOpenAI,
		StartDate:  startDate,
		EndDate:    endDate,
		DataPoints: dataPoints,
		TotalCost:  totalCost,
	}, nil
}

// AnthropicUsageResponse represents the Anthropic usage API response
type AnthropicUsageResponse struct {
	Data []struct {
		Date         string  `json:"date"`
		InputTokens  int64   `json:"input_tokens"`
		OutputTokens int64   `json:"output_tokens"`
		CostUSD      float64 `json:"cost_usd"`
	} `json:"data"`
}

// GetAnthropicUsageHistory fetches usage history from Anthropic
func (s *UsageHistoryService) GetAnthropicUsageHistory(
	ctx context.Context,
	apiKey string,
	startDate, endDate time.Time,
) (*domain.UsageHistoryResponse, error) {
	// Anthropic requires Admin API key
	// For now, return a placeholder response
	// TODO: Implement once Admin API key is available
	return &domain.UsageHistoryResponse{
		Provider:   domain.ProviderAnthropic,
		StartDate:  startDate,
		EndDate:    endDate,
		DataPoints: []domain.UsageHistoryPoint{},
		TotalCost:  0,
	}, nil
}
