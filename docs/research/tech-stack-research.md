# Token-Viz: Tech Stack Research Report

**Date**: 2026-02-27
**Prepared by**: COO Agent
**Project**: token-viz — Real-time AI model token consumption & context window visualization

---

## Executive Summary

### Key Findings

1. **Claude API** provides rich token tracking natively per-message response (`input_tokens`, `output_tokens`, `cache_creation_input_tokens`, `cache_read_input_tokens`) with no additional cost. An Admin API provides org-level aggregation with 1-minute granularity.

2. **Go SDK** (official: `anthropic-sdk-go` v1.26.0) is production-ready and supports full token tracking and streaming. **Python SDK** is the most mature reference. For this project, **Go backend is the recommended choice** given performance and concurrency needs for a real-time dashboard.

3. **Next.js (App Router) + Recharts** is the optimal frontend stack for 2026. SSE (Server-Sent Events) is the right transport — simpler than WebSocket, native to Next.js 15 App Router, ideal for server-to-client token streaming.

4. **Deployment**: Vercel (frontend) + Railway (Go backend) is the sweet spot. Combined cost at seed stage: ~$5-15/mo.

5. **Market gap**: Existing tools (Langfuse, LangSmith) are observability platforms for entire LLM stacks. There is no purpose-built, visual-first, real-time token consumption dashboard for individual developers.

### Recommended Stack

| Layer | Choice | Rationale |
|---|---|---|
| Frontend | Next.js 15 (App Router) + TypeScript | SSE support, Vercel deploy, ecosystem |
| Charts | Recharts | Best React performance for real-time updates |
| Real-time | SSE via Next.js Route Handlers | Simpler than WS, perfect for server→client token push |
| Backend | Go + `anthropic-sdk-go` v1.26.0 | Concurrency, performance, official SDK |
| DB (optional) | SQLite (dev) / Turso (prod) | Lightweight, edge-compatible |
| Hosting | Vercel (Next.js) + Railway (Go) | Best DX, $5-15/mo total |

---

## 1. Claude API Token Tracking

### 1.1 Per-Message Response Fields

Every Claude API `messages` response includes a `usage` object:

```json
{
  "id": "msg_01XFDUDYJgAACzvnptvVoYEL",
  "type": "message",
  "role": "assistant",
  "content": [...],
  "model": "claude-opus-4-6",
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 2048,
    "output_tokens": 503,
    "cache_creation_input_tokens": 248,
    "cache_read_input_tokens": 1800
  }
}
```

| Field | Description | Billing |
|---|---|---|
| `input_tokens` | Non-cached input tokens processed | Full price |
| `output_tokens` | Tokens generated in response | Full price |
| `cache_creation_input_tokens` | Tokens written to cache (new cache entry) | 25% surcharge |
| `cache_read_input_tokens` | Tokens read from cache hit | 90% discount |

**Key insight for token-viz**: The total context consumed = `input_tokens + cache_creation_input_tokens + cache_read_input_tokens`. This represents real context window usage.

### 1.2 Pre-flight Token Counting API

Count tokens **before** sending a message — free of charge:

```bash
POST https://api.anthropic.com/v1/messages/count_tokens

{
  "model": "claude-opus-4-6",
  "system": "You are a helpful assistant",
  "messages": [{"role": "user", "content": "Hello"}]
}

# Response:
{ "input_tokens": 14 }
```

- Free (no cost incurred)
- Rate-limited separately from Messages API
- Returns estimate (may differ by small amount from actual)
- Supports: text, tools, images, PDFs, extended thinking

Rate limits by tier:

| Tier | Requests/min |
|---|---|
| 1 | 100 |
| 2 | 2,000 |
| 3 | 4,000 |
| 4 | 8,000 |

### 1.3 Admin API: Organization-Level Usage

Requires Admin API key (`sk-ant-admin...`). Provides historical aggregated data:

**Usage Endpoint**: `GET /v1/organizations/usage_report/messages`

```bash
curl "https://api.anthropic.com/v1/organizations/usage_report/messages?\
starting_at=2026-02-20T00:00:00Z&\
ending_at=2026-02-27T00:00:00Z&\
group_by[]=model&\
bucket_width=1h" \
  --header "x-api-key: $ADMIN_API_KEY"
```

| Granularity | Default buckets | Max buckets | Use case |
|---|---|---|---|
| `1m` | 60 | 1440 | Real-time monitoring |
| `1h` | 24 | 168 | Daily patterns |
| `1d` | 7 | 31 | Weekly/monthly reports |

