# Multi-Provider LLM Support: Feasibility & Architecture Research

**Date**: 2026-02-27
**Author**: COO AI (startup-coo)
**Status**: Final
**Decision required by**: CTO + Founder

---

## Executive Summary

**Feasibility verdict: YES — multi-provider support is technically feasible and commercially valuable.**

The top 6 providers all expose token usage in their API responses, have either official or stable community Go SDKs, and their token structures are sufficiently similar to normalize into a unified data model. The key differentiators — cache tokens for Claude/Gemini, reasoning tokens for OpenAI o-series/Gemini thinking models, and ultra-low-latency metrics for Groq — are additive fields that don't break a unified schema.

**Recommended architecture: Option C (Hybrid)** — direct SDK integration for the top 3 providers (OpenAI, Anthropic, Gemini), LiteLLM proxy for the long tail (Cohere, Mistral, Groq, Bedrock, etc.).

**Recommended rollout**:
- Phase 1: OpenAI (largest market) + existing Anthropic
- Phase 2: Gemini (Google ecosystem)
- Phase 3: Long tail via LiteLLM proxy

**Estimated engineering effort**: 3-4 weeks for Phase 1+2 with an AI-agent team.

---

## 1. Provider Token Tracking Analysis

### 1.1 OpenAI (GPT-4o, GPT-4.1, o1, o3-mini)

**Token fields in API responses:**

```json
{
  "usage": {
    "prompt_tokens": 100,
    "completion_tokens": 50,
    "total_tokens": 150,
    "prompt_tokens_details": {
      "cached_tokens": 20,
      "audio_tokens": 0
    },
    "completion_tokens_details": {
      "reasoning_tokens": 10,
      "audio_tokens": 0,
      "accepted_prediction_tokens": 0,
      "rejected_prediction_tokens": 0
    }
  }
}
```

**Key characteristics:**
- Field names use `prompt_tokens` / `completion_tokens` (not input/output)
- Cached tokens are nested inside `prompt_tokens_details` (50% price discount)
- Reasoning tokens (o1/o3) billed at output token rate but separate tracking
- Streaming: requires `stream_options: {"include_usage": true}` to get usage in stream

**Historical usage API**: Yes — `/v1/usage` endpoint, requires Admin API key, granular by model/day.

**Rate limits**: Tier-based (Tier 1: 500 RPM for GPT-4o). Monitored via `x-ratelimit-*` response headers.

---

### 1.2 Anthropic (Claude 3.5 Sonnet, Claude 3 Opus/Haiku)

**Token fields in API responses:**

```json
{
  "usage": {
    "input_tokens": 100,
    "output_tokens": 50,
    "cache_creation_input_tokens": 20,
    "cache_read_input_tokens": 30
  }
}
```

**Key characteristics:**
- Uses `input_tokens` / `output_tokens` (different from OpenAI naming)
- Cache write pricing: 1.25x (5-min TTL) or 2x (1-hour TTL) base input price
- Cache read pricing: 0.1x base input price (90% discount — highest among all providers)
- Extended thinking (claude-3-7-sonnet): separate `thinking` block token tracking
- Streaming: usage included in final `message_delta` event

**Historical usage API**: Admin API (beta) — `/v1/usage` with Bearer token.

---

### 1.3 Google Gemini (Gemini 2.0 Flash, 1.5 Pro/Flash)

**Token fields in API responses:**

```json
{
  "usageMetadata": {
    "promptTokenCount": 100,
    "candidatesTokenCount": 50,
    "totalTokenCount": 150,
    "cachedContentTokenCount": 20,
    "thoughtsTokenCount": 10
  }
}
```

Note: Newer Interactions API format uses `usage` object:

```json
{
  "usage": {
    "input_tokens_by_modality": {"text": 100},
    "total_cached_tokens": 20,
    "total_input_tokens": 100,
    "total_output_tokens": 50,
    "total_thought_tokens": 10,
    "total_tokens": 150
  }
}
```

**Key characteristics:**
- Field names use camelCase `usageMetadata` in legacy API, snake_case in new Interactions API
- `thoughtsTokenCount` for Gemini 2.0 thinking models (billed as output tokens)
- Context caching (server-side): `cachedContentTokenCount` tracked separately
- Free tier: 15 RPM, 1 million TPM for Gemini 1.5 Flash
- 1M token context window is a significant differentiator

---

### 1.4 Cohere (Command R+, Command R)

**Token fields in API responses:**

```json
{
  "meta": {
    "billed_units": {
      "input_tokens": 100,
      "output_tokens": 50
    },
    "tokens": {
      "input_tokens": 120,
      "output_tokens": 55
    }
  }
}
```

**Key characteristics:**
- Dual tracking: `billed_units` (for billing) vs `tokens` (actual usage — may differ due to overhead)
- No explicit cache token tracking in current API
- Official Go SDK: `github.com/cohere-ai/cohere-go/v2` (Fern-generated, well-maintained)
- OpenAI compatibility endpoint available: can use via OpenAI SDK with base URL change
- Free tier: 1,000 API calls/month (trial key)

---

### 1.5 Mistral AI (Mistral Large 3, Mistral Small, Codestral)

**Token fields in API responses:**

```json
{
  "usage": {
    "prompt_tokens": 100,
    "completion_tokens": 50,
    "total_tokens": 150
  }
}
```

**Key characteristics:**
- Same field names as OpenAI (`prompt_tokens` / `completion_tokens`) — OpenAI-compatible API
- No cache token tracking (as of early 2026)
- Community Go SDK: `github.com/gage-technologies/mistral-go` (active, not official)
- Can use OpenAI Go SDK via base URL override (`https://api.mistral.ai/v1`)
- Function/tool calling supported

---

