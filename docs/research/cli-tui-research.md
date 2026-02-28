# CLI/TUI Real-Time Token Monitor - Research Report

**Date**: 2026-02-27
**Status**: Research Complete
**Author**: COO AI (startup-coo)

---

## Executive Summary

**Feasibility**: HIGH. Claude Code stores complete token usage data locally in JSONL files. Real-time monitoring via file watching is proven by multiple existing tools (ccusage, toktrack, clog).

**Recommended Approach**: **Separate project** (`tokwatch` or `llm-viz-cli`), not integrated into llm-viz.

**Recommended Stack**: Go + Bubbletea + asciigraph + JSONL file watching (no API keys required)

**Key Insight**: Claude Code writes token usage (input_tokens, output_tokens, cache tokens) to `~/.claude/projects/<project>/<session-id>.jsonl` after every API call. This data is immediately available for real-time parsing — zero API key needed, zero latency overhead, zero proxy required.

**Why Separate Project**:
- Different tech stack (Go TUI vs Next.js web)
- Different distribution (single binary vs web app)
- Different data source (local JSONL vs API calls)
- Claude Code-specific use case doesn't overlap with multi-provider web dashboard
- Faster iteration, no monorepo complexity

**Time to MVP**: 1-2 weeks (Go + Bubbletea + JSONL parsing)

---

## 1. TUI Framework Comparison

| Framework | Language | GitHub Stars | Architecture | Rendering | Learning Curve | Best For | Recommended? |
|-----------|----------|-------------|--------------|-----------|----------------|----------|--------------|
| **bubbletea** | Go | 30,000+ | Elm (MVU) | Retained-mode | Low-Medium | Most TUI apps, rapid dev | **YES (Primary)** |
| **ratatui** | Rust | 12,000+ | Immediate-mode | Immediate-mode | Medium-High | Performance-critical TUIs | Yes (if Rust) |
| **tview** | Go | 12,460+ | Widget-based | Retained-mode | Low | Form-heavy, data grids | Alternative |
| **termui** | Go | 13,000+ | Dashboard widgets | Retained-mode | Low | Simple dashboards | Partial |
| **textual** | Python | 27,000+ | Web-inspired CSS | Delta-update | Low | Python teams, 120 FPS | No (wrong lang) |
| **cursive** | Rust | 4,000+ | Widget-based | Retained-mode | Medium | Dialog-heavy UIs | No |

### Winner: bubbletea (Go)

**Rationale**:
- llm-viz backend already uses Go — same language, shared tooling
- 30,000+ stars, largest Go TUI ecosystem (18,000+ apps built with it)
- Elm architecture naturally handles real-time state updates (file changes -> model update -> re-render)
- Rich ecosystem: Bubbles (components), Lip Gloss (styling), ntcharts (terminal charts)
- Single binary distribution (go build = cross-platform binary)
- Real-time dashboards: use `tea.Every()` command for polling, `tea.Cmd` for file watching

**bubbletea vs ratatui (Performance)**:
- ratatui uses 30-40% less memory, 15% lower CPU for 1,000 data points/second
- For token monitoring (1-2 updates/second), bubbletea performance is more than sufficient
- Go's build speed and ecosystem advantages outweigh Rust's raw performance for this use case

### Supporting Libraries

```
github.com/charmbracelet/bubbletea    # Core TUI framework
github.com/charmbracelet/lipgloss     # Styling (colors, borders, padding)
github.com/charmbracelet/bubbles      # Pre-built components (table, spinner, progress)
github.com/guptarohit/asciigraph      # ASCII line charts
github.com/NimbleMarkets/ntcharts     # BubbleTea-native charts (sparklines, bar charts)
```

---

## 2. Claude Code Integration Options

### Option A: Parse Claude Code JSONL Logs (RECOMMENDED)

**Location**: `~/.claude/projects/<encoded-project-path>/<session-id>.jsonl`

**JSONL Schema** (one JSON object per line):

