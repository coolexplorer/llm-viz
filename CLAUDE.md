# LLM-Viz - Claude Code Instructions

## Project Overview

Real-time multi-provider LLM token consumption and context window visualization dashboard. Helps developers monitor API usage across OpenAI, Anthropic, Gemini, Mistral, Groq, and OpenRouter—tracking costs, comparing providers, and optimizing token efficiency.

**Supported Providers**: OpenAI, Anthropic, Gemini, Mistral, Groq, OpenRouter

**Key Features**:
- Real-time token counter (input/output/cache) per provider
- Context window usage gauge with model-specific limits
- Cache hit/miss visualization (Anthropic, Gemini, OpenAI)
- Cost tracker with USD estimates and cross-provider comparison
- Usage timeline charts with provider breakdown

## Tech Stack

- **Frontend**: Next.js 15 (App Router) + TypeScript + Tailwind CSS + Recharts
- **Backend**: Go 1.22+ with provider SDKs:
  - `anthropic-sdk-go` v1.26.0 (Anthropic)
  - `openai-go` v1.x (OpenAI, Mistral, Groq, OpenRouter via base URL override)
  - `generative-ai-go` v1.x (Gemini)
- **Real-time**: SSE (Server-Sent Events) via Next.js Route Handlers
- **Deployment**: Vercel (frontend) + Railway (backend)
- **Database**: Turso SQLite (Phase 3, optional)

### Key Paths
- `frontend/` - Next.js app (dashboard, API routes)
- `backend/` - Go server (token tracking, SSE)
- `docs/` - Documentation
- `shared/` - Shared types/schemas

## Documentation Standards

All docs MUST go in `docs/` folder:
```
docs/plans/        docs/architecture/     docs/decisions/
docs/guides/       docs/research/
```
- kebab-case filenames, relative cross-references
- Never save docs to `~/.claude/plans/` or project root

## Quality Standards

### Frontend Quality Checks

```bash
cd frontend && npm run lint         # Zero errors
cd frontend && npm run type-check   # Zero TypeScript errors
cd frontend && npm run test         # 100% pass rate
cd frontend && npm run build        # Production build success
```

### Backend Quality Checks

```bash
cd backend && go test ./...         # All tests pass
cd backend && go vet ./...          # No issues
cd backend && golangci-lint run     # Linting (optional)
```

### Task Completion Criteria

A task is done when:
1. All deliverables exist and work (no placeholders/TODOs)
2. All quality checks pass (Frontend + Backend)
3. API integration verified (Anthropic SDK, SSE streams)
4. Next dependent task can start without questions

## Conventions

### Commits (Conventional Commits)

| Prefix | Example |
|--------|---------|
| `feat:` | `feat: add context window gauge component` |
| `fix:` | `fix: correct cache hit rate calculation` |
| `docs:`, `test:`, `refactor:`, `chore:` | `test: add token calculator unit tests` |

### Branches

`feat/<name>`, `fix/<name>`, `refactor/<name>`, `test/<name>`

## Agent Team Operations

### Organization

```
Main Assistant (Orchestrator) — Direct team management
  ├─ CEO AI (startup-ceo-advisor) — Strategy (when needed)
  ├─ CTO AI (startup-cto-advisor) — Architecture (when needed)
  ├─ COO AI (startup-coo) — Research (when needed)
  └─ Dynamic teammates (spawned per task)
```

**CRITICAL**: Main Assistant acts as Orchestrator directly.
- ❌ NEVER spawn engineer-team-leader agent
- ✅ Main Assistant directly: TeamCreate → Task spawn → Verify → TeamDelete

### Orchestrator Role (Main Assistant)

Main Assistant performs orchestration directly: Plan → Delegate → Verify → Integrate.

**Workflow**:
```
Request → Analyze → Break down tasks → User approval
  → Spawn agents (parallel/sequential) → Verify results → Integrate → Report
```

**Delegation Template**: Task(1 sentence) + Input(context) + Output format + Constraints + Success criteria

**Prohibitions**: Vague instructions, 2 tasks in 1 delegation, agent-to-agent communication without orchestrator

### Dynamic Team Composition

| Task Type | subagent_type | model |
|-----------|---------------|-------|
| Frontend (Next.js, React) | `general-purpose` | sonnet |
| Backend (Go API) | `general-purpose` | sonnet |
| Test writing | `general-purpose` | sonnet |
| Verification (lint, build) | `general-purpose` | haiku |
| Codebase exploration | `Explore` | haiku |

### Team Lifecycle

```
TeamCreate → TaskCreate(tasks) → Task(spawn teammates, parallel)
  → TaskUpdate(track progress) → SendMessage(communicate)
  → SendMessage(shutdown_request) → TeamDelete
```

### Validation

