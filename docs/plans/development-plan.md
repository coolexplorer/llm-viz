# LLM-Viz - Development Plan

**Date**: 2026-02-27
**Status**: Initial Planning
**Target**: Production-ready MVP in 2 weeks

---

## 1. Product Goal

Build a **visual-first, real-time token consumption dashboard** for multi-provider LLM developers.

**Target Audience**: Individual developers and small teams using LLM APIs who want to:
- Monitor real-time token usage across providers
- Compare costs between OpenAI, Anthropic, Gemini, and others
- Optimize cache efficiency (Anthropic, Gemini, OpenAI)
- Calculate API costs accurately with provider-specific pricing

**Supported Providers**: OpenAI, Anthropic, Gemini, Mistral, Groq, OpenRouter

**Differentiation**: Unlike Langfuse/LangSmith (full observability platforms) or Helicone/Portkey (proxy-based gateways), llm-viz is SDK-based (zero latency overhead), cost-first, and purpose-built for multi-provider token visualization with zero setup.

---

## 2. Architecture Overview

```
User Browser
    |
    | HTTPS
    v
[Vercel — Next.js Frontend]
    | (Phase 1: API Routes only)
    | (Phase 2: SSE to Go backend)
    v
[Railway — Go Backend] (Phase 2+)
    |
    | HTTPS
    v
[Anthropic API]
```

**Phase 1**: Next.js frontend + API Routes (Vercel-only, $0)
**Phase 2**: Next.js frontend + Go backend for SSE (Vercel + Railway, $5/mo)

---

## 3. Tech Stack

### 3.1 Frontend
- **Framework**: Next.js 15 (App Router)
- **Language**: TypeScript
- **Styling**: Tailwind CSS
- **Charts**: Recharts (67.6% faster than Chart.js for real-time)
- **State**: React hooks (useState, useEffect)
- **Real-time**: SSE via EventSource API

### 3.2 Backend (Phase 2)
- **Language**: Go 1.22+
- **SDKs**:
  - `anthropic-sdk-go` v1.26.0 (Anthropic)
  - `openai-go` v1.x (OpenAI, Mistral, Groq, OpenRouter)
  - `generative-ai-go` v1.x (Gemini)
- **Server**: net/http (native)
- **Concurrency**: Goroutines for per-session tracking
- **DB**: In-memory (Phase 2), Turso SQLite (Phase 3)

### 3.3 Deployment
- **Frontend**: Vercel (Hobby tier, free)
- **Backend**: Railway (Starter, $5/mo)
- **CI/CD**: GitHub Actions (optional)

---

## 4. Core Features

### 4.1 Dashboard Components

1. **Token Counter** (Real-time)
   - Input tokens
   - Output tokens
   - Cache creation tokens
   - Cache read tokens
   - Total tokens per request

2. **Context Window Gauge**
   - Current usage (%)
   - Remaining capacity
   - Warning threshold (80%)
   - Critical threshold (95%)
   - Model-specific limits

3. **Cache Efficiency Chart**
   - Cache hit rate (%)
   - Cache miss rate (%)
   - Savings calculator (USD)

4. **Cost Tracker**
   - Per-request cost (USD)
   - Session total cost
   - Daily/weekly estimates
   - Breakdown by model

5. **Usage Timeline**
   - Line chart (Recharts)
   - Last 100 requests
   - Input/output trends
   - Cache hit patterns

### 4.2 Provider Selector

**Input**:
- Provider selection (OpenAI / Anthropic / Gemini / Mistral / Groq / OpenRouter)
- API key per selected provider (browser-only, not stored)
- Model selection (provider-specific)

**Output**:
- Real-time SSE stream of token data
- Visual charts and gauges
- Downloadable usage report (JSON/CSV)

---

## 5. Implementation Plan

### Phase 1: MVP + OpenAI (Week 1-3, Vercel-only, $0)

**Goal**: Functional dashboard with real-time token tracking for OpenAI and Anthropic.

**Providers**: OpenAI (GPT-4o, o1, o3-mini) + Anthropic (Claude 3.5 Sonnet/Haiku, Claude 3 Opus)