```json
// Assistant message (contains token usage)
{
  "type": "assistant",
  "message": {
    "role": "assistant",
    "content": "...",
    "usage": {
      "input_tokens": 2048,
      "output_tokens": 503,
      "cache_creation_input_tokens": 248,
      "cache_read_input_tokens": 1800
    },
    "model": "claude-opus-4-6-20260301"
  },
  "uuid": "msg-uuid-here",
  "parentUuid": "parent-msg-uuid",
  "sessionId": "session-uuid",
  "cwd": "/Users/user/project",
  "version": "1.x.x",
  "timestamp": "2026-02-27T10:30:00.000Z"
}

// User message (no usage data)
{
  "type": "user",
  "message": {
    "role": "user",
    "content": "Hello, claude-code!"
  },
  "uuid": "msg-uuid",
  "sessionId": "session-uuid",
  "timestamp": "2026-02-27T10:29:00.000Z"
}
```

**Key Fields for Token Monitoring**:
- `message.usage.input_tokens` — input token count
- `message.usage.output_tokens` — output token count
- `message.usage.cache_creation_input_tokens` — cache write tokens
- `message.usage.cache_read_input_tokens` — cache read tokens
- `message.model` — model name (for context window limit lookup)
- `timestamp` — for session time tracking

**Important Note**: As of Claude Code v1.0.9+, `costUSD` field is no longer included in JSONL files on Max plan. Cost must be calculated from token counts using pricing data (same approach as ccusage and toktrack).

**File Discovery**:
```go
// Find active session: most recently modified JSONL in ~/.claude/projects/**/*.jsonl
pattern := filepath.Join(os.Getenv("HOME"), ".claude", "projects", "**", "*.jsonl")
```

**Real-Time Watching**: Use `fsnotify` (Go file system notifications) to watch the JSONL file and trigger re-parse on Write events.

**Pros**: Zero API keys, zero latency, proven approach (ccusage, toktrack, clog all use this)
**Cons**: JSONL format could theoretically change between Claude Code versions

---

### Option B: Claude Code Hooks (ADVANCED)

Claude Code supports lifecycle hooks that fire on events like `Stop`, `PostToolUse`.

**Hook config** (`~/.claude/settings.json`):
```json
{
  "hooks": {
    "Stop": [{
      "matcher": "",
      "hooks": [{
        "type": "command",
        "command": "curl -X POST http://localhost:7890/hook -d @-"
      }]
    }]
  }
}
```

**Hook payload** (stdin JSON):
```json
{
  "session_id": "...",
  "transcript_path": "~/.claude/projects/.../session.jsonl",
  "cwd": "/path/to/project",
  "hook_event_name": "Stop"
}
```

**Architecture**: tokwatch runs a local HTTP server on port 7890. Claude Code hooks POST to it on session stop/tool use. tokwatch reads the `transcript_path` from the payload.

**Pros**: Event-driven (no polling), exact timing
**Cons**: Requires hook configuration by user, not zero-config

---

### Option C: API Proxy (NOT RECOMMENDED)

Intercept Anthropic API calls via a local proxy (MITM). Users would need to change their API base URL.

**Cons**: Requires SSL certificate setup, changes to Claude Code config, privacy concerns, complex setup.

---

### Option D: Anthropic Admin API Polling (NOT RECOMMENDED for real-time)

Poll the Admin API for usage data. Has a 5-minute delay. Not suitable for real-time monitoring.

---

**Decision**: Option A (JSONL parsing) as primary, Option B (hooks) as optional enhancement.

---

## 3. Data Visualization in Terminal

### Graph Types and Libraries

#### 3.1 ASCII Line Chart (asciigraph)

Token usage over time:

```go
import "github.com/guptarohit/asciigraph"

data := []float64{100, 250, 180, 420, 310, 580, 490}
graph := asciigraph.Plot(data,
    asciigraph.Height(8),
    asciigraph.Width(40),
    asciigraph.Caption("Input Tokens Over Time"),
    asciigraph.SeriesColors(asciigraph.Blue),
)
fmt.Println(graph)
// Output:
// 580.00 ┤      ╭╮
// 507.33 ┤     ╭╯│
// 434.67 ┤    ╭╯ ╰╮
// 362.00 ┤   ╭╯   │
// 289.33 ┤  ╭╯    ╰╮
// ...
```

#### 3.2 ASCII Progress Bar (Context Window Gauge)

Custom implementation — simple and effective:

```go
func contextGauge(used, total int) string {
    pct := float64(used) / float64(total)
    width := 30
    filled := int(pct * float64(width))

    color := "\033[32m" // green
    if pct > 0.8 { color = "\033[33m" } // yellow
    if pct > 0.95 { color = "\033[31m" } // red

    bar := color + strings.Repeat("█", filled) + "\033[0m" + strings.Repeat("░", width-filled)
    return fmt.Sprintf("Context [%s] %.1f%% (%dk/%dk)", bar, pct*100, used/1000, total/1000)
}
// Output: Context [████████████░░░░░░░░░░░░░░░░░░] 40.0% (80k/200k)
```

