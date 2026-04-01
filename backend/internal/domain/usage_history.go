package domain

import "time"

// UsageHistoryPoint represents a single data point in usage history
type UsageHistoryPoint struct {
	Date         time.Time `json:"date"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	TotalTokens  int64     `json:"total_tokens"`
	CostUSD      float64   `json:"cost_usd"`
	RequestCount int       `json:"request_count,omitempty"`
}

// UsageHistoryResponse represents the response for historical usage data
type UsageHistoryResponse struct {
	Provider ProviderID           `json:"provider"`
	StartDate time.Time            `json:"start_date"`
	EndDate   time.Time            `json:"end_date"`
	DataPoints []UsageHistoryPoint `json:"data_points"`
	TotalCost  float64             `json:"total_cost"`
}
