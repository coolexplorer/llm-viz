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
// handleStats dispatch
// ---------------------------------------------------------------------------

func TestHandleStats_MethodNotAllowed(t *testing.T) {
	srv, _, _ := defaultTestServer()
	for _, method := range []string{"POST", "PUT", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			rr := callHandler(srv, srv.handleStats, method, "/api/stats", "")
			assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
		})
	}
}

// ---------------------------------------------------------------------------
// handleSessionStats
// ---------------------------------------------------------------------------

func TestHandleStats_SessionID_ReturnsHistory(t *testing.T) {
	srv, repo, _ := defaultTestServer()

	// Pre-seed a usage record
	repo.saved = []domain.NormalizedUsage{
		{ID: "u1", SessionID: "sess-abc", Provider: domain.ProviderOpenAI, Model: "gpt-4o"},
		{ID: "u2", SessionID: "sess-abc", Provider: domain.ProviderOpenAI, Model: "gpt-4o"},
		{ID: "u3", SessionID: "sess-other", Provider: domain.ProviderAnthropic, Model: "claude-opus-4-6"},
	}

	rr := callHandler(srv, srv.handleStats, "GET", "/api/stats?session_id=sess-abc", "")
	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "sess-abc", resp["session_id"])
	records, ok := resp["records"].([]any)
	require.True(t, ok)
	assert.Len(t, records, 2)
	assert.Equal(t, float64(2), resp["count"])
}

func TestHandleStats_SessionID_WithLimit(t *testing.T) {
	srv, repo, _ := defaultTestServer()

	for i := 0; i < 5; i++ {
		repo.saved = append(repo.saved, domain.NormalizedUsage{
			ID:        "u" + string(rune('0'+i)),
			SessionID: "s1",
		})
	}

	rr := callHandler(srv, srv.handleStats, "GET", "/api/stats?session_id=s1&limit=2", "")
	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	records := resp["records"].([]any)
	assert.Len(t, records, 2)
}

func TestHandleStats_SessionID_InvalidLimitIgnored(t *testing.T) {
	srv, _, _ := defaultTestServer()

	// Invalid limit should fall back to the default (100)
	rr := callHandler(srv, srv.handleStats, "GET", "/api/stats?session_id=s1&limit=notanumber", "")
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleStats_SessionID_EmptyResults(t *testing.T) {
	srv, _, _ := defaultTestServer()

	rr := callHandler(srv, srv.handleStats, "GET", "/api/stats?session_id=nonexistent", "")
	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, float64(0), resp["count"])
}

// ---------------------------------------------------------------------------
// handleProviderStats
// ---------------------------------------------------------------------------

func TestHandleStats_ProviderStats(t *testing.T) {
	srv, repo, _ := defaultTestServer()

	repo.saved = []domain.NormalizedUsage{
		{Provider: domain.ProviderOpenAI, Usage: domain.TokenUsage{InputTokens: 100, OutputTokens: 50}, CostUSD: 0.001},
		{Provider: domain.ProviderOpenAI, Usage: domain.TokenUsage{InputTokens: 200, OutputTokens: 80}, CostUSD: 0.002},
		{Provider: domain.ProviderAnthropic, Usage: domain.TokenUsage{InputTokens: 500, OutputTokens: 200}, CostUSD: 0.01},
	}

	rr := callHandler(srv, srv.handleStats, "GET", "/api/stats", "")
	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	providers, ok := resp["providers"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, providers, "openai")
	assert.Contains(t, providers, "anthropic")
}

func TestHandleStats_ProviderStats_WithStartFilter(t *testing.T) {
	srv, _, _ := defaultTestServer()

	start := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	rr := callHandler(srv, srv.handleStats, "GET", "/api/stats?start="+start, "")
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleStats_ProviderStats_WithEndFilter(t *testing.T) {
	srv, _, _ := defaultTestServer()

	end := time.Now().Format(time.RFC3339)
	rr := callHandler(srv, srv.handleStats, "GET", "/api/stats?end="+end, "")
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleStats_ProviderStats_WithProviderFilter(t *testing.T) {
	srv, repo, _ := defaultTestServer()

	repo.saved = []domain.NormalizedUsage{
		{Provider: domain.ProviderOpenAI, Usage: domain.TokenUsage{InputTokens: 100}},
		{Provider: domain.ProviderAnthropic, Usage: domain.TokenUsage{InputTokens: 200}},
	}

	rr := callHandler(srv, srv.handleStats, "GET", "/api/stats?provider=openai", "")
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleStats_ProviderStats_InvalidTimestampsIgnored(t *testing.T) {
	srv, _, _ := defaultTestServer()

	rr := callHandler(srv, srv.handleStats, "GET", "/api/stats?start=not-a-time&end=also-not-a-time", "")
	require.Equal(t, http.StatusOK, rr.Code)
}

// ---------------------------------------------------------------------------
// handleModels
// ---------------------------------------------------------------------------

func TestHandleModels_Success(t *testing.T) {
	srv, _, _ := defaultTestServer()

	rr := callHandler(srv, srv.handleModels, "GET", "/api/models", "")
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	_, hasModels := resp["models"]
	assert.True(t, hasModels)
	_, hasProviders := resp["available_providers"]
	assert.True(t, hasProviders)
}

func TestHandleModels_MethodNotAllowed(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := callHandler(srv, srv.handleModels, "POST", "/api/models", "")
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

// ---------------------------------------------------------------------------
// handleHealth
// ---------------------------------------------------------------------------

func TestHandleHealth_Success(t *testing.T) {
	srv, _, _ := defaultTestServer()

	rr := callHandler(srv, srv.handleHealth, "GET", "/api/health", "")
	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "ok", resp["status"])
	_, hasProviders := resp["providers"]
	assert.True(t, hasProviders)
}

// ---------------------------------------------------------------------------
// Full routing via mux
// ---------------------------------------------------------------------------

func TestRouting_Health(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := serveRequest(srv, "GET", "/api/health", "")
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRouting_Models(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := serveRequest(srv, "GET", "/api/models", "")
	assert.Equal(t, http.StatusOK, rr.Code)
}
