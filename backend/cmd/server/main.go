package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/kimseunghwan/llm-viz/backend/internal/adapter/pricing/jsonfile"
	"github.com/kimseunghwan/llm-viz/backend/internal/adapter/provider/anthropic"
	"github.com/kimseunghwan/llm-viz/backend/internal/adapter/provider/openai"
	"github.com/kimseunghwan/llm-viz/backend/internal/adapter/storage/memory"
	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/port"
	"github.com/kimseunghwan/llm-viz/backend/internal/service"
	httpserver "github.com/kimseunghwan/llm-viz/backend/transport/http"
	ssebroadcaster "github.com/kimseunghwan/llm-viz/backend/transport/sse"
)

func main() {
	// Load .env file if present (non-fatal if missing).
	_ = godotenv.Load()

	// Structured JSON logger.
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// --- Build provider adapters (only for configured API keys) ---
	providers := make(map[domain.ProviderID]port.LLMProvider)

	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		providers[domain.ProviderAnthropic] = anthropic.New(key)
		logger.Info("provider configured", "provider", domain.ProviderAnthropic)
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		providers[domain.ProviderOpenAI] = openai.New(key)
		logger.Info("provider configured", "provider", domain.ProviderOpenAI)
	}

	if len(providers) == 0 {
		logger.Warn("no provider API keys configured — set ANTHROPIC_API_KEY and/or OPENAI_API_KEY")
	}

	// --- Build infrastructure ---
	repo := memory.NewRepository()

	pricingPath := os.Getenv("PRICING_FILE")
	if pricingPath == "" {
		pricingPath = "data/pricing.json" // relative to working directory
	}
	pricingRepo, err := jsonfile.NewRepository(pricingPath)
	if err != nil {
		logger.Error("failed to load pricing data", "path", pricingPath, "error", err)
		os.Exit(1)
	}
	logger.Info("pricing data loaded", "path", pricingPath)

	broadcaster := ssebroadcaster.NewBroadcaster()

	// --- Build service ---
	tracker := service.NewTokenTracker(providers, repo, pricingRepo, broadcaster, logger)

	// --- Build HTTP server ---
	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = "8080"
	}
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:3000"
	}

	addr := fmt.Sprintf(":%s", httpPort)
	srv := httpserver.NewServer(tracker, broadcaster, pricingRepo, logger, addr, allowedOrigin)

	// Graceful shutdown on SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("server starting", "addr", addr, "allowed_origin", allowedOrigin)
		if err := srv.ListenAndServe(); err != nil {
			// http.ErrServerClosed is expected on graceful shutdown.
			logger.Error("server stopped", "error", err)
		}
	}()

	<-quit
	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped cleanly")
}
