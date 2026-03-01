# Multi-Provider Architecture — llm-viz Backend

**Date**: 2026-02-27
**Author**: CTO AI (startup-cto-advisor)
**Status**: Approved for Implementation
**Decision**: Hexagonal Architecture + Manual DI + Direct SDK adapters

---

## 1. Executive Summary

### Architecture Approach

**Hexagonal Architecture (Ports & Adapters)** — the core domain defines interfaces (ports), and all external integrations (LLM providers, storage, SSE) implement those interfaces (adapters). The domain is completely isolated from any provider SDK.

**Key decisions**:

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Architecture | Hexagonal (Ports & Adapters) | Domain isolation, mock-friendly, extensible |
| Provider strategy | Option C Hybrid | Direct SDK for OAI/Anthropic/Gemini; OAI-compat for Mistral/Groq/OpenRouter |
| DI framework | Manual (Pure DI) | No magic, explicit, zero overhead — add Wire if complexity grows |
| Logging | `log/slog` (stdlib) | Go 1.21+, structured, zero dependencies |
| Config | `os.Getenv` + `godotenv` | Simple, 12-factor, no framework lock-in |
| Error handling | Sentinel errors + `fmt.Errorf("%w")` | Idiomatic Go, `errors.Is/As` compatible |
| Storage Phase 1-2 | In-memory | Zero ops complexity; swap to Turso in Phase 3 |
| Pricing | `data/pricing.json` flat file | Simple, no DB needed at seed stage |

### Why Hexagonal?

The domain never imports provider SDKs. A new provider = a new adapter file. Tests use mock adapters — no real API calls. The transport layer (HTTP/SSE) is also an adapter, swappable without touching business logic.

---

## 2. Architecture Principles

### Principle 1: Reusability (DRY)

- One `OAICompatAdapter` covers Mistral + Groq + OpenRouter (3 providers, 1 implementation)
- `domain.CalculateCost()` is a pure function reused by all providers
- `domain.TokenUsage` is the single canonical data model
- Shared `mapError()` utility per adapter for consistent error mapping

### Principle 2: Testability

- All external dependencies are behind interfaces → swap with mocks in tests
- Domain logic (cost calculation, cache hit rate) is pure functions → no mocking needed
- Provider adapters have integration tests (tagged `//go:build integration`) — skipped in CI unless keys present
- `MockProvider` is 10 lines, implements the `port.LLMProvider` interface

### Principle 3: Extensibility

- **New provider in ~1 hour**: create `adapter/provider/{name}/adapter.go`, implement `LLMProvider` interface, register in `main.go`
- **New storage backend**: implement `port.UsageRepository`, register in `main.go`
- **New transport**: add a handler, wire to existing service — no domain changes
- Pricing updates: edit `data/pricing.json` — no code change required

### Principle 4: Readability

- Package names match their role: `domain`, `port`, `service`, `adapter/provider/anthropic`
- Interfaces have 3-5 methods max (Go convention)
- No magic: `main.go` explicitly wires every dependency — you can read the full dependency graph in one file
- Comments only where behavior isn't obvious from names

---

## 3. High-Level Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                         HTTP Client (Next.js)                       │
└──────────────┬────────────────────────────────────┬────────────────┘
               │ POST /api/complete                  │ GET /api/sse
               ▼                                     ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Transport Layer (HTTP + SSE)                    │
│  CompletionHandler          SSEHandler                              │
└──────────────┬────────────────────────────────────┬────────────────┘
               │                                     │
               ▼                                     │
┌─────────────────────────────────────────────────────────────────────┐
│                    Service Layer (Use Cases)                         │
│                       TokenTracker                                  │
│   TrackCompletion()  GetSessionHistory()  GetProviderStats()        │
└──┬──────────────┬────────────────────────┬──────────────────────────┘
   │              │                        │
   ▼              ▼                        ▼
┌──────────┐  ┌──────────────┐  ┌──────────────────┐
│  port.   │  │   port.      │  │  port.           │
│ LLMProv- │  │  UsageRepo   │  │  EventBroadcaster│
│  ider   │  │              │  │  (SSE)           │
└──────────┘  └──────────────┘  └──────────────────┘
   │(interface)    │(interface)       │(interface)
   ▼              ▼                  ▼
┌──────────────────────────────────────────────────────────────────────┐
│                      Adapter Layer                                   │
│                                                                      │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌─────────────────┐  │
│  │ Anthropic  │ │   OpenAI   │ │   Gemini   │ │  OAICompat      │  │
│  │  Adapter   │ │  Adapter   │ │  Adapter   │ │  (Mistral/Groq/ │  │
│  │            │ │            │ │            │ │   OpenRouter)   │  │
│  └────────────┘ └────────────┘ └────────────┘ └─────────────────┘  │
│                                                                      │
│  ┌────────────────────────┐  ┌─────────────────────────────────┐    │
│  │  storage/memory        │  │  pricing/jsonfile               │    │
│  │  (Phase 1-2)           │  │  (data/pricing.json)            │    │
│  │  storage/turso (P3)    │  │                                 │    │
│  └────────────────────────┘  └─────────────────────────────────┘    │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  transport/sse  — SSE Broadcaster (goroutine-safe)           │   │
│  └──────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────┘

           │ Provider SDKs (external)
           ▼
   ┌───────────────────────────────────────────────────┐
   │ anthropic-sdk-go │ openai-go │ generative-ai-go   │
   └───────────────────────────────────────────────────┘