Filter/group dimensions: model, workspace, API key, service tier, context window, inference_geo, speed

**Data freshness**: Typically within 5 minutes of request completion.
**Recommended polling**: Once per minute for sustained use.

**Cost Endpoint**: `GET /v1/organizations/cost_report`
- USD costs (decimal strings in cents)
- Daily granularity only (`1d`)
- Covers: token usage, web search, code execution

### 1.4 Prompt Caching Impact on Token Counts

Caching changes how to interpret token counts:

```
Effective context window usage = input_tokens + cache_creation_input_tokens + cache_read_input_tokens

Cache hit rate = cache_read_input_tokens / (cache_read_input_tokens + cache_creation_input_tokens)

Actual cost:
  = (input_tokens * base_price)
  + (cache_creation_input_tokens * base_price * 1.25)
  + (cache_read_input_tokens * base_price * 0.10)
  + (output_tokens * output_price)
```

Token-viz should visualize cache hit/miss ratio as a key metric — this is where users save money.

---

## 2. Backend Tech Stack: Go vs Python

### 2.1 Go — Official SDK

**Repository**: `github.com/anthropics/anthropic-sdk-go` (v1.26.0)
**Requirements**: Go 1.22+

```bash
go get -u 'github.com/anthropics/anthropic-sdk-go@v1.26.0'
```

**Token tracking example**:

```go
package main

import (
    "context"
    "fmt"
    "github.com/anthropics/anthropic-sdk-go"
    "github.com/anthropics/anthropic-sdk-go/option"
)

func main() {
    client := anthropic.NewClient(
        option.WithAPIKey("your-api-key"),
    )

    message, err := client.Messages.New(context.TODO(), anthropic.MessageNewParams{
        Model:     anthropic.ModelClaudeOpus4_6,
        MaxTokens: 1024,
        Messages: []anthropic.MessageParam{
            anthropic.NewUserMessage(anthropic.NewTextBlock("Explain quantum computing")),
        },
    })
    if err != nil {
        panic(err)
    }

    // Token tracking
    fmt.Printf("Input tokens:          %d\n", message.Usage.InputTokens)
    fmt.Printf("Output tokens:         %d\n", message.Usage.OutputTokens)
    fmt.Printf("Cache creation tokens: %d\n", message.Usage.CacheCreationInputTokens)
    fmt.Printf("Cache read tokens:     %d\n", message.Usage.CacheReadInputTokens)

    // Context window usage
    totalContext := message.Usage.InputTokens +
        message.Usage.CacheCreationInputTokens +
        message.Usage.CacheReadInputTokens
    fmt.Printf("Total context used:    %d / 200000\n", totalContext)
}
```

**SSE streaming with token tracking**:

```go
stream := client.Messages.NewStreaming(context.TODO(), anthropic.MessageNewParams{
    Model:     anthropic.ModelClaudeOpus4_6,
    MaxTokens: 1024,
    Messages:  []anthropic.MessageParam{
        anthropic.NewUserMessage(anthropic.NewTextBlock("Hello")),
    },
})

accumulated := anthropic.Message{}
for stream.Next() {
    event := stream.Current()
    accumulated.Accumulate(event)
    // Access usage after stream completes via accumulated.Usage
}
// Final token counts available in accumulated.Usage
```

### 2.2 Python — Official SDK

**Repository**: `pip install anthropic`

```python
import anthropic

client = anthropic.Anthropic(api_key="your-api-key")

message = client.messages.create(
    model="claude-opus-4-6",
    max_tokens=1024,
    messages=[{"role": "user", "content": "Hello"}]
)

# Token tracking
print(f"Input tokens:          {message.usage.input_tokens}")
print(f"Output tokens:         {message.usage.output_tokens}")
print(f"Cache creation tokens: {message.usage.cache_creation_input_tokens}")
print(f"Cache read tokens:     {message.usage.cache_read_input_tokens}")
```

### 2.3 Comparison Table