#### 3.3 Sparkline (ntcharts + bubbletea)

Compact trend inline with stats:

```go
import "github.com/NimbleMarkets/ntcharts/sparkline"

// Sparkline shows last 20 token counts compactly
sl := sparkline.New(20, 3)
sl.Push(tokenCounts...)
view := sl.View() // Renders as small bar graph
// Output: ▁▂▃▄▅▆▇█▇▆▅▄▃▂▁▂▃▄▅▆
```

#### 3.4 Table (bubbles/table)

Session history:

```go
import "github.com/charmbracelet/bubbles/table"

columns := []table.Column{
    {Title: "Time",    Width: 8},
    {Title: "Input",   Width: 8},
    {Title: "Output",  Width: 8},
    {Title: "Cache%",  Width: 7},
    {Title: "Cost",    Width: 8},
}
rows := []table.Row{
    {"10:30:01", "2,048", "503", "87%", "$0.012"},
    {"10:28:45", "1,800", "320", "91%", "$0.009"},
}
```

#### 3.5 Complete Dashboard Layout (Lip Gloss + bubbletea)

```
╭─────────────── tokwatch ────────────────────────╮
│  Provider: Anthropic  Model: claude-opus-4-6     │
│  Session: 10:25 → now  Duration: 00:32:15        │
╰─────────────────────────────────────────────────╯
╭── Context Window ──────────────────────────────╮
│ [████████████░░░░░░░░░░░░░░░░░░] 40.0% (80k/200k)│
╰────────────────────────────────────────────────╯
╭── Tokens ──────────────────────────────────────╮
│  Input:  12,450    Output: 3,210               │
│  Cache:   9,800    Miss:   2,650               │
│  Cache Hit Rate: 78.7%  Saved: $0.045          │
╰────────────────────────────────────────────────╯
╭── Session Cost ────────────────────────────────╮
│  This request: $0.012                          │
│  Session total: $0.187                         │
│  Rate: $0.35/hr                                │
╰────────────────────────────────────────────────╯
╭── Token Timeline ──────────────────────────────╮
│ 580 ┤      ╭╮                                  │
│ 435 ┤    ╭─╯╰─╮                                │
│ 290 ┤  ╭─╯    ╰──╮                             │
│ 145 ┤╭─╯         ╰──                           │
╰────────────────────────────────────────────────╯
╭── Last 5 Requests ─────────────────────────────╮
│  Time     Input  Output Cache%  Cost           │
│  10:30:01  2,048    503   87%  $0.012          │
│  10:28:45  1,800    320   91%  $0.009          │
│  10:25:12  3,200    810   65%  $0.021          │
╰────────────────────────────────────────────────╯
  q: quit  r: reset  t: toggle provider  ?: help
```

---

## 4. Real-Time Update Architecture

### Refresh Strategy

| Method | Latency | CPU | Recommended |
|--------|---------|-----|-------------|
| File watch (fsnotify) + parse on event | <100ms | Minimal | YES |
| Polling every 500ms | ~500ms | Low | Acceptable |
| Polling every 100ms | ~100ms | Medium | Avoid |
| Hook-based (HTTP POST) | Event-driven | Minimal | Advanced |

### bubbletea Real-Time Pattern

```go
// Tick every second to check for file changes
func tickCmd() tea.Cmd {
    return tea.Every(time.Second, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

// File watch command (non-blocking)
func watchFileCmd(path string) tea.Cmd {
    return func() tea.Msg {
        // Use fsnotify watcher
        return fileChangedMsg{path: path}
    }
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case fileChangedMsg:
        // Re-parse JSONL tail (only new lines)
        newData := parseJSONLTail(msg.path, m.lastOffset)
        m.tokenData = append(m.tokenData, newData...)
        return m, watchFileCmd(msg.path)  // Continue watching
    case tickMsg:
        return m, tickCmd()  // Continue ticking
    }
}
```

### Terminal Compatibility