```

**Dependency rule**: Arrows point inward. Domain has zero external imports. Service imports only domain + port. Adapters import domain + port + external SDKs.

---

## 4. Directory Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go                   # Entry point: manual DI wiring
│
├── internal/
│   ├── domain/                       # Core entities — ZERO external imports
│   │   ├── provider.go               # ProviderID, ModelInfo, CompletionRequest/Result
│   │   ├── usage.go                  # TokenUsage, NormalizedUsage, ContextWindowStatus
│   │   ├── pricing.go                # PricingEntry, CalculateCost (pure function)
│   │   └── errors.go                 # Sentinel domain errors
│   │
│   ├── port/                         # Interfaces (contracts between layers)
│   │   ├── provider.go               # LLMProvider interface
│   │   ├── repository.go             # UsageRepository, PricingRepository interfaces
│   │   └── broadcaster.go            # EventBroadcaster interface
│   │
│   ├── service/
│   │   └── tracker.go                # TokenTracker — main business logic
│   │
│   └── adapter/
│       ├── provider/
│       │   ├── anthropic/
│       │   │   ├── adapter.go        # Anthropic SDK → LLMProvider
│       │   │   └── adapter_test.go
│       │   ├── openai/
│       │   │   ├── adapter.go        # OpenAI SDK → LLMProvider
│       │   │   └── adapter_test.go
│       │   ├── gemini/
│       │   │   ├── adapter.go        # Gemini SDK → LLMProvider (Phase 2)
│       │   │   └── adapter_test.go
│       │   └── oaicompat/
│       │       ├── adapter.go        # OpenAI SDK (base URL) → Mistral/Groq/OpenRouter
│       │       └── adapter_test.go
│       │
│       ├── storage/
│       │   ├── memory/
│       │   │   └── repository.go     # In-memory UsageRepository (Phase 1-2)
│       │   └── turso/
│       │       └── repository.go     # Turso SQLite (Phase 3)
│       │
│       └── pricing/
│           └── jsonfile/
│               └── repository.go     # Reads data/pricing.json
│
├── transport/
│   ├── http/
│   │   ├── server.go                 # net/http server setup + routes
│   │   ├── handler_completion.go     # POST /api/complete
│   │   ├── handler_stats.go          # GET /api/stats
│   │   └── middleware.go             # CORS, request logging
│   └── sse/
│       └── broadcaster.go            # goroutine-safe SSE broadcaster
│
├── data/
│   └── pricing.json                  # Flat pricing DB (updated via GitHub Actions)
│
├── go.mod
└── go.sum
```

---

## 5. Core Interfaces

```go
// internal/port/provider.go

// LLMProvider is the outbound port for any LLM provider.
// Each provider adapter must implement this interface.
type LLMProvider interface {
    // Complete sends a completion request and returns normalized usage data.
    Complete(ctx context.Context, req domain.CompletionRequest) (*domain.CompletionResult, error)

    // Stream sends a streaming completion request, calling handler for each chunk.
    // The final chunk carries the TokenUsage; earlier chunks have nil Usage.
    Stream(ctx context.Context, req domain.CompletionRequest, handler StreamHandler) error

    // ProviderID returns the canonical identifier for this provider.
    ProviderID() domain.ProviderID

    // SupportedModels returns the list of models available from this provider.
    SupportedModels() []domain.ModelInfo
}

// StreamHandler is invoked for each streaming chunk.
// Return an error to abort the stream.
type StreamHandler func(chunk domain.StreamChunk) error
```

```go
// internal/port/repository.go

// UsageRepository is the outbound port for persisting token usage records.
type UsageRepository interface {
    Save(ctx context.Context, usage domain.NormalizedUsage) error
    FindBySession(ctx context.Context, sessionID string, limit int) ([]domain.NormalizedUsage, error)
    FindByTimeRange(ctx context.Context, filter TimeRangeFilter) ([]domain.NormalizedUsage, error)
    // SumByProvider returns aggregated token counts grouped by provider.
    SumByProvider(ctx context.Context, filter TimeRangeFilter) (map[domain.ProviderID]domain.UsageSummary, error)
}

// PricingRepository is the outbound port for model pricing data.
type PricingRepository interface {
    GetPricing(ctx context.Context, provider domain.ProviderID, modelID string) (*domain.PricingEntry, error)
    ListModels(ctx context.Context) ([]domain.ModelInfo, error)
}

type TimeRangeFilter struct {
    Start      time.Time
    End        time.Time
    ProviderID domain.ProviderID // optional, empty = all providers
    SessionID  string            // optional
}
```

```go
// internal/port/broadcaster.go

// EventBroadcaster is the outbound port for real-time usage event delivery (SSE).
type EventBroadcaster interface {
    // Subscribe creates a buffered channel for a client session.
    // Caller must call Unsubscribe when done to prevent goroutine leaks.
    Subscribe(sessionID string) <-chan domain.UsageEvent

    // Unsubscribe closes and removes the channel for the given session.
    Unsubscribe(sessionID string)

    // Publish sends an event to the subscriber of the matching sessionID.
    // Non-blocking: events are dropped if the subscriber buffer is full.
    Publish(event domain.UsageEvent)
}
```

---

## 6. Data Models