### 1.6 Groq (LLaMA 3.3 70B, Llama 3.1 8B via Groq)

**Token fields in API responses:**

```json
{
  "usage": {
    "prompt_tokens": 100,
    "completion_tokens": 50,
    "total_tokens": 150,
    "prompt_time": 0.03,
    "completion_time": 0.07,
    "total_time": 0.10,
    "queue_time": 0.001
  }
}
```

**Key characteristics:**
- Unique: includes timing fields (`prompt_time`, `completion_time`, `queue_time` in seconds)
- These timing fields are critical for Groq's positioning (ultra-fast inference)
- OpenAI-compatible API — use OpenAI SDK with `https://api.groq.com/openai/v1`
- Community Go SDK: `github.com/jpoz/groq` or `github.com/magicx-ai/groq-go`
- Batch API: 50% discount on real-time rates

---

### 1.7 AWS Bedrock (Multi-model gateway)

**Token fields via Converse API:**

```json
{
  "usage": {
    "inputTokens": 100,
    "outputTokens": 50,
    "totalTokens": 150,
    "cacheReadInputTokensCount": 20,
    "cacheWriteInputTokensCount": 10
  }
}
```

**Key characteristics:**
- Unified Converse API wraps Claude, Llama, Mistral, Titan, Cohere models
- Official Go SDK: `github.com/aws/aws-sdk-go-v2/service/bedrockruntime`
- Token usage varies by underlying model (Bedrock normalizes it)
- Requires AWS account + IAM setup — higher friction for token-viz users
- Pricing: model pricing + Bedrock markup (typically 10-20% on top)

**Recommendation**: Deprioritize. Target enterprise customers only. Implement in Phase 3+ via LiteLLM.

---

### 1.8 OpenRouter (API Gateway — 400+ models)

**Token fields in API responses (OpenAI-compatible):**

```json
{
  "usage": {
    "prompt_tokens": 100,
    "completion_tokens": 50,
    "total_tokens": 150,
    "reasoning_tokens": 10,
    "cached_tokens": 20,
    "cache_write_tokens": 5,
    "cost": 0.0025,
    "upstream_inference_cost": 0.002
  }
}
```

**Key characteristics:**
- Automatic cost field included in every response — no manual pricing calculation needed
- Acts as a drop-in OpenAI replacement (same SDK, just change base URL)
- Access to 400+ models via single API key
- Usage accounting built-in: no additional params required
- BYOK (Bring Your Own Key): `upstream_inference_cost` tracked separately
- **High strategic value for token-viz**: use OpenRouter as an optional "all other providers" adapter

---

## 2. Go SDK Support Matrix

| Provider | Official Go SDK | Community SDK | OpenAI-Compatible | Token Tracking | Streaming Usage |
|----------|----------------|---------------|------------------|----------------|-----------------|
| **OpenAI** | `github.com/openai/openai-go` v1.x | `sashabaranov/go-openai` (11k+ stars) | Native | Full (incl. reasoning, cache) | Yes (stream_options flag) |
| **Anthropic** | `github.com/anthropics/anthropic-sdk-go` v1.26.0 | - | No | Full (incl. cache write/read) | Yes (final delta event) |
| **Gemini** | `github.com/google/generative-ai-go` | - | No (separate format) | Full (incl. cached, thoughts) | Yes |
| **Cohere** | `github.com/cohere-ai/cohere-go/v2` | - | Partial (compat endpoint) | Full (billed_units + tokens) | Yes |
| **Mistral** | None | `gage-technologies/mistral-go` | Yes (same format as OAI) | Basic (no cache) | Yes |
| **Groq** | None | `jpoz/groq`, `magicx-ai/groq-go` | Yes (+ timing fields) | Full (incl. timing metrics) | Yes |
| **AWS Bedrock** | `aws-sdk-go-v2/bedrockruntime` | - | No (separate Converse API) | Full (incl. cache) | Yes |
| **OpenRouter** | None | Use any OpenAI Go SDK | Yes (drop-in) | Full (incl. cost field) | Yes |

**SDK Quality Assessment:**
- Tier 1 (production-ready): OpenAI official, Anthropic official, Gemini official, Cohere official, AWS Bedrock official
- Tier 2 (stable community): `sashabaranov/go-openai` (de facto standard before official SDK)
- Tier 3 (use via OpenAI SDK): Mistral, Groq, OpenRouter (all OpenAI-compatible)

---

## 3. Unified Integration Libraries

### 3.1 LiteLLM

**What it does**: Python SDK + proxy server that calls 100+ LLMs in OpenAI format. Acts as an AI gateway with cost tracking, guardrails, load balancing, and logging.

**Token tracking capabilities**:
- Tracks `spend`, `prompt_tokens`, `completion_tokens`, `total_tokens` per request
- Stores by model, provider, API key, team, tag
- Granular daily usage endpoint: `/usage?group_by=model&start_date=...`
- Built-in pricing database for all supported models

**Go integration approach**:
- Run LiteLLM as a sidecar proxy (Docker: `ghcr.io/berriai/litellm:main`)
- token-viz Go server sends all requests to `http://localhost:4000/v1` (OpenAI format)
- LiteLLM handles provider routing, retries, token tracking
- token-viz reads LiteLLM spend logs via API

**Pros**:
- Zero provider-specific code in token-viz
- Automatic new provider support as LiteLLM adds them
- Built-in pricing database (maintained by LiteLLM team)
- Excellent for Cohere, Mistral, Groq, Bedrock, 90+ other providers

**Cons**:
- Adds a Python runtime dependency (ops complexity)
- Network hop adds ~5-20ms latency
- Token data lives in LiteLLM DB, not token-viz DB (data ownership issue)
- LiteLLM proxy is overkill if users just want to track their own usage
- CRITICAL: token-viz's value proposition is tracking, not proxying — LiteLLM proxy conflicts with the product model if token-viz is meant to be a monitoring sidebar, not a request router

