package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kimseunghwan/llm-viz/backend/internal/adapter/provider/anthropic"
	"github.com/kimseunghwan/llm-viz/backend/internal/adapter/provider/openai"
	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/port"
)

// TrackRequest is the input for a tracked completion call.
type TrackRequest struct {
	Provider   domain.ProviderID
	Model      string
	Messages   []domain.Message
	MaxTokens  int
	Stream     bool
	SessionID  string
	APIKey     string // optional: hashed for storage, not stored raw
	ProjectTag string
}

func (r TrackRequest) toCompletionRequest() domain.CompletionRequest {
	maxTokens := r.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	return domain.CompletionRequest{
		Model:      r.Model,
		Messages:   r.Messages,
		MaxTokens:  maxTokens,
		Stream:     r.Stream,
		SessionID:  r.SessionID,
		ProjectTag: r.ProjectTag,
	}
}

// TokenTracker is the application-level service that orchestrates:
// provider call → cost calculation → persistence → SSE broadcast.
type TokenTracker struct {
	providers   map[domain.ProviderID]port.LLMProvider
	repo        port.UsageRepository
	pricing     port.PricingRepository
	broadcaster port.EventBroadcaster
	logger      *slog.Logger
}

// NewTokenTracker constructs a TokenTracker with explicit dependencies (manual DI).
func NewTokenTracker(
	providers map[domain.ProviderID]port.LLMProvider,
	repo port.UsageRepository,
	pricing port.PricingRepository,
	broadcaster port.EventBroadcaster,
	logger *slog.Logger,
) *TokenTracker {
	return &TokenTracker{
		providers:   providers,
		repo:        repo,
		pricing:     pricing,
		broadcaster: broadcaster,
		logger:      logger,
	}
}

// TrackCompletion calls the provider, calculates cost, persists usage, and broadcasts via SSE.
func (t *TokenTracker) TrackCompletion(ctx context.Context, req TrackRequest) (*domain.NormalizedUsage, error) {
	var provider port.LLMProvider
	var ok bool

	// If request has API key, create runtime provider adapter
	if req.APIKey != "" {
		provider = t.createProviderWithKey(req.Provider, req.APIKey)
		if provider == nil {
			return nil, fmt.Errorf("%w: %s", domain.ErrUnknownProvider, req.Provider)
		}
	} else {
		// Use default provider from initialization
		provider, ok = t.providers[req.Provider]
		if !ok {
			return nil, fmt.Errorf("%w: %s", domain.ErrUnknownProvider, req.Provider)
		}
	}

	t.logger.Info("calling provider", "provider", req.Provider, "model", req.Model, "session_id", req.SessionID)

	result, err := provider.Complete(ctx, req.toCompletionRequest())
	if err != nil {
		return nil, err // already wrapped by adapter's mapError
	}

	t.logger.Info("provider response", "provider", req.Provider, "input_tokens", result.Usage.InputTokens, "output_tokens", result.Usage.OutputTokens)

	pricing, err := t.pricing.GetPricing(ctx, req.Provider, req.Model)
	if err != nil {
		// Non-fatal: log and continue with $0 cost.
		t.logger.Warn("pricing not found, cost set to 0",
			"provider", req.Provider, "model", req.Model)
		pricing = &domain.PricingEntry{} // zero pricing
	}

	normalized := &domain.NormalizedUsage{
		ID:         uuid.NewString(),
		Timestamp:  time.Now().UTC(),
		Provider:   req.Provider,
		Model:      req.Model,
		SessionID:  req.SessionID,
		APIKeyHash: hashAPIKey(req.APIKey),
		Usage:      result.Usage,
		CostUSD:    domain.CalculateCost(result.Usage, *pricing),
		ProjectTag: req.ProjectTag,
	}

	if err := t.repo.Save(ctx, *normalized); err != nil {
		// Non-fatal: log, still return result to caller.
		t.logger.Error("failed to persist usage", "error", err)
	}

	event := domain.UsageEvent{
		SessionID: req.SessionID,
		Usage:     *normalized,
	}
	t.logger.Info("broadcasting event", "session_id", req.SessionID, "input", normalized.Usage.InputTokens, "output", normalized.Usage.OutputTokens)
	t.broadcaster.Publish(event)

	return normalized, nil
}

// GetSessionHistory returns stored usage records for a session (most recent first).
func (t *TokenTracker) GetSessionHistory(ctx context.Context, sessionID string, limit int) ([]domain.NormalizedUsage, error) {
	return t.repo.FindBySession(ctx, sessionID, limit)
}

// GetProviderStats returns aggregated usage statistics grouped by provider.
func (t *TokenTracker) GetProviderStats(ctx context.Context, filter port.TimeRangeFilter) (map[domain.ProviderID]domain.UsageSummary, error) {
	return t.repo.SumByProvider(ctx, filter)
}

// ListAvailableProviders returns the IDs of configured providers.
func (t *TokenTracker) ListAvailableProviders() []domain.ProviderID {
	ids := make([]domain.ProviderID, 0, len(t.providers))
	for id := range t.providers {
		ids = append(ids, id)
	}
	return ids
}

// Repo exposes the underlying usage repository (used for analytics wiring).
func (t *TokenTracker) Repo() port.UsageRepository {
	return t.repo
}

// createProviderWithKey creates a runtime provider adapter with the given API key.
func (t *TokenTracker) createProviderWithKey(providerID domain.ProviderID, apiKey string) port.LLMProvider {
	switch providerID {
	case domain.ProviderAnthropic:
		return anthropic.New(apiKey)
	case domain.ProviderOpenAI:
		return openai.New(apiKey)
	default:
		return nil
	}
}

// hashAPIKey returns the first 8 hex chars of SHA-256 for safe logging/storage.
func hashAPIKey(key string) string {
	if key == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:8]
}
