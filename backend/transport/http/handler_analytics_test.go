// Package http — analytics HTTP handler tests (Red phase, TDD).
//
// Tests use serveRequest (through the full router) so they compile without
// referencing non-existent handler methods.  They fail at runtime with 404
// until the analytics routes are registered in NewServer (Task #6).
//
// Routes expected after implementation:
//   GET /api/analytics/cumulative  → { "data": [...], "count": N }
//   GET /api/analytics/providers   → { "providers": {...} }
//   GET /api/analytics/models      → { "models": [...], "count": N }
//   GET /api/analytics/hourly      → { "buckets": [...], "count": N }
//
// Time-range query params: start, end (RFC3339), provider (optional)
package http

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
)

// ---------------------------------------------------------------------------
// /api/analytics/cumulative
// ---------------------------------------------------------------------------

func TestHandleAnalyticsCumulative_Success_EmptyData(t *testing.T) {
	srv, _, _ := defaultTestServer()

	rr := serveRequest(srv, "GET", "/api/analytics/cumulative", "")

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	_, hasData := resp["data"]
	assert.True(t, hasData, "response must contain 'data' field")
	_, hasCount := resp["count"]
	assert.True(t, hasCount, "response must contain 'count' field")
}

func TestHandleAnalyticsCumulative_WithData(t *testing.T) {
	srv, repo, _ := defaultTestServer()

	base := time.Now().UTC().Add(-2 * time.Hour)
	repo.saved = []domain.NormalizedUsage{
		{
			Timestamp: base,
			Provider:  domain.ProviderOpenAI,
			Usage:     domain.TokenUsage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150},
			CostUSD:   0.002,
		},
		{
			Timestamp: base.Add(time.Hour),
			Provider:  domain.ProviderOpenAI,
			Usage:     domain.TokenUsage{InputTokens: 200, OutputTokens: 100, TotalTokens: 300},
			CostUSD:   0.004,
		},
	}

	rr := serveRequest(srv, "GET", "/api/analytics/cumulative", "")

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	data, ok := resp["data"].([]any)
	require.True(t, ok)
	assert.Len(t, data, 2)
	assert.Equal(t, float64(2), resp["count"])
}

func TestHandleAnalyticsCumulative_MethodNotAllowed(t *testing.T) {
	srv, _, _ := defaultTestServer()

	for _, method := range []string{"POST", "PUT", "DELETE", "PATCH"} {
		t.Run(method, func(t *testing.T) {
			rr := serveRequest(srv, method, "/api/analytics/cumulative", "")
			assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
		})
	}
}