---

### 3.2 LangChain (langchaingo)

**Go port**: `github.com/tmc/langchaingo`

**Assessment**:
- Provides multi-provider chat interfaces but is primarily an application framework
- Token counting utilities exist but are not the focus
- Too heavy — brings application scaffolding when we only need token normalization
- **Not recommended** for token-viz's specific use case

---

### 3.3 OpenRouter

**What it does**: API gateway for 400+ models. OpenAI-compatible. Auto-includes cost in every response.

**For token-viz**:
- If users configure token-viz with their OpenRouter API key, a single adapter handles all 400+ models
- Cost field eliminates need for pricing DB maintenance
- **Recommended as optional "catch-all" provider** in Phase 3

---

### 3.4 Other Gateways

| Tool | Focus | Token Tracking | Go Support | Verdict for token-viz |
|------|-------|---------------|------------|----------------------|
| **Cloudflare AI Gateway** | Caching + routing | Basic | Via REST | No — limited provider support |
| **Portkey** | Enterprise gateway | Full | Via REST | Maybe Phase 3 — good for enterprise |
| **Helicone** | Observability | Full | Via REST proxy | Competitor, not a dep |
| **LangSmith** | LangChain ecosystem | Full | Via Python | No — Python-only |

---

## 4. Token Data Standardization

### 4.1 Raw Format Comparison

```
Provider    | Field Names                           | Nested Details
------------|---------------------------------------|---------------------------------------------
OpenAI      | prompt_tokens, completion_tokens      | cached_tokens, reasoning_tokens (nested)
Anthropic   | input_tokens, output_tokens           | cache_creation_input_tokens, cache_read_input_tokens (flat)
Gemini      | promptTokenCount, candidatesTokenCount | cachedContentTokenCount, thoughtsTokenCount
Cohere      | billed_units.input_tokens, ...        | tokens.input_tokens (dual: billed vs actual)
Mistral     | prompt_tokens, completion_tokens      | (none)
Groq        | prompt_tokens, completion_tokens      | prompt_time, completion_time (timing, not tokens)
Bedrock     | inputTokens, outputTokens             | cacheReadInputTokensCount, cacheWriteInputTokensCount
OpenRouter  | prompt_tokens, completion_tokens      | cached_tokens, cache_write_tokens, cost (flat)
```

### 4.2 Proposed Unified Data Model (Go)

```go
// TokenUsage represents normalized token usage across all providers
type TokenUsage struct {
    // Core fields — present for all providers
    InputTokens  int64 `json:"input_tokens" db:"input_tokens"`
    OutputTokens int64 `json:"output_tokens" db:"output_tokens"`
    TotalTokens  int64 `json:"total_tokens" db:"total_tokens"`

    // Cache fields — Anthropic, Gemini, OpenAI, OpenRouter, Bedrock
    CacheReadTokens  int64 `json:"cache_read_tokens,omitempty" db:"cache_read_tokens"`
    CacheWriteTokens int64 `json:"cache_write_tokens,omitempty" db:"cache_write_tokens"`
    CacheHitTokens   int64 `json:"cache_hit_tokens,omitempty" db:"cache_hit_tokens"` // alias for read

    // Reasoning/thinking tokens — OpenAI o-series, Gemini thinking, Claude extended thinking
    ReasoningTokens int64 `json:"reasoning_tokens,omitempty" db:"reasoning_tokens"`

    // Timing — Groq-specific
    PromptTimeMs     float64 `json:"prompt_time_ms,omitempty" db:"prompt_time_ms"`
    CompletionTimeMs float64 `json:"completion_time_ms,omitempty" db:"completion_time_ms"`
    TotalTimeMs      float64 `json:"total_time_ms,omitempty" db:"total_time_ms"`

    // Cost — OpenRouter provides natively; others computed from pricing DB
    CostUSD float64 `json:"cost_usd,omitempty" db:"cost_usd"`
}

// NormalizedUsage is the full record stored in token-viz DB
type NormalizedUsage struct {
    ID         string    `json:"id" db:"id"`
    Timestamp  time.Time `json:"timestamp" db:"timestamp"`
    Provider   Provider  `json:"provider" db:"provider"`
    Model      string    `json:"model" db:"model"`
    APIKeyHash string    `json:"api_key_hash" db:"api_key_hash"` // SHA256, never store raw key
    Usage      TokenUsage `json:"usage"`

    // Metadata
    RequestID   string `json:"request_id,omitempty" db:"request_id"`
    StreamMode  bool   `json:"stream_mode" db:"stream_mode"`
    ProjectTag  string `json:"project_tag,omitempty" db:"project_tag"`
}

type Provider string
const (
    ProviderOpenAI    Provider = "openai"
    ProviderAnthropic Provider = "anthropic"
    ProviderGemini    Provider = "gemini"
    ProviderCohere    Provider = "cohere"
    ProviderMistral   Provider = "mistral"
    ProviderGroq      Provider = "groq"
    ProviderBedrock   Provider = "bedrock"
    ProviderOpenRouter Provider = "openrouter"
)
```

### 4.3 Provider-Specific Adapters (Normalization Layer)