- Output format matches, success criteria met, interface compatibility verified
- 1st failure: Retry with specific instructions
- 2nd consecutive failure: Report to user

### Response Format

- Progress: `[Done n/Total n] Task name`
- Internal processing: Summary only (no verbose logs)
- Errors: 1 line cause + 1 line solution

## Context & Token Management (MANDATORY)

**All agents MUST follow these rules.**

### Orchestrator Token Rules

1. **Team Size**: MAX 3 members (exceeding causes token explosion)
2. **Teammate Models**: sonnet or haiku only (NO Opus)
3. **Prompts**: Essential context only, never send full files
4. **Parallel Spawn**: Independent tasks in single message with multiple Task calls
5. **Compaction**: `/compact [current task summary]` after 15 turns
6. **Shutdown**: `shutdown_request` immediately when teammate finishes (minimize idle time)

### Teammate Token Rules

1. **File Reading**: Only necessary files. NEVER read `node_modules`/`.next`/`dist`/`__pycache__`
2. **Exploration**: Glob/Grep within 3 attempts. Use Explore agent if failing
3. **Output**: Code → Verify → Report results. No unnecessary explanations
4. **Isolation**: `isolation: "worktree"` required for code changes
5. **Re-reading**: Refer to line numbers for already-read files, don't re-read

### Context Recovery

1. `/compact Focus on [task]` → 2. Complete in-progress work → 3. New session if needed (after commit)

## API Integration

### Multi-Provider Token Tracking

All providers return token usage in API responses. Each is normalized into a unified `TokenUsage` struct via provider-specific adapters.

**Anthropic** (`input_tokens`, `output_tokens`, `cache_creation_input_tokens`, `cache_read_input_tokens`):
```json
{ "usage": { "input_tokens": 2048, "output_tokens": 503, "cache_creation_input_tokens": 248, "cache_read_input_tokens": 1800 } }
```

**OpenAI** (`prompt_tokens`, `completion_tokens`, nested `cached_tokens`, `reasoning_tokens`):
```json
{ "usage": { "prompt_tokens": 100, "completion_tokens": 50, "prompt_tokens_details": { "cached_tokens": 20 } } }
```

**Gemini** (`promptTokenCount`, `candidatesTokenCount`, `cachedContentTokenCount`, `thoughtsTokenCount`):
```json
{ "usageMetadata": { "promptTokenCount": 100, "candidatesTokenCount": 50, "cachedContentTokenCount": 20 } }
```

**Mistral / Groq / OpenRouter**: OpenAI-compatible format via base URL override.

**Unified Context Formula**:
```
Cache hit rate (Anthropic) = cache_read_input_tokens / (cache_read + cache_creation)
Cache hit rate (OpenAI)    = cached_tokens / prompt_tokens
```

### Provider Architecture (Hybrid — Option C)

```
Go Backend
├── Direct SDK adapters: OpenAI, Anthropic, Gemini
├── OAI-compat adapter: Mistral, Groq, OpenRouter (base URL override)
└── UsageNormalizer → unified TokenUsage → CostCalculator (pricing.json)
```

### Model Context Limits (2026-02)

| Provider | Model | Context Window |
|----------|-------|---------------|
| Anthropic | Claude Opus 4.6 | 200K (1M beta) |
| Anthropic | Claude Sonnet 4.6 | 200K (1M beta) |
| Anthropic | Claude Haiku 4.5 | 200K |
| OpenAI | GPT-4o | 128K |
| OpenAI | o1 / o3-mini | 200K |
| Gemini | Gemini 2.0 Flash | 1M |
| Gemini | Gemini 1.5 Pro | 2M |

## Development Phases

### Phase 1: MVP + OpenAI (Week 1-3, Vercel-only, $0)
- Providers: OpenAI + Anthropic
- Real-time token counter with provider selector
- Context window gauge
- Cache efficiency chart (Anthropic + OpenAI)
- Cost calculator with per-provider pricing
- Timeline (last 100 requests)

### Phase 2: Gemini (Week 4-6, Railway, $5/mo)
- Provider: Google Gemini (2.0 Flash, 1.5 Pro/Flash)
- Persistent SSE connections via Go backend
- Session aggregation + multi-user support
- Cache efficiency chart (Gemini context caching)

### Phase 3: Long-Tail Providers (Month 2, Turso, $0 additional)
- Providers: Mistral, Groq, OpenRouter (via OAI-compat adapter)
- Token history storage
- Provider comparison charts
- Groq latency metrics visualization

## Notes

- Multi-provider from day one: OpenAI + Anthropic in Phase 1
- User provides their own API keys per provider (no platform costs)
- Focus on visual-first, real-time experience with provider comparison
- Differentiate from Langfuse/LangSmith/Helicone: SDK-based (no proxy), cost-first, multi-provider
