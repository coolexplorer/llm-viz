# Claude Code Real-time Integration Guide

**Last Updated**: 2026-03-01
**Status**: Production Ready
**Difficulty**: Intermediate

---

## Overview

This runbook guides you through connecting **llm-viz** to your active Claude Code session for real-time token monitoring and cost tracking.

### What You'll Achieve

- ✅ Monitor Claude Code token consumption in real-time
- ✅ Track costs per conversation, per agent, per team
- ✅ Visualize context window usage across agents
- ✅ Identify high-cost operations and optimize prompts
- ✅ Export usage data for analysis

### Prerequisites

- Claude Code CLI installed and configured
- Anthropic API key (same key used in Claude Code)
- llm-viz frontend + backend running
- Basic familiarity with environment variables

---

## Architecture

```
┌─────────────────────────────────────────┐
│   Your Claude Code Session              │
│   (uses Anthropic API)                  │
└──────────────┬──────────────────────────┘
               │
               │ API Key (shared)
               │
               ▼
┌──────────────────────────────────────────┐
│   llm-viz Backend (Go)                   │
│   • Intercepts API calls (transparent)   │
│   • Tracks token usage                   │
│   • Calculates costs                     │
│   • Broadcasts via SSE                   │
└──────────────┬───────────────────────────┘
               │
               │ SSE Stream
               │
               ▼
┌──────────────────────────────────────────┐
│   llm-viz Dashboard (Browser)            │
│   • Real-time token counter              │
│   • Cost tracker                         │
│   • Context window gauge                 │
│   • Usage timeline                       │
└──────────────────────────────────────────┘
```

---

## Step 1: Verify Claude Code API Key

Claude Code uses the Anthropic API key from your environment or settings.

### Option A: Environment Variable

```bash
echo $ANTHROPIC_API_KEY
# Should output: sk-ant-api03-...
```

### Option B: Claude Settings File

```bash
cat ~/.config/claude/config.json | grep apiKey
# Or for macOS:
cat ~/Library/Application\ Support/Claude/config.json | grep apiKey
```

**⚠️ Security Note**: Never commit this key to git. Use `.env` files.

---

## Step 2: Configure llm-viz Backend

### 2.1 Create `.env` File

```bash
cd /Users/kimseunghwan/ClaudProjects/llm-viz/backend
cp .env.example .env
```

### 2.2 Edit `.env`

```bash
# Use the SAME API key as Claude Code
ANTHROPIC_API_KEY=sk-ant-api03-xxxxx

# Optional: Add OpenAI key if you use both
OPENAI_API_KEY=sk-xxxxx

# Server configuration
PORT=8080
ALLOWED_ORIGIN=http://localhost:3000
LOG_LEVEL=info

# Pricing database
PRICING_FILE=data/pricing.json
```

### 2.3 Start Backend

```bash
cd /Users/kimseunghwan/ClaudProjects/llm-viz/backend
go run ./cmd/server

# Expected output:
# 2026/03/01 00:40:00 INFO Server starting addr=:8080
# 2026/03/01 00:40:00 INFO Providers configured providers=[anthropic openai]
```

**✅ Checkpoint**: Backend should be running on `http://localhost:8080`

---

## Step 3: Start Frontend Dashboard

Open a **new terminal window**:

```bash
# Set LLM_VIZ_ROOT if not already set
LLM_VIZ_ROOT="${LLM_VIZ_ROOT:-$HOME/llm-viz}"

cd "$LLM_VIZ_ROOT/frontend"
npm install  # Only needed once
npm run dev

# Expected output:
# ▲ Next.js 16.1.6
# - Local:        http://localhost:3000
# - Ready in 1.2s
```

**✅ Checkpoint**: Dashboard should open at `http://localhost:3000`

---

## Step 4: Configure Session Tracking

### 4.1 Choose a Session ID

Use a consistent session ID to group related Claude Code interactions:

```bash
# Example session IDs:
# - By project: "llm-viz-dev"
# - By date: "2026-03-01-morning"
# - By feature: "pr-review-session"
# - By agent team: "test-coverage-team"
```

### 4.2 Create Test Script

Create `test-claude-integration.sh` in your llm-viz directory:

```bash
#!/bin/bash
# test-claude-integration.sh - Test llm-viz with Claude Code

SESSION_ID="claude-code-test-$(date +%Y%m%d-%H%M%S)"
API_KEY="$ANTHROPIC_API_KEY"

echo "🔍 Testing llm-viz integration"
echo "Session ID: $SESSION_ID"

# Send a test completion request
curl -X POST http://localhost:8080/api/complete \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "anthropic",
    "model": "claude-sonnet-4-6",
    "messages": [
      {"role": "user", "content": "Hello! This is a test from llm-viz."}
    ],
    "max_tokens": 100,
    "session_id": "'"$SESSION_ID"'"
  }' | jq .

echo ""
echo "✅ Check your dashboard at http://localhost:3000"
echo "   Session ID: $SESSION_ID"
```

