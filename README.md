# LLM-Viz

**Real-time multi-provider LLM token consumption and cost visualization dashboard.**

Monitor your LLM API usage across OpenAI, Anthropic, Gemini, Mistral, Groq, and OpenRouter—compare costs, optimize cache efficiency, and track token spend in real-time.

**Providers**: `OpenAI` · `Anthropic` · `Gemini` · `Mistral` · `Groq` · `OpenRouter`

---

## Features

- 📊 **Real-time Token Counter** — Input/output/cache tokens per request, per provider
- 🔀 **Provider Comparison** — Cost per 1K tokens across OpenAI, Anthropic, Gemini, and more
- 🎯 **Context Window Gauge** — Visual capacity tracking with model-specific limits
- 💰 **Cost Tracker** — Accurate USD estimates with provider-specific pricing and cache savings
- 📈 **Usage Timeline** — Historical trends with provider breakdown (Recharts)
- ⚡ **Cache Efficiency** — Hit/miss rates for Anthropic, Gemini, and OpenAI
- ⚡ **Groq Latency Metrics** — Tokens/second and queue time visualization (Phase 3)

---

## Tech Stack

- **Frontend**: Next.js 15 + TypeScript + Tailwind CSS + Recharts
- **Backend**: Go 1.22+ with multi-provider SDKs (`openai-go`, `anthropic-sdk-go`, `generative-ai-go`)
- **Real-time**: Server-Sent Events (SSE)
- **Deploy**: Vercel (frontend) + Railway (backend)

---

## Quick Start

### Phase 1: MVP (Vercel-only)

```bash
# Frontend setup
cd frontend
npm install
npm run dev

# Open http://localhost:3000
```

### Phase 2: With Go Backend

```bash
# Backend setup
cd backend
go mod download
go run main.go

# Frontend (in another terminal)
cd frontend
npm run dev
```

---

## Development

See [Development Plan](docs/plans/development-plan.md) for roadmap.

### Quality Checks

**Frontend**:
```bash
npm run lint        # ESLint
npm run type-check  # TypeScript
npm run test        # Vitest
npm run build       # Production build
```

**Backend**:
```bash
go test ./...       # Tests
go vet ./...        # Static analysis
```

---

## Documentation

- [Tech Stack Research](docs/research/tech-stack-research.md)
- [Development Plan](docs/plans/development-plan.md)
- [CLAUDE.md](CLAUDE.md) - Orchestrator guide

---

## Roadmap

- [x] **Phase 1**: MVP — OpenAI + Anthropic with provider selector (Week 1-3)
- [ ] **Phase 2**: Go backend + Gemini support (Week 4-6)
- [ ] **Phase 3**: Mistral, Groq, OpenRouter + Turso DB + comparison charts (Month 2)

---

## Contributing

This is an AI-first project managed by AI agents. See [CLAUDE.md](CLAUDE.md) for orchestration guidelines.

---

## License

MIT

---

**Built with Claude Code**