```go
// internal/domain/provider.go

type ProviderID string

const (
    ProviderAnthropic  ProviderID = "anthropic"
    ProviderOpenAI     ProviderID = "openai"
    ProviderGemini     ProviderID = "gemini"
    ProviderMistral    ProviderID = "mistral"
    ProviderGroq       ProviderID = "groq"
    ProviderOpenRouter ProviderID = "openrouter"
)

type ModelInfo struct {
    ID              string     `json:"id"`
    DisplayName     string     `json:"display_name"`
    Provider        ProviderID `json:"provider"`
    ContextWindow   int64      `json:"context_window"`
    MaxOutputTokens int64      `json:"max_output_tokens"`
}

type Message struct {
    Role    string `json:"role"` // "user" | "assistant" | "system"
    Content string `json:"content"`
}

type CompletionRequest struct {
    Model      string    // Provider-specific model ID
    Messages   []Message
    MaxTokens  int
    Stream     bool
    SessionID  string // Used for SSE routing
    ProjectTag string // Optional grouping label
}

type CompletionResult struct {
    ID      string
    Content string
    Usage   TokenUsage
}

type StreamChunk struct {
    Delta   string     // Incremental text content
    IsFinal bool       // True for the last chunk
    Usage   *TokenUsage // Non-nil only in the final chunk
}
```

```go
// internal/domain/usage.go

// TokenUsage is the canonical normalized token structure across all providers.
// Fields not applicable to a provider are zero-valued.
type TokenUsage struct {
    // Core — present for all providers
    InputTokens  int64 `json:"input_tokens"`
    OutputTokens int64 `json:"output_tokens"`
    TotalTokens  int64 `json:"total_tokens"`

    // Cache — Anthropic, Gemini, OpenAI, OpenRouter
    CacheReadTokens  int64 `json:"cache_read_tokens,omitempty"`
    CacheWriteTokens int64 `json:"cache_write_tokens,omitempty"`

    // Reasoning — OpenAI o-series, Gemini thinking, Claude extended thinking
    ReasoningTokens int64 `json:"reasoning_tokens,omitempty"`

    // Timing — Groq only (milliseconds)
    PromptTimeMs     float64 `json:"prompt_time_ms,omitempty"`
    CompletionTimeMs float64 `json:"completion_time_ms,omitempty"`
}

// CacheHitRate returns cache efficiency ratio (0.0–1.0).
// Returns 0 if no cache activity.
func (u TokenUsage) CacheHitRate() float64 {
    total := u.CacheReadTokens + u.CacheWriteTokens
    if total == 0 {
        return 0
    }
    return float64(u.CacheReadTokens) / float64(total)
}

// EffectiveContextUsed returns total context window tokens consumed.
// Accounts for cached tokens which still count against the context window.
func (u TokenUsage) EffectiveContextUsed() int64 {
    return u.InputTokens + u.CacheReadTokens + u.CacheWriteTokens
}

// NormalizedUsage is the persisted record stored after each LLM call.
type NormalizedUsage struct {
    ID         string     `json:"id"`
    Timestamp  time.Time  `json:"timestamp"`
    Provider   ProviderID `json:"provider"`
    Model      string     `json:"model"`
    SessionID  string     `json:"session_id"`
    APIKeyHash string     `json:"api_key_hash"` // SHA256 of user's API key — never store raw
    Usage      TokenUsage `json:"usage"`
    CostUSD    float64    `json:"cost_usd"`
    StreamMode bool       `json:"stream_mode"`
    ProjectTag string     `json:"project_tag,omitempty"`
}

// UsageSummary aggregates token counts for reporting.
type UsageSummary struct {
    Provider     ProviderID `json:"provider"`
    TotalInput   int64      `json:"total_input"`
    TotalOutput  int64      `json:"total_output"`
    TotalCost    float64    `json:"total_cost"`
    RequestCount int64      `json:"request_count"`
}

// ContextWindowStatus describes how full a model's context window is.
type ContextWindowStatus struct {
    Model              string
    MaxTokens          int64
    UsedTokens         int64
    UtilizationPct     float64
    WarningThreshold   bool // >80%
    CriticalThreshold  bool // >95%
}

func NewContextWindowStatus(used, max int64, model string) ContextWindowStatus {
    pct := float64(used) / float64(max) * 100
    return ContextWindowStatus{
        Model:             model,
        MaxTokens:         max,
        UsedTokens:        used,
        UtilizationPct:    pct,
        WarningThreshold:  pct > 80,
        CriticalThreshold: pct > 95,
    }
}

// UsageEvent is published to the SSE broadcaster after each LLM call.
type UsageEvent struct {
    SessionID string          `json:"session_id"`
    Usage     NormalizedUsage `json:"usage"`
}
```

```go
// internal/domain/pricing.go

// PricingEntry holds per-model token pricing. Prices in USD per million tokens.
type PricingEntry struct {
    Provider            ProviderID `json:"provider"`
    ModelID             string     `json:"model_id"`
    DisplayName         string     `json:"display_name"`
    InputPricePerM      float64    `json:"input_price_per_m"`
    OutputPricePerM     float64    `json:"output_price_per_m"`
    CacheReadPricePerM  float64    `json:"cache_read_price_per_m,omitempty"`
    CacheWritePricePerM float64    `json:"cache_write_price_per_m,omitempty"`
    ContextWindow       int64      `json:"context_window"`
    EffectiveDate       time.Time  `json:"effective_date"`
    IsActive            bool       `json:"is_active"`
}

// CalculateCost is a pure function — no side effects, easily testable.
func CalculateCost(usage TokenUsage, p PricingEntry) float64 {
    perM := func(tokens int64, pricePerM float64) float64 {
        return float64(tokens) / 1_000_000 * pricePerM
    }
    return perM(usage.InputTokens, p.InputPricePerM) +
        perM(usage.OutputTokens, p.OutputPricePerM) +
        perM(usage.CacheReadTokens, p.CacheReadPricePerM) +
        perM(usage.CacheWriteTokens, p.CacheWritePricePerM)
}
```