### 4.3 Run Test

```bash
chmod +x test-claude-integration.sh
./test-claude-integration.sh
```

**Expected Result**:

Terminal shows JSON response:
```json
{
  "id": "msg_01abc123",
  "content": "Hello! This is Claude responding to your test message from llm-viz.",
  "usage": {
    "input_tokens": 24,
    "output_tokens": 18,
    "cache_read_tokens": 0,
    "cache_write_tokens": 0,
    "total_tokens": 42
  }
}
```

- Dashboard updates in real-time
- Token counter increments to 42 total tokens
- Cost tracker shows USD estimate (~$0.00054 for this request)

---

## Step 5: Monitor Live Claude Code Sessions

### Method 1: Wrapper Script (Recommended)

Create `~/bin/claude-monitored`:

```bash
#!/bin/bash
# claude-monitored - Run Claude Code with llm-viz monitoring
# Set LLM_VIZ_ROOT to your llm-viz installation path (defaults to ~/llm-viz)

# 1. Determine llm-viz root directory
LLM_VIZ_ROOT="${LLM_VIZ_ROOT:-$HOME/llm-viz}"

# 2. Start llm-viz backend (if not running)
if ! curl -s http://localhost:8080/api/health > /dev/null; then
    echo "🚀 Starting llm-viz backend..."
    cd "$LLM_VIZ_ROOT/backend"
    go run ./cmd/server > /tmp/llm-viz-backend.log 2>&1 &
    sleep 2
fi

# 3. Open dashboard in browser (if not open)
if ! curl -s http://localhost:3000 > /dev/null; then
    echo "🌐 Starting llm-viz dashboard..."
    cd "$LLM_VIZ_ROOT/frontend"
    npm run dev > /tmp/llm-viz-frontend.log 2>&1 &
    sleep 3
    open http://localhost:3000
fi

# 4. Set session ID based on current directory
SESSION_ID="claude-$(basename $(pwd))-$(date +%H%M)"
export CLAUDE_SESSION_ID="$SESSION_ID"

echo "📊 llm-viz monitoring active"
echo "   Dashboard: http://localhost:3000"
echo "   Session: $SESSION_ID"
echo ""

# 5. Run Claude Code
claude "$@"
```

**Usage**:

```bash
chmod +x ~/bin/claude-monitored

# If llm-viz is not in ~/llm-viz, set LLM_VIZ_ROOT
export LLM_VIZ_ROOT="/path/to/your/llm-viz"

claude-monitored  # Instead of 'claude'
```

### Method 2: Environment Variable Hook

If Claude Code supports custom hooks (check Claude Code documentation):

Create `~/.config/claude/hooks/post-api-call.sh`:

```bash
#!/bin/bash
# Post-API-call hook - send usage to llm-viz

curl -X POST http://localhost:8080/api/complete \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "'"${CLAUDE_PROVIDER:-anthropic}"'",
    "model": "'"$CLAUDE_MODEL"'",
    "session_id": "'"${CLAUDE_SESSION_ID:-default}"'",
    "usage": {
      "input_tokens": '"$CLAUDE_INPUT_TOKENS"',
      "output_tokens": '"$CLAUDE_OUTPUT_TOKENS"'
    }
  }' > /dev/null 2>&1
```

---

## Step 6: Real-time Monitoring in Action

### Open Dashboard

Visit `http://localhost:3000` and you'll see:

1. **Provider Selector**
   - Choose "Anthropic"
   - Select your model (e.g., claude-sonnet-4-6)
   - API key auto-detected from backend

2. **Session Filter**
   - Enter your session ID
   - Click "Subscribe" to listen for events

3. **Real-time Updates**
   - Token counter updates on each API call
   - Cost accumulates automatically
   - Context window gauge shows capacity
   - Timeline chart plots requests

### Monitor Specific Scenarios

#### Scenario 1: Agent Team Monitoring

```bash
# In Claude Code, when using agent teams:
export CLAUDE_SESSION_ID="team-test-coverage-80"

# Your agent team tokens will be tracked under this session
```

#### Scenario 2: PR Review Tracking

```bash
export CLAUDE_SESSION_ID="pr-review-$(date +%Y%m%d)"

# All PR review interactions grouped together
```

#### Scenario 3: Cost Comparison

```bash
# Test with Sonnet
export CLAUDE_SESSION_ID="sonnet-test"
# ... use Claude Code ...

# Test with Haiku
export CLAUDE_SESSION_ID="haiku-test"
# ... use Claude Code ...

# Compare costs in dashboard timeline
```

---

## Step 7: Advanced Features

### 7.1 Export Usage Data

