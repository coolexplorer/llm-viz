package port

import (
	"context"
	"time"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
)

// TimeRangeFilter narrows repository queries by time, provider, and session.
type TimeRangeFilter struct {
	Start     time.Time
	End       time.Time
	Provider  domain.ProviderID // optional, empty = all providers
	SessionID string            // optional
}

// UsageRepository is the outbound port for persisting token usage records.
type UsageRepository interface {
	Save(ctx context.Context, usage domain.NormalizedUsage) error
	FindBySession(ctx context.Context, sessionID string, limit int) ([]domain.NormalizedUsage, error)
	FindByTimeRange(ctx context.Context, filter TimeRangeFilter) ([]domain.NormalizedUsage, error)
	// SumByProvider returns aggregated counts grouped by provider.
	SumByProvider(ctx context.Context, filter TimeRangeFilter) (map[domain.ProviderID]domain.UsageSummary, error)
}

// PricingRepository is the outbound port for model pricing data.
type PricingRepository interface {
	GetPricing(ctx context.Context, provider domain.ProviderID, modelID string) (*domain.PricingEntry, error)
	ListModels(ctx context.Context) ([]domain.ModelInfo, error)
}
