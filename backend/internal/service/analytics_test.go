// Package service_test — analytics service tests (Red phase, TDD).
//
// NOTE: This file will cause a compilation failure until service.AnalyticsService
// is implemented (Task #6). That is intentional — implement the analytics service
// to make these tests compile and pass (Green phase).
//
// Expected types to be defined in package service:
//
//	AnalyticsService struct
//	NewAnalyticsService(repo port.UsageRepository) *AnalyticsService
//	CumulativeDataPoint  { Timestamp, TotalTokens, TotalCost, RequestCount }
//	ProviderStats        { Provider, TotalInput, TotalOutput, TotalCost, RequestCount, AvgCostPerRequest }
//	ModelPerformance     { Model, Provider, AvgInputTokens, AvgOutputTokens, TotalCost, RequestCount }
//	HourlyBucket        { Hour, RequestCount, TotalTokens, TotalCost }
package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/port"
	"github.com/kimseunghwan/llm-viz/backend/internal/service"
)

// ---------------------------------------------------------------------------
// analyticsRepo — time-range-aware in-memory mock for analytics tests.
// (Distinct from mockRepo in tracker_test.go which doesn't filter by time.)
// ---------------------------------------------------------------------------

type analyticsRepo struct {
	records []domain.NormalizedUsage
	findErr error
}

func (r *analyticsRepo) Save(_ context.Context, _ domain.NormalizedUsage) error { return nil }

func (r *analyticsRepo) FindBySession(_ context.Context, _ string, _ int) ([]domain.NormalizedUsage, error) {
	return nil, nil
}

func (r *analyticsRepo) FindByTimeRange(_ context.Context, filter port.TimeRangeFilter) ([]domain.NormalizedUsage, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	var out []domain.NormalizedUsage
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
		out = append(out, rec)
	}
	return out, nil
}

func (r *analyticsRepo) SumByProvider(_ context.Context, _ port.TimeRangeFilter) (map[domain.ProviderID]domain.UsageSummary, error) {
	return nil, nil
}

// newAnalyticsSvc creates an AnalyticsService with a properly-filtering in-memory repo.
func newAnalyticsSvc(records []domain.NormalizedUsage) *service.AnalyticsService {
	return service.NewAnalyticsService(&analyticsRepo{records: records})
}

// ---------------------------------------------------------------------------
// GetCumulativeUsage — time-series with running cumulative sums
// ---------------------------------------------------------------------------

func TestGetCumulativeUsage_EmptyData(t *testing.T) {
	svc := newAnalyticsSvc(nil)

	result, err := svc.GetCumulativeUsage(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetCumulativeUsage_SingleRecord(t *testing.T) {
	ts := time.Date(2024, 1, 1, 10, 30, 0, 0, time.UTC)
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{
			Timestamp: ts,
			Provider:  domain.ProviderOpenAI,
			Usage:     domain.TokenUsage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150},
			CostUSD:   0.005,
		},
	})

	result, err := svc.GetCumulativeUsage(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, ts, result[0].Timestamp)
	assert.Equal(t, int64(150), result[0].TotalTokens)
	assert.InDelta(t, 0.005, result[0].TotalCost, 0.0001)
	assert.Equal(t, int64(1), result[0].RequestCount)
}

func TestGetCumulativeUsage_MultipleRecords_RunningSum(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Timestamp: base, Usage: domain.TokenUsage{TotalTokens: 100}, CostUSD: 0.001},
		{Timestamp: base.Add(time.Hour), Usage: domain.TokenUsage{TotalTokens: 200}, CostUSD: 0.002},
		{Timestamp: base.Add(2 * time.Hour), Usage: domain.TokenUsage{TotalTokens: 300}, CostUSD: 0.003},
	})

	result, err := svc.GetCumulativeUsage(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	require.Len(t, result, 3)
	// Each data point holds a running sum, not just its own value.
	assert.Equal(t, int64(100), result[0].TotalTokens)
	assert.Equal(t, int64(1), result[0].RequestCount)
	assert.Equal(t, int64(300), result[1].TotalTokens)
	assert.Equal(t, int64(2), result[1].RequestCount)
	assert.Equal(t, int64(600), result[2].TotalTokens)
	assert.Equal(t, int64(3), result[2].RequestCount)
}