- **ANSI colors**: Supported by all modern terminals (iTerm2, Terminal.app, Windows Terminal, WSL2)
- **Unicode box-drawing**: `╭╮╰╯│─` — requires UTF-8 locale (default on macOS/Linux)
- **Windows**: Works in Windows Terminal (WT) but not cmd.exe. Acceptable tradeoff.
- **Fallback**: Use ASCII `+--+` if Unicode detection fails (check `$TERM` and `$LANG`)

### Performance Targets

| Metric | Target | Approach |
|--------|--------|----------|
| CPU usage | <1% (idle) | File watch (not polling) |
| Memory | <20MB | Sliding window, keep last 100 requests |
| Render latency | <50ms | Only re-render on data change |
| File parse latency | <5ms | Parse only new JSONL lines (offset tracking) |

---

## 5. Existing Tool Analysis

### 5.1 toktrack (mag123c/toktrack)

**Language**: Rust
**Distribution**: npm (`npx toktrack`) — wraps Rust binary
**Stars**: Growing (launched 2025, Product Hunt featured)

**Architecture**:
- Cold path: Full glob scan → parallel SIMD JSON parsing (simd-json) → build cache → aggregate
- Warm path: Load `~/.toktrack/` cache → parse only recent files (mtime filter) → merge
- Performance: ~3 GiB/s JSONL throughput, 3,500+ files in 40ms on Apple Silicon

**TUI**: 3-tab dashboard (Overview, Stats, Models) — daily/weekly/monthly views
**Scope**: Historical aggregation, NOT real-time session monitoring
**Multi-CLI**: Claude Code, Codex CLI, Gemini CLI, OpenCode

**Lessons Learned**:
- SIMD JSON parsing is overkill for real-time monitoring (single file, <1MB)
- Caching daily summaries independently (survives Claude Code log deletion)
- User demand for multi-CLI support is real — design for extensibility
- npm wrapper for Rust binary is clever for distribution (no cargo install required)

**Gap vs tokwatch**:
- toktrack = historical analysis (past sessions)
- tokwatch = real-time monitoring (active session, live updates)

---

### 5.2 ccusage (ryoppippi/ccusage)

**Language**: TypeScript/Bun
**Distribution**: npm (`npx ccusage`)

**Architecture**:
- Reads `~/.claude/projects/**/*.jsonl` glob
- Parses JSONL into usage records: input, output, cache_creation, cache_read tokens
- Groups by: daily, weekly, monthly, 5-hour billing blocks
- Calculates cost from token counts × pricing table (not from costUSD field)

**Key Features**:
- Session/project breakdown
- 5-hour billing block analysis (Anthropic Max plan billing cycle)
- JSON output mode for piping
- Interactive TUI with keyboard navigation

**Lessons Learned**:
- 5-hour billing block is a unique Anthropic Max plan concept — worth supporting
- Users want both session-level and daily/monthly aggregation
- JSON output mode enables downstream tooling (piping to jq, etc.)

---

### 5.3 clog (HillviewCap/clog)

**Language**: TypeScript (web-based)
**Architecture**: Web viewer with real-time file watching
**Key Feature**: "Automatic detection of changes to JSONL files and live updates when new log entries are added"

**Lessons Learned**:
- Real-time JSONL watching is already validated — not a technical risk
- Web-based approach has disadvantage: requires browser open separately
- Terminal-native is simpler UX (no browser tab needed)

---

### 5.4 claude-token-monitor (crates.io)

**Language**: Rust
**Distribution**: cargo install
**Key Feature**: File-based monitoring of Claude Code sessions

**Architecture**: Watches JSONL file, displays token counts in terminal
**Lessons Learned**: Simpler tool validates the file-watch approach works

---

## 6. Project Structure Decision

**Recommendation: Separate Project (`tokwatch`)**

### Decision Matrix

| Criterion | Integrated (`llm-viz/cli/`) | Separate (`tokwatch`) |
|-----------|---------------------------|----------------------|
| Tech stack fit | Poor (Next.js vs Go TUI) | Good (Go only) |
| Distribution | Awkward (web + binary) | Clean (single binary) |
| Data source | Shared pricing logic only | Different (JSONL vs API) |
| Iteration speed | Slow (monorepo overhead) | Fast |
| Branding | Unified | Different but linked |
| Code reuse | pricing.json, cost calc | Minimal |
| Maintenance | Shared CI/CD complexity | Independent |

**Verdict**: The tools serve different use cases. llm-viz is a web dashboard for developers testing API calls with their own keys. tokwatch is a CLI monitor for Claude Code power users watching their active session. Shared code is minimal (pricing data only — can duplicate or reference shared JSON file).