| Criterion | Go | Python |
|---|---|---|
| Official SDK | Yes (v1.26.0) | Yes (mature) |
| Token tracking support | Full | Full |
| Streaming support | Yes | Yes |
| Concurrency model | Goroutines (native) | asyncio (requires async) |
| Performance (throughput) | Excellent | Good |
| SSE server implementation | Simple (net/http) | FastAPI / Flask |
| Cold start time | Fast (~10ms) | Slower (~200ms) |
| Memory footprint | Low | Higher |
| Prototyping speed | Moderate | Fast |
| Library ecosystem | Growing | Mature |
| Type safety | Strong (compile-time) | Optional (mypy) |
| Community tools | Fewer | More (Langfuse, LangSmith SDKs) |

**Recommendation**: **Go** for this project.
- Concurrency is critical for real-time multi-user token tracking
- SSE endpoints benefit from Go's goroutine model
- Lower memory footprint matters when tracking many concurrent sessions
- Official SDK is production-ready at v1.26.0

**Exception**: If the founder is more comfortable with Python and wants to ship in 1-2 days, Python + FastAPI is acceptable. Switch to Go in Phase 2 if concurrency becomes a bottleneck.

---

## 3. Frontend Tech Stack

### 3.1 Framework: Next.js 15 (App Router)

**Recommendation**: Next.js 15 with App Router.

Why not plain React:
- SSE implementation is built-in via Route Handlers
- Vercel deployment is zero-config
- API proxying protects Anthropic API key from client exposure
- File-based routing reduces boilerplate

### 3.2 Visualization Libraries Comparison

| Library | Performance (real-time) | React Integration | Learning Curve | Bundle Size | Recommendation |
|---|---|---|---|---|---|
| **Recharts** | Excellent | Native | Low | Medium (~80KB) | **Best for this project** |
| Chart.js | Good (SVG), poor at 10K+ pts | Via wrapper | Low | Small (~60KB) | Good for simple charts |
| D3.js | Excellent (Canvas option) | Manual | High | Large (~150KB) | Too complex for MVP |
| Visx | Excellent | Native (Airbnb) | High | Medium | Overkill at this stage |
| ECharts | Best for 1M+ pts | Via wrapper | Medium | Large | Server-side rendering issues |

**Winner: Recharts**
- 67.6% faster renders than Chart.js for datasets >10K points
- 3x faster updates for 100K point real-time updates
- React-native differential re-renders (only changed data points)
- Simple declarative API

```tsx
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

interface TokenDataPoint {
  timestamp: number;
  inputTokens: number;
  outputTokens: number;
  cacheHits: number;
  contextUsed: number;
}

export function TokenUsageChart({ data }: { data: TokenDataPoint[] }) {
  return (
    <ResponsiveContainer width="100%" height={300}>
      <LineChart data={data}>
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis dataKey="timestamp" tickFormatter={(ts) => new Date(ts).toLocaleTimeString()} />
        <YAxis />
        <Tooltip />
        <Line type="monotone" dataKey="inputTokens" stroke="#0D9488" dot={false} />
        <Line type="monotone" dataKey="outputTokens" stroke="#4F46E5" dot={false} />
        <Line type="monotone" dataKey="cacheHits" stroke="#F59E0B" dot={false} />
      </LineChart>
    </ResponsiveContainer>
  );
}
```

### 3.3 Real-time Transport: SSE vs WebSocket vs Polling

| Method | Direction | Complexity | Next.js 15 Support | Best For |
|---|---|---|---|---|
| **SSE** | Server → Client | Low | Native (Route Handlers) | **Token stream push** |
| WebSocket | Bidirectional | High | Needs additional setup | Chat, bidirectional |
| Long polling | Client pulls | Medium | Simple fetch loop | Low-frequency updates |
| Short polling | Client pulls | Low | Simple fetch loop | Status checks |

**Recommendation: SSE** for real-time token push.

Token consumption flows in one direction: API → Backend → Dashboard. SSE is purpose-built for this.

```typescript
// app/api/token-stream/route.ts
export async function GET() {
  const encoder = new TextEncoder();

  const stream = new ReadableStream({
    async start(controller) {
      // Poll or subscribe to token updates
      const interval = setInterval(() => {
        const data = JSON.stringify({
          timestamp: Date.now(),
          inputTokens: getCurrentInputTokens(),
          outputTokens: getCurrentOutputTokens(),
          contextWindowUsage: getContextWindowUsage(),
        });
        controller.enqueue(encoder.encode(`data: ${data}\n\n`));
      }, 1000);

      return () => clearInterval(interval);
    },
  });

  return new Response(stream, {
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      'Connection': 'keep-alive',
    },
  });
}
```