**Deliverables**:
- [ ] Next.js 15 project setup (App Router, TypeScript, Tailwind)
- [ ] Provider selector + API key input form (browser localStorage)
- [ ] Multi-provider API proxy (Next.js API Route)
- [ ] Token Counter component (with provider-specific field normalization)
- [ ] Context Window Gauge component (model-specific limits)
- [ ] Cache Efficiency Chart — Anthropic + OpenAI (Recharts)
- [ ] Cost Tracker with per-provider pricing
- [ ] Usage Timeline (Recharts LineChart) with provider breakdown
- [ ] SSE endpoint (Next.js Route Handler)
- [ ] Real-time data updates (EventSource)

**File Structure**:
```
llm-viz/
├── frontend/
│   ├── app/
│   │   ├── page.tsx                # Dashboard
│   │   ├── layout.tsx              # Root layout
│   │   ├── api/
│   │   │   ├── proxy/route.ts      # Multi-provider API proxy
│   │   │   └── stream/route.ts     # SSE endpoint
│   │   └── components/
│   │       ├── ProviderSelector.tsx
│   │       ├── TokenCounter.tsx
│   │       ├── ContextGauge.tsx
│   │       ├── CacheChart.tsx
│   │       ├── CostTracker.tsx
│   │       └── UsageTimeline.tsx
│   ├── lib/
│   │   ├── token-calculator.ts
│   │   ├── cost-calculator.ts
│   │   └── model-limits.ts
│   ├── hooks/
│   │   └── useTokenStream.ts
│   └── types/
│       └── token-data.ts
├── docs/
└── CLAUDE.md
```

**Quality Checks**:
```bash
npm run lint        # ESLint (zero errors)
npm run type-check  # TypeScript (zero errors)
npm run test        # Vitest (100% pass)
npm run build       # Production build success
```

**Success Criteria**:
- User can select provider and input API key
- Dashboard displays real-time token counts for OpenAI and Anthropic
- Context gauge updates per request with correct model limits
- Cost tracker shows USD estimates with per-provider pricing
- Timeline chart renders smoothly with provider breakdown

### Phase 2: Go Backend + Gemini (Week 4-6, Railway, $5/mo)

**Goal**: Persistent SSE connections, multi-user support, Gemini provider added.

**Providers added**: Google Gemini (2.0 Flash, 1.5 Pro/Flash)

**Deliverables**:
- [ ] Go HTTP server setup
- [ ] Provider adapters: Anthropic, OpenAI, Gemini (unified `NormalizedUsage` model)
- [ ] SSE endpoint with goroutines
- [ ] Session management (in-memory)
- [ ] Aggregated metrics endpoint (per-provider)
- [ ] Pricing JSON (`data/pricing.json`) for all Phase 1-2 providers
- [ ] Railway deployment config
- [ ] Frontend → Backend SSE connection

**File Structure**:
```
backend/
├── main.go                # Server entry point
├── handlers/
│   ├── stream.go          # SSE handler
│   ├── proxy.go           # Multi-provider API proxy
│   └── metrics.go         # Aggregated stats
├── adapters/
│   ├── anthropic.go       # Anthropic adapter
│   ├── openai.go          # OpenAI adapter
│   ├── gemini.go          # Gemini adapter
│   └── openai_compat.go   # Mistral/Groq/OpenRouter (Phase 3)
├── services/
│   ├── token_tracker.go   # Token tracking + normalization
│   └── session_store.go   # In-memory sessions
├── models/
│   └── token_data.go      # NormalizedUsage, TokenUsage, Provider types
├── data/
│   └── pricing.json       # Per-model pricing (all providers)
└── utils/
    ├── cost.go            # Cost calculator
    └── limits.go          # Model context limits
```

**Quality Checks**:
```bash
go test ./...       # All tests pass
go vet ./...        # No issues
```

**Success Criteria**:
- Go server runs on Railway with 3-provider support
- SSE streams persist beyond 10s
- Multiple users supported concurrently
- No memory leaks under load

### Phase 3: Long-Tail Providers (Month 2, Turso, $0)

**Goal**: Mistral, Groq, OpenRouter support; historical tracking; provider comparison charts.

**Providers added**: Mistral, Groq, OpenRouter (all via OpenAI-compatible adapter)