func TestGetCumulativeUsage_CostRunningSum(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Timestamp: base, CostUSD: 0.001},
		{Timestamp: base.Add(time.Hour), CostUSD: 0.002},
		{Timestamp: base.Add(2 * time.Hour), CostUSD: 0.003},
	})

	result, err := svc.GetCumulativeUsage(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	require.Len(t, result, 3)
	assert.InDelta(t, 0.001, result[0].TotalCost, 0.0001)
	assert.InDelta(t, 0.003, result[1].TotalCost, 0.0001)
	assert.InDelta(t, 0.006, result[2].TotalCost, 0.0001)
}

func TestGetCumulativeUsage_SortedByTimestampAscending(t *testing.T) {
	// Records inserted in reverse chronological order — output must be sorted.
	t3 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	t1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Timestamp: t3, Usage: domain.TokenUsage{TotalTokens: 300}},
		{Timestamp: t1, Usage: domain.TokenUsage{TotalTokens: 100}},
		{Timestamp: t2, Usage: domain.TokenUsage{TotalTokens: 200}},
	})

	result, err := svc.GetCumulativeUsage(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	require.Len(t, result, 3)
	assert.True(t, result[0].Timestamp.Before(result[1].Timestamp))
	assert.True(t, result[1].Timestamp.Before(result[2].Timestamp))
}

func TestGetCumulativeUsage_TimeRangeFilter(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Timestamp: base, Usage: domain.TokenUsage{TotalTokens: 100}, CostUSD: 0.001},
		{Timestamp: base.Add(2 * time.Hour), Usage: domain.TokenUsage{TotalTokens: 200}, CostUSD: 0.002},
		{Timestamp: base.Add(4 * time.Hour), Usage: domain.TokenUsage{TotalTokens: 300}, CostUSD: 0.003},
	})

	filter := port.TimeRangeFilter{
		Start: base.Add(time.Hour),
		End:   base.Add(3 * time.Hour),
	}
	result, err := svc.GetCumulativeUsage(context.Background(), filter)

	require.NoError(t, err)
	// Only the record at base+2h falls within [base+1h, base+3h].
	require.Len(t, result, 1)
	assert.Equal(t, int64(200), result[0].TotalTokens)
}

func TestGetCumulativeUsage_FutureFilter_ReturnsEmpty(t *testing.T) {
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Timestamp: time.Now().Add(-time.Hour), Usage: domain.TokenUsage{TotalTokens: 100}},
	})

	filter := port.TimeRangeFilter{
		Start: time.Now().Add(time.Hour),
		End:   time.Now().Add(2 * time.Hour),
	}
	result, err := svc.GetCumulativeUsage(context.Background(), filter)

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetCumulativeUsage_RepositoryError(t *testing.T) {
	repo := &analyticsRepo{findErr: errors.New("db unavailable")}
	svc := service.NewAnalyticsService(repo)

	_, err := svc.GetCumulativeUsage(context.Background(), port.TimeRangeFilter{})

	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetProviderComparison — aggregated cost/request stats per provider
// ---------------------------------------------------------------------------

func TestGetProviderComparison_EmptyData(t *testing.T) {
	svc := newAnalyticsSvc(nil)

	result, err := svc.GetProviderComparison(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetProviderComparison_SingleProvider_Aggregates(t *testing.T) {
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Provider: domain.ProviderOpenAI, Usage: domain.TokenUsage{InputTokens: 100, OutputTokens: 50}, CostUSD: 0.002},
		{Provider: domain.ProviderOpenAI, Usage: domain.TokenUsage{InputTokens: 200, OutputTokens: 80}, CostUSD: 0.004},
	})

	result, err := svc.GetProviderComparison(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	require.Contains(t, result, domain.ProviderOpenAI)
	oai := result[domain.ProviderOpenAI]
	assert.Equal(t, domain.ProviderOpenAI, oai.Provider)
	assert.Equal(t, int64(300), oai.TotalInput)
	assert.Equal(t, int64(130), oai.TotalOutput)
	assert.InDelta(t, 0.006, oai.TotalCost, 0.0001)
	assert.Equal(t, int64(2), oai.RequestCount)
	assert.InDelta(t, 0.003, oai.AvgCostPerRequest, 0.0001)
}

func TestGetProviderComparison_MultipleProviders(t *testing.T) {
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Provider: domain.ProviderOpenAI, Usage: domain.TokenUsage{InputTokens: 100, OutputTokens: 50}, CostUSD: 0.002},
		{Provider: domain.ProviderAnthropic, Usage: domain.TokenUsage{InputTokens: 500, OutputTokens: 200}, CostUSD: 0.01},
		{Provider: domain.ProviderAnthropic, Usage: domain.TokenUsage{InputTokens: 300, OutputTokens: 100}, CostUSD: 0.005},
	})

	result, err := svc.GetProviderComparison(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	assert.Len(t, result, 2)

	ant := result[domain.ProviderAnthropic]
	assert.Equal(t, int64(800), ant.TotalInput)
	assert.Equal(t, int64(300), ant.TotalOutput)
	assert.Equal(t, int64(2), ant.RequestCount)
	assert.InDelta(t, 0.015, ant.TotalCost, 0.0001)
	assert.InDelta(t, 0.0075, ant.AvgCostPerRequest, 0.0001)
}