```typescript
// Client-side SSE hook
export function useTokenStream() {
  const [tokens, setTokens] = useState<TokenDataPoint[]>([]);

  useEffect(() => {
    const es = new EventSource('/api/token-stream');
    es.onmessage = (event) => {
      const data = JSON.parse(event.data);
      setTokens(prev => [...prev.slice(-100), data]); // Keep last 100 points
    };
    return () => es.close();
  }, []);

  return tokens;
}
```

---

## 4. Context Window Tracking

### 4.1 Current Model Limits (February 2026)

| Model | Context Window | Max Output | Input Price/MTok | Output Price/MTok |
|---|---|---|---|---|
| Claude Opus 4.6 | 200K (1M beta) | 128K | $5 | $25 |
| Claude Sonnet 4.6 | 200K (1M beta) | 64K | $3 | $15 |
| Claude Haiku 4.5 | 200K | 64K | $1 | $5 |
| Claude Sonnet 4.5 | 200K (1M beta) | 64K | $3 | $15 |
| Claude Opus 4.5 | 200K | 64K | $5 | $25 |

Note: 1M context window beta requires `context-1m-2025-08-07` header. Available for Opus 4.6, Sonnet 4.6, Sonnet 4.5, Sonnet 4. Usage tier 4+ or custom rate limits required.

### 4.2 Context Usage Calculation

```typescript
interface ContextWindowStatus {
  model: string;
  maxTokens: number;           // Model limit
  currentUsed: number;         // input + cache tokens
  inputTokens: number;
  cacheCreationTokens: number;
  cacheReadTokens: number;
  outputTokens: number;
  utilizationPercent: number;
  remainingTokens: number;
  warningThreshold: boolean;   // >80% usage
  criticalThreshold: boolean;  // >95% usage
}

function calculateContextStatus(
  usage: MessageUsage,
  model: string
): ContextWindowStatus {
  const maxTokens = MODEL_CONTEXT_LIMITS[model] ?? 200_000;
  const currentUsed = usage.inputTokens +
    usage.cacheCreationInputTokens +
    usage.cacheReadInputTokens;

  return {
    model,
    maxTokens,
    currentUsed,
    inputTokens: usage.inputTokens,
    cacheCreationTokens: usage.cacheCreationInputTokens,
    cacheReadTokens: usage.cacheReadInputTokens,
    outputTokens: usage.outputTokens,
    utilizationPercent: (currentUsed / maxTokens) * 100,
    remainingTokens: maxTokens - currentUsed,
    warningThreshold: currentUsed / maxTokens > 0.80,
    criticalThreshold: currentUsed / maxTokens > 0.95,
  };
}

const MODEL_CONTEXT_LIMITS: Record<string, number> = {
  'claude-opus-4-6':               200_000,
  'claude-sonnet-4-6':             200_000,
  'claude-haiku-4-5':              200_000,
  'claude-sonnet-4-5':             200_000,
  'claude-opus-4-5':               200_000,
  // 1M beta (requires special header)
  'claude-opus-4-6-1m':          1_000_000,
  'claude-sonnet-4-6-1m':        1_000_000,
};
```

### 4.3 Sliding Window vs Fixed Window

| Approach | Description | Use Case |
|---|---|---|
| Fixed window | Track single conversation's context from start | Single-session monitoring |
| Sliding window | Rolling N-turn window, older turns evicted | Multi-turn agent tracking |
| Cumulative | Total tokens across all calls (org-level) | Cost tracking, billing |

For token-viz MVP: Fixed window per session + cumulative across sessions.

---

## 5. AI Monitoring Landscape (2026 Trends)

### 5.1 Anthropic Console Native Features

Anthropic Console provides:
- Usage dashboard with daily/monthly token breakdown by model
- Cost reports with cache efficiency metrics
- API key-level attribution
- Integration with Grafana Cloud, Datadog, CloudZero, Honeycomb, Vantage

**Gap**: Console shows historical aggregates. No real-time per-session visualization. No context window gauge. No visual cost breakdown per call.

### 5.2 Major Observability Platforms

| Platform | Type | Token Tracking | Claude Support | Cost |
|---|---|---|---|---|
| **Langfuse** | Open source (Apache 2.0) | Full (input, output, cached) | Yes | Free self-hosted |
| **LangSmith** | SaaS (LangChain) | Full + P50/P99 latency | Yes | Free tier |
| **Weights & Biases** | SaaS | Via LLM integration | Yes | Free tier |
| **Datadog** | Enterprise SaaS | Auto-tracing | Yes | Expensive |
| **Grafana Cloud** | Hybrid | Agentless Anthropic integration | Yes | Free tier available |
| **Honeycomb** | SaaS | OpenTelemetry-based | Yes | Free tier |

