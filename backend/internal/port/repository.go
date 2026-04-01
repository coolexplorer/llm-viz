package port

import (
	"context"
	"errors"
	"time"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
)

// ErrKeyNotFound is returned when an API key does not exist in the repository.
var ErrKeyNotFound = errors.New("api key not found")

// KeyRepository is the outbound port for encrypted API key persistence.
type KeyRepository interface {
	Save(ctx context.Context, key domain.APIKey) error
	Get(ctx context.Context, id string) (*domain.APIKey, error)
	GetByHash(ctx context.Context, hash string) (*domain.APIKey, error)
	List(ctx context.Context, provider domain.ProviderID) ([]domain.APIKey, error)
	Delete(ctx context.Context, id string) error
}

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
