package jsonfile_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kimseunghwan/llm-viz/backend/internal/adapter/pricing/jsonfile"
	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
)

// writeTempPricing writes a pricing JSON file to a temp directory and returns its path.
func writeTempPricing(t *testing.T, data map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(data)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "pricing.json")
	require.NoError(t, os.WriteFile(path, raw, 0o644))
	return path
}

// minimalPricingData returns a small but valid pricing JSON structure.
func minimalPricingData() map[string]any {
	return map[string]any{
		"openai": map[string]any{
			"gpt-4o": map[string]any{
				"display_name":          "GPT-4o",
				"input_price_per_m":     2.50,
				"output_price_per_m":    10.00,
				"cache_read_price_per_m": 1.25,
				"context_window":        128000,
				"is_active":             true,
			},
			"gpt-4o-mini": map[string]any{
				"display_name":       "GPT-4o mini",
				"input_price_per_m":  0.15,
				"output_price_per_m": 0.60,
				"context_window":     128000,
				"is_active":          true,
			},
		},
		"anthropic": map[string]any{
			"claude-opus-4-6": map[string]any{
				"display_name":          "Claude Opus 4.6",
				"input_price_per_m":     15.00,
				"output_price_per_m":    75.00,
				"cache_read_price_per_m": 1.50,
				"cache_write_price_per_m": 18.75,
				"context_window":        200000,
				"is_active":             true,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// NewRepository
// ---------------------------------------------------------------------------

func TestNewRepository_Success(t *testing.T) {
	path := writeTempPricing(t, minimalPricingData())
	repo, err := jsonfile.NewRepository(path)

	require.NoError(t, err)
	require.NotNil(t, repo)
}

func TestNewRepository_FileNotFound(t *testing.T) {
	_, err := jsonfile.NewRepository("/nonexistent/path/pricing.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pricing: read file")
}

func TestNewRepository_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte(`{invalid json`), 0o644))

	_, err := jsonfile.NewRepository(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pricing: parse JSON")
}

func TestNewRepository_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	require.NoError(t, os.WriteFile(path, []byte(`{}`), 0o644))

	repo, err := jsonfile.NewRepository(path)
	require.NoError(t, err)
	require.NotNil(t, repo)
}

// ---------------------------------------------------------------------------
// GetPricing
// ---------------------------------------------------------------------------

func TestGetPricing_Found_OpenAI(t *testing.T) {
	path := writeTempPricing(t, minimalPricingData())
	repo, err := jsonfile.NewRepository(path)
	require.NoError(t, err)

	entry, err := repo.GetPricing(context.Background(), domain.ProviderOpenAI, "gpt-4o")
	require.NoError(t, err)
	require.NotNil(t, entry)

	assert.Equal(t, domain.ProviderOpenAI, entry.Provider)
	assert.Equal(t, "gpt-4o", entry.ModelID)
	assert.Equal(t, "GPT-4o", entry.DisplayName)
	assert.InDelta(t, 2.50, entry.InputPricePerM, 0.001)
	assert.InDelta(t, 10.00, entry.OutputPricePerM, 0.001)
	assert.InDelta(t, 1.25, entry.CacheReadPricePerM, 0.001)
	assert.Equal(t, int64(128000), entry.ContextWindow)
	assert.True(t, entry.IsActive)
}

func TestGetPricing_Found_Anthropic(t *testing.T) {
	path := writeTempPricing(t, minimalPricingData())
	repo, err := jsonfile.NewRepository(path)
	require.NoError(t, err)

	entry, err := repo.GetPricing(context.Background(), domain.ProviderAnthropic, "claude-opus-4-6")
	require.NoError(t, err)
	require.NotNil(t, entry)

	assert.Equal(t, domain.ProviderAnthropic, entry.Provider)
	assert.InDelta(t, 15.00, entry.InputPricePerM, 0.001)
	assert.InDelta(t, 75.00, entry.OutputPricePerM, 0.001)
	assert.InDelta(t, 1.50, entry.CacheReadPricePerM, 0.001)
	assert.InDelta(t, 18.75, entry.CacheWritePricePerM, 0.001)
}

func TestGetPricing_NotFound_UnknownModel(t *testing.T) {
	path := writeTempPricing(t, minimalPricingData())
	repo, err := jsonfile.NewRepository(path)
	require.NoError(t, err)

	_, err = repo.GetPricing(context.Background(), domain.ProviderOpenAI, "nonexistent-model")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrModelNotFound)
}

func TestGetPricing_NotFound_UnknownProvider(t *testing.T) {
	path := writeTempPricing(t, minimalPricingData())
	repo, err := jsonfile.NewRepository(path)
	require.NoError(t, err)

	_, err = repo.GetPricing(context.Background(), domain.ProviderGemini, "gemini-pro")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrModelNotFound)
}

func TestGetPricing_DefaultsIsActiveWhenPriceSet(t *testing.T) {
	// A model without explicit is_active=true but with input_price > 0
	// should default to active.
	data := map[string]any{
		"openai": map[string]any{
			"gpt-default-active": map[string]any{
				"display_name":       "Default Active",
				"input_price_per_m":  1.00,
				"output_price_per_m": 2.00,
				"context_window":     100000,
				// is_active omitted — defaults based on price > 0
			},
		},
	}
	path := writeTempPricing(t, data)
	repo, err := jsonfile.NewRepository(path)
	require.NoError(t, err)

	entry, err := repo.GetPricing(context.Background(), domain.ProviderOpenAI, "gpt-default-active")
	require.NoError(t, err)
	assert.True(t, entry.IsActive)
}

// ---------------------------------------------------------------------------
// ListModels
// ---------------------------------------------------------------------------

func TestListModels_ReturnsAll(t *testing.T) {
	path := writeTempPricing(t, minimalPricingData())
	repo, err := jsonfile.NewRepository(path)
	require.NoError(t, err)

	models, err := repo.ListModels(context.Background())
	require.NoError(t, err)
	assert.Len(t, models, 3) // gpt-4o, gpt-4o-mini, claude-opus-4-6
}

func TestListModels_ContainsModelInfo(t *testing.T) {
	path := writeTempPricing(t, minimalPricingData())
	repo, err := jsonfile.NewRepository(path)
	require.NoError(t, err)

	models, err := repo.ListModels(context.Background())
	require.NoError(t, err)

	modelIDs := make(map[string]domain.ModelInfo)
	for _, m := range models {
		modelIDs[m.ID] = m
	}

	gpt4o, ok := modelIDs["gpt-4o"]
	require.True(t, ok)
	assert.Equal(t, "GPT-4o", gpt4o.DisplayName)
	assert.Equal(t, domain.ProviderOpenAI, gpt4o.Provider)
	assert.Equal(t, int64(128000), gpt4o.ContextWindow)
}

func TestListModels_EmptyWhenNoData(t *testing.T) {
	path := writeTempPricing(t, map[string]any{})
	repo, err := jsonfile.NewRepository(path)
	require.NoError(t, err)

	models, err := repo.ListModels(context.Background())
	require.NoError(t, err)
	assert.Empty(t, models)
}

// ---------------------------------------------------------------------------
// Integration: pricing.json on disk
// ---------------------------------------------------------------------------

func TestNewRepository_WithRealPricingFile(t *testing.T) {
	// Use the real data/pricing.json if available (skips if not found).
	realPath := "../../../../../data/pricing.json"
	if _, err := os.Stat(realPath); os.IsNotExist(err) {
		t.Skip("real pricing.json not found, skipping integration test")
	}

	repo, err := jsonfile.NewRepository(realPath)
	require.NoError(t, err)

	models, err := repo.ListModels(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, models)
}