func TestGetProviderComparison_ProviderFilter(t *testing.T) {
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Provider: domain.ProviderOpenAI, Usage: domain.TokenUsage{InputTokens: 100}, CostUSD: 0.002},
		{Provider: domain.ProviderAnthropic, Usage: domain.TokenUsage{InputTokens: 500}, CostUSD: 0.01},
	})

	filter := port.TimeRangeFilter{Provider: domain.ProviderOpenAI}
	result, err := svc.GetProviderComparison(context.Background(), filter)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Contains(t, result, domain.ProviderOpenAI)
	assert.NotContains(t, result, domain.ProviderAnthropic)
}

func TestGetProviderComparison_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		records      []domain.NormalizedUsage
		wantProviders []domain.ProviderID
		wantCounts   map[domain.ProviderID]int64
	}{
		{
			name:         "no data",
			records:      nil,
			wantProviders: nil,
		},
		{
			name: "three providers",
			records: []domain.NormalizedUsage{
				{Provider: domain.ProviderOpenAI},
				{Provider: domain.ProviderAnthropic},
				{Provider: domain.ProviderGemini},
			},
			wantProviders: []domain.ProviderID{domain.ProviderOpenAI, domain.ProviderAnthropic, domain.ProviderGemini},
			wantCounts: map[domain.ProviderID]int64{
				domain.ProviderOpenAI:    1,
				domain.ProviderAnthropic: 1,
				domain.ProviderGemini:    1,
			},
		},
		{
			name: "one provider multiple requests",
			records: []domain.NormalizedUsage{
				{Provider: domain.ProviderGroq},
				{Provider: domain.ProviderGroq},
				{Provider: domain.ProviderGroq},
			},
			wantProviders: []domain.ProviderID{domain.ProviderGroq},
			wantCounts: map[domain.ProviderID]int64{
				domain.ProviderGroq: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newAnalyticsSvc(tt.records)
			result, err := svc.GetProviderComparison(context.Background(), port.TimeRangeFilter{})
			require.NoError(t, err)
			assert.Len(t, result, len(tt.wantProviders))
			for provider, wantCount := range tt.wantCounts {
				assert.Equal(t, wantCount, result[provider].RequestCount)
			}
		})
	}
}

func TestGetProviderComparison_RepositoryError(t *testing.T) {
	repo := &analyticsRepo{findErr: errors.New("timeout")}
	svc := service.NewAnalyticsService(repo)

	_, err := svc.GetProviderComparison(context.Background(), port.TimeRangeFilter{})

	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetModelPerformance — avg tokens and cost per model, sorted by cost desc
// ---------------------------------------------------------------------------

func TestGetModelPerformance_EmptyData(t *testing.T) {
	svc := newAnalyticsSvc(nil)

	result, err := svc.GetModelPerformance(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetModelPerformance_SingleModel_Averages(t *testing.T) {
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Provider: domain.ProviderOpenAI, Model: "gpt-4o", Usage: domain.TokenUsage{InputTokens: 100, OutputTokens: 50}, CostUSD: 0.002},
		{Provider: domain.ProviderOpenAI, Model: "gpt-4o", Usage: domain.TokenUsage{InputTokens: 200, OutputTokens: 100}, CostUSD: 0.004},
	})

	result, err := svc.GetModelPerformance(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	require.Len(t, result, 1)
	m := result[0]
	assert.Equal(t, "gpt-4o", m.Model)
	assert.Equal(t, domain.ProviderOpenAI, m.Provider)
	assert.InDelta(t, 150.0, m.AvgInputTokens, 0.01)  // (100+200)/2
	assert.InDelta(t, 75.0, m.AvgOutputTokens, 0.01)  // (50+100)/2
	assert.Equal(t, int64(2), m.RequestCount)
	assert.InDelta(t, 0.006, m.TotalCost, 0.0001)
}

func TestGetModelPerformance_AverageCalculation_ThreeRequests(t *testing.T) {
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Model: "model-a", Usage: domain.TokenUsage{InputTokens: 100, OutputTokens: 50}},
		{Model: "model-a", Usage: domain.TokenUsage{InputTokens: 300, OutputTokens: 150}},
		{Model: "model-a", Usage: domain.TokenUsage{InputTokens: 200, OutputTokens: 100}},
	})

	result, err := svc.GetModelPerformance(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.InDelta(t, 200.0, result[0].AvgInputTokens, 0.01)  // (100+300+200)/3
	assert.InDelta(t, 100.0, result[0].AvgOutputTokens, 0.01) // (50+150+100)/3
}