**Langfuse** is the open-source leader (19K+ GitHub stars). Tracks:
- `input_tokens`, `output_tokens`
- `cached_tokens`, `audio_tokens`, `image_tokens`
- Predefined tokenizers for Anthropic models
- Cost calculation with model-specific pricing

**Key differentiator for token-viz**: These tools are for entire LLM stack observability. Token-viz targets individual developers who want a focused, real-time, visual dashboard for their Claude API usage — simpler, faster, purpose-built.

### 5.3 Reference Open Source Projects

| Project | Language | Description | GitHub Stars |
|---|---|---|---|
| **toktrack** | Rust | Ultra-fast token tracker for Claude Code, Codex CLI (SIMD-JSON + parallel) | Active |
| **ccusage** | TypeScript | CLI for analyzing Claude Code JSONL usage files | Active |
| **tokencost** (AgentOps) | Python | USD cost calculation for 400+ LLMs | 1K+ stars |
| **llm-token-tracker** | TypeScript | MCP server for token/cost tracking | Active |
| **tokenx** | Python | Decorator-based cost tracking (Anthropic + OpenAI) | Active |
| **token-tally** | JavaScript | Web-based cost calculator for major LLM APIs | Active |
| **Langfuse** | TypeScript | Full LLM observability platform | 19K+ stars |

These are the competitive reference projects. token-viz's differentiation: **visual-first, real-time, context window gauge, session-scoped**.

---

## 6. Deployment Options

### 6.1 Recommended Architecture

```
User Browser
    |
    | HTTPS
    v
[Vercel — Next.js Frontend]
    |
    | HTTP/SSE
    v
[Railway — Go Backend]
    |
    | HTTPS
    v
[Anthropic API]
```

### 6.2 Platform Comparison

| Platform | Best For | Next.js | Go Backend | Free Tier | Est. Cost (seed) |
|---|---|---|---|---|---|
| **Vercel** | Next.js frontend | Native | No (serverless functions only) | Yes (Hobby) | $0-20/mo |
| **Railway** | Go backend / full-stack | Yes | Yes (Docker) | $5/mo credit | $5-15/mo |
| **Fly.io** | Globally distributed Go | Yes | Yes (Docker) | No | $2-10/mo |
| **Render** | Simple full-stack | Yes | Yes | Yes (slow cold start) | $0-7/mo |
| **Vercel + Railway** | Split frontend/backend | Native | Native | Partial | $5-15/mo total |

### 6.3 Recommended Deployment Plan

**Phase 1 (MVP, Day 1-30)**: Vercel only
- Next.js frontend + API Routes on Vercel
- No separate backend (Go backend proxied through Next.js API routes)
- Cost: $0 (Hobby tier)
- Limitation: No persistent WebSocket, 10s function timeout

**Phase 2 (Growth, Day 30+)**: Vercel + Railway
- Next.js on Vercel
- Go backend on Railway ($5/mo starter)
- SSE via Go backend for persistent connections
- Cost: $5-20/mo

**Phase 3 (Scale)**: Add Turso (SQLite edge database) for session persistence
- Turso free tier: 500 DBs, 1B row reads/mo
- Stores token history per session
- Cost: $0 additional

---

## 7. Implementation Path

### Phase 1: MVP (Week 1-2, Vercel-only)

**Core features**:
- [ ] Real-time token counter (input/output per call)
- [ ] Context window gauge (% used)
- [ ] Cache hit/miss visualization
- [ ] Cost calculator (real-time USD estimate)
- [ ] Last 100 requests timeline (Recharts LineChart)

**Data flow**:
```
User enters API key in browser
  → Next.js API Route proxies to Anthropic
  → Response usage fields captured
  → SSE pushes to dashboard
  → Recharts updates in real-time
```

**File structure**:
```
token-viz/
├── app/
│   ├── page.tsx                    # Main dashboard
│   ├── api/
│   │   ├── proxy/route.ts          # Anthropic API proxy
│   │   └── token-stream/route.ts   # SSE endpoint
│   └── components/
│       ├── TokenCounter.tsx
│       ├── ContextWindowGauge.tsx
│       ├── CostTracker.tsx
│       ├── CacheEfficiency.tsx
│       └── UsageTimeline.tsx
├── lib/
│   ├── token-calculator.ts
│   ├── cost-calculator.ts
│   └── model-limits.ts
└── hooks/
    └── useTokenStream.ts
```

