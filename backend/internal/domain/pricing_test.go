package domain_test

import (
	"testing"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		name    string
		usage   domain.TokenUsage
		pricing domain.PricingEntry
		want    float64
	}{
		{
			name: "basic input and output",
			usage: domain.TokenUsage{
				InputTokens:  1000,
				OutputTokens: 500,
			},
			pricing: domain.PricingEntry{
				InputPricePerM:  3.00,
				OutputPricePerM: 15.00,
			},
			// 1000/1M * 3.00 + 500/1M * 15.00 = 0.003 + 0.0075 = 0.0105
			want: 0.0105,
		},
		{
			name: "with cache tokens",
			usage: domain.TokenUsage{
				InputTokens:      1000,
				OutputTokens:     500,
				CacheReadTokens:  200,
				CacheWriteTokens: 100,
			},
			pricing: domain.PricingEntry{
				InputPricePerM:      3.00,
				OutputPricePerM:     15.00,
				CacheReadPricePerM:  0.30,
				CacheWritePricePerM: 3.75,
			},
			// 0.003 + 0.0075 + 0.00006 + 0.000375 = 0.010935
			want: 0.010935,
		},
		{
			name: "zero usage",
			usage: domain.TokenUsage{
				InputTokens:  0,
				OutputTokens: 0,
			},
			pricing: domain.PricingEntry{
				InputPricePerM:  3.00,
				OutputPricePerM: 15.00,
			},
			want: 0.0,
		},
		{
			name: "zero pricing",
			usage: domain.TokenUsage{
				InputTokens:  1000,
				OutputTokens: 500,
			},
			pricing: domain.PricingEntry{},
			want:    0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.CalculateCost(tt.usage, tt.pricing)
			assert.InDelta(t, tt.want, got, 0.000001)
		})
	}
}

func TestTokenUsage_CacheHitRate(t *testing.T) {
	tests := []struct {
		name  string
		usage domain.TokenUsage
		want  float64
	}{
		{
			name:  "no cache activity",
			usage: domain.TokenUsage{},
			want:  0.0,
		},
		{
			name:  "all cache hits",
			usage: domain.TokenUsage{CacheReadTokens: 100, CacheWriteTokens: 0},
			want:  1.0,
		},
		{
			name:  "80% cache hit rate",
			usage: domain.TokenUsage{CacheReadTokens: 80, CacheWriteTokens: 20},
			want:  0.8,
		},
		{
			name:  "all cache misses (writes only)",
			usage: domain.TokenUsage{CacheReadTokens: 0, CacheWriteTokens: 100},
			want:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.usage.CacheHitRate()
			assert.InDelta(t, tt.want, got, 0.000001)
		})
	}
}

func TestTokenUsage_EffectiveContextUsed(t *testing.T) {
	u := domain.TokenUsage{
		InputTokens:      1000,
		CacheReadTokens:  200,
		CacheWriteTokens: 100,
	}
	assert.Equal(t, int64(1300), u.EffectiveContextUsed())
}

func TestNewContextWindowStatus(t *testing.T) {
	status := domain.NewContextWindowStatus(160_000, 200_000, "claude-sonnet-4-6")
	assert.Equal(t, float64(80), status.UtilizationPct)
	assert.True(t, status.WarningThreshold)
	assert.False(t, status.CriticalThreshold)
}
