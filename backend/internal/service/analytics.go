package service

import (
	"context"
	"sort"
	"time"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/port"
)

// AnalyticsService provides aggregated analytics over usage data.
type AnalyticsService struct {
	repo port.UsageRepository
}

// NewAnalyticsService constructs an AnalyticsService.
func NewAnalyticsService(repo port.UsageRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

// CumulativeDataPoint holds a running cumulative sum of tokens and cost at a point in time.
type CumulativeDataPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	TotalTokens  int64     `json:"total_tokens"`
	TotalCost    float64   `json:"total_cost"`
	RequestCount int64     `json:"request_count"`
}

// ProviderStats holds aggregated token/cost stats for a single provider.
type ProviderStats struct {
	Provider          domain.ProviderID `json:"provider"`
	TotalInput        int64             `json:"total_input"`
	TotalOutput       int64             `json:"total_output"`
	TotalCost         float64           `json:"total_cost"`
	RequestCount      int64             `json:"request_count"`
	AvgCostPerRequest float64           `json:"avg_cost_per_request"`
}

// ModelPerformance holds per-model analytics data.
type ModelPerformance struct {
	Model           string            `json:"model"`
	Provider        domain.ProviderID `json:"provider"`
	AvgInputTokens  float64           `json:"avg_input_tokens"`
	AvgOutputTokens float64           `json:"avg_output_tokens"`
	TotalCost       float64           `json:"total_cost"`
	RequestCount    int64             `json:"request_count"`
}

// HourlyBucket holds request counts and token usage bucketed by hour.
type HourlyBucket struct {
	Hour         time.Time `json:"hour"`
	RequestCount int64     `json:"request_count"`
	TotalTokens  int64     `json:"total_tokens"`
	TotalCost    float64   `json:"total_cost"`
}

// GetCumulativeUsage returns time-ordered data points with running cumulative sums.
func (a *AnalyticsService) GetCumulativeUsage(ctx context.Context, filter port.TimeRangeFilter) ([]CumulativeDataPoint, error) {
	records, err := a.repo.FindByTimeRange(ctx, filter)
	if err != nil {
		return nil, err
	}

	records = applyTimeRangeFilter(records, filter)
	if len(records) == 0 {
		return nil, nil
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp.Before(records[j].Timestamp)
	})

	result := make([]CumulativeDataPoint, len(records))
	var totalTokens int64
	var totalCost float64
	var requestCount int64
	for i, rec := range records {
		totalTokens += rec.Usage.TotalTokens
		totalCost += rec.CostUSD
		requestCount++
		result[i] = CumulativeDataPoint{
			Timestamp:    rec.Timestamp,
			TotalTokens:  totalTokens,
			TotalCost:    totalCost,
			RequestCount: requestCount,
		}
	}
	return result, nil
}

// GetProviderComparison returns aggregated stats grouped by provider.
func (a *AnalyticsService) GetProviderComparison(ctx context.Context, filter port.TimeRangeFilter) (map[domain.ProviderID]ProviderStats, error) {
	records, err := a.repo.FindByTimeRange(ctx, filter)
	if err != nil {
		return nil, err
	}

	records = applyTimeRangeFilter(records, filter)
	if len(records) == 0 {
		return nil, nil
	}

	stats := make(map[domain.ProviderID]ProviderStats)
	for _, rec := range records {
		s := stats[rec.Provider]
		s.Provider = rec.Provider
		s.TotalInput += rec.Usage.InputTokens
		s.TotalOutput += rec.Usage.OutputTokens
		s.TotalCost += rec.CostUSD
		s.RequestCount++
		stats[rec.Provider] = s
	}
	for p, s := range stats {
		if s.RequestCount > 0 {
			s.AvgCostPerRequest = s.TotalCost / float64(s.RequestCount)
			stats[p] = s
		}
	}
	return stats, nil
}

type modelKey struct {
	model    string
	provider domain.ProviderID
}

// GetModelPerformance returns per-model analytics sorted by total cost descending.
func (a *AnalyticsService) GetModelPerformance(ctx context.Context, filter port.TimeRangeFilter) ([]ModelPerformance, error) {
	records, err := a.repo.FindByTimeRange(ctx, filter)
	if err != nil {
		return nil, err
	}

	records = applyTimeRangeFilter(records, filter)
	if len(records) == 0 {
		return nil, nil
	}

	type accumulator struct {
		provider    domain.ProviderID
		model       string
		totalInput  int64
		totalOutput int64
		totalCost   float64
		count       int64
	}

	acc := make(map[modelKey]*accumulator)
	for _, rec := range records {
		key := modelKey{model: rec.Model, provider: rec.Provider}
		if acc[key] == nil {
			acc[key] = &accumulator{provider: rec.Provider, model: rec.Model}
		}
		entry := acc[key]
		entry.totalInput += rec.Usage.InputTokens
		entry.totalOutput += rec.Usage.OutputTokens
		entry.totalCost += rec.CostUSD
		entry.count++
	}

	result := make([]ModelPerformance, 0, len(acc))
	for _, entry := range acc {
		result = append(result, ModelPerformance{
			Model:           entry.model,
			Provider:        entry.provider,
			AvgInputTokens:  float64(entry.totalInput) / float64(entry.count),
			AvgOutputTokens: float64(entry.totalOutput) / float64(entry.count),
			TotalCost:       entry.totalCost,
			RequestCount:    entry.count,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalCost > result[j].TotalCost
	})
	return result, nil
}

// GetHourlyUsage returns request counts bucketed by hour, sorted ascending.
func (a *AnalyticsService) GetHourlyUsage(ctx context.Context, filter port.TimeRangeFilter) ([]HourlyBucket, error) {
	records, err := a.repo.FindByTimeRange(ctx, filter)
	if err != nil {
		return nil, err
	}

	records = applyTimeRangeFilter(records, filter)
	if len(records) == 0 {
		return nil, nil
	}

	buckets := make(map[time.Time]*HourlyBucket)
	for _, rec := range records {
		hour := rec.Timestamp.Truncate(time.Hour)
		if buckets[hour] == nil {
			buckets[hour] = &HourlyBucket{Hour: hour}
		}
		b := buckets[hour]
		b.RequestCount++
		b.TotalTokens += rec.Usage.TotalTokens
		b.TotalCost += rec.CostUSD
	}

	result := make([]HourlyBucket, 0, len(buckets))
	for _, b := range buckets {
		result = append(result, *b)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Hour.Before(result[j].Hour)
	})
	return result, nil
}

// applyTimeRangeFilter filters records by the given time range and provider.
// Applied as a safety net in case the underlying repository doesn't fully support filtering.
func applyTimeRangeFilter(records []domain.NormalizedUsage, filter port.TimeRangeFilter) []domain.NormalizedUsage {
	if filter.Start.IsZero() && filter.End.IsZero() && filter.Provider == "" {
		return records
	}
	out := make([]domain.NormalizedUsage, 0, len(records))
	for _, rec := range records {
		if !filter.Start.IsZero() && rec.Timestamp.Before(filter.Start) {
			continue
		}
		if !filter.End.IsZero() && rec.Timestamp.After(filter.End) {
			continue
		}
		if filter.Provider != "" && rec.Provider != filter.Provider {
			continue
		}
		out = append(out, rec)
	}
	return out
}
