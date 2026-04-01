package http

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"

	memstore "github.com/kimseunghwan/llm-viz/backend/internal/adapter/storage/memory"
	"github.com/kimseunghwan/llm-viz/backend/internal/port"
	"github.com/kimseunghwan/llm-viz/backend/internal/service"
)

// Server holds dependencies for all HTTP handlers.
type Server struct {
	tracker     *service.TokenTracker
	analytics   *service.AnalyticsService
	keyManager  *service.KeyManager
	broadcaster port.EventBroadcaster
	pricing     port.PricingRepository
	logger      *slog.Logger
	httpServer  *http.Server
}

// NewServer creates the HTTP server with all routes registered.
// A default in-memory KeyManager is created automatically; use SetKeyManager
// to replace it with a production instance before calling ListenAndServe.
func NewServer(
	tracker *service.TokenTracker,
	broadcaster port.EventBroadcaster,
	pricing port.PricingRepository,
	logger *slog.Logger,
	addr string,
	allowedOrigin string,
) *Server {
	s := &Server{
		tracker:     tracker,
		analytics:   service.NewAnalyticsService(tracker.Repo()),
		broadcaster: broadcaster,
		pricing:     pricing,
		logger:      logger,
	}

	// Create a default (ephemeral) key manager used in tests and local dev.
	var defaultKey [32]byte
	if _, err := rand.Read(defaultKey[:]); err != nil {
		panic("failed to generate default encryption key: " + err.Error())
	}
	keyRepo := memstore.NewKeyRepository()
	km, _ := service.NewKeyManager(keyRepo, defaultKey[:])
	s.keyManager = km

	mux := http.NewServeMux()

	// Core routes
	mux.HandleFunc("/api/complete", s.handleCompletion)
	mux.HandleFunc("/api/sse", s.handleSSE)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/models", s.handleModels)
	mux.HandleFunc("/api/health", s.handleHealth)

	// Analytics routes
	mux.HandleFunc("/api/analytics/cumulative", s.handleAnalyticsCumulative)
	mux.HandleFunc("/api/analytics/providers", s.handleAnalyticsProviders)
	mux.HandleFunc("/api/analytics/models", s.handleAnalyticsModels)
	mux.HandleFunc("/api/analytics/hourly", s.handleAnalyticsHourly)

	// Usage history route
	mux.HandleFunc("/api/usage/history", s.handleUsageHistory)

	// Key management routes
	// /api/keys        → collection (GET list, POST save, others → 405)
	// /api/keys/       → item subtree (DELETE /{id}, others → 405)
	mux.HandleFunc("/api/keys", s.handleCollection)
	mux.HandleFunc("/api/keys/", s.handleItem)

	// Apply middleware: CORS → logging → router
	handler := corsMiddleware(mux, allowedOrigin)
	handler = loggingMiddleware(handler, logger)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	return s
}

// SetKeyManager replaces the default key manager with a production instance.
func (s *Server) SetKeyManager(km *service.KeyManager) {
	s.keyManager = km
}

// ListenAndServe starts the HTTP server (blocking).
func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
