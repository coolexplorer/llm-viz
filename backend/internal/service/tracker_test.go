package service_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/port"
	"github.com/kimseunghwan/llm-viz/backend/internal/service"
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

type mockProviderB struct{}

func (m *mockProviderB) Complete(_ context.Context, _ domain.CompletionRequest) (*domain.CompletionResult, error) {
	return &domain.CompletionResult{}, nil
}
func (m *mockProviderB) Stream(_ context.Context, _ domain.CompletionRequest, _ port.StreamHandler) error {
	return nil
}
func (m *mockProviderB) ProviderID() domain.ProviderID      { return "mock2" }
func (m *mockProviderB) SupportedModels() []domain.ModelInfo { return nil }

type mockRepo struct {
	saved   []domain.NormalizedUsage
	saveErr error
}

func (r *mockRepo) Save(_ context.Context, u domain.NormalizedUsage) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.saved = append(r.saved, u)
	return nil
}
func (r *mockRepo) FindBySession(_ context.Context, sessionID string, limit int) ([]domain.NormalizedUsage, error) {
	var out []domain.NormalizedUsage
	for i := len(r.saved) - 1; i >= 0; i-- {
		if r.saved[i].SessionID == sessionID {
			out = append(out, r.saved[i])
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}
func (r *mockRepo) FindByTimeRange(_ context.Context, _ port.TimeRangeFilter) ([]domain.NormalizedUsage, error) {
	return r.saved, nil
}
func (r *mockRepo) SumByProvider(_ context.Context, _ port.TimeRangeFilter) (map[domain.ProviderID]domain.UsageSummary, error) {
	sums := make(map[domain.ProviderID]domain.UsageSummary)
	for _, u := range r.saved {
		s := sums[u.Provider]
		s.Provider = u.Provider
		s.TotalInput += u.Usage.InputTokens
		s.TotalOutput += u.Usage.OutputTokens
		s.TotalCost += u.CostUSD
		s.RequestCount++
		sums[u.Provider] = s
	}
	return sums, nil
}

type mockPricingRepo struct {
	entry *domain.PricingEntry
	err   error
}

func (p *mockPricingRepo) GetPricing(_ context.Context, _ domain.ProviderID, _ string) (*domain.PricingEntry, error) {
	if p.err != nil {
		return nil, p.err
	}
	if p.entry != nil {
		return p.entry, nil
	}
	return &domain.PricingEntry{InputPricePerM: 3.00, OutputPricePerM: 15.00}, nil
}
func (p *mockPricingRepo) ListModels(_ context.Context) ([]domain.ModelInfo, error) {
	return nil, nil
}

type mockBroadcaster struct {
	published []domain.UsageEvent
}

func (b *mockBroadcaster) Subscribe(_ string) <-chan domain.UsageEvent {
	return make(chan domain.UsageEvent, 1)
}
func (b *mockBroadcaster) Unsubscribe(_ string) {}
func (b *mockBroadcaster) Publish(event domain.UsageEvent) {
	b.published = append(b.published, event)
}

// --- Helpers ---

func newTracker(provider port.LLMProvider) (*service.TokenTracker, *mockRepo, *mockBroadcaster) {
	repo := &mockRepo{}
	broadcaster := &mockBroadcaster{}
	providers := map[domain.ProviderID]port.LLMProvider{provider.ProviderID(): provider}
	tracker := service.NewTokenTracker(providers, repo, &mockPricingRepo{}, broadcaster, slog.Default())
	return tracker, repo, broadcaster
}

func newTrackRequest(overrides ...func(*service.TrackRequest)) service.TrackRequest {
	req := service.TrackRequest{
		Provider:  "mock",
		Model:     "mock-model",
		SessionID: "default-session",
		Messages:  []domain.Message{{Role: "user", Content: "Hello"}},
	}
	for _, o := range overrides {
		o(&req)
	}
	return req
}

func defaultProvider() *mockProvider {
	return &mockProvider{result: &domain.CompletionResult{
		ID:      "test-id",
		Content: "Hello, world!",
		Usage:   domain.TokenUsage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150},
	}}
}

// --- Tests ---

func TestTrackCompletion_HappyPath(t *testing.T) {
	tracker, repo, broadcaster := newTracker(defaultProvider())

	usage, err := tracker.TrackCompletion(context.Background(), newTrackRequest())

	require.NoError(t, err)
	assert.NotEmpty(t, usage.ID)
	assert.Equal(t, int64(100), usage.Usage.InputTokens)
	assert.Equal(t, int64(50), usage.Usage.OutputTokens)
	assert.Equal(t, "default-session", usage.SessionID)
	assert.Greater(t, usage.CostUSD, 0.0)
	assert.Len(t, repo.saved, 1)
	assert.Len(t, broadcaster.published, 1)
	assert.Equal(t, "default-session", broadcaster.published[0].SessionID)
}

func TestTrackCompletion_UnknownProvider(t *testing.T) {
	tracker, _, _ := newTracker(defaultProvider())

	_, err := tracker.TrackCompletion(context.Background(), service.TrackRequest{
		Provider: "nonexistent",
		Model:    "some-model",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnknownProvider)
}

func TestTrackCompletion_ProviderError(t *testing.T) {
	tracker, _, _ := newTracker(&mockProvider{err: domain.ErrRateLimited})

	_, err := tracker.TrackCompletion(context.Background(), newTrackRequest())

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRateLimited)
}

func TestTrackCompletion_PricingNotFound_ContinuesWithZeroCost(t *testing.T) {
	repo := &mockRepo{}
	broadcaster := &mockBroadcaster{}
	pricing := &mockPricingRepo{err: fmt.Errorf("%w: mock/mock-model", domain.ErrModelNotFound)}
	tracker := service.NewTokenTracker(
		map[domain.ProviderID]port.LLMProvider{"mock": defaultProvider()},
		repo, pricing, broadcaster, slog.Default(),
	)

	usage, err := tracker.TrackCompletion(context.Background(), newTrackRequest())

	require.NoError(t, err)
	assert.Equal(t, 0.0, usage.CostUSD)
	assert.Len(t, repo.saved, 1)
}

func TestTrackCompletion_RepoSaveFailure_NonFatal(t *testing.T) {
	repo := &mockRepo{saveErr: errors.New("disk full")}
	broadcaster := &mockBroadcaster{}
	tracker := service.NewTokenTracker(
		map[domain.ProviderID]port.LLMProvider{"mock": defaultProvider()},
		repo, &mockPricingRepo{}, broadcaster, slog.Default(),
	)

	usage, err := tracker.TrackCompletion(context.Background(), newTrackRequest())

	require.NoError(t, err)
	assert.NotNil(t, usage)
	assert.Len(t, broadcaster.published, 1)
}

func TestTrackCompletion_APIKeyHashing(t *testing.T) {
	tracker, repo, _ := newTracker(defaultProvider())

	_, err := tracker.TrackCompletion(context.Background(), service.TrackRequest{
		Provider:  "mock",
		Model:     "mock-model",
		Messages:  []domain.Message{{Role: "user", Content: "hi"}},
		SessionID: "s1",
		APIKey:    "sk-super-secret-key-must-never-be-stored-raw",
	})
	require.NoError(t, err)
	require.Len(t, repo.saved, 1)

	saved := repo.saved[0]
	assert.NotEqual(t, "sk-super-secret-key-must-never-be-stored-raw", saved.APIKeyHash)
	assert.Len(t, saved.APIKeyHash, 8)
}

func TestTrackCompletion_EmptyAPIKey_NoHash(t *testing.T) {
	tracker, repo, _ := newTracker(defaultProvider())

	_, _ = tracker.TrackCompletion(context.Background(), service.TrackRequest{
		Provider:  "mock",
		Model:     "mock-model",
		Messages:  []domain.Message{{Role: "user", Content: "hi"}},
		SessionID: "s2",
		APIKey:    "",
	})

	require.Len(t, repo.saved, 1)
	assert.Equal(t, "", repo.saved[0].APIKeyHash)
}

func TestTrackCompletion_TimestampIsSet(t *testing.T) {
	before := time.Now().UTC()
	tracker, _, _ := newTracker(defaultProvider())

	usage, err := tracker.TrackCompletion(context.Background(), newTrackRequest())
	require.NoError(t, err)

	after := time.Now().UTC()
	assert.True(t, !usage.Timestamp.Before(before))
	assert.True(t, !usage.Timestamp.After(after))
}

func TestGetSessionHistory_WithLimit(t *testing.T) {
	_, repo, _ := newTracker(defaultProvider())
	for i := range 10 {
		repo.saved = append(repo.saved, domain.NormalizedUsage{
			ID:        fmt.Sprintf("u%d", i),
			SessionID: "sess",
		})
	}
	tracker := service.NewTokenTracker(
		map[domain.ProviderID]port.LLMProvider{"mock": defaultProvider()},
		repo, &mockPricingRepo{}, &mockBroadcaster{}, slog.Default(),
	)

	history, err := tracker.GetSessionHistory(context.Background(), "sess", 3)
	require.NoError(t, err)
	assert.Len(t, history, 3)
}

func TestGetSessionHistory_MostRecentFirst(t *testing.T) {
	_, repo, _ := newTracker(defaultProvider())
	repo.saved = []domain.NormalizedUsage{
		{ID: "oldest", SessionID: "sess"},
		{ID: "middle", SessionID: "sess"},
		{ID: "newest", SessionID: "sess"},
	}
	tracker := service.NewTokenTracker(
		map[domain.ProviderID]port.LLMProvider{"mock": defaultProvider()},
		repo, &mockPricingRepo{}, &mockBroadcaster{}, slog.Default(),
	)

	history, err := tracker.GetSessionHistory(context.Background(), "sess", 0)
	require.NoError(t, err)
	assert.Equal(t, "newest", history[0].ID)
	assert.Equal(t, "oldest", history[2].ID)
}

func TestGetProviderStats_Aggregation(t *testing.T) {
	_, repo, _ := newTracker(defaultProvider())
	repo.saved = []domain.NormalizedUsage{
		{Provider: domain.ProviderOpenAI, Usage: domain.TokenUsage{InputTokens: 100, OutputTokens: 50}, CostUSD: 0.001},
		{Provider: domain.ProviderOpenAI, Usage: domain.TokenUsage{InputTokens: 200, OutputTokens: 80}, CostUSD: 0.002},
		{Provider: domain.ProviderAnthropic, Usage: domain.TokenUsage{InputTokens: 500, OutputTokens: 200}, CostUSD: 0.01},
	}
	tracker := service.NewTokenTracker(
		map[domain.ProviderID]port.LLMProvider{"mock": defaultProvider()},
		repo, &mockPricingRepo{}, &mockBroadcaster{}, slog.Default(),
	)

	stats, err := tracker.GetProviderStats(context.Background(), port.TimeRangeFilter{})
	require.NoError(t, err)

	oai := stats[domain.ProviderOpenAI]
	assert.Equal(t, int64(300), oai.TotalInput)
	assert.Equal(t, int64(130), oai.TotalOutput)
	assert.InDelta(t, 0.003, oai.TotalCost, 0.0001)
	assert.Equal(t, int64(2), oai.RequestCount)
}

func TestListAvailableProviders(t *testing.T) {
	tracker, _, _ := newTracker(defaultProvider())

	providers := tracker.ListAvailableProviders()
	assert.Len(t, providers, 1)
	assert.Contains(t, providers, domain.ProviderID("mock"))
}

func TestListAvailableProviders_Multiple(t *testing.T) {
	providers := map[domain.ProviderID]port.LLMProvider{
		"mock":  defaultProvider(),
		"mock2": &mockProviderB{},
	}
	tracker := service.NewTokenTracker(providers, &mockRepo{}, &mockPricingRepo{}, &mockBroadcaster{}, slog.Default())

	ids := tracker.ListAvailableProviders()
	assert.Len(t, ids, 2)
}

// --- Suite ---

type TrackerSuite struct {
	suite.Suite
	tracker     *service.TokenTracker
	repo        *mockRepo
	broadcaster *mockBroadcaster
}

func (s *TrackerSuite) SetupTest() {
	s.repo = &mockRepo{}
	s.broadcaster = &mockBroadcaster{}
	s.tracker = service.NewTokenTracker(
		map[domain.ProviderID]port.LLMProvider{"mock": defaultProvider()},
		s.repo, &mockPricingRepo{}, s.broadcaster, slog.Default(),
	)
}

func (s *TrackerSuite) TestHappyPath() {
	usage, err := s.tracker.TrackCompletion(context.Background(), newTrackRequest())
	s.Require().NoError(err)
	s.Equal(int64(100), usage.Usage.InputTokens)
}

func (s *TrackerSuite) TestPersistenceAndBroadcast() {
	_, err := s.tracker.TrackCompletion(context.Background(), newTrackRequest())
	s.Require().NoError(err)
	s.Len(s.repo.saved, 1)
	s.Len(s.broadcaster.published, 1)
}

func (s *TrackerSuite) TestCostCalculation() {
	usage, err := s.tracker.TrackCompletion(context.Background(), newTrackRequest())
	s.Require().NoError(err)
	// 100 input @ $3/M + 50 output @ $15/M = $0.0003 + $0.00075 = $0.00105
	s.InDelta(0.00105, usage.CostUSD, 0.0001)
}

func (s *TrackerSuite) TestProviderAndModelPropagated() {
	usage, err := s.tracker.TrackCompletion(context.Background(), newTrackRequest())
	s.Require().NoError(err)
	s.Equal(domain.ProviderID("mock"), usage.Provider)
	s.Equal("mock-model", usage.Model)
}

func TestTrackerSuite(t *testing.T) {
	suite.Run(t, new(TrackerSuite))
}