```go
// internal/domain/errors.go

var (
    ErrUnknownProvider     = errors.New("unknown provider")
    ErrProviderUnavailable = errors.New("provider unavailable")
    ErrRateLimited         = errors.New("rate limited by provider")
    ErrInvalidAPIKey       = errors.New("invalid API key")
    ErrContextExceeded     = errors.New("context window exceeded")
    ErrModelNotFound       = errors.New("model not found")
)
```

---

## 7. Provider Adapter Implementations

### 7.1 Anthropic Adapter

```go
// internal/adapter/provider/anthropic/adapter.go
package anthropic

import (
    "context"
    "errors"
    "fmt"

    sdk "github.com/anthropics/anthropic-sdk-go"
    "github.com/anthropics/anthropic-sdk-go/option"
    "github.com/your-org/llm-viz/backend/internal/domain"
)

type Adapter struct {
    client *sdk.Client
    models []domain.ModelInfo
}

func New(apiKey string) *Adapter {
    return &Adapter{
        client: sdk.NewClient(option.WithAPIKey(apiKey)),
        models: supportedModels(),
    }
}

func (a *Adapter) ProviderID() domain.ProviderID { return domain.ProviderAnthropic }
func (a *Adapter) SupportedModels() []domain.ModelInfo { return a.models }

func (a *Adapter) Complete(ctx context.Context, req domain.CompletionRequest) (*domain.CompletionResult, error) {
    resp, err := a.client.Messages.New(ctx, sdk.MessageNewParams{
        Model:     sdk.F(req.Model),
        MaxTokens: sdk.F(int64(req.MaxTokens)),
        Messages:  sdk.F(convertMessages(req.Messages)),
    })
    if err != nil {
        return nil, mapError(err)
    }

    content := ""
    for _, block := range resp.Content {
        if block.Type == sdk.ContentBlockTypeText {
            content += block.Text
        }
    }

    return &domain.CompletionResult{
        ID:      resp.ID,
        Content: content,
        Usage: domain.TokenUsage{
            InputTokens:      int64(resp.Usage.InputTokens),
            OutputTokens:     int64(resp.Usage.OutputTokens),
            TotalTokens:      int64(resp.Usage.InputTokens + resp.Usage.OutputTokens),
            CacheWriteTokens: int64(resp.Usage.CacheCreationInputTokens),
            CacheReadTokens:  int64(resp.Usage.CacheReadInputTokens),
        },
    }, nil
}

func (a *Adapter) Stream(ctx context.Context, req domain.CompletionRequest, handler domain.StreamHandler) error {
    stream := a.client.Messages.NewStreaming(ctx, sdk.MessageNewParams{
        Model:     sdk.F(req.Model),
        MaxTokens: sdk.F(int64(req.MaxTokens)),
        Messages:  sdk.F(convertMessages(req.Messages)),
    })

    accumulated := sdk.Message{}
    for stream.Next() {
        event := stream.Current()
        accumulated.Accumulate(event)

        chunk := domain.StreamChunk{IsFinal: false}
        // Extract text delta from event if available
        if err := handler(chunk); err != nil {
            return err
        }
    }
    if err := stream.Err(); err != nil {
        return mapError(err)
    }

    // Final chunk with usage
    return handler(domain.StreamChunk{
        IsFinal: true,
        Usage: &domain.TokenUsage{
            InputTokens:      int64(accumulated.Usage.InputTokens),
            OutputTokens:     int64(accumulated.Usage.OutputTokens),
            TotalTokens:      int64(accumulated.Usage.InputTokens + accumulated.Usage.OutputTokens),
            CacheWriteTokens: int64(accumulated.Usage.CacheCreationInputTokens),
            CacheReadTokens:  int64(accumulated.Usage.CacheReadInputTokens),
        },
    })
}

// mapError translates Anthropic SDK errors to domain-level errors.
func mapError(err error) error {
    var apiErr *sdk.Error
    if errors.As(err, &apiErr) {
        switch apiErr.StatusCode {
        case 401:
            return fmt.Errorf("%w: anthropic", domain.ErrInvalidAPIKey)
        case 429:
            return fmt.Errorf("%w: anthropic", domain.ErrRateLimited)
        case 400:
            if apiErr.Message == "prompt is too long" {
                return fmt.Errorf("%w: anthropic", domain.ErrContextExceeded)
            }
        }
    }
    return fmt.Errorf("anthropic: %w", err)
}

func convertMessages(msgs []domain.Message) []sdk.MessageParam {
    out := make([]sdk.MessageParam, len(msgs))
    for i, m := range msgs {
        switch m.Role {
        case "user":
            out[i] = sdk.NewUserMessage(sdk.NewTextBlock(m.Content))
        case "assistant":
            out[i] = sdk.NewAssistantMessage(sdk.NewTextBlock(m.Content))
        }
    }
    return out
}

func supportedModels() []domain.ModelInfo {
    return []domain.ModelInfo{
        {ID: "claude-opus-4-6", DisplayName: "Claude Opus 4.6", Provider: domain.ProviderAnthropic, ContextWindow: 200_000, MaxOutputTokens: 128_000},
        {ID: "claude-sonnet-4-6", DisplayName: "Claude Sonnet 4.6", Provider: domain.ProviderAnthropic, ContextWindow: 200_000, MaxOutputTokens: 64_000},
        {ID: "claude-haiku-4-5", DisplayName: "Claude Haiku 4.5", Provider: domain.ProviderAnthropic, ContextWindow: 200_000, MaxOutputTokens: 64_000},
    }
}
```

