// Package http — test-only mock implementations shared across handler test files.
package http

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/port"
	"github.com/kimseunghwan/llm-viz/backend/internal/service"
)

// ---------------------------------------------------------------------------
// Mock port implementations
// ---------------------------------------------------------------------------

type mockProvider struct {
	id     domain.ProviderID
	result *domain.CompletionResult
	err    error
}

func (m *mockProvider) Complete(_ context.Context, _ domain.CompletionRequest) (*domain.CompletionResult, error) {
	return m.result, m.err
}
func (m *mockProvider) Stream(_ context.Context, _ domain.CompletionRequest, _ port.StreamHandler) error {
	return m.err
}
func (m *mockProvider) ProviderID() domain.ProviderID      { return m.id }
func (m *mockProvider) SupportedModels() []domain.ModelInfo { return nil }

type mockRepo struct {
	saved   []domain.NormalizedUsage
	saveErr error
	findErr error
}

func (r *mockRepo) Save(_ context.Context, u domain.NormalizedUsage) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.saved = append(r.saved, u)
	return nil
}
func (r *mockRepo) FindBySession(_ context.Context, sessionID string, limit int) ([]domain.NormalizedUsage, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	var out []domain.NormalizedUsage
	for _, u := range r.saved {
		if u.SessionID == sessionID {
			out = append(out, u)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
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
	return &domain.PricingEntry{
		InputPricePerM:  3.00,
		OutputPricePerM: 15.00,
	}, nil
}
func (p *mockPricingRepo) ListModels(_ context.Context) ([]domain.ModelInfo, error) {
	return []domain.ModelInfo{
		{ID: "gpt-4o", Provider: domain.ProviderOpenAI, ContextWindow: 128_000},
	}, nil
}

type mockBroadcaster struct {
	published []domain.UsageEvent
	ch        chan domain.UsageEvent
}

func newMockBroadcaster() *mockBroadcaster {
	return &mockBroadcaster{ch: make(chan domain.UsageEvent, 16)}
}
func (b *mockBroadcaster) Subscribe(_ string) <-chan domain.UsageEvent { return b.ch }
func (b *mockBroadcaster) Unsubscribe(_ string)                       {}
func (b *mockBroadcaster) Publish(event domain.UsageEvent) {
	b.published = append(b.published, event)
	select {
	case b.ch <- event:
	default:
	}
}

// ---------------------------------------------------------------------------
// flushableRecorder implements http.Flusher for SSE handler tests.
// ---------------------------------------------------------------------------

type flushableRecorder struct {
	*httptest.ResponseRecorder
	flushed int
}

func newFlushableRecorder() *flushableRecorder {
	return &flushableRecorder{ResponseRecorder: httptest.NewRecorder()}
}
func (f *flushableRecorder) Flush() { f.flushed++ }

// ---------------------------------------------------------------------------
// Server construction helpers
// ---------------------------------------------------------------------------

// newTestServer constructs a fully-wired Server with mock dependencies.
func newTestServer(provider port.LLMProvider, repo *mockRepo, broadcaster *mockBroadcaster) *Server {
	providers := map[domain.ProviderID]port.LLMProvider{}
	if provider != nil {
		providers[provider.ProviderID()] = provider
	}
	pricing := &mockPricingRepo{}
	tracker := service.NewTokenTracker(providers, repo, pricing, broadcaster, slog.Default())
	return NewServer(tracker, broadcaster, pricing, slog.Default(), ":0", "*")
}

// defaultTestServer returns a server wired with a happy-path "mock" provider.
func defaultTestServer() (*Server, *mockRepo, *mockBroadcaster) {
	repo := &mockRepo{}
	broadcaster := newMockBroadcaster()
	provider := &mockProvider{
		id: "mock",
		result: &domain.CompletionResult{
			ID:      "test-completion-id",
			Content: "Hello from mock!",
			Usage: domain.TokenUsage{
				InputTokens:  100,
				OutputTokens: 50,
				TotalTokens:  150,
			},
		},
	}
	srv := newTestServer(provider, repo, broadcaster)
	return srv, repo, broadcaster
}

// serveRequest sends a request through the full server handler (including middleware).
func serveRequest(srv *Server, method, target, body string) *httptest.ResponseRecorder {
	var buf *bytes.Buffer
	if body != "" {
		buf = bytes.NewBufferString(body)
	} else {
		buf = &bytes.Buffer{}
	}
	req := httptest.NewRequest(method, target, buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)
	return rr
}

// serveRequestWithOrigin adds an Origin header (for CORS tests).
func serveRequestWithOrigin(srv *Server, method, target, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("Origin", origin)
	rr := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)
	return rr
}

// callHandler invokes a handler method directly without going through the mux.
// Useful for fine-grained handler unit tests.
func callHandler(srv *Server, handler func(http.ResponseWriter, *http.Request), method, target, body string) *httptest.ResponseRecorder {
	var buf *bytes.Buffer
	if body != "" {
		buf = bytes.NewBufferString(body)
	} else {
		buf = &bytes.Buffer{}
	}
	req := httptest.NewRequest(method, target, buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}