### Phase 2: Go Backend (Week 3-4, Railway)

- Go HTTP server with goroutine-per-session token tracking
- SSE endpoint for persistent connections
- In-memory session store (Redis optional)
- Aggregated metrics endpoint

### Phase 3: Persistence + Admin API (Month 2)

- Turso SQLite for token history
- Anthropic Admin API polling for org-level data
- Historical trend charts (daily/weekly)
- Cost anomaly detection

---

## 8. Cost Estimation for token-viz Development

### API Costs (Running token-viz itself)

token-viz proxies user's own Anthropic API key — no API cost to the platform.

If token-viz uses Claude internally (e.g., for cost optimization suggestions):

| Volume | Model | Estimated Monthly API Cost |
|---|---|---|
| 1K requests/mo | Haiku 4.5 | ~$0.01 |
| 10K requests/mo | Haiku 4.5 | ~$0.10 |
| 100K requests/mo | Sonnet 4.6 | ~$3.00 |

### Infrastructure Costs

| Stage | Setup | Monthly Cost |
|---|---|---|
| MVP | Vercel Hobby | $0 |
| Growth | Vercel Hobby + Railway Starter | $5 |
| Scale | Vercel Pro + Railway + Turso | $25-40 |

---

## Sources & References

**Anthropic Official Docs**:
- [Token Counting API](https://platform.claude.com/docs/en/build-with-claude/token-counting)
- [Usage and Cost Admin API](https://platform.claude.com/docs/en/build-with-claude/usage-cost-api)
- [Models Overview](https://platform.claude.com/docs/en/about-claude/models/overview)
- [Prompt Caching](https://platform.claude.com/docs/en/build-with-claude/prompt-caching)
- [Context Windows](https://platform.claude.com/docs/en/build-with-claude/context-windows)

**SDKs**:
- [anthropic-sdk-go (Official)](https://github.com/anthropics/anthropic-sdk-go)
- [Go SDK Docs](https://platform.claude.com/docs/en/api/sdks/go)

**Visualization**:
- [Best React Chart Libraries 2025 — LogRocket](https://blog.logrocket.com/best-react-chart-libraries-2025/)
- [Recharts vs D3.js Comparison](https://solutions.lykdat.com/blog/recharts-vs-d3-js/)
- [SSE in Next.js 15 — HackerNoon](https://hackernoon.com/streaming-in-nextjs-15-websockets-vs-server-sent-events)

**Observability Tools**:
- [Langfuse Token Tracking](https://langfuse.com/docs/observability/features/token-and-cost-tracking)
- [LLM Observability Tools 2026 — Firecrawl](https://www.firecrawl.dev/blog/best-llm-observability-tools)
- [Top AI Observability Platforms 2026 — Maxim](https://www.getmaxim.ai/articles/top-5-ai-observability-platforms-for-production-ai-systems-in-2026/)

**Reference Projects**:
- [toktrack — Rust token tracker](https://github.com/mag123c/toktrack)
- [ccusage — Claude Code usage CLI](https://github.com/ryoppippi/ccusage)
- [tokencost — 400+ LLM cost calculator](https://github.com/AgentOps-AI/tokencost)
- [llm-token-tracker — MCP token tracker](https://github.com/wn01011/llm-token-tracker)
- [Langfuse — Open source LLM observability](https://github.com/langfuse/langfuse)

**Deployment**:
- [Deployment Platforms Comparison 2025](https://www.jasonsy.dev/blog/comparing-deployment-platforms-2025)
- [Fly.io vs Vercel](https://ritza.co/articles/gen-articles/cloud-hosting-providers/fly-io-vs-vercel/)
- [Railway vs Vercel](https://ritza.co/articles/gen-articles/cloud-hosting-providers/railway-vs-vercel/)

**Monitoring Integrations**:
- [Grafana Cloud Anthropic Integration](https://grafana.com/blog/how-to-monitor-claude-usage-and-costs-introducing-the-anthropic-integration-for-grafana-cloud/)
- [Datadog Anthropic Integration](https://www.datadoghq.com/blog/anthropic-usage-and-costs/)