### 7.2 OpenAI Adapter

```go
// internal/adapter/provider/openai/adapter.go
package openai

import (
    "context"
    "errors"
    "fmt"

    sdk "github.com/openai/openai-go"
    "github.com/openai/openai-go/option"
    "github.com/your-org/llm-viz/backend/internal/domain"
)

type Adapter struct {
    client *sdk.Client
    models []domain.ModelInfo
}

func New(apiKey string) *Adapter {
    return &Adapter{
        client: sdk.NewClient(option.WithAPIKey(apiKey)),
        models: supportedModels(),
    }
}

func (a *Adapter) ProviderID() domain.ProviderID { return domain.ProviderOpenAI }
func (a *Adapter) SupportedModels() []domain.ModelInfo { return a.models }

func (a *Adapter) Complete(ctx context.Context, req domain.CompletionRequest) (*domain.CompletionResult, error) {
    resp, err := a.client.Chat.Completions.New(ctx, sdk.ChatCompletionNewParams{
        Model:    sdk.F(req.Model),
        Messages: sdk.F(convertMessages(req.Messages)),
    })
    if err != nil {
        return nil, mapError(err)
    }

    content := ""
    if len(resp.Choices) > 0 {
        content = resp.Choices[0].Message.Content
    }

    return &domain.CompletionResult{
        ID:      resp.ID,
        Content: content,
        Usage: domain.TokenUsage{
            InputTokens:     resp.Usage.PromptTokens,
            OutputTokens:    resp.Usage.CompletionTokens,
            TotalTokens:     resp.Usage.TotalTokens,
            CacheReadTokens: resp.Usage.PromptTokensDetails.CachedTokens,
            ReasoningTokens: resp.Usage.CompletionTokensDetails.ReasoningTokens,
        },
    }, nil
}

func mapError(err error) error {
    var apiErr *sdk.Error
    if errors.As(err, &apiErr) {
        switch apiErr.StatusCode {
        case 401:
            return fmt.Errorf("%w: openai", domain.ErrInvalidAPIKey)
        case 429:
            return fmt.Errorf("%w: openai", domain.ErrRateLimited)
        }
    }
    return fmt.Errorf("openai: %w", err)
}
```

### 7.3 OpenAI-Compatible Adapter (Mistral, Groq, OpenRouter)

```go
// internal/adapter/provider/oaicompat/adapter.go
package oaicompat

import (
    "context"
    "fmt"

    sdk "github.com/openai/openai-go"
    "github.com/openai/openai-go/option"
    "github.com/your-org/llm-viz/backend/internal/domain"
)

// Config parameterizes the adapter for any OpenAI-compatible provider.
type Config struct {
    Provider domain.ProviderID
    APIKey   string
    BaseURL  string
    Models   []domain.ModelInfo
}

type Adapter struct {
    client   *sdk.Client
    provider domain.ProviderID
    models   []domain.ModelInfo
}

func New(cfg Config) *Adapter {
    return &Adapter{
        client: sdk.NewClient(
            option.WithAPIKey(cfg.APIKey),
            option.WithBaseURL(cfg.BaseURL),
        ),
        provider: cfg.Provider,
        models:   cfg.Models,
    }
}

// Pre-configured factories — one line to add a new OAI-compat provider.

func NewMistral(apiKey string) *Adapter {
    return New(Config{
        Provider: domain.ProviderMistral,
        APIKey:   apiKey,
        BaseURL:  "https://api.mistral.ai/v1",
        Models:   mistralModels(),
    })
}

func NewGroq(apiKey string) *Adapter {
    return New(Config{
        Provider: domain.ProviderGroq,
        APIKey:   apiKey,
        BaseURL:  "https://api.groq.com/openai/v1",
        Models:   groqModels(),
    })
}

func NewOpenRouter(apiKey string) *Adapter {
    return New(Config{
        Provider: domain.ProviderOpenRouter,
        APIKey:   apiKey,
        BaseURL:  "https://openrouter.ai/api/v1",
        Models:   nil, // OpenRouter has 400+ models — list fetched dynamically
    })
}

func (a *Adapter) ProviderID() domain.ProviderID      { return a.provider }
func (a *Adapter) SupportedModels() []domain.ModelInfo { return a.models }

func (a *Adapter) Complete(ctx context.Context, req domain.CompletionRequest) (*domain.CompletionResult, error) {
    resp, err := a.client.Chat.Completions.New(ctx, sdk.ChatCompletionNewParams{
        Model:    sdk.F(req.Model),
        Messages: sdk.F(convertMessages(req.Messages)),
    })
    if err != nil {
        return nil, fmt.Errorf("%s: %w", a.provider, err)
    }

    content := ""
    if len(resp.Choices) > 0 {
        content = resp.Choices[0].Message.Content
    }

    usage := domain.TokenUsage{
        InputTokens:  resp.Usage.PromptTokens,
        OutputTokens: resp.Usage.CompletionTokens,
        TotalTokens:  resp.Usage.TotalTokens,
    }

    // Groq: timing data is in the x-groq-usage header (available via raw response).
    // OpenRouter: cost field is in resp.Usage — requires raw JSON extraction.
    // Both are additive; the base struct is valid without them.

    return &domain.CompletionResult{
        ID:      resp.ID,
        Content: content,
        Usage:   usage,
    }, nil
}
```

