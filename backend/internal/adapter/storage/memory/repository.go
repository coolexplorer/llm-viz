package memory

import (
	"context"
	"sync"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/port"
)

// Repository is a thread-safe in-memory implementation of port.UsageRepository.
// Data resets on server restart — suitable for Phase 1-2.
type Repository struct {
	mu      sync.RWMutex
	records []domain.NormalizedUsage
}

// NewRepository creates an empty in-memory repository.
func NewRepository() *Repository {
	return &Repository{
		records: make([]domain.NormalizedUsage, 0),
	}
}

// Save appends a usage record to the in-memory store.
func (r *Repository) Save(_ context.Context, usage domain.NormalizedUsage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, usage)
	return nil
}

// FindBySession returns the most recent `limit` records for a session.
func (r *Repository) FindBySession(_ context.Context, sessionID string, limit int) ([]domain.NormalizedUsage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.NormalizedUsage
	// Iterate in reverse to get most recent first.
	for i := len(r.records) - 1; i >= 0; i-- {
		if r.records[i].SessionID == sessionID {
			result = append(result, r.records[i])
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// FindByTimeRange returns records matching the given filter.
func (r *Repository) FindByTimeRange(_ context.Context, filter port.TimeRangeFilter) ([]domain.NormalizedUsage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.NormalizedUsage
	for _, rec := range r.records {
		if !filter.Start.IsZero() && rec.Timestamp.Before(filter.Start) {
			continue
		}
		if !filter.End.IsZero() && rec.Timestamp.After(filter.End) {
			continue
		}
		if filter.Provider != "" && rec.Provider != filter.Provider {
			continue
		}
		if filter.SessionID != "" && rec.SessionID != filter.SessionID {
			continue
		}
		result = append(result, rec)
	}
	return result, nil
}

// SumByProvider aggregates token counts and costs grouped by provider.
func (r *Repository) SumByProvider(_ context.Context, filter port.TimeRangeFilter) (map[domain.ProviderID]domain.UsageSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sums := make(map[domain.ProviderID]domain.UsageSummary)
	for _, rec := range r.records {
		if !filter.Start.IsZero() && rec.Timestamp.Before(filter.Start) {
			continue
		}
		if !filter.End.IsZero() && rec.Timestamp.After(filter.End) {
			continue
		}
		if filter.Provider != "" && rec.Provider != filter.Provider {
			continue
		}

		s := sums[rec.Provider]
		s.Provider = rec.Provider
		s.TotalInput += rec.Usage.InputTokens
		s.TotalOutput += rec.Usage.OutputTokens
		s.TotalCost += rec.CostUSD
		s.RequestCount++
		sums[rec.Provider] = s
	}
	return sums, nil
}