**Project name options**: `tokwatch`, `ccwatch`, `claude-monitor`
**Recommended name**: `tokwatch` (generic, not Claude-specific, room to expand to Codex/Gemini)

---

## 7. MVP Feature Set

### P0 (Must Have for Week 1-2 Launch)

| Feature | Implementation | Complexity |
|---------|---------------|------------|
| Real-time token counter (input/output/cache) | JSONL tail parse | Low |
| Context window gauge (ASCII progress bar) | Model limits table + gauge | Low |
| Session cost tracker | Token × pricing.json | Low |
| Cache hit rate display | cache_read / (cache_read + cache_creation) | Low |
| Active session auto-detection | Find most recent .jsonl by mtime | Low |
| Model detection | Parse `message.model` field | Low |

### P1 (Week 3-4, post-launch)

| Feature | Implementation | Complexity |
|---------|---------------|------------|
| Session history table (last 10 requests) | bubbles/table | Medium |
| Token sparkline (last 20 data points) | ntcharts sparkline | Medium |
| Multiple session support | Session picker UI | Medium |
| Cost rate ($/hour estimate) | Rolling 5-min average | Low |

### P2 (Month 2, based on user feedback)

| Feature | Implementation | Complexity |
|---------|---------------|------------|
| Multi-CLI support (Codex, Gemini CLI) | Different JSONL paths/schemas | Medium |
| Hooks integration (Option B) | Local HTTP server | Medium |
| Export session report (JSON/CSV) | File write | Low |
| Cost alerts (threshold notifications) | Configurable thresholds | Low |
| Daily/weekly aggregation | ccusage-style summary | Medium |
| Config file (refresh rate, alerts) | TOML/YAML config | Low |

**Cut from MVP** (validated as post-MVP):
- MCP server integration (complex, unclear user demand)
- Customizable dashboard layouts
- Browser extension
- Provider comparison (llm-viz handles this)

---

## 8. Technical Challenges and Solutions

### Challenge 1: JSONL Format Changes Between Versions

**Risk**: Anthropic changes JSONL schema (costUSD was removed in v1.0.9)

**Solution**:
- Parse defensively with `omitempty` / optional fields
- Only depend on core fields: `type`, `message.usage.*`, `message.model`, `timestamp`
- Version detection via `version` field in JSONL entries
- Test matrix across Claude Code versions (v1.0.x, v1.1.x, latest)

```go
type AssistantMessage struct {
    Type    string `json:"type"`
    Message struct {
        Usage *TokenUsage `json:"usage,omitempty"` // Pointer = optional
        Model string      `json:"model,omitempty"`
    } `json:"message"`
    Timestamp time.Time `json:"timestamp"`
}
```

---

### Challenge 2: Finding the Active Session

**Risk**: Multiple projects open simultaneously, which JSONL to watch?

**Solution**:
1. Detect by most recently modified `.jsonl` file (mtime)
2. Optional: Accept `--project` flag to specify project path
3. Optional: Read `$CLAUDE_CODE_PROJECT` env var if Claude Code sets it

```go
func findActiveSession() (string, error) {
    pattern := filepath.Join(os.Getenv("HOME"), ".claude", "projects", "*", "*.jsonl")
    files, _ := filepath.Glob(pattern)

    var newest string
    var newestTime time.Time
    for _, f := range files {
        info, _ := os.Stat(f)
        if info.ModTime().After(newestTime) {
            newestTime = info.ModTime()
            newest = f
        }
    }
    return newest, nil
}
```

---

### Challenge 3: Real-Time File Watching Cross-Platform

**Risk**: `fsnotify` behaves differently on macOS (kqueue) vs Linux (inotify) vs Windows

**Solution**: Use `github.com/fsnotify/fsnotify` — it abstracts platform differences. Fallback to 1-second polling if fsnotify fails (tail -f style).

```go
import "github.com/fsnotify/fsnotify"

watcher, _ := fsnotify.NewWatcher()
watcher.Add(jsonlPath)

go func() {
    for event := range watcher.Events {
        if event.Op&fsnotify.Write == fsnotify.Write {
            // Re-parse new lines from last offset
            newLines := readNewLines(jsonlPath, lastOffset)
            updateChan <- newLines
        }
    }
}()
```

---

### Challenge 4: Context Window Limits by Model

