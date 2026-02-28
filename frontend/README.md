# llm-viz Frontend

Real-time multi-provider LLM token consumption dashboard built with Next.js App Router.

## Features

- **Provider Selector**: OpenAI and Anthropic support with model selection
- **Token Counter**: Real-time input/output/cache token display
- **Context Gauge**: Circular gauge showing context window utilization per model
- **Cost Tracker**: USD cost breakdown per request and session total
- **Cache Chart**: Cache hit/miss visualization (Recharts PieChart)
- **Usage Timeline**: Last 100 requests as a Recharts LineChart

## Getting Started

### Prerequisites

- Node.js 18+
- API key from OpenAI or Anthropic

### Installation

```bash
npm install
```

### Development

```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000).

### Production Build

```bash
npm run build
npm run start
```

## Environment Variables

No environment variables required for Phase 1. API keys are entered in the browser UI and never stored server-side.

## Architecture

```
app/
├── page.tsx                 # Main dashboard (client component)
├── layout.tsx               # Root layout
├── globals.css              # Global styles
├── api/
│   ├── proxy/route.ts       # Multi-provider API proxy
│   └── stream/route.ts      # SSE endpoint for real-time updates
└── components/
    ├── ProviderSelector.tsx  # Provider/model/API key selection
    ├── TokenCounter.tsx      # Token count display
    ├── ContextGauge.tsx      # Context window circular gauge
    ├── CostTracker.tsx       # USD cost breakdown
    ├── CacheChart.tsx        # Cache hit/miss pie chart
    ├── UsageTimeline.tsx     # Line chart (last 100 requests)
    └── ChatInput.tsx         # Message input to trigger API calls
hooks/
└── useTokenStream.ts        # SSE client hook
lib/
├── model-limits.ts          # Model context window limits
├── cost-calculator.ts       # Per-provider USD cost formulas
└── token-calculator.ts      # Token aggregation helpers
types/
└── token-data.ts            # TypeScript interfaces
```

## API Key Security

API keys are stored **only in browser memory** for the session duration. They are:
- Never stored in `localStorage` or `sessionStorage`
- Never logged or persisted server-side
- Cleared on page refresh

## Quality Checks

```bash
npm run lint        # ESLint (zero errors)
npm run type-check  # TypeScript strict mode (zero errors)
npm run build       # Production build
```

## Tech Stack

- **Framework**: Next.js (App Router)
- **Language**: TypeScript (strict mode)
- **Styling**: Tailwind CSS v4
- **Charts**: Recharts
- **Real-time**: SSE via `EventSource` API
