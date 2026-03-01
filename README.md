<div align="center">

# 🔍 LLM-Viz

**Real-time multi-provider LLM token consumption and cost visualization dashboard**

[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](#-license)
[![Go Version](https://img.shields.io/badge/go-1.23+-00ADD8.svg)](https://go.dev/)
[![Next.js](https://img.shields.io/badge/Next.js-16-black.svg)](https://nextjs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.0+-3178C6.svg)](https://www.typescriptlang.org/)

[Features](#-features) • [Quick Start](#-quick-start) • [Documentation](#-documentation) • [API](#-api) • [Contributing](#-contributing)

---

**Monitor your LLM API usage across OpenAI and Anthropic (Phase 1) — with Gemini, Mistral, Groq, and OpenRouter coming soon**

Track token consumption, compare costs, optimize cache efficiency, and visualize spending in real-time.

</div>

---

## ✨ Features

### 📊 Real-time Monitoring
- **Live Token Counter** — Input/output/cache tokens per request, broken down by provider
- **Context Window Gauge** — Visual capacity tracking with model-specific limits (128K–2M tokens)
- **Cost Tracker** — Accurate USD estimates with provider-specific pricing and cache savings

### 🔀 Multi-Provider Support
- **2 Providers (Phase 1)** — OpenAI, Anthropic _(+4 more in Phases 2-3: Gemini, Mistral, Groq, OpenRouter)_
- **Provider Comparison** — Cost per 1K tokens across configured providers
- **Unified Interface** — Switch providers without changing your code

### ⚡ Performance Insights
- **Cache Efficiency** — Hit/miss rates for Anthropic, Gemini, and OpenAI prompt caching
- **Usage Timeline** — Historical trends with provider breakdown (last 100 requests)
- **Groq Latency Metrics** — Tokens/second and queue time visualization _(Phase 3)_

### 🎨 Developer Experience
- **Real-time Updates** — Server-Sent Events (SSE) for instant feedback
- **Clean Architecture** — Hexagonal architecture (Ports & Adapters) in Go backend
- **Type-Safe** — Full TypeScript support with strict mode
- **Beautiful UI** — Modern dashboard built with Next.js 16 + Tailwind CSS + Recharts

---

## 🚀 Quick Start

### Prerequisites

- **Node.js** 18+ (for frontend)
- **Go** 1.22+ (for backend)
- **API Keys** from one or more providers:
  - [OpenAI API Key](https://platform.openai.com/api-keys)
  - [Anthropic API Key](https://console.anthropic.com/)
  - [Google AI Studio](https://makersuite.google.com/app/apikey) (Gemini)

### Frontend Only (Development)

Perfect for testing the UI with mock data:

```bash
# Clone the repository
git clone https://github.com/coolexplorer/llm-viz.git
cd llm-viz/frontend

# Install dependencies
npm install

# Start development server
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

### Full Stack (Production-Ready)

Run both frontend and backend for real API integration:

**Terminal 1 - Backend:**
```bash
cd backend

# Set up environment variables
cp .env.example .env
# Edit .env and add your API keys

# Install dependencies
go mod download

# Run the server
go run ./cmd/server

# Server starts on http://localhost:8080
```

**Terminal 2 - Frontend:**
```bash
cd frontend

# Install dependencies
npm install

# Start Next.js
npm run dev

# Open http://localhost:3000
```

---

## 🎯 Usage

### Basic Usage

1. **Select Provider** — Choose OpenAI, Anthropic, or Gemini from the dropdown
2. **Choose Model** — Pick a specific model (e.g., `gpt-4o`, `claude-sonnet-4-6`)
3. **Enter API Key** — Your key is stored only in browser memory (never persisted)
4. **Send Messages** — Type a message and watch tokens/cost update in real-time

### API Integration Example

```typescript
// Send a completion request
const response = await fetch('http://localhost:8080/api/complete', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    provider: 'anthropic',
    model: 'claude-sonnet-4-6',
    messages: [
      { role: 'user', content: 'Explain quantum computing in 3 sentences' }
    ],
    max_tokens: 1024,
    session_id: 'my-session-123'
  })
});

const data = await response.json();
console.log(data);
// {
//   id: "msg_123",
//   content: "Hello! How can I help you?",
//   provider: "anthropic",
//   model: "claude-sonnet-4-6",
//   usage: {
//     id: "uuid",
//     timestamp: "2026-03-01T00:00:00Z",
//     provider: "anthropic",
//     model: "claude-sonnet-4-6",
//     session_id: "my-session-123",
//     usage: {
//       input_tokens: 12,
//       output_tokens: 58,
//       total_tokens: 70,
//       cache_read_tokens: 0
//     },
//     cost_usd: 0.000234
//   }
// }
```

### Real-time Streaming

```typescript
const eventSource = new EventSource(
  'http://localhost:8080/api/sse?session_id=my-session-123'
);

eventSource.addEventListener('usage', (event) => {
  const usage = JSON.parse(event.data);
  console.log('New token usage:', usage);
});
```

---

## 📖 Documentation

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Frontend (Next.js 15)                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ Components   │  │ Hooks        │  │ API Routes   │  │
│  │ (React)      │  │ (useToken    │  │ (/api/proxy) │  │
│  │              │  │  Stream)     │  │              │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
                           │
                           │ SSE / HTTP
                           ▼
┌─────────────────────────────────────────────────────────┐
│                    Backend (Go 1.22+)                    │
│  ┌──────────────────────────────────────────────────┐   │
│  │              Transport Layer (HTTP)              │   │
│  │  • Handlers  • Middleware  • SSE Broadcaster     │   │
│  └──────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────┐   │
│  │               Service Layer                      │   │
│  │  • TokenTracker  • Session Management            │   │
│  └──────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────┐   │
│  │               Adapter Layer                      │   │
│  │  • OpenAI  • Anthropic  • Gemini  • Storage      │   │
│  └──────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────┐   │
│  │               Domain Layer (Core)                │   │
│  │  • Entities  • Pricing  • Errors                 │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
              ┌────────────────────────┐
              │  Provider APIs         │
              │  • OpenAI  • Anthropic │
              │  • Gemini  • Mistral   │
              └────────────────────────┘
```

**Hexagonal Architecture** (Ports & Adapters):
- **Domain** — Core business logic (zero external dependencies)
- **Ports** — Interfaces (LLMProvider, UsageRepository, EventBroadcaster)
- **Service** — Use cases (TokenTracker)
- **Adapters** — Provider SDKs, storage, pricing
- **Transport** — HTTP handlers, SSE, middleware

### Project Structure

```
llm-viz/
├── frontend/                 # Next.js 16 application
│   ├── app/
│   │   ├── components/       # React components
│   │   ├── api/             # API routes (proxy, stream)
│   │   └── page.tsx         # Main dashboard
│   ├── hooks/               # Custom React hooks
│   ├── lib/                 # Utilities (cost calc, limits)
│   └── types/               # TypeScript definitions
│
├── backend/                 # Go 1.23+ server
│   ├── cmd/server/          # Entry point (main.go)
│   ├── internal/
│   │   ├── domain/          # Core entities
│   │   ├── port/            # Interfaces
│   │   ├── service/         # Business logic
│   │   └── adapter/         # Provider implementations
│   ├── transport/           # HTTP/SSE layer
│   └── data/                # pricing.json
│
└── docs/                    # Documentation
    ├── architecture/        # Design docs
    ├── research/            # Tech stack research
    └── plans/               # Development roadmap
```

### Tech Stack

| Category | Technologies |
|----------|-------------|
| **Frontend** | Next.js 16 (App Router), TypeScript 5.0+, Tailwind CSS v4, Recharts |
| **Backend** | Go 1.23+, `anthropic-sdk-go` v1.26.0, `openai-go` v1.12.0 |
| **Real-time** | Server-Sent Events (SSE) |
| **Testing** | Vitest, Playwright (E2E), testify (Go), React Testing Library |
| **Deployment** | Vercel (frontend), Railway (backend) |
| **Database** | In-memory (Phase 1), Turso SQLite (Phase 3) |

---

## 🔌 API

### Endpoints

#### `POST /api/complete`
Proxy a completion request to the selected provider.

**Request:**
```json
{
  "provider": "anthropic",
  "model": "claude-sonnet-4-6",
  "messages": [
    { "role": "user", "content": "Hello!" }
  ],
  "max_tokens": 1024,
  "session_id": "my-session-123"
}
```

**Response:**
```json
{
  "id": "msg_123",
  "content": "Hello! How can I help you?",
  "usage": {
    "input_tokens": 12,
    "output_tokens": 8,
    "cache_read_tokens": 0,
    "total_tokens": 20,
    "cost_usd": 0.000156
  }
}
```

#### `GET /api/sse?session_id=<id>`
Subscribe to real-time token usage events via SSE.

**Events:**
- `ping` — Connection confirmation
- `usage` — Token usage update (JSON)
- `: heartbeat` — Keepalive (every 30s)

#### `GET /api/stats?session_id=<id>&limit=100`
Get session history (most recent first).

#### `GET /api/models`
List available models and configured providers.

#### `GET /api/health`
Health check endpoint.

See [Backend API Documentation](backend/README.md) for full details.

---

## 🧪 Development

### Quality Checks

**Frontend:**
```bash
npm run lint        # ESLint (zero errors)
npm run type-check  # TypeScript strict mode
npm run test        # Vitest + React Testing Library
npm run test:e2e    # Playwright E2E tests
npm run build       # Production build
```

**Backend:**
```bash
go test ./...              # All tests
go test -cover ./...       # With coverage
go vet ./...               # Static analysis
go build ./cmd/server      # Build binary
```

### Environment Variables

**Backend** (`.env`):
```bash
# Required (at least one)
ANTHROPIC_API_KEY=sk-ant-xxx
OPENAI_API_KEY=sk-xxx
GEMINI_API_KEY=xxx

# Optional
PORT=8080
ALLOWED_ORIGIN=http://localhost:3000
PRICING_FILE=data/pricing.json
LOG_LEVEL=info
```

**Frontend** (`.env.local`):
```bash
# Optional - defaults to http://localhost:8080
NEXT_PUBLIC_API_URL=http://localhost:8080
```

---

## 🗺️ Roadmap

- [x] **Phase 1** — MVP with OpenAI + Anthropic _(Completed)_
  - Real-time token counter
  - Context window gauge
  - Cost tracker with cache savings
  - Provider comparison UI
  - SSE integration

- [ ] **Phase 2** — Gemini + Backend Deployment _(In Progress)_
  - Google Gemini support
  - Railway backend deployment
  - Session persistence
  - Multi-user support

- [ ] **Phase 3** — Extended Providers + Analytics _(Planned)_
  - Mistral, Groq, OpenRouter support
  - Turso database integration
  - Historical analytics dashboard
  - Provider cost comparison charts
  - Export to CSV/JSON

See [Development Plan](docs/plans/development-plan.md) for detailed timeline.

---

## 🤝 Contributing

This is an **AI-first project** managed by AI agents using Claude Code. We welcome contributions!

### How to Contribute

1. **Fork the repository**
2. **Create a feature branch** (`git checkout -b feat/amazing-feature`)
3. **Make your changes**
4. **Run quality checks** (lint, type-check, tests)
5. **Commit** using [Conventional Commits](https://www.conventionalcommits.org/)
   ```bash
   git commit -m "feat: add Mistral provider support"
   ```
6. **Push to your fork** (`git push origin feat/amazing-feature`)
7. **Open a Pull Request**

### Development Guidelines

- See [CLAUDE.md](CLAUDE.md) for AI agent orchestration guidelines
- Follow existing code structure (Hexagonal Architecture for backend)
- Maintain 80%+ test coverage
- Use TypeScript strict mode
- All PRs must pass CI checks (lint, type-check, tests, build)

---

## 📋 Requirements

- **Node.js** 18.0.0 or higher
- **Go** 1.23.0 or higher
- **npm** or **yarn** or **pnpm**
- **Git** (for version control)
- **API Keys** from at least one supported provider

---

## 📄 License

This project is licensed under the **MIT License**.

---

## 🙏 Acknowledgments

- **Provider SDKs**: [anthropic-sdk-go](https://github.com/anthropics/anthropic-sdk-go), [openai-go](https://github.com/openai/openai-go), [generative-ai-go](https://github.com/google/generative-ai-go)
- **Charting**: [Recharts](https://recharts.org/)
- **UI Inspiration**: [shadcn/ui](https://ui.shadcn.com/), [Next.js Examples](https://github.com/vercel/next.js/tree/canary/examples)
- **Architecture**: Inspired by [Hexagonal Architecture](https://netflixtechblog.com/ready-for-changes-with-hexagonal-architecture-b315ec967749) and Clean Architecture principles

---

## 💡 FAQ

### Q: Is my API key secure?
**A:** Yes. API keys are stored only in browser memory for the session duration. They are never persisted to `localStorage`, `sessionStorage`, or logged server-side.

### Q: Which providers are supported?
**A:** Currently OpenAI and Anthropic (Phase 1). Gemini coming in Phase 2, with Mistral/Groq/OpenRouter in Phase 3.

### Q: Can I self-host this?
**A:** Absolutely! Deploy the frontend to Vercel/Netlify and the backend to Railway/Fly.io. See deployment docs _(coming soon)_.

### Q: How accurate is the cost calculation?
**A:** Very accurate. We use official provider pricing (updated 2026-02) and account for cache discounts (50–90% savings for cached tokens).

### Q: Does this work with streaming responses?
**A:** Phase 1 uses non-streaming. Streaming support (real-time token counting) is planned for Phase 2.

---

<div align="center">

**Built with ❤️ using [Claude Code](https://claude.com/claude-code)**

[⬆ Back to Top](#-llm-viz)

</div>