**Risk**: New models added frequently, hardcoded limits go stale

**Solution**: Maintain a `models.json` with limits. Check weekly via GitHub Actions. Same approach as llm-viz pricing.json.

```json
{
  "claude-opus-4-6": 200000,
  "claude-sonnet-4-6": 200000,
  "claude-haiku-4-5": 200000,
  "claude-3-5-sonnet-20241022": 200000,
  "gpt-4o": 128000
}
```

---

### Challenge 5: No costUSD in JSONL (Max Plan)

**Risk**: Users on Max plan don't have costUSD; API plan users might have it or not

**Solution**: Always calculate cost from tokens × pricing, never rely on costUSD field. Same approach as ccusage and toktrack.

```go
func calculateCost(usage TokenUsage, model string, pricing PricingData) float64 {
    p := pricing[model]
    return float64(usage.InputTokens) * p.InputPerToken +
           float64(usage.OutputTokens) * p.OutputPerToken +
           float64(usage.CacheWriteTokens) * p.CacheWritePerToken +
           float64(usage.CacheReadTokens) * p.CacheReadPerToken
}
```

---

## 9. Phased Rollout Plan

### Phase 1: Static Snapshot CLI (Week 1)

**Goal**: Working tool users can install and run today.

**Deliverables**:
- `tokwatch snapshot` — one-shot display of current session stats
- JSONL parsing (find active session, parse all lines)
- Token counter, cost, cache hit rate display
- Context window gauge
- Plain terminal output (no TUI framework yet)

**Distribution**: `go install github.com/yourusername/tokwatch@latest`

**Success Criteria**: Accurate token count matching Claude Code's `/cost` command output

---

### Phase 2: Real-Time TUI (Week 2)

**Goal**: Live-updating dashboard in second terminal window.

**Deliverables**:
- `tokwatch` (default command) — launches TUI
- bubbletea integration
- fsnotify file watching (real-time JSONL tail)
- Token timeline sparkline (ntcharts)
- Session history table (bubbles/table)
- Lip Gloss styling (colors, borders)
- Keyboard: `q` quit, `r` reset, `?` help

**Success Criteria**: Updates within 1 second of Claude Code making an API call

---

### Phase 3: Multi-Level Views + Command Tracking (Month 2)

**Goal**: Advanced monitoring with multi-level views and command-to-token tracking.

**Deliverables**:

#### 3.1 Multi-Level View System

Interactive navigation between 4 view levels:

**Global View** (전체):
- Total tokens across all sessions
- Total USD cost
- Number of sessions
- Active agents count
- Active teams count
- Overall cache hit rate

**Session View** (세션별):
- List of all sessions with:
  - Session ID, timestamp, duration
  - Total tokens, cost
  - Number of commands
  - Agent/team participation
- Select session to drill down into details

**Agent View** (에이전트별):
- List of all agents (teammates) with:
  - Agent name/ID
  - Total tokens consumed
  - USD cost
  - Tasks completed
  - Percentage of total session tokens
  - Status (active/idle/completed)
- Filter by team or session

**Team View** (팀 그룹별):
- List of all teams with:
  - Team name
  - Number of members
  - Total tokens consumed (team aggregate)
  - USD cost
  - Tasks (pending/in_progress/completed)
  - Team status (active/completed)
- Drill down to see team members

**Navigation**:
- Keyboard: `g` (global), `s` (sessions), `a` (agents), `t` (teams), Tab (cycle views)
- Mouse support: click to switch views

#### 3.2 Command Tracking

**User Command → Token Consumption Mapping**:

Track which user commands trigger token usage:

```
Command History View:
┌─────────────────────────────────────────────────────┐
│ 22:00 │ "Phase 1 MVP 개발 시작해줘"                  │
│       ├─ TeamCreate: 150 tokens                     │
│       ├─ TaskCreate (2): 300 tokens                 │
│       ├─ Task spawn (2 agents): 1,200 tokens        │
│       ├─ Agents: backend-engineer (23K)             │
│       │          frontend-engineer (21K)            │
│       └─ Total: 45,650 tokens → $2.28               │
├─────────────────────────────────────────────────────┤
│ 21:30 │ "multi-provider 리서치해줘"                   │
│       ├─ COO spawn: 2,000 tokens                    │
│       ├─ Research: 45,376 tokens                    │
│       └─ Total: 47,376 tokens → $2.31               │
└─────────────────────────────────────────────────────┘
```