**Deliverables**:
- [ ] OpenAI-compatible adapter (covers Mistral, Groq, OpenRouter with base URL override)
- [ ] Groq latency metrics visualization (tokens/second, queue time)
- [ ] OpenRouter cost passthrough support
- [ ] Turso SQLite setup + token history schema
- [ ] Provider comparison charts (cost by provider, token volume, cost/1K tokens)
- [ ] Historical trend charts

---

## 6. Development Timeline

| Week | Phase | Providers | Deliverables | Cost |
|------|-------|-----------|--------------|------|
| 1-3 | MVP + OpenAI | OpenAI, Anthropic | Next.js dashboard, provider selector, SSE, Recharts | $0 |
| 4-6 | Go Backend + Gemini | + Gemini | Go server, adapter pattern, Railway deploy | $5/mo |
| 7-10 | Long-Tail | + Mistral, Groq, OpenRouter | OAI-compat adapter, Turso DB, comparison charts | $0 |

**Total Estimated Cost (3 months)**: ~$15

---

## 7. KPIs

**Technical**:
- Token tracking accuracy: 100% match with API responses
- SSE latency: <100ms from API response to UI update
- Chart render time: <50ms for 100 data points
- Cost calculation error: <0.1% vs actual billing

**Product**:
- User retention: 3+ sessions per user
- API key input rate: 80%+ of visitors
- Multi-provider usage: 30%+ of users connect 2+ providers
- Provider comparison feature usage: 40%+ of multi-provider sessions
- Export usage: 20%+ of sessions
- Feature usage: All 5 components used per session

---

## 8. Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Vercel 10s timeout | High | Move SSE to Go backend (Phase 2) |
| API key security | Critical | Browser-only, never stored server-side |
| Cost miscalculation | Medium | Unit tests for all pricing formulas |
| Chart performance | Medium | Recharts + data windowing (last 100) |

---

## 9. Future Enhancements (Post-MVP)

- Enterprise providers: AWS Bedrock, Azure OpenAI, Cohere (Phase 4)
- Team dashboards with shared provider keys
- Budget alerts per provider / per project
- Cost optimization suggestions (AI-powered)
- Browser extension
- Mobile app

### CLI/TUI Version: `tokwatch` (Separate Project)

**Status**: Research complete (2026-02-27). Feasibility: HIGH.

**Concept**: Real-time token monitoring in a second terminal window while using Claude Code.

```
Terminal 1: Claude Code (working)
Terminal 2: tokwatch (live token/cost dashboard)
```

**Key Finding**: Claude Code writes full token usage to `~/.claude/projects/**/*.jsonl` after every API call. No API key required. Latency <100ms via file watching.

**Recommended Stack**:
- Language: Go 1.22+
- TUI: bubbletea (charmbracelet, 30K stars)
- Charts: asciigraph + ntcharts
- Data Source: JSONL file watching (fsnotify)
- Distribution: Single binary (`go install`)

**MVP Features** (2 weeks):
- Real-time token counter (input/output/cache)
- Context window gauge (ASCII progress bar)
- Session cost tracker (tokens × pricing.json)
- Cache hit rate display
- Active session auto-detection

**Why Separate Project** (not integrated here):
- Different tech stack (Go TUI vs Next.js web)
- Different data source (local JSONL vs LLM API calls)
- Different distribution (binary vs web app)
- No user overlap with llm-viz web dashboard

**Competitive Position**: toktrack = historical analysis. ccusage = batch reporting. tokwatch = real-time live monitoring. No direct competitor in the real-time TUI space.

**Full Research**: See [CLI TUI Research](../research/cli-tui-research.md)

---

## 10. References

See [Tech Stack Research](../research/tech-stack-research.md) for detailed analysis.

**Key Sources**:
- [Claude API Token Counting](https://platform.claude.com/docs/en/build-with-claude/token-counting)
- [anthropic-sdk-go](https://github.com/anthropics/anthropic-sdk-go)
- [Recharts Performance](https://blog.logrocket.com/best-react-chart-libraries-2025/)
- [Next.js SSE](https://hackernoon.com/streaming-in-nextjs-15-websockets-vs-server-sent-events)
