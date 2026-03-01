package service_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/port"
	"github.com/kimseunghwan/llm-viz/backend/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock implementations ---

type mockProvider struct {
	result *domain.CompletionResult
	err    error
}

func (m *mockProvider) Complete(_ context.Context, _ domain.CompletionRequest) (*domain.CompletionResult, error) {
	return m.result, m.err
}
func (m *mockProvider) Stream(_ context.Context, _ domain.CompletionRequest, _ port.StreamHandler) error {
	return nil
}
func (m *mockProvider) ProviderID() domain.ProviderID      { return "mock" }
func (m *mockProvider) SupportedModels() []domain.ModelInfo { return nil }

type mockRepo struct {
	saved []domain.NormalizedUsage
}

func (r *mockRepo) Save(_ context.Context, u domain.NormalizedUsage) error {
	r.saved = append(r.saved, u)
	return nil
}
func (r *mockRepo) FindBySession(_ context.Context, _ string, _ int) ([]domain.NormalizedUsage, error) {
	return r.saved, nil
}
func (r *mockRepo) FindByTimeRange(_ context.Context, _ port.TimeRangeFilter) ([]domain.NormalizedUsage, error) {
	return r.saved, nil
}
func (r *mockRepo) SumByProvider(_ context.Context, _ port.TimeRangeFilter) (map[domain.ProviderID]domain.UsageSummary, error) {
	return nil, nil
}

type mockPricingRepo struct{}

func (p *mockPricingRepo) GetPricing(_ context.Context, _ domain.ProviderID, _ string) (*domain.PricingEntry, error) {
	return &domain.PricingEntry{
		InputPricePerM:  3.00,
		OutputPricePerM: 15.00,
	}, nil
}
func (p *mockPricingRepo) ListModels(_ context.Context) ([]domain.ModelInfo, error) {
	return nil, nil
}

type mockBroadcaster struct {
	published []domain.UsageEvent
}

func (b *mockBroadcaster) Subscribe(_ string) <-chan domain.UsageEvent {
	ch := make(chan domain.UsageEvent, 1)
	return ch
}
func (b *mockBroadcaster) Unsubscribe(_ string) {}
func (b *mockBroadcaster) Publish(event domain.UsageEvent) {
	b.published = append(b.published, event)
}

// --- Tests ---

func newTracker(provider port.LLMProvider) (*service.TokenTracker, *mockRepo, *mockBroadcaster) {
	repo := &mockRepo{}
	broadcaster := &mockBroadcaster{}
	providers := map[domain.ProviderID]port.LLMProvider{
		provider.ProviderID(): provider,
	}
	tracker := service.NewTokenTracker(
		providers,
		repo,
		&mockPricingRepo{},
		broadcaster,
		slog.Default(),
	)
	return tracker, repo, broadcaster
}

func TestTrackCompletion_HappyPath(t *testing.T) {
	mock := &mockProvider{
		result: &domain.CompletionResult{
			ID:      "test-id",
			Content: "Hello, world!",
			Usage: domain.TokenUsage{
				InputTokens:  100,
				OutputTokens: 50,
				TotalTokens:  150,
			},
		},
	}
	tracker, repo, broadcaster := newTracker(mock)

	usage, err := tracker.TrackCompletion(context.Background(), service.TrackRequest{
		Provider:  "mock",
		Model:     "mock-model",
		SessionID: "test-session",
		Messages:  []domain.Message{{Role: "user", Content: "Hello"}},
	})

	require.NoError(t, err)
	assert.NotEmpty(t, usage.ID)
	assert.Equal(t, int64(100), usage.Usage.InputTokens)
	assert.Equal(t, int64(50), usage.Usage.OutputTokens)
	assert.Equal(t, "test-session", usage.SessionID)
	assert.Greater(t, usage.CostUSD, 0.0)

	// Verify persistence
	assert.Len(t, repo.saved, 1)

	// Verify SSE broadcast
	assert.Len(t, broadcaster.published, 1)
	assert.Equal(t, "test-session", broadcaster.published[0].SessionID)
}

func TestTrackCompletion_UnknownProvider(t *testing.T) {
	mock := &mockProvider{result: &domain.CompletionResult{}}
	tracker, _, _ := newTracker(mock)

	_, err := tracker.TrackCompletion(context.Background(), service.TrackRequest{
		Provider: "nonexistent",
		Model:    "some-model",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnknownProvider)
}

func TestTrackCompletion_ProviderError(t *testing.T) {
	mock := &mockProvider{err: domain.ErrRateLimited}
	tracker, _, _ := newTracker(mock)

	_, err := tracker.TrackCompletion(context.Background(), service.TrackRequest{
		Provider: "mock",
		Model:    "mock-model",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRateLimited)
}

func TestListAvailableProviders(t *testing.T) {
	mock := &mockProvider{result: &domain.CompletionResult{}}
	tracker, _, _ := newTracker(mock)

	providers := tracker.ListAvailableProviders()
	assert.Len(t, providers, 1)
	assert.Contains(t, providers, domain.ProviderID("mock"))
}