**Implementation**:
- Parse JSONL to find user messages (type: "user")
- Associate subsequent assistant messages + agent messages
- Aggregate token usage per user command
- Display in chronological order with expandable details

#### 3.3 Claude Code Agent Teams Integration

**JSONL Fields to Parse** (if available):
- `sessionId` - Session UUID
- `agentId` - Agent/teammate ID (if spawned via Task tool)
- `teamName` - Team name (if agent belongs to a team)
- `parentUuid` - Parent message UUID (for command tracking)

**Agent Detection**:
- Check JSONL file path: `~/.claude/projects/<project>/<session>/subagents/agent-<id>.jsonl`
- Parse agent-specific JSONL files
- Aggregate tokens per agent

**Team Detection**:
- Check `~/.claude/teams/<team-name>/config.json` for team roster
- Map agents to teams
- Aggregate tokens per team

#### 3.4 Other Phase 3 Features

- Multi-CLI support (Codex CLI, Gemini CLI)
- Claude Code hooks integration (Option B — event-driven updates)
- `tokwatch report` — daily/weekly cost summary
- Cost alerts (`--alert 5.00` warns at $5.00 session cost)
- Config file (`~/.tokwatch.toml`)
- Export: `tokwatch export --format json`
- GitHub Actions: weekly model pricing update

**Success Criteria**: 100+ GitHub stars, multi-level views working, command tracking accurate

---

## 10. Recommended Tech Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| **Language** | Go 1.22+ | Same as llm-viz backend, single binary, fast build |
| **TUI Framework** | bubbletea (charmbracelet) | 30K stars, Elm architecture for real-time state, Go ecosystem leader |
| **Styling** | lipgloss (charmbracelet) | Native bubbletea companion, terminal color/layout |
| **Components** | bubbles (charmbracelet) | Table, progress bar, spinner — no build-from-scratch |
| **Line Charts** | asciigraph (guptarohit) | Zero-dependency, simple API, proven terminal output |
| **Sparklines** | ntcharts (NimbleMarkets) | BubbleTea-native, sparkline + bar chart |
| **Data Source** | JSONL file watching | Proven (ccusage, toktrack, clog), zero API keys, <100ms latency |
| **File Watcher** | fsnotify | Cross-platform, standard Go file watching library |
| **Distribution** | Single binary | `go build` produces cross-platform binary, `go install` for users |
| **Pricing Data** | Local pricing.json | Weekly GH Actions update, same pattern as llm-viz |
| **Config** | TOML (BurntSushi/toml) | Simple, human-readable, standard Go config format |

**Why not Python/Textual**: Wrong language for the Go ecosystem. Binary distribution harder.
**Why not Rust/Ratatui**: Higher build complexity, slower iteration. Performance advantage unnecessary at 1-2 updates/sec.
**Why not tview**: Less active development than bubbletea, weaker ecosystem.

---

## References

- [bubbletea - charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea)
- [toktrack - mag123c/toktrack](https://github.com/mag123c/toktrack)
- [ccusage - ryoppippi/ccusage](https://github.com/ryoppippi/ccusage)
- [clog - HillviewCap/clog](https://github.com/HillviewCap/clog)
- [asciigraph - guptarohit/asciigraph](https://github.com/guptarohit/asciigraph)
- [ntcharts - NimbleMarkets/ntcharts](https://github.com/NimbleMarkets/ntcharts)
- [ratatui vs bubbletea comparison - DEV Community](https://dev.to/dev-tngsh/go-vs-rust-for-tui-development-a-deep-dive-into-bubbletea-and-ratatui-2b7)
- [Claude Code token usage in JSONL - Shipyard](https://shipyard.build/blog/claude-code-track-usage/)
- [Claude Code JSONL analysis - Liam ERD](https://liambx.com/blog/claude-code-log-analysis-with-duckdb)
- [toktrack on Hacker News](https://news.ycombinator.com/item?id=46855551)
- [Textual Python TUI Framework](https://textual.textualize.io/)
- [tview - rivo/tview](https://github.com/rivo/tview)
- [Claude Code Hooks Reference](https://code.claude.com/docs/en/hooks)
- [fsnotify - file watching Go library](https://github.com/fsnotify/fsnotify)
- [bubbles - charmbracelet/bubbles](https://pkg.go.dev/github.com/charmbracelet/bubbletea)