---

## 8. Service Layer

```go
// internal/service/tracker.go
package service

// TokenTracker is the application-level service (inbound port implementation).
// It orchestrates: provider call → cost calculation → persistence → SSE broadcast.
type TokenTracker struct {
    providers   map[domain.ProviderID]port.LLMProvider
    repo        port.UsageRepository
    pricing     port.PricingRepository
    broadcaster port.EventBroadcaster
    logger      *slog.Logger
}

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

func (t *TokenTracker) TrackCompletion(ctx context.Context, req port.TrackRequest) (*domain.NormalizedUsage, error) {
    provider, ok := t.providers[req.Provider]
    if !ok {
        return nil, fmt.Errorf("%w: %s", domain.ErrUnknownProvider, req.Provider)
    }

    result, err := provider.Complete(ctx, req.ToCompletionRequest())
    if err != nil {
        return nil, err // already wrapped by adapter's mapError
    }

    pricing, err := t.pricing.GetPricing(ctx, req.Provider, req.Model)
    if err != nil {
        // Non-fatal: log and continue with $0 cost
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
        // Non-fatal: log, still return result to caller
        t.logger.Error("failed to persist usage", "error", err)
    }

    t.broadcaster.Publish(domain.UsageEvent{
        SessionID: req.SessionID,
        Usage:     *normalized,
    })

    return normalized, nil
}

// hashAPIKey returns the first 8 chars of SHA256 hex for safe logging/storage.
func hashAPIKey(key string) string {
    sum := sha256.Sum256([]byte(key))
    return hex.EncodeToString(sum[:])[:8]
}
```

---

## 9. Error Handling Strategy

```
Provider SDK Error
       │
       ▼
adapter/mapError()          ← Maps HTTP status codes to domain errors
       │
       ▼
domain.ErrRateLimited       ← Sentinel error (errors.Is compatible)
       │
       ▼
service/tracker.go           ← Wraps with context: fmt.Errorf("anthropic: %w", err)
       │
       ▼
transport/handler.go         ← Maps to HTTP status + JSON error response
```

**Rules**:
1. Adapters translate provider errors to domain sentinel errors
2. Service layer adds context via `fmt.Errorf("context: %w", err)` — never swallows errors
3. HTTP handlers use `errors.Is(err, domain.ErrRateLimited)` to pick response code
4. Callers never check error strings — only `errors.Is/As`

```go
// transport/http/handler_completion.go — error mapping example
func writeError(w http.ResponseWriter, err error) {
    status := http.StatusInternalServerError
    switch {
    case errors.Is(err, domain.ErrInvalidAPIKey):
        status = http.StatusUnauthorized
    case errors.Is(err, domain.ErrRateLimited):
        status = http.StatusTooManyRequests
    case errors.Is(err, domain.ErrUnknownProvider):
        status = http.StatusBadRequest
    case errors.Is(err, domain.ErrContextExceeded):
        status = http.StatusUnprocessableEntity
    }
    http.Error(w, err.Error(), status)
}
```

---

## 10. Testing Strategy

### Unit Tests — Domain Logic (no mocks)

```go
// internal/domain/pricing_test.go
func TestCalculateCost(t *testing.T) {
    usage := domain.TokenUsage{
        InputTokens:     1000,
        OutputTokens:    500,
        CacheReadTokens: 200,
    }
    pricing := domain.PricingEntry{
        InputPricePerM:    3.00,
        OutputPricePerM:   15.00,
        CacheReadPricePerM: 0.30,
    }
    got := domain.CalculateCost(usage, pricing)
    want := 0.003 + 0.0075 + 0.000060 // = $0.010560
    assert.InDelta(t, want, got, 0.000001)
}

func TestCacheHitRate(t *testing.T) {
    u := domain.TokenUsage{CacheReadTokens: 80, CacheWriteTokens: 20}
    assert.Equal(t, 0.8, u.CacheHitRate())
}
```

### Service Tests — With Mock Adapters

