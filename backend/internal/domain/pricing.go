package domain

import "time"

// PricingEntry holds per-model token pricing. Prices are in USD per million tokens.
type PricingEntry struct {
	Provider            ProviderID `json:"provider"`
	ModelID             string     `json:"model_id"`
	DisplayName         string     `json:"display_name"`
	InputPricePerM      float64    `json:"input_price_per_m"`
	OutputPricePerM     float64    `json:"output_price_per_m"`
	CacheReadPricePerM  float64    `json:"cache_read_price_per_m,omitempty"`
	CacheWritePricePerM float64    `json:"cache_write_price_per_m,omitempty"`
	ContextWindow       int64      `json:"context_window"`
	EffectiveDate       time.Time  `json:"effective_date"`
	IsActive            bool       `json:"is_active"`
}

// CalculateCost is a pure function that computes the USD cost for a given usage and pricing entry.
// No side effects — easily testable.
func CalculateCost(usage TokenUsage, p PricingEntry) float64 {
	perM := func(tokens int64, pricePerM float64) float64 {
		return float64(tokens) / 1_000_000 * pricePerM
	}
	return perM(usage.InputTokens, p.InputPricePerM) +
		perM(usage.OutputTokens, p.OutputPricePerM) +
		perM(usage.CacheReadTokens, p.CacheReadPricePerM) +
		perM(usage.CacheWriteTokens, p.CacheWritePricePerM)
}