```go
// Adapter interface
type UsageAdapter interface {
    Normalize(raw interface{}) (*TokenUsage, error)
    Provider() Provider
}

// OpenAI adapter example
type OpenAIAdapter struct{}

func (a *OpenAIAdapter) Normalize(raw interface{}) (*TokenUsage, error) {
    r := raw.(*openai.CompletionUsage)
    return &TokenUsage{
        InputTokens:     int64(r.PromptTokens),
        OutputTokens:    int64(r.CompletionTokens),
        TotalTokens:     int64(r.TotalTokens),
        CacheReadTokens: int64(r.PromptTokensDetails.CachedTokens),
        ReasoningTokens: int64(r.CompletionTokensDetails.ReasoningTokens),
    }, nil
}

// Anthropic adapter example
type AnthropicAdapter struct{}

func (a *AnthropicAdapter) Normalize(raw interface{}) (*TokenUsage, error) {
    r := raw.(*anthropic.Usage)
    return &TokenUsage{
        InputTokens:      int64(r.InputTokens),
        OutputTokens:     int64(r.OutputTokens),
        TotalTokens:      int64(r.InputTokens + r.OutputTokens),
        CacheWriteTokens: int64(r.CacheCreationInputTokens),
        CacheReadTokens:  int64(r.CacheReadInputTokens),
    }, nil
}

// Gemini adapter example
type GeminiAdapter struct{}

func (a *GeminiAdapter) Normalize(raw interface{}) (*TokenUsage, error) {
    r := raw.(*genai.GenerateContentResponse)
    m := r.UsageMetadata
    return &TokenUsage{
        InputTokens:     int64(m.PromptTokenCount),
        OutputTokens:    int64(m.CandidatesTokenCount),
        TotalTokens:     int64(m.TotalTokenCount),
        CacheReadTokens: int64(m.CachedContentTokenCount),
        ReasoningTokens: int64(m.ThoughtsTokenCount),
    }, nil
}
```

---

## 5. Pricing Database

### 5.1 Pricing Comparison Table (per million tokens, Feb 2026)

| Provider | Model | Input $/M | Output $/M | Cache Read $/M | Cache Write $/M | Free Tier |
|----------|-------|-----------|------------|----------------|-----------------|-----------|
| **OpenAI** | GPT-4o | $2.50 | $10.00 | $1.25 | N/A | No |
| **OpenAI** | GPT-4o mini | $0.15 | $0.60 | $0.075 | N/A | No |
| **OpenAI** | o3-mini | $1.10 | $4.40 | $0.55 | N/A | No |
| **OpenAI** | o1 | $15.00 | $60.00 | $7.50 | N/A | No |
| **Anthropic** | Claude 3.5 Sonnet | $3.00 | $15.00 | $0.30 | $3.75 (5min) | No |
| **Anthropic** | Claude 3.5 Haiku | $0.80 | $4.00 | $0.08 | $1.00 (5min) | No |
| **Anthropic** | Claude 3 Opus | $15.00 | $75.00 | $1.50 | $18.75 (5min) | No |
| **Gemini** | Gemini 2.0 Flash | $0.10 | $0.40 | $0.025 | N/A | Yes (15 RPM) |
| **Gemini** | Gemini 1.5 Pro | $1.25 | $5.00 | $0.3125 | N/A | Yes (2 RPM) |
| **Gemini** | Gemini 1.5 Flash | $0.075 | $0.30 | N/A | N/A | Yes (15 RPM) |
| **Cohere** | Command R+ (08-24) | $2.50 | $10.00 | N/A | N/A | 1K calls/mo |
| **Cohere** | Command R | $0.15 | $0.60 | N/A | N/A | 1K calls/mo |
| **Mistral** | Mistral Large 3 | $0.80 | $2.40 | N/A | N/A | No |
| **Mistral** | Mistral Small | $0.10 | $0.30 | N/A | N/A | No |
| **Groq** | LLaMA 3.3 70B | $0.59 | $0.79 | N/A | N/A | Limited free |
| **Groq** | LLaMA 3.1 8B | $0.05 | $0.08 | N/A | N/A | Limited free |

### 5.2 Pricing DB Schema

```go
// PricingEntry stores per-model token pricing
type PricingEntry struct {
    Provider        Provider  `json:"provider" db:"provider"`
    ModelID         string    `json:"model_id" db:"model_id"`
    ModelDisplayName string   `json:"model_display_name" db:"model_display_name"`

    // Per million token pricing (USD)
    InputPricePerM       float64 `json:"input_price_per_m" db:"input_price_per_m"`
    OutputPricePerM      float64 `json:"output_price_per_m" db:"output_price_per_m"`
    CacheReadPricePerM   float64 `json:"cache_read_price_per_m,omitempty" db:"cache_read_price_per_m"`
    CacheWritePricePerM  float64 `json:"cache_write_price_per_m,omitempty" db:"cache_write_price_per_m"`

    // Context window
    ContextWindowTokens int64 `json:"context_window_tokens" db:"context_window_tokens"`

    // Metadata
    EffectiveDate time.Time `json:"effective_date" db:"effective_date"`
    IsActive      bool      `json:"is_active" db:"is_active"`
    Source        string    `json:"source" db:"source"` // "official_docs" | "community"
}

// CostCalculator computes cost from TokenUsage + PricingEntry
func CalculateCost(usage TokenUsage, pricing PricingEntry) float64 {
    inputCost  := float64(usage.InputTokens)       / 1_000_000 * pricing.InputPricePerM
    outputCost := float64(usage.OutputTokens)      / 1_000_000 * pricing.OutputPricePerM
    cacheRead  := float64(usage.CacheReadTokens)   / 1_000_000 * pricing.CacheReadPricePerM
    cacheWrite := float64(usage.CacheWriteTokens)  / 1_000_000 * pricing.CacheWritePricePerM
    return inputCost + outputCost + cacheRead + cacheWrite
}
```

**Maintenance strategy**: JSON flat file in repo (`/data/pricing.json`). GitHub Actions cron checks official pricing pages weekly. PR raised when changes detected. No database needed at seed stage.

---

## 6. Architecture Options

### Option A: Direct SDK Integration