```go
// internal/service/tracker_test.go

// MockProvider implements port.LLMProvider for testing — no real API calls.
type MockProvider struct {
    result *domain.CompletionResult
    err    error
}

func (m *MockProvider) Complete(_ context.Context, _ domain.CompletionRequest) (*domain.CompletionResult, error) {
    return m.result, m.err
}
func (m *MockProvider) Stream(_ context.Context, _ domain.CompletionRequest, _ port.StreamHandler) error {
    return nil
}
func (m *MockProvider) ProviderID() domain.ProviderID      { return "mock" }
func (m *MockProvider) SupportedModels() []domain.ModelInfo { return nil }

func TestTrackCompletion_HappyPath(t *testing.T) {
    mock := &MockProvider{result: &domain.CompletionResult{
        Usage: domain.TokenUsage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150},
    }}
    tracker := service.NewTokenTracker(
        map[domain.ProviderID]port.LLMProvider{"mock": mock},
        memory.NewRepository(),
        &mockPricingRepo{},
        &mockBroadcaster{},
        slog.Default(),
    )

    usage, err := tracker.TrackCompletion(context.Background(), port.TrackRequest{
        Provider:  "mock",
        Model:     "mock-model",
        SessionID: "test-session",
    })
    require.NoError(t, err)
    assert.Equal(t, int64(100), usage.Usage.InputTokens)
}
```

### Integration Tests — Real Provider Calls (tagged)

```go
// internal/adapter/provider/anthropic/adapter_test.go

//go:build integration
// +build integration

func TestAnthropicAdapter_RealAPI(t *testing.T) {
    key := os.Getenv("ANTHROPIC_API_KEY")
    if key == "" {
        t.Skip("ANTHROPIC_API_KEY not set")
    }
    adapter := anthropic.New(key)
    result, err := adapter.Complete(context.Background(), domain.CompletionRequest{
        Model:     "claude-haiku-4-5",
        Messages:  []domain.Message{{Role: "user", Content: "Say 'hi'"}},
        MaxTokens: 10,
    })
    require.NoError(t, err)
    assert.Positive(t, result.Usage.InputTokens)
}
```

Run unit tests: `go test ./...`
Run integration tests: `go test -tags integration ./...`

---

## 11. Technology Stack Recommendations

| Component | Choice | Alternative | Decision Rationale |
|-----------|--------|-------------|-------------------|
| HTTP server | `net/http` (stdlib) | Gin, Echo, Chi | No framework lock-in; Chi is acceptable if routing gets complex |
| Logging | `log/slog` (stdlib, Go 1.21+) | zerolog, zap | Zero deps, structured, good enough for seed stage |
| Config | `os.Getenv` + `godotenv` | viper, envconfig | Simple 12-factor; add envconfig for type safety if config grows |
| DI | Manual (pure DI in `main.go`) | Wire, Fx | Explicit wiring, no magic, readable startup; add Wire at 10+ services |
| UUID | `github.com/google/uuid` | stdlib crypto/rand | Standard, tiny, widely used |
| Testing assertions | `github.com/stretchr/testify` | stdlib only | `assert.Equal` reduces test boilerplate significantly |
| SSE | Custom broadcaster (stdlib) | Mercure | Single-writer, bounded channels, zero overhead |
| Storage P1-2 | In-memory (`sync.Map` or mutex map) | Redis | No ops at seed stage; data resets on restart (acceptable) |
| Storage P3 | Turso SQLite | PostgreSQL | Edge-compatible, free tier, simple ops |
| Pricing | `data/pricing.json` flat file | DB table | Updated via GitHub Actions, no DB needed |

**Go module additions by phase**:

```go
// Phase 1 (Week 1-3)
require (
    github.com/anthropics/anthropic-sdk-go v1.26.0
    github.com/openai/openai-go v1.x
    github.com/google/uuid v1.x
    github.com/stretchr/testify v1.x
    github.com/joho/godotenv v1.x
)

// Phase 2 (Month 2)
require (
    google.golang.org/genai v1.x  // Gemini
)

// Phase 3 (Month 3+) — Mistral/Groq/OpenRouter use openai-go (already present)
// Turso: libsql driver tursodatabase/libsql-client-go (if chosen)
```

---

## 12. SSE Broadcaster Implementation

```go
// transport/sse/broadcaster.go
package sse

import "sync"

// Broadcaster routes usage events to active SSE client sessions.
// One goroutine per client session; channels are buffered to prevent blocking.
type Broadcaster struct {
    mu          sync.RWMutex
    subscribers map[string]chan domain.UsageEvent
}

func NewBroadcaster() *Broadcaster {
    return &Broadcaster{subscribers: make(map[string]chan domain.UsageEvent)}
}

func (b *Broadcaster) Subscribe(sessionID string) <-chan domain.UsageEvent {
    b.mu.Lock()
    defer b.mu.Unlock()
    ch := make(chan domain.UsageEvent, 16) // buffered: handles burst without blocking
    b.subscribers[sessionID] = ch
    return ch
}

func (b *Broadcaster) Unsubscribe(sessionID string) {
    b.mu.Lock()
    defer b.mu.Unlock()
    if ch, ok := b.subscribers[sessionID]; ok {
        close(ch)
        delete(b.subscribers, sessionID)
    }
}

func (b *Broadcaster) Publish(event domain.UsageEvent) {
    b.mu.RLock()
    defer b.mu.RUnlock()
    ch, ok := b.subscribers[event.SessionID]
    if !ok {
        return
    }
    select {
    case ch <- event: // deliver
    default:          // drop if buffer full — non-blocking
    }
}
```

---

## 13. Main.go — Manual Dependency Injection

