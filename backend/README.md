# llm-viz Backend

Go backend for the llm-viz token tracking dashboard. Implements Hexagonal Architecture (Ports & Adapters) with OpenAI and Anthropic provider support.

## Quick Start

```bash
# 1. Copy and configure environment variables
cp .env.example .env
# Edit .env and add your API keys

# 2. Run from the backend directory (pricing.json path is relative)
cd backend
go run ./cmd/server

# Server starts on http://localhost:8080
```

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `ANTHROPIC_API_KEY` | At least one | — | Anthropic API key |
| `OPENAI_API_KEY` | At least one | — | OpenAI API key |
| `PORT` | No | `8080` | HTTP server port |
| `ALLOWED_ORIGIN` | No | `http://localhost:3000` | CORS allowed origin |
| `PRICING_FILE` | No | `data/pricing.json` | Path to pricing JSON |
| `LOG_LEVEL` | No | `info` | `info` or `debug` |

## API Endpoints

### `POST /api/complete`
Proxy a completion request to the selected provider and track token usage.

**Request body:**
```json
{
  "provider": "anthropic",
  "model": "claude-sonnet-4-6",
  "messages": [
    { "role": "user", "content": "Hello!" }
  ],
  "max_tokens": 1024,
  "session_id": "my-session-123",
  "project_tag": "my-project"
}
```

**Response:**
```json
{
  "id": "uuid",
  "content": "Hello! How can I help you?",
  "provider": "anthropic",
  "model": "claude-sonnet-4-6",
  "usage": {
    "id": "uuid",
    "timestamp": "2026-02-27T12:00:00Z",
    "provider": "anthropic",
    "model": "claude-sonnet-4-6",
    "session_id": "my-session-123",
    "usage": {
      "input_tokens": 12,
      "output_tokens": 8,
      "total_tokens": 20
    },
    "cost_usd": 0.000156
  }
}
```

### `GET /api/sse?session_id=<id>`
Subscribe to real-time token usage events for a session via Server-Sent Events.

Events emitted:
- `ping` — connection confirmation on connect
- `usage` — token usage event after each completion
- `: heartbeat` — keepalive comment every 30 seconds

### `GET /api/stats?session_id=<id>&limit=100`
Get session history (most recent first).

### `GET /api/stats?provider=anthropic&start=<RFC3339>&end=<RFC3339>`
Get aggregated stats by provider.

### `GET /api/models`
List available models and configured providers.

### `GET /api/health`
Health check. Returns `{"status":"ok","providers":[...]}`.

## Architecture

```
internal/
  domain/        # Core entities — zero external imports
  port/          # Interfaces (LLMProvider, UsageRepository, EventBroadcaster)
  service/       # TokenTracker use case
  adapter/
    provider/    # OpenAI and Anthropic adapters
    storage/     # In-memory repository (Phase 1-2)
    pricing/     # pricing.json reader
transport/
  http/          # HTTP handlers, CORS, logging middleware
  sse/           # SSE broadcaster (goroutine-safe)
cmd/server/      # Entry point — manual DI wiring
data/
  pricing.json   # Per-model pricing database
```

**Dependency rule**: Domain → no external imports. Port → domain only. Service → domain + port. Adapter → domain + port + SDK. Transport → service + port.

## Running Tests

```bash
# Unit tests (no API keys needed)
go test ./...

# Integration tests (real API calls — requires keys in environment)
go test -tags integration ./...

# Vet
go vet ./...
```

## Supported Providers (Phase 1)

| Provider | Models |
|---|---|
| Anthropic | claude-opus-4-6, claude-sonnet-4-6, claude-haiku-4-5 |
| OpenAI | gpt-4o, gpt-4o-mini, o1, o3-mini |

Phase 2 adds Gemini. Phase 3 adds Mistral, Groq, OpenRouter.