```
token-viz Go server
├── /adapters/openai.go   → github.com/openai/openai-go
├── /adapters/anthropic.go → github.com/anthropics/anthropic-sdk-go
├── /adapters/gemini.go   → github.com/google/generative-ai-go
├── /adapters/cohere.go   → github.com/cohere-ai/cohere-go/v2
├── /adapters/mistral.go  → OpenAI SDK (base URL override)
└── /adapters/groq.go     → OpenAI SDK (base URL override)
```

**Pros**:
- Full control over token data — lives in token-viz DB
- No external dependencies at runtime
- Best performance (no proxy hop)
- Simplest deployment model (single binary)

**Cons**:
- Must import each SDK individually
- Pricing DB is your responsibility to maintain
- Adding new provider = new adapter code

---

### Option B: LiteLLM Proxy

```
token-viz Go server
    ↓ (OpenAI format)
LiteLLM Proxy (Python, Docker)
    ├── OpenAI
    ├── Anthropic
    ├── Gemini
    ├── Cohere
    └── ... (100+ providers)
```

**Pros**:
- Single HTTP adapter in token-viz
- LiteLLM maintains pricing DB
- 100+ providers automatically

**Cons**:
- Python runtime dependency (Docker required for users)
- LiteLLM proxy conflicts with token-viz's positioning: if token-viz IS the proxy, it loses appeal as a monitoring sidebar
- token-viz becomes a proxy product, not a monitoring product
- Token data stored in LiteLLM's DB (not token-viz's)
- Cold start / reliability concerns for user-deployed instances

**Verdict**: Not recommended as primary architecture. OK as an optional mode for power users.

---

### Option C: Hybrid (RECOMMENDED)

```
token-viz Go server
├── Provider Adapters (direct SDK):
│   ├── OpenAI adapter    → openai/openai-go (official)
│   ├── Anthropic adapter → anthropics/anthropic-sdk-go (official)
│   └── Gemini adapter    → google/generative-ai-go (official)
│
├── OpenAI-Compatible Adapter (reusable):
│   ├── Mistral (base URL override)
│   ├── Groq (base URL override + timing field extraction)
│   └── OpenRouter (base URL override + cost field extraction)
│
└── [Phase 3] LiteLLM sidecar (optional, for enterprise Bedrock/Cohere/others)
```

**Pros**:
- 6+ providers covered with ~4 adapter implementations
- Mistral + Groq + OpenRouter share the same OpenAI-compatible adapter
- Full data ownership — all token data in token-viz DB
- Single binary deployment for MVP (no Docker sidecar required)
- Clean separation of concerns

**Cons**:
- Long-tail providers (AWS Bedrock, Azure OpenAI, Together AI) require LiteLLM in Phase 3
- Pricing DB maintenance (mitigated by JSON flat file + weekly cron)

---

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│                   token-viz Go Server                    │
│                                                         │
│  ┌──────────────┐    ┌──────────────────────────────┐  │
│  │  HTTP Handler │───▶│     Provider Router          │  │
│  └──────────────┘    └──────────────────────────────┘  │
│                              │                          │
│         ┌────────────────────┼────────────────────┐     │
│         ▼                    ▼                    ▼     │
│   ┌──────────┐       ┌──────────────┐     ┌──────────┐ │
│   │ Anthropic │       │   OpenAI     │     │  Gemini  │ │
│   │ Adapter  │       │  Adapter     │     │ Adapter  │ │
│   └──────────┘       └──────────────┘     └──────────┘ │
│         │                    │                    │     │
│         │            ┌───────┴───────┐            │     │
│         │            ▼               ▼            │     │
│         │      ┌──────────┐  ┌──────────┐        │     │
│         │      │  Mistral │  │   Groq   │        │     │
│         │      │ (OAI compat)│ (OAI compat+time)│ │     │
│         │      └──────────┘  └──────────┘        │     │
│         ▼                                         ▼     │
│   ┌─────────────────────────────────────────────────┐  │
│   │              UsageNormalizer                    │  │
│   │   raw usage → TokenUsage → NormalizedUsage      │  │
│   └─────────────────────────────────────────────────┘  │
│                         │                               │
│                         ▼                               │
│   ┌─────────────────────────────────────────────────┐  │
│   │         CostCalculator (pricing.json)           │  │
│   └─────────────────────────────────────────────────┘  │
│                         │                               │
│                         ▼                               │
│                    ┌──────────┐                         │
│                    │    DB    │                         │
│                    └──────────┘                         │
└─────────────────────────────────────────────────────────┘
```

---

## 7. Working Code Examples

### 7.1 OpenAI Go SDK — Token Extraction

```go
package adapters

import (
    "context"
    "github.com/openai/openai-go"
    "github.com/openai/openai-go/option"
)

type OpenAIAdapter struct {
    client *openai.Client
}

func NewOpenAIAdapter(apiKey string) *OpenAIAdapter {
    client := openai.NewClient(option.WithAPIKey(apiKey))
    return &OpenAIAdapter{client: client}
}

func (a *OpenAIAdapter) Complete(ctx context.Context, model, prompt string) (*NormalizedUsage, error) {
    resp, err := a.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
        Model: openai.F(openai.ChatModelGPT4o),
        Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
            openai.UserMessage(prompt),
        }),
    })
    if err != nil {
        return nil, err
    }

    usage := &TokenUsage{
        InputTokens:     resp.Usage.PromptTokens,
        OutputTokens:    resp.Usage.CompletionTokens,
        TotalTokens:     resp.Usage.TotalTokens,
        CacheReadTokens: resp.Usage.PromptTokensDetails.CachedTokens,
        ReasoningTokens: resp.Usage.CompletionTokensDetails.ReasoningTokens,
    }

    return &NormalizedUsage{
        Provider: ProviderOpenAI,
        Model:    model,
        Usage:    *usage,
    }, nil
}