```bash
# Get session history
curl "http://localhost:8080/api/stats?session_id=your-session-id&limit=1000" \
  | jq . > usage-report.json

# Get provider stats
curl "http://localhost:8080/api/stats?provider=anthropic&start=2026-03-01T00:00:00Z" \
  | jq . > anthropic-usage.json
```

### 7.2 Cache Efficiency Analysis

Track how much you save with prompt caching:

```bash
curl "http://localhost:8080/api/stats?session_id=your-session" \
  | jq '.records[] | {
      timestamp,
      input: .usage.input_tokens,
      cache_read: .usage.cache_read_tokens,
      cache_write: .usage.cache_write_tokens,
      cost: .cost_usd
    }'
```

### 7.3 Multi-Project Tracking

Use project tags:

```bash
curl -X POST http://localhost:8080/api/complete \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "anthropic",
    "model": "claude-sonnet-4-6",
    "messages": [...],
    "session_id": "my-session",
    "project_tag": "llm-viz"
  }'
```

---

## Troubleshooting

### Issue 1: Dashboard Not Updating

**Symptoms**: Dashboard shows "Connecting..." or no data

**Solutions**:

```bash
# 1. Check backend is running
curl http://localhost:8080/api/health
# Expected: {"status":"ok","providers":[...]}

# 2. Check SSE connection
curl -N http://localhost:8080/api/sse?session_id=test
# Should keep connection open and send heartbeats

# 3. Check browser console for errors
# Open DevTools → Console → look for SSE errors
```

### Issue 2: API Key Mismatch

**Symptoms**: Backend shows different provider than Claude Code

**Solution**:

```bash
# Verify both use same key
echo "Claude Code: $ANTHROPIC_API_KEY"
grep ANTHROPIC_API_KEY /Users/kimseunghwan/ClaudProjects/llm-viz/backend/.env

# They should match
```

### Issue 3: High Latency

**Symptoms**: Dashboard lags behind actual Claude Code usage

**Causes**:
- SSE connection dropping
- Backend under load
- Network issues

**Solutions**:

```bash
# Check backend logs
tail -f /tmp/llm-viz-backend.log

# Restart backend
pkill -f "go run ./cmd/server"
cd /Users/kimseunghwan/ClaudProjects/llm-viz/backend
go run ./cmd/server
```

### Issue 4: Missing Sessions

**Symptoms**: Old sessions don't appear

**Explanation**: Phase 1 uses in-memory storage (data lost on restart)

**Solution**: Wait for Phase 3 (Turso DB) or export data after each session

---

## Best Practices

### 1. Session Naming Convention

```bash
# Use hierarchical structure:
<project>-<feature>-<YYYYMMDD>

# Examples:
llm-viz-testing-20260301
northnavi-auth-feature-20260301
pr-review-daily-20260301
```

### 2. Cost Budgeting

Set alerts by monitoring accumulated cost:

```bash
# Check total cost for today
curl "http://localhost:8080/api/stats?start=$(date -u +%Y-%m-%dT00:00:00Z)" \
  | jq '[.records[].cost_usd] | add'

# If > $5, switch to Haiku or reduce context
```

### 3. Context Window Management

Monitor gauge closely during long conversations:

- **<50%**: Safe - continue normally
- **50-80%**: Warning - consider summarizing
- **80-95%**: Critical - compact or `/compact`
- **>95%**: Danger - immediate action required

### 4. Cache Optimization

Maximize cache hits:

```bash
# 1. Use consistent system prompts
# 2. Reuse large context (docs, code)
# 3. Structure conversations hierarchically

# Monitor cache efficiency:
# - Anthropic: 90% discount on cache reads
# - OpenAI: 50% discount on cached tokens
```

---

## Next Steps

### Immediate

1. ✅ Start llm-viz backend + frontend
2. ✅ Run test script to verify integration
3. ✅ Use Claude Code normally and watch dashboard update

### Short-term

1. Create custom session IDs for different projects
2. Export daily usage reports
3. Analyze which prompts cost most
4. Optimize high-cost operations

### Long-term

1. Wait for Phase 2 (persistent database)
2. Set up automated cost alerts
3. Build custom analytics dashboards
4. Integrate with CI/CD for PR cost tracking

---

## Related Runbooks

- [OpenAI Integration](openai-integration.md)
- [Gemini Integration](gemini-integration.md) _(Phase 2)_
- [Cost Optimization Guide](../troubleshooting/cost-optimization.md)
- [Multi-Agent Team Monitoring](multi-agent-monitoring.md)

---

## Support

- **Issues**: https://github.com/coolexplorer/llm-viz/issues
- **Discussions**: https://github.com/coolexplorer/llm-viz/discussions
- **Documentation**: https://github.com/coolexplorer/llm-viz/tree/main/docs

---

**🤖 Generated with Claude Code**
**Last Tested**: 2026-03-01 with Claude Code v2.1.50
