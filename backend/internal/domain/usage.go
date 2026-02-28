package domain

import "time"

// TokenUsage is the canonical normalized token structure across all providers.
// Fields not applicable to a provider are zero-valued.
type TokenUsage struct {
	// Core — present for all providers.
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	TotalTokens  int64 `json:"total_tokens"`

	// Cache — Anthropic, Gemini, OpenAI, OpenRouter.
	CacheReadTokens  int64 `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens int64 `json:"cache_write_tokens,omitempty"`

	// Reasoning — OpenAI o-series, Claude extended thinking.
	ReasoningTokens int64 `json:"reasoning_tokens,omitempty"`

	// Timing — Groq only (milliseconds).
	PromptTimeMs     float64 `json:"prompt_time_ms,omitempty"`
	CompletionTimeMs float64 `json:"completion_time_ms,omitempty"`
}

// CacheHitRate returns the cache efficiency ratio (0.0–1.0).
func (u TokenUsage) CacheHitRate() float64 {
	total := u.CacheReadTokens + u.CacheWriteTokens
	if total == 0 {
		return 0
	}
	return float64(u.CacheReadTokens) / float64(total)
}

// EffectiveContextUsed returns total context window tokens consumed.
func (u TokenUsage) EffectiveContextUsed() int64 {
	return u.InputTokens + u.CacheReadTokens + u.CacheWriteTokens
}

// NormalizedUsage is the persisted record stored after each LLM call.
type NormalizedUsage struct {
	ID         string     `json:"id"`
	Timestamp  time.Time  `json:"timestamp"`
	Provider   ProviderID `json:"provider"`
	Model      string     `json:"model"`
	SessionID  string     `json:"session_id"`
	APIKeyHash string     `json:"api_key_hash"` // SHA-256 prefix — never store raw key
	Usage      TokenUsage `json:"usage"`
	CostUSD    float64    `json:"cost_usd"`
	StreamMode bool       `json:"stream_mode"`
	ProjectTag string     `json:"project_tag,omitempty"`
}

// UsageSummary aggregates token counts for reporting.
type UsageSummary struct {
	Provider     ProviderID `json:"provider"`
	TotalInput   int64      `json:"total_input"`
	TotalOutput  int64      `json:"total_output"`
	TotalCost    float64    `json:"total_cost"`
	RequestCount int64      `json:"request_count"`
}

// ContextWindowStatus describes how full a model's context window is.
type ContextWindowStatus struct {
	Model             string
	MaxTokens         int64
	UsedTokens        int64
	UtilizationPct    float64
	WarningThreshold  bool // >80%
	CriticalThreshold bool // >95%
}

// NewContextWindowStatus computes context window utilization metrics.
func NewContextWindowStatus(used, max int64, model string) ContextWindowStatus {
	pct := float64(used) / float64(max) * 100
	return ContextWindowStatus{
		Model:             model,
		MaxTokens:         max,
		UsedTokens:        used,
		UtilizationPct:    pct,
		WarningThreshold:  pct >= 80,
		CriticalThreshold: pct >= 95,
	}
}

// UsageEvent is published to the SSE broadcaster after each LLM call.
type UsageEvent struct {
	SessionID string          `json:"session_id"`
	Usage     NormalizedUsage `json:"usage"`
}