// Streaming with usage — requires stream_options
func (a *OpenAIAdapter) StreamComplete(ctx context.Context, model, prompt string) (*TokenUsage, error) {
    stream := a.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
        Model: openai.F(openai.ChatModelGPT4o),
        Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
            openai.UserMessage(prompt),
        }),
        StreamOptions: openai.F(openai.ChatCompletionStreamOptionsParam{
            IncludeUsage: openai.F(true), // Required for usage in stream
        }),
    })

    var lastUsage *TokenUsage
    for stream.Next() {
        chunk := stream.Current()
        if chunk.Usage.TotalTokens > 0 {
            lastUsage = &TokenUsage{
                InputTokens:  chunk.Usage.PromptTokens,
                OutputTokens: chunk.Usage.CompletionTokens,
                TotalTokens:  chunk.Usage.TotalTokens,
            }
        }
    }
    return lastUsage, stream.Err()
}
```

### 7.2 Anthropic Go SDK — Token Extraction

```go
package adapters

import (
    "context"
    "github.com/anthropics/anthropic-sdk-go"
    "github.com/anthropics/anthropic-sdk-go/option"
)

type AnthropicAdapter struct {
    client *anthropic.Client
}

func NewAnthropicAdapter(apiKey string) *AnthropicAdapter {
    client := anthropic.NewClient(option.WithAPIKey(apiKey))
    return &AnthropicAdapter{client: client}
}

func (a *AnthropicAdapter) Complete(ctx context.Context, model, prompt string) (*NormalizedUsage, error) {
    resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
        Model:     anthropic.F(anthropic.ModelClaude3_5SonnetLatest),
        MaxTokens: anthropic.F(int64(1024)),
        Messages: anthropic.F([]anthropic.MessageParam{
            anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
        }),
    })
    if err != nil {
        return nil, err
    }

    usage := &TokenUsage{
        InputTokens:      int64(resp.Usage.InputTokens),
        OutputTokens:     int64(resp.Usage.OutputTokens),
        TotalTokens:      int64(resp.Usage.InputTokens + resp.Usage.OutputTokens),
        CacheWriteTokens: int64(resp.Usage.CacheCreationInputTokens),
        CacheReadTokens:  int64(resp.Usage.CacheReadInputTokens),
    }

    return &NormalizedUsage{
        Provider: ProviderAnthropic,
        Model:    model,
        Usage:    *usage,
    }, nil
}
```

### 7.3 Gemini Go SDK — Token Extraction

```go
package adapters

import (
    "context"
    "google.golang.org/genai"
)

type GeminiAdapter struct {
    client *genai.Client
}

func NewGeminiAdapter(ctx context.Context, apiKey string) (*GeminiAdapter, error) {
    client, err := genai.NewClient(ctx, &genai.ClientConfig{
        APIKey:  apiKey,
        Backend: genai.BackendGeminiAPI,
    })
    if err != nil {
        return nil, err
    }
    return &GeminiAdapter{client: client}, nil
}

func (a *GeminiAdapter) Complete(ctx context.Context, model, prompt string) (*NormalizedUsage, error) {
    resp, err := a.client.Models.GenerateContent(ctx, model,
        genai.Text(prompt), nil)
    if err != nil {
        return nil, err
    }

    m := resp.UsageMetadata
    usage := &TokenUsage{
        InputTokens:     int64(m.PromptTokenCount),
        OutputTokens:    int64(m.CandidatesTokenCount),
        TotalTokens:     int64(m.TotalTokenCount),
        CacheReadTokens: int64(m.CachedContentTokenCount),
        ReasoningTokens: int64(m.ThoughtsTokenCount),
    }

    return &NormalizedUsage{
        Provider: ProviderGemini,
        Model:    model,
        Usage:    *usage,
    }, nil
}
```

### 7.4 OpenAI-Compatible Adapter (covers Mistral, Groq, OpenRouter)

```go
package adapters

import (
    "context"
    "github.com/openai/openai-go"
    "github.com/openai/openai-go/option"
)

type OpenAICompatAdapter struct {
    client   *openai.Client
    provider Provider
}

func NewOpenAICompatAdapter(provider Provider, apiKey, baseURL string) *OpenAICompatAdapter {
    client := openai.NewClient(
        option.WithAPIKey(apiKey),
        option.WithBaseURL(baseURL),
    )
    return &OpenAICompatAdapter{client: client, provider: provider}
}

// Factory for common providers
func NewMistralAdapter(apiKey string) *OpenAICompatAdapter {
    return NewOpenAICompatAdapter(ProviderMistral, apiKey, "https://api.mistral.ai/v1")
}

func NewGroqAdapter(apiKey string) *OpenAICompatAdapter {
    return NewOpenAICompatAdapter(ProviderGroq, apiKey, "https://api.groq.com/openai/v1")
}

func NewOpenRouterAdapter(apiKey string) *OpenAICompatAdapter {
    return NewOpenAICompatAdapter(ProviderOpenRouter, apiKey, "https://openrouter.ai/api/v1")
}