func TestGetModelPerformance_MultipleModels(t *testing.T) {
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Provider: domain.ProviderOpenAI, Model: "gpt-4o", Usage: domain.TokenUsage{InputTokens: 100, OutputTokens: 50}, CostUSD: 0.002},
		{Provider: domain.ProviderAnthropic, Model: "claude-opus-4-6", Usage: domain.TokenUsage{InputTokens: 500, OutputTokens: 200}, CostUSD: 0.01},
		{Provider: domain.ProviderOpenAI, Model: "gpt-4o-mini", Usage: domain.TokenUsage{InputTokens: 50, OutputTokens: 30}, CostUSD: 0.0001},
	})

	result, err := svc.GetModelPerformance(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestGetModelPerformance_SortedByTotalCostDescending(t *testing.T) {
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Model: "cheap-model", CostUSD: 0.001},
		{Model: "expensive-model", CostUSD: 0.1},
		{Model: "mid-model", CostUSD: 0.01},
	})

	result, err := svc.GetModelPerformance(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	require.Len(t, result, 3)
	assert.Equal(t, "expensive-model", result[0].Model)
	assert.Equal(t, "mid-model", result[1].Model)
	assert.Equal(t, "cheap-model", result[2].Model)
}

func TestGetModelPerformance_SameModelDifferentProviders_Separate(t *testing.T) {
	// "gpt-4o" from openai vs openrouter should be tracked separately.
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Provider: domain.ProviderOpenAI, Model: "gpt-4o", CostUSD: 0.002},
		{Provider: domain.ProviderOpenRouter, Model: "gpt-4o", CostUSD: 0.001},
	})

	result, err := svc.GetModelPerformance(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGetModelPerformance_RepositoryError(t *testing.T) {
	repo := &analyticsRepo{findErr: errors.New("connection reset")}
	svc := service.NewAnalyticsService(repo)

	_, err := svc.GetModelPerformance(context.Background(), port.TimeRangeFilter{})

	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetHourlyUsage — request count bucketed by hour of day
// ---------------------------------------------------------------------------

func TestGetHourlyUsage_EmptyData(t *testing.T) {
	svc := newAnalyticsSvc(nil)

	result, err := svc.GetHourlyUsage(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetHourlyUsage_SameHour_SingleBucket(t *testing.T) {
	hour := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Timestamp: hour.Add(10 * time.Minute), Usage: domain.TokenUsage{TotalTokens: 100}, CostUSD: 0.001},
		{Timestamp: hour.Add(30 * time.Minute), Usage: domain.TokenUsage{TotalTokens: 200}, CostUSD: 0.002},
		{Timestamp: hour.Add(59 * time.Minute), Usage: domain.TokenUsage{TotalTokens: 300}, CostUSD: 0.003},
	})

	result, err := svc.GetHourlyUsage(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	require.Len(t, result, 1)
	bucket := result[0]
	assert.Equal(t, hour, bucket.Hour)
	assert.Equal(t, int64(3), bucket.RequestCount)
	assert.Equal(t, int64(600), bucket.TotalTokens)
	assert.InDelta(t, 0.006, bucket.TotalCost, 0.0001)
}

func TestGetHourlyUsage_DifferentHours_MultipleBuckets(t *testing.T) {
	base := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Timestamp: base.Add(10 * time.Minute), Usage: domain.TokenUsage{TotalTokens: 100}},
		{Timestamp: base.Add(time.Hour + 10*time.Minute), Usage: domain.TokenUsage{TotalTokens: 200}},
		{Timestamp: base.Add(2*time.Hour + 10*time.Minute), Usage: domain.TokenUsage{TotalTokens: 300}},
	})

	result, err := svc.GetHourlyUsage(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	require.Len(t, result, 3)
	assert.Equal(t, base, result[0].Hour)
	assert.Equal(t, base.Add(time.Hour), result[1].Hour)
	assert.Equal(t, base.Add(2*time.Hour), result[2].Hour)
}

