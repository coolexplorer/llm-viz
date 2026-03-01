# Multi-Agent Team Monitoring Guide

**Track token consumption across Claude Code agent teams**

---

## Overview

When using Claude Code's agent teams feature, llm-viz helps you:

- Track total team token consumption
- Identify which agents consume the most tokens
- Monitor costs per task/project
- Optimize agent delegation strategies

---

## Session ID Naming Strategy

Use hierarchical session IDs to organize multi-agent data:

```bash
# Format: <team-name>-<task>-<YYYYMMDD>

# Examples:
test-coverage-80-20260301        # Team for test coverage task
mvp-phase1-development-20260301   # MVP development team
pr-review-process-20260301        # PR review team
```

---

## Monitoring Scenarios

### Scenario 1: Test Coverage Team

You have a team with:
- Frontend Test Engineer
- Backend Test Engineer
- Verification Engineer

**Setup**:

```bash
# Before starting team
export CLAUDE_SESSION_ID="test-coverage-80-$(date +%Y%m%d)"

# In Claude Code
TeamCreate("test-coverage-80")
# Spawn engineers...
```

**Dashboard View**:

```
Session: test-coverage-80-20260301

┌─────────────────────────────────────────┐
│ Total Team Tokens                        │
│ Input:  245,892  Cache Read:  102,445   │
│ Output: 156,234  Cache Write:  45,123   │
│ Cost:   $42.56                          │
└─────────────────────────────────────────┘

Timeline:
10:00  Frontend Engineer spawn  →  +45K tokens  $6.23
10:15  Backend Engineer spawn   →  +67K tokens  $9.45
10:45  Verification             →  +12K tokens  $1.67
11:00  Frontend complete        →  +89K tokens  $12.78
```

### Scenario 2: Feature Development Team

**Setup**:

```bash
export CLAUDE_SESSION_ID="feature-auth-$(date +%Y%m%d)"

# Spawn:
# - Frontend Engineer (UI components)
# - Backend Engineer (API endpoints)
# - Test Engineer (E2E tests)
```

**Cost Breakdown**:

```
Feature: Authentication System
Total Cost: $87.23

Frontend (UI):    $32.45  (37%)
Backend (API):    $41.23  (47%)  ← Most expensive
Testing (E2E):    $13.55  (16%)

Recommendation: Backend engineer used many retries.
Consider breaking down tasks into smaller pieces.
```

---

## Best Practices

### 1. Pre-Task Cost Estimation

Before spawning agents, estimate cost:

```bash
# Average costs (Phase 1, Sonnet 4.6):
# - Frontend task:  $15-30
# - Backend task:   $20-40
# - Test task:      $10-20
# - Review task:    $5-10

# For a 3-agent team:
# Expected: $45-90
# Budget:   $100 (add 10% buffer)
```

### 2. Real-time Budget Tracking

Monitor during execution:

```bash
# Check accumulated cost
curl "http://localhost:8080/api/stats?session_id=test-coverage-80-20260301" \
  | jq '.records | map(.usage.cost_usd) | add'

# If approaching budget, consider:
# - Switching to Haiku for verification tasks
# - Reducing context in prompts
# - Pausing non-critical agents
```

### 3. Post-Task Analysis

After team completes:

```bash
# Generate cost report
curl "http://localhost:8080/api/stats?session_id=test-coverage-80-20260301&limit=1000" \
  | jq '{
      total_input: [.records[].usage.usage.input_tokens] | add,
      total_output: [.records[].usage.usage.output_tokens] | add,
      total_cost: [.records[].usage.cost_usd] | add,
      request_count: (.records | length)
    }'

# Output:
# {
#   "total_input": 245892,
#   "total_output": 156234,
#   "total_cost": 42.56,
#   "request_count": 47
# }
```

### 4. Agent Efficiency Comparison

Compare different agent approaches:

```bash
# Approach A: Monolithic (single agent)
Session: feature-monolithic-20260301
Cost: $75.23
Time: 45min

# Approach B: Team of 3 (parallel agents)
Session: feature-team-20260301
Cost: $87.45 (16% more expensive)
Time: 18min (60% faster)

# ROI: Parallel worth it for urgent features
```

---

## Integration with Claude Code Workflow

### Method 1: Automatic Session Naming

Create a wrapper that auto-generates session IDs:

```bash
# ~/bin/claude-team
#!/bin/bash

TEAM_NAME="$1"
shift  # Remove first arg

SESSION_ID="${TEAM_NAME}-$(date +%Y%m%d-%H%M)"
export CLAUDE_SESSION_ID="$SESSION_ID"

echo "🎯 Team session: $SESSION_ID"
echo "📊 Dashboard: http://localhost:3000?session=$SESSION_ID"

claude "$@"
```

Usage:

```bash
claude-team test-coverage
# Session auto-set to: test-coverage-20260301-1045
```

### Method 2: Task-based Sessions

```bash
# In MEMORY.md or task list:
# Task #1: Frontend tests - Session: test-coverage-80-frontend
# Task #2: Backend tests  - Session: test-coverage-80-backend

# Spawn with explicit session:
Task(
  name="frontend-test-engineer",
  session_id="test-coverage-80-frontend",
  ...
)
```

---

## Advanced: Cross-Team Analysis

Compare costs across different teams:

```bash
# All test coverage teams this month
curl "http://localhost:8080/api/stats?start=2026-03-01T00:00:00Z" \
  | jq '.records[] | select(.usage.session_id | startswith("test-coverage"))' \
  | jq -s 'group_by(.usage.session_id) | map({
      session: .[0].usage.session_id,
      cost: map(.usage.cost_usd) | add
    })' \
  | jq 'sort_by(.cost) | reverse'

# Output:
# [
#   { "session": "test-coverage-80-20260301", "cost": 42.56 },
#   { "session": "test-coverage-phase2-20260228", "cost": 38.12 },
#   ...
# ]
```

---

## Optimization Tips

### Reduce Redundant Context

**Before** (wasteful):

```
Each agent reads full codebase independently:
- Frontend: 200K tokens input
- Backend:  200K tokens input
- Test:     200K tokens input
Total: 600K tokens = $9.00
```

**After** (optimized):

```
Main Assistant reads codebase once (cached):
- Main reads: 200K tokens
- Frontend refs cache: 200K cache_read (90% discount)
- Backend refs cache:  200K cache_read (90% discount)
- Test refs cache:     200K cache_read (90% discount)
Total cost: ~$2.00 (78% savings)
```

### Use Haiku for Simple Tasks

```bash
# Verification tasks don't need Sonnet:
# Sonnet: $15/M input, $75/M output
# Haiku:  $0.80/M input, $4/M output

# For 50K verification task:
# Sonnet: $3.75
# Haiku:  $0.20
# Savings: $3.55 per verification (95% cheaper)
```

---

## Related Runbooks

- [Claude Code Integration](claude-code-integration.md)
- [Cost Optimization](../troubleshooting/cost-optimization.md)
- [Environment Configuration](../setup/environment-config.md)

---

**Last Updated**: 2026-03-01