func (a *OpenAICompatAdapter) Complete(ctx context.Context, model, prompt string) (*NormalizedUsage, error) {
    resp, err := a.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
        Model: openai.F(model),
        Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
            openai.UserMessage(prompt),
        }),
    })
    if err != nil {
        return nil, err
    }

    usage := &TokenUsage{
        InputTokens:  resp.Usage.PromptTokens,
        OutputTokens: resp.Usage.CompletionTokens,
        TotalTokens:  resp.Usage.TotalTokens,
    }

    // Groq: extract timing from metadata (available in raw JSON)
    // OpenRouter: extract cost field if present

    return &NormalizedUsage{
        Provider: a.provider,
        Model:    model,
        Usage:    *usage,
    }, nil
}
```

---

## 8. Competitor Analysis

### 8.1 Langfuse

**Provider support**: OpenAI, Anthropic, Gemini, Cohere, Mistral, Bedrock, Azure OpenAI, 20+ others via OpenAI-compatible format.

**Token tracking**: Full token counts per request, cost tracking (requires pricing config), traces/spans model.

**Gaps for token-viz**:
- Langfuse is observability-first (traces, evals, prompts) — token cost is secondary
- No real-time dashboard focused purely on token spend visualization
- Self-hosted or cloud (Langfuse Cloud: $59/mo Pro tier)

**token-viz advantage**: Purpose-built for financial visibility of LLM costs, not observability.

---

### 8.2 Helicone

**Provider support**: OpenAI-focused, expanding via proxy. Supports Anthropic, Azure, others.

**Token tracking**: Full token counts, cost tracking per API key, team, user ID.

**Gaps**:
- Proxy-based architecture (adds latency, users must route through Helicone servers)
- Less granular cache token visualization
- $20/mo for 100K requests (then $0.0002/req)

**token-viz advantage**: Local-first or self-hosted — no data leaves user's infrastructure.

---

### 8.3 Portkey

**Provider support**: 200+ LLMs via AI Gateway.

**Token tracking**: Full costs, cache hit rates, latency per provider.

**Gaps**:
- Enterprise positioning ($49/mo+)
- Gateway-first — requires routing all traffic through Portkey
- Complex setup for solo developers

**token-viz advantage**: Lightweight, developer-first, no routing required.

---

### 8.4 OpenMeter

**Focus**: Usage-based billing infrastructure — not LLM-specific.

**Assessment**: Infrastructure layer (Kafka, ClickHouse). Too heavy for token-viz's current stage. Monitor for Phase 3 enterprise billing features.

---

### 8.5 Market Gap Analysis

All existing tools are either:
1. **Gateway/proxy-based** (Helicone, Portkey): require routing traffic through their servers
2. **Observability-heavy** (Langfuse, LangSmith): trace/eval focused, cost is secondary
3. **Enterprise-first** (pricing starts $20-59/mo)

**token-viz's unique positioning**:
- SDK-based (no proxy required — zero latency overhead)
- Cost-first visualization (daily/monthly spend by model, provider, project)
- Multi-provider with provider-specific features (cache efficiency, reasoning tokens)
- Developer-friendly (open source, self-hostable, free tier viable)

---

## 9. Technical Challenges

### 9.1 Token Counting Inconsistencies

| Challenge | Detail | Mitigation |
|-----------|--------|------------|
| Field naming variance | `prompt_tokens` vs `input_tokens` vs `promptTokenCount` | Normalize in adapter layer |
| Billed vs actual (Cohere) | `billed_units` may differ from `tokens` | Track both, display billed |
| Reasoning tokens billing | OpenAI o1/o3: reasoning tokens billed as output | Mark as `output_type: reasoning` in UI |
| Streaming usage gaps | OpenAI: requires `include_usage` flag; some providers don't send mid-stream | Require final message, flag if unavailable |
| Gemini free tier quirks | Free tier tokenizer may differ from paid | Flag free tier model in DB |

### 9.2 Streaming Support Differences

| Provider | Streaming Usage Support | Notes |
|----------|------------------------|-------|
| OpenAI | Yes (flag required) | `stream_options.include_usage: true` |
| Anthropic | Yes (final event) | `message_delta` with `usage` |
| Gemini | Yes | Included in final chunk |
| Cohere | Yes | Final `message_end` event |
| Mistral | Yes | Final chunk |
| Groq | Yes | Final chunk |

**Mitigation**: Adapter layer handles stream accumulation and usage extraction uniformly.

### 9.3 Rate Limit Handling

Providers expose rate limit info in response headers:

```
OpenAI:    x-ratelimit-limit-requests, x-ratelimit-remaining-requests
Anthropic: anthropic-ratelimit-requests-limit, anthropic-ratelimit-tokens-remaining
Gemini:    Via quota error responses (no headers)
```

**Mitigation**: token-viz should capture rate limit headers and surface them in the dashboard. Implement exponential backoff in adapters.

### 9.4 Cost Calculation Accuracy

- Pricing changes without notice (Anthropic changed cache write TTL pricing in 2025)
- Per-model pricing varies (GPT-4o vs GPT-4o-mini)
- Batch API discounts not always tracked

**Mitigation**: Pricing JSON with `effective_date` field. Weekly GitHub Actions check against official docs. Alert founder when pricing mismatch detected.

---

## 10. Phased Rollout Plan

### Phase 1: MVP Multi-Provider (Week 1-3 of implementation)

**Goal**: OpenAI support added to existing Anthropic baseline.

**Providers**:
- OpenAI (GPT-4o, GPT-4o mini, o1, o3-mini)
- Anthropic (existing — refactor into adapter pattern)

**Engineering tasks**:
1. Refactor Anthropic integration into adapter interface (1 day)
2. Implement OpenAI adapter with `openai/openai-go` official SDK (2 days)
3. Unified `NormalizedUsage` DB schema migration (1 day)
4. Pricing JSON for OpenAI models (0.5 day)
5. UI: provider selector + per-provider stats (2 days)

**Effort**: ~6-7 agent-days
**Unlock criteria**: 10+ users requesting OpenAI support

---

### Phase 2: Google Gemini (Month 2)

**Goal**: Cover the 3 major providers (OpenAI + Anthropic + Google = ~85% of market).

**Providers**:
- Gemini 2.0 Flash, 1.5 Pro/Flash

**Engineering tasks**:
1. Gemini adapter with `google/generative-ai-go` SDK (1.5 days)
2. Cache efficiency metrics (Gemini context caching) (1 day)
3. Pricing JSON update (0.5 day)
4. UI: cache efficiency chart (Gemini + Claude side-by-side) (1 day)

**Effort**: ~4 agent-days
**Unlock criteria**: Phase 1 stable + 5+ Gemini requests

---

### Phase 3: OpenAI-Compatible Providers (Month 3)

**Goal**: Cover Groq (speed), Mistral (European market), OpenRouter (catch-all).

**Providers**:
- Groq (LLaMA 3.3 70B, LLaMA 3.1 8B)
- Mistral (Mistral Large 3, Mistral Small)
- OpenRouter (optional — covers 400+ models)

**Engineering tasks**:
1. OpenAI-compat adapter (single adapter for all three) (1 day)
2. Groq timing metrics visualization (0.5 day)
3. OpenRouter cost passthrough (0.5 day)
4. Pricing JSON for Groq + Mistral (0.5 day)

**Effort**: ~2.5 agent-days (reuse of OAI-compat adapter is high leverage)

---

### Phase 4: Long Tail (Month 4+)

**Goal**: Enterprise providers (Bedrock, Azure OpenAI, Cohere).

**Approach**:
- Option A: LiteLLM sidecar for enterprise users who want it
- Option B: Cohere official Go SDK (well-maintained)
- Option C: AWS Bedrock via `aws-sdk-go-v2` for enterprise customers

**Unlock criteria**: Enterprise customer demand, revenue justification

---

### Provider Priority Ranking

```
Priority 1 (MVP): OpenAI ████████████ 45% of LLM API market
Priority 2 (MVP): Anthropic █████████ 30% of LLM API market (existing)
Priority 3 (P2): Gemini █████ 15% of LLM API market
Priority 4 (P3): Groq ███ high developer mindshare, speed use cases
Priority 5 (P3): Mistral ██ European market, cost-sensitive users
Priority 6 (P3): OpenRouter █ catch-all, power users
Priority 7 (P4): Cohere, Bedrock — enterprise, defer
```

---

## 11. UI/UX Implications

### Dashboard Changes Required

**Provider selector**:
```
[ All Providers v ] [ Last 30 days v ] [ All Models v ]
```

**Multi-provider comparison chart**:
- Cost by provider (stacked bar, daily)
- Token volume by provider (line chart)
- Cost per 1K tokens by model (sortable table — efficiency ranking)

**Provider-specific panels** (conditional rendering):
- Claude / Gemini: Cache efficiency panel (cache hit rate, cost saved)
- OpenAI o1/o3 / Gemini thinking: Reasoning token breakdown
- Groq: Latency panel (tokens/second, queue time)

**Unified vs provider-specific views**:
```
Unified view: Total spend across all providers
Provider view: Deep-dive per provider (model breakdown, cache, rate limits)
```

**New metric: Cost Efficiency Score**
- Effective cost per output token (accounting for cache hits)
- Enables cross-provider comparison even when pricing structures differ

---

## 12. Final Recommendation

**Architecture**: Option C (Hybrid Direct SDK + OpenAI-compat)

**Go module dependencies to add**:
```go
// go.mod additions
require (
    github.com/openai/openai-go v1.x          // Phase 1
    github.com/google/generative-ai-go v1.x    // Phase 2
    // Cohere: github.com/cohere-ai/cohere-go/v2  (Phase 4 only)
    // Bedrock: github.com/aws/aws-sdk-go-v2       (Phase 4 only)
    // Mistral, Groq, OpenRouter: use openai-go with base URL override
)
```

**Data model**: Unified `NormalizedUsage` struct (defined above) covers all 8 providers with zero data loss.

**Pricing DB**: `data/pricing.json` flat file + weekly GitHub Actions check.

**Monday morning action**: Create GitHub Issue for Phase 1 (OpenAI adapter + adapter interface refactor). Assign to backend agent. Target: 1-week sprint.

---

## Sources

- [OpenAI API Reference — Responses](https://platform.openai.com/docs/api-reference/responses)
- [OpenAI API Reference — Usage](https://platform.openai.com/docs/api-reference/usage)
- [OpenAI official Go SDK — github.com/openai/openai-go](https://github.com/openai/openai-go)
- [Anthropic Go SDK — github.com/anthropics/anthropic-sdk-go](https://github.com/anthropics/anthropic-sdk-go)
- [Google Gemini API — Understand and count tokens](https://ai.google.dev/gemini-api/docs/tokens)
- [Google Generative AI Go SDK](https://github.com/google/generative-ai-go)
- [Cohere Go SDK — github.com/cohere-ai/cohere-go](https://github.com/cohere-ai/cohere-go)
- [Mistral AI — Pricing guide](https://www.binstellar.com/blog/what-are-tokens-in-mistral-ai-how-pricing-works-explained-in-simple-words/)
- [Groq API Reference](https://console.groq.com/docs/api-reference)
- [OpenRouter — Usage Accounting](https://openrouter.ai/docs/guides/guides/usage-accounting)
- [AWS Bedrock — Converse API](https://docs.aws.amazon.com/bedrock/latest/userguide/conversation-inference-call.html)
- [LiteLLM — Spend Tracking](https://docs.litellm.ai/docs/proxy/cost_tracking)
- [Gemini Developer API Pricing](https://ai.google.dev/gemini-api/docs/pricing)
- [OpenAI Pricing](https://openai.com/api/pricing/)
- [Groq Pricing](https://groq.com/pricing)
- [Cohere Pricing](https://cohere.com/pricing)
- [Helicone — LLM Observability Guide](https://www.helicone.ai/blog/the-complete-guide-to-LLM-observability-platforms)