func TestGetHourlyUsage_SortedByHourAscending(t *testing.T) {
	// Records inserted in reverse chronological order — output must be sorted ascending.
	t3 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	t1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Timestamp: t3, Usage: domain.TokenUsage{TotalTokens: 300}},
		{Timestamp: t1, Usage: domain.TokenUsage{TotalTokens: 100}},
		{Timestamp: t2, Usage: domain.TokenUsage{TotalTokens: 200}},
	})

	result, err := svc.GetHourlyUsage(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	require.Len(t, result, 3)
	assert.True(t, result[0].Hour.Before(result[1].Hour))
	assert.True(t, result[1].Hour.Before(result[2].Hour))
}

func TestGetHourlyUsage_BucketTimeTruncatedToHour(t *testing.T) {
	// A record at 14:37:22 must produce a bucket at 14:00:00.
	ts := time.Date(2024, 1, 15, 14, 37, 22, 0, time.UTC)
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Timestamp: ts, Usage: domain.TokenUsage{TotalTokens: 100}},
	})

	result, err := svc.GetHourlyUsage(context.Background(), port.TimeRangeFilter{})

	require.NoError(t, err)
	require.Len(t, result, 1)
	expected := time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, result[0].Hour)
}

func TestGetHourlyUsage_TimeRangeFilter(t *testing.T) {
	base := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	svc := newAnalyticsSvc([]domain.NormalizedUsage{
		{Timestamp: base.Add(10 * time.Minute), Usage: domain.TokenUsage{TotalTokens: 100}},
		{Timestamp: base.Add(time.Hour + 10*time.Minute), Usage: domain.TokenUsage{TotalTokens: 200}},
		{Timestamp: base.Add(2*time.Hour + 10*time.Minute), Usage: domain.TokenUsage{TotalTokens: 300}},
	})

	// Only the record at base+1h10m (11:10) falls within [10:30, 11:30].
	filter := port.TimeRangeFilter{
		Start: base.Add(30 * time.Minute),
		End:   base.Add(90 * time.Minute),
	}
	result, err := svc.GetHourlyUsage(context.Background(), filter)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, base.Add(time.Hour), result[0].Hour)
	assert.Equal(t, int64(1), result[0].RequestCount)
	assert.Equal(t, int64(200), result[0].TotalTokens)
}

func TestGetHourlyUsage_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		records     []domain.NormalizedUsage
		wantBuckets int
	}{
		{
			name:        "empty data",
			records:     nil,
			wantBuckets: 0,
		},
		{
			name: "all in same hour",
			records: []domain.NormalizedUsage{
				{Timestamp: time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC)},
				{Timestamp: time.Date(2024, 6, 1, 9, 30, 0, 0, time.UTC)},
				{Timestamp: time.Date(2024, 6, 1, 9, 59, 0, 0, time.UTC)},
			},
			wantBuckets: 1,
		},
		{
			name: "spans three days same hour",
			records: []domain.NormalizedUsage{
				{Timestamp: time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC)},
				{Timestamp: time.Date(2024, 6, 2, 9, 0, 0, 0, time.UTC)},
				{Timestamp: time.Date(2024, 6, 3, 9, 0, 0, 0, time.UTC)},
			},
			wantBuckets: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newAnalyticsSvc(tt.records)
			result, err := svc.GetHourlyUsage(context.Background(), port.TimeRangeFilter{})
			require.NoError(t, err)
			assert.Len(t, result, tt.wantBuckets)
		})
	}
}

func TestGetHourlyUsage_RepositoryError(t *testing.T) {
	repo := &analyticsRepo{findErr: errors.New("query failed")}
	svc := service.NewAnalyticsService(repo)

	_, err := svc.GetHourlyUsage(context.Background(), port.TimeRangeFilter{})

	require.Error(t, err)
}
