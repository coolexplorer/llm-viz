package jsonfile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
)

// pricingModel is the JSON structure for a single model entry.
type pricingModel struct {
	DisplayName         string  `json:"display_name"`
	InputPricePerM      float64 `json:"input_price_per_m"`
	OutputPricePerM     float64 `json:"output_price_per_m"`
	CacheReadPricePerM  float64 `json:"cache_read_price_per_m"`
	CacheWritePricePerM float64 `json:"cache_write_price_per_m"`
	ContextWindow       int64   `json:"context_window"`
	IsActive            bool    `json:"is_active"`
}

// Repository loads pricing data from a JSON flat file.
type Repository struct {
	entries map[string]*domain.PricingEntry // key: "provider:modelID"
	models  []domain.ModelInfo
}

// NewRepository reads and parses the pricing JSON file at the given path.
func NewRepository(filePath string) (*Repository, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("pricing: read file: %w", err)
	}

	// Parse as map[provider]map[modelID]pricingModel
	var raw map[string]map[string]pricingModel
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("pricing: parse JSON: %w", err)
	}

	repo := &Repository{
		entries: make(map[string]*domain.PricingEntry),
		models:  make([]domain.ModelInfo, 0),
	}

	effectiveDate := time.Now().UTC()

	for providerStr, models := range raw {
		providerID := domain.ProviderID(providerStr)
		for modelID, pm := range models {
			isActive := pm.IsActive
			// Default to active if not explicitly set.
			if !isActive && pm.InputPricePerM > 0 {
				isActive = true
			}
			entry := &domain.PricingEntry{
				Provider:            providerID,
				ModelID:             modelID,
				DisplayName:         pm.DisplayName,
				InputPricePerM:      pm.InputPricePerM,
				OutputPricePerM:     pm.OutputPricePerM,
				CacheReadPricePerM:  pm.CacheReadPricePerM,
				CacheWritePricePerM: pm.CacheWritePricePerM,
				ContextWindow:       pm.ContextWindow,
				EffectiveDate:       effectiveDate,
				IsActive:            isActive,
			}
			key := providerStr + ":" + modelID
			repo.entries[key] = entry

			repo.models = append(repo.models, domain.ModelInfo{
				ID:            modelID,
				DisplayName:   pm.DisplayName,
				Provider:      providerID,
				ContextWindow: pm.ContextWindow,
			})
		}
	}

	return repo, nil
}

// GetPricing returns the pricing entry for a provider+model combination.
func (r *Repository) GetPricing(_ context.Context, provider domain.ProviderID, modelID string) (*domain.PricingEntry, error) {
	key := string(provider) + ":" + modelID
	entry, ok := r.entries[key]
	if !ok {
		return nil, fmt.Errorf("%w: %s/%s", domain.ErrModelNotFound, provider, modelID)
	}
	return entry, nil
}

// ListModels returns all known models from the pricing file.
func (r *Repository) ListModels(_ context.Context) ([]domain.ModelInfo, error) {
	return r.models, nil
}