func TestHandleAnalyticsCumulative_WithStartFilter(t *testing.T) {
	srv, _, _ := defaultTestServer()

	start := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	rr := serveRequest(srv, "GET", "/api/analytics/cumulative?start="+start, "")

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleAnalyticsCumulative_WithEndFilter(t *testing.T) {
	srv, _, _ := defaultTestServer()

	end := time.Now().Format(time.RFC3339)
	rr := serveRequest(srv, "GET", "/api/analytics/cumulative?end="+end, "")

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleAnalyticsCumulative_WithBothFilters(t *testing.T) {
	srv, _, _ := defaultTestServer()

	start := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	end := time.Now().Format(time.RFC3339)
	rr := serveRequest(srv, "GET", "/api/analytics/cumulative?start="+start+"&end="+end, "")

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleAnalyticsCumulative_InvalidTimestamp_Ignored(t *testing.T) {
	srv, _, _ := defaultTestServer()

	// Malformed timestamps should be silently ignored (fall back to no filter).
	rr := serveRequest(srv, "GET", "/api/analytics/cumulative?start=not-a-time&end=also-bad", "")

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleAnalyticsCumulative_FutureDates_ReturnsEmptyData(t *testing.T) {
	srv, repo, _ := defaultTestServer()
	repo.saved = []domain.NormalizedUsage{
		{Timestamp: time.Now().Add(-time.Hour), Usage: domain.TokenUsage{TotalTokens: 100}},
	}

	future := time.Now().Add(time.Hour).Format(time.RFC3339)
	rr := serveRequest(srv, "GET", "/api/analytics/cumulative?start="+future, "")

	require.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	data := resp["data"].([]any)
	assert.Empty(t, data)
}

// ---------------------------------------------------------------------------
// /api/analytics/providers
// ---------------------------------------------------------------------------

func TestHandleAnalyticsProviders_Success_EmptyData(t *testing.T) {
	srv, _, _ := defaultTestServer()

	rr := serveRequest(srv, "GET", "/api/analytics/providers", "")

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	_, hasProviders := resp["providers"]
	assert.True(t, hasProviders, "response must contain 'providers' field")
}

func TestHandleAnalyticsProviders_WithData(t *testing.T) {
	srv, repo, _ := defaultTestServer()

	repo.saved = []domain.NormalizedUsage{
		{Provider: domain.ProviderOpenAI, Usage: domain.TokenUsage{InputTokens: 100, OutputTokens: 50}, CostUSD: 0.002},
		{Provider: domain.ProviderOpenAI, Usage: domain.TokenUsage{InputTokens: 200, OutputTokens: 80}, CostUSD: 0.004},
		{Provider: domain.ProviderAnthropic, Usage: domain.TokenUsage{InputTokens: 500, OutputTokens: 200}, CostUSD: 0.01},
	}

	rr := serveRequest(srv, "GET", "/api/analytics/providers", "")

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	providers, ok := resp["providers"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, providers, "openai")
	assert.Contains(t, providers, "anthropic")
}

func TestHandleAnalyticsProviders_ProviderStats_Fields(t *testing.T) {
	srv, repo, _ := defaultTestServer()

	repo.saved = []domain.NormalizedUsage{
		{Provider: domain.ProviderOpenAI, Usage: domain.TokenUsage{InputTokens: 100, OutputTokens: 50}, CostUSD: 0.002},
		{Provider: domain.ProviderOpenAI, Usage: domain.TokenUsage{InputTokens: 200, OutputTokens: 80}, CostUSD: 0.004},
	}

	rr := serveRequest(srv, "GET", "/api/analytics/providers", "")

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	providers, ok := resp["providers"].(map[string]any)
	require.True(t, ok)

	oai, ok := providers["openai"].(map[string]any)
	require.True(t, ok, "openai stats must be an object")
	// Verify all expected fields are present in the response.
	assert.Contains(t, oai, "provider")
	assert.Contains(t, oai, "total_input")
	assert.Contains(t, oai, "total_output")
	assert.Contains(t, oai, "total_cost")
	assert.Contains(t, oai, "request_count")
	assert.Contains(t, oai, "avg_cost_per_request")
	assert.Equal(t, float64(2), oai["request_count"])
}

func TestHandleAnalyticsProviders_MethodNotAllowed(t *testing.T) {
	srv, _, _ := defaultTestServer()

	for _, method := range []string{"POST", "PUT", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			rr := serveRequest(srv, method, "/api/analytics/providers", "")
			assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
		})
	}
}

func TestHandleAnalyticsProviders_WithTimeFilter(t *testing.T) {
	srv, _, _ := defaultTestServer()

	start := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	end := time.Now().Format(time.RFC3339)
	rr := serveRequest(srv, "GET", "/api/analytics/providers?start="+start+"&end="+end, "")

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleAnalyticsProviders_WithProviderFilter(t *testing.T) {
	srv, repo, _ := defaultTestServer()

	repo.saved = []domain.NormalizedUsage{
		{Provider: domain.ProviderOpenAI, Usage: domain.TokenUsage{InputTokens: 100}},
		{Provider: domain.ProviderAnthropic, Usage: domain.TokenUsage{InputTokens: 200}},
	}

	rr := serveRequest(srv, "GET", "/api/analytics/providers?provider=openai", "")

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleAnalyticsProviders_InvalidTimestampsIgnored(t *testing.T) {
	srv, _, _ := defaultTestServer()

	rr := serveRequest(srv, "GET", "/api/analytics/providers?start=bad&end=worse", "")

	require.Equal(t, http.StatusOK, rr.Code)
}

// ---------------------------------------------------------------------------
// /api/analytics/models
// ---------------------------------------------------------------------------

func TestHandleAnalyticsModels_Success_EmptyData(t *testing.T) {
	srv, _, _ := defaultTestServer()

	rr := serveRequest(srv, "GET", "/api/analytics/models", "")

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	_, hasModels := resp["models"]
	assert.True(t, hasModels, "response must contain 'models' field")
	_, hasCount := resp["count"]
	assert.True(t, hasCount, "response must contain 'count' field")
}

func TestHandleAnalyticsModels_WithData(t *testing.T) {
	srv, repo, _ := defaultTestServer()

	repo.saved = []domain.NormalizedUsage{
		{Provider: domain.ProviderOpenAI, Model: "gpt-4o", Usage: domain.TokenUsage{InputTokens: 100, OutputTokens: 50}, CostUSD: 0.002},
		{Provider: domain.ProviderOpenAI, Model: "gpt-4o", Usage: domain.TokenUsage{InputTokens: 200, OutputTokens: 80}, CostUSD: 0.004},
		{Provider: domain.ProviderAnthropic, Model: "claude-opus-4-6", Usage: domain.TokenUsage{InputTokens: 500, OutputTokens: 200}, CostUSD: 0.01},
	}

	rr := serveRequest(srv, "GET", "/api/analytics/models", "")

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	models, ok := resp["models"].([]any)
	require.True(t, ok)
	assert.Len(t, models, 2)
	assert.Equal(t, float64(2), resp["count"])
}

func TestHandleAnalyticsModels_ModelStats_Fields(t *testing.T) {
	srv, repo, _ := defaultTestServer()

	repo.saved = []domain.NormalizedUsage{
		{Provider: domain.ProviderOpenAI, Model: "gpt-4o", Usage: domain.TokenUsage{InputTokens: 100, OutputTokens: 50}, CostUSD: 0.002},
	}

	rr := serveRequest(srv, "GET", "/api/analytics/models", "")

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	models, ok := resp["models"].([]any)
	require.True(t, ok)
	require.Len(t, models, 1)

	m, ok := models[0].(map[string]any)
	require.True(t, ok)
	// Verify all expected fields are present.
	assert.Contains(t, m, "model")
	assert.Contains(t, m, "provider")
	assert.Contains(t, m, "avg_input_tokens")
	assert.Contains(t, m, "avg_output_tokens")
	assert.Contains(t, m, "total_cost")
	assert.Contains(t, m, "request_count")
}

func TestHandleAnalyticsModels_MethodNotAllowed(t *testing.T) {
	srv, _, _ := defaultTestServer()

	for _, method := range []string{"POST", "PUT", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			rr := serveRequest(srv, method, "/api/analytics/models", "")
			assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
		})
	}
}

func TestHandleAnalyticsModels_WithTimeFilter(t *testing.T) {
	srv, _, _ := defaultTestServer()

	start := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	rr := serveRequest(srv, "GET", "/api/analytics/models?start="+start, "")

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleAnalyticsModels_SortedByCostDescending(t *testing.T) {
	srv, repo, _ := defaultTestServer()

	repo.saved = []domain.NormalizedUsage{
		{Model: "cheap", CostUSD: 0.001},
		{Model: "expensive", CostUSD: 0.1},
		{Model: "mid", CostUSD: 0.01},
	}

	rr := serveRequest(srv, "GET", "/api/analytics/models", "")

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	models, ok := resp["models"].([]any)
	require.True(t, ok)
	require.Len(t, models, 3)

	first := models[0].(map[string]any)
	assert.Equal(t, "expensive", first["model"])
}

// ---------------------------------------------------------------------------
// /api/analytics/hourly
// ---------------------------------------------------------------------------

func TestHandleAnalyticsHourly_Success_EmptyData(t *testing.T) {
	srv, _, _ := defaultTestServer()

	rr := serveRequest(srv, "GET", "/api/analytics/hourly", "")

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	_, hasBuckets := resp["buckets"]
	assert.True(t, hasBuckets, "response must contain 'buckets' field")
	_, hasCount := resp["count"]
	assert.True(t, hasCount, "response must contain 'count' field")
}

func TestHandleAnalyticsHourly_WithData(t *testing.T) {
	srv, repo, _ := defaultTestServer()

	base := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	repo.saved = []domain.NormalizedUsage{
		{Timestamp: base.Add(10 * time.Minute), Usage: domain.TokenUsage{TotalTokens: 100}, CostUSD: 0.001},
		{Timestamp: base.Add(30 * time.Minute), Usage: domain.TokenUsage{TotalTokens: 200}, CostUSD: 0.002},
		{Timestamp: base.Add(time.Hour + 5*time.Minute), Usage: domain.TokenUsage{TotalTokens: 300}, CostUSD: 0.003},
	}

	rr := serveRequest(srv, "GET", "/api/analytics/hourly", "")

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	buckets, ok := resp["buckets"].([]any)
	require.True(t, ok)
	assert.Len(t, buckets, 2) // two distinct hours
}

func TestHandleAnalyticsHourly_BucketFields(t *testing.T) {
	srv, repo, _ := defaultTestServer()

	hour := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	repo.saved = []domain.NormalizedUsage{
		{Timestamp: hour.Add(15 * time.Minute), Usage: domain.TokenUsage{TotalTokens: 100}, CostUSD: 0.001},
	}

	rr := serveRequest(srv, "GET", "/api/analytics/hourly", "")

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	buckets, ok := resp["buckets"].([]any)
	require.True(t, ok)
	require.Len(t, buckets, 1)

	b, ok := buckets[0].(map[string]any)
	require.True(t, ok)
	// Verify all expected bucket fields.
	assert.Contains(t, b, "hour")
	assert.Contains(t, b, "request_count")
	assert.Contains(t, b, "total_tokens")
	assert.Contains(t, b, "total_cost")
	assert.Equal(t, float64(1), b["request_count"])
	assert.Equal(t, float64(100), b["total_tokens"])
}

func TestHandleAnalyticsHourly_MethodNotAllowed(t *testing.T) {
	srv, _, _ := defaultTestServer()

	for _, method := range []string{"POST", "PUT", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			rr := serveRequest(srv, method, "/api/analytics/hourly", "")
			assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
		})
	}
}

func TestHandleAnalyticsHourly_WithTimeFilter(t *testing.T) {
	srv, _, _ := defaultTestServer()

	start := time.Now().Add(-6 * time.Hour).Format(time.RFC3339)
	end := time.Now().Format(time.RFC3339)
	rr := serveRequest(srv, "GET", "/api/analytics/hourly?start="+start+"&end="+end, "")

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleAnalyticsHourly_WithProviderFilter(t *testing.T) {
	srv, repo, _ := defaultTestServer()

	base := time.Now().Add(-2 * time.Hour)
	repo.saved = []domain.NormalizedUsage{
		{Timestamp: base, Provider: domain.ProviderOpenAI, Usage: domain.TokenUsage{TotalTokens: 100}},
		{Timestamp: base.Add(30 * time.Minute), Provider: domain.ProviderAnthropic, Usage: domain.TokenUsage{TotalTokens: 200}},
	}

	rr := serveRequest(srv, "GET", "/api/analytics/hourly?provider=openai", "")

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleAnalyticsHourly_InvalidTimestampsIgnored(t *testing.T) {
	srv, _, _ := defaultTestServer()

	rr := serveRequest(srv, "GET", "/api/analytics/hourly?start=garbage&end=nonsense", "")

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleAnalyticsHourly_SortedByHourAscending(t *testing.T) {
	srv, repo, _ := defaultTestServer()

	// Records in reverse chronological order — response should be sorted ascending.
	repo.saved = []domain.NormalizedUsage{
		{Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), Usage: domain.TokenUsage{TotalTokens: 300}},
		{Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), Usage: domain.TokenUsage{TotalTokens: 100}},
		{Timestamp: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC), Usage: domain.TokenUsage{TotalTokens: 200}},
	}

	rr := serveRequest(srv, "GET", "/api/analytics/hourly", "")

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	buckets, ok := resp["buckets"].([]any)
	require.True(t, ok)
	require.Len(t, buckets, 3)

	// Parse and verify ascending order.
	b0 := buckets[0].(map[string]any)
	b1 := buckets[1].(map[string]any)
	b2 := buckets[2].(map[string]any)

	h0, _ := time.Parse(time.RFC3339, b0["hour"].(string))
	h1, _ := time.Parse(time.RFC3339, b1["hour"].(string))
	h2, _ := time.Parse(time.RFC3339, b2["hour"].(string))
	assert.True(t, h0.Before(h1))
	assert.True(t, h1.Before(h2))
}

// ---------------------------------------------------------------------------
// Full routing — ensure routes are registered and reachable
// ---------------------------------------------------------------------------

func TestRouting_AnalyticsCumulative(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := serveRequest(srv, "GET", "/api/analytics/cumulative", "")
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRouting_AnalyticsProviders(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := serveRequest(srv, "GET", "/api/analytics/providers", "")
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRouting_AnalyticsModels(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := serveRequest(srv, "GET", "/api/analytics/models", "")
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRouting_AnalyticsHourly(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := serveRequest(srv, "GET", "/api/analytics/hourly", "")
	assert.Equal(t, http.StatusOK, rr.Code)
}
