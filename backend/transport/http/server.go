package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/kimseunghwan/llm-viz/backend/internal/port"
	"github.com/kimseunghwan/llm-viz/backend/internal/service"
)

// Server holds dependencies for all HTTP handlers.
type Server struct {
	tracker     *service.TokenTracker
	broadcaster port.EventBroadcaster
	pricing     port.PricingRepository
	logger      *slog.Logger
	httpServer  *http.Server
}

// NewServer creates the HTTP server with all routes registered.
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
		broadcaster: broadcaster,
		pricing:     pricing,
		logger:      logger,
	}

	mux := http.NewServeMux()

	// Routes
	mux.HandleFunc("/api/complete", s.handleCompletion)
	mux.HandleFunc("/api/sse", s.handleSSE)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/models", s.handleModels)
	mux.HandleFunc("/api/health", s.handleHealth)

	// Apply middleware: CORS → logging → router
	handler := corsMiddleware(mux, allowedOrigin)
	handler = loggingMiddleware(handler, logger)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	return s
}

// ListenAndServe starts the HTTP server (blocking).
func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