```go
// cmd/server/main.go
func main() {
    cfg := config.Load() // reads env vars

    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))

    // Build provider adapters — only for configured API keys
    providers := make(map[domain.ProviderID]port.LLMProvider)
    if cfg.AnthropicAPIKey != "" {
        providers[domain.ProviderAnthropic] = anthropicAdapter.New(cfg.AnthropicAPIKey)
    }
    if cfg.OpenAIAPIKey != "" {
        providers[domain.ProviderOpenAI] = openaiAdapter.New(cfg.OpenAIAPIKey)
    }
    if cfg.GeminiAPIKey != "" {
        g, err := geminiAdapter.New(context.Background(), cfg.GeminiAPIKey)
        if err != nil {
            logger.Error("failed to init Gemini adapter", "error", err)
        } else {
            providers[domain.ProviderGemini] = g
        }
    }
    // Phase 3: OAI-compat providers
    if cfg.GroqAPIKey != "" {
        providers[domain.ProviderGroq] = oaicompat.NewGroq(cfg.GroqAPIKey)
    }

    // Build infrastructure
    repo := memory.NewRepository()
    pricingRepo, err := jsonfile.NewRepository(cfg.PricingFilePath)
    if err != nil {
        logger.Error("failed to load pricing", "error", err)
        os.Exit(1)
    }
    broadcaster := sse.NewBroadcaster()

    // Build service
    tracker := service.NewTokenTracker(providers, repo, pricingRepo, broadcaster, logger)

    // Build HTTP server
    srv := transport.NewServer(tracker, broadcaster, logger)

    logger.Info("server starting", "port", cfg.Port)
    if err := srv.ListenAndServe(fmt.Sprintf(":%d", cfg.Port)); err != nil {
        logger.Error("server error", "error", err)
        os.Exit(1)
    }
}
```

**Reading the full dependency graph takes 60 seconds — that's the point of manual DI.**

---

## 14. Migration Path

### Phase 1 — MVP (Week 1-3): OpenAI + Anthropic

**Target**: Real-time token dashboard for 2 providers, Vercel-only (Next.js API routes proxy to Go backend on Railway)

**Deliverables**:
- [ ] `domain/` package (all entities, errors, CalculateCost)
- [ ] `port/` interfaces (LLMProvider, UsageRepository, EventBroadcaster)
- [ ] `adapter/provider/anthropic/` — refactor existing into adapter pattern
- [ ] `adapter/provider/openai/` — new OpenAI adapter
- [ ] `adapter/storage/memory/` — in-memory repository
- [ ] `adapter/pricing/jsonfile/` — pricing.json loader
- [ ] `service/tracker.go` — TokenTracker
- [ ] `transport/http/` — POST /api/complete, GET /api/stats
- [ ] `transport/sse/` — SSE broadcaster
- [ ] `data/pricing.json` — OpenAI + Anthropic pricing
- [ ] Unit tests for domain, service tests with mocks
- [ ] `cmd/server/main.go` — manual DI wiring

**Architecture invariants to enforce in PR review**:
- `domain/` has zero imports from `adapter/` or external SDKs
- `port/` has zero imports from `adapter/`
- `service/` imports only `domain/` and `port/`

---

### Phase 2 — Gemini (Month 2)

**Target**: Cover 3 major providers (~85% market)

**Deliverables**:
- [ ] `adapter/provider/gemini/` using `google.golang.org/genai`
- [ ] Gemini models added to `data/pricing.json`
- [ ] Context window for Gemini 1M token models displayed correctly
- [ ] `thoughtsTokenCount` → `ReasoningTokens` normalization

**No changes required to**: domain, port, service, transport layers.

---

### Phase 3 — Long-Tail Providers + Persistence (Month 3)

**Target**: Mistral, Groq, OpenRouter + token history storage

**Deliverables**:
- [ ] `adapter/provider/oaicompat/` — one adapter, three providers
- [ ] `adapter/storage/turso/` — Turso SQLite repository (swap from memory)
- [ ] Groq timing metrics → `PromptTimeMs`, `CompletionTimeMs` extraction
- [ ] OpenRouter cost passthrough → `CostUSD` field populated directly
- [ ] `data/pricing.json` — Mistral + Groq entries

**No changes to domain, port, or service layers.**

---

## Appendix: Key Model Context Limits (Feb 2026)

```go
// Reference values for ContextWindowStatus calculations
var ModelContextWindows = map[string]int64{
    // Anthropic
    "claude-opus-4-6":   200_000,
    "claude-sonnet-4-6": 200_000,
    "claude-haiku-4-5":  200_000,
    // OpenAI
    "gpt-4o":            128_000,
    "gpt-4o-mini":       128_000,
    "o1":                200_000,
    "o3-mini":           200_000,
    // Gemini
    "gemini-2.0-flash":  1_000_000,
    "gemini-1.5-pro":    2_000_000,
    "gemini-1.5-flash":  1_000_000,
    // Groq (via hosted LLaMA)
    "llama-3.3-70b-versatile": 128_000,
    "llama-3.1-8b-instant":    128_000,
}
```

---

*Sources consulted: [Clean Architecture in Go (Three Dots Labs)](https://threedots.tech/post/introducing-clean-architecture/), [Hexagonal Architecture in Go (Mike Polo)](https://medium.com/@mike_polo/structuring-a-golang-project-hexagonal-architecture-43b4de480c14), [Go DI: Wire vs Fx vs Pure DI (Geison)](https://medium.com/@geisonfgfg/dependency-injection-in-go-fx-vs-wire-vs-pure-di-structuring-maintainable-testable-applications-61c13939fd66), [golang-standards/project-layout](https://github.com/golang-standards/project-layout)*
