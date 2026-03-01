package domain

// ProviderID is the canonical identifier for an LLM provider.
type ProviderID string

const (
	ProviderAnthropic  ProviderID = "anthropic"
	ProviderOpenAI     ProviderID = "openai"
	ProviderGemini     ProviderID = "gemini"
	ProviderMistral    ProviderID = "mistral"
	ProviderGroq       ProviderID = "groq"
	ProviderOpenRouter ProviderID = "openrouter"
)

// ModelInfo describes a model available from a provider.
type ModelInfo struct {
	ID              string     `json:"id"`
	DisplayName     string     `json:"display_name"`
	Provider        ProviderID `json:"provider"`
	ContextWindow   int64      `json:"context_window"`
	MaxOutputTokens int64      `json:"max_output_tokens"`
}

// Message is a single chat turn.
type Message struct {
	Role    string `json:"role"`    // "user" | "assistant" | "system"
	Content string `json:"content"`
}

// CompletionRequest is the normalized request sent to any provider adapter.
type CompletionRequest struct {
	Model      string
	Messages   []Message
	MaxTokens  int
	Stream     bool
	SessionID  string
	ProjectTag string
}

// CompletionResult holds the response from a non-streaming completion call.
type CompletionResult struct {
	ID      string
	Content string
	Usage   TokenUsage
}

// StreamChunk is a single incremental chunk from a streaming completion.
type StreamChunk struct {
	Delta   string     // incremental text content
	IsFinal bool       // true for the last chunk
	Usage   *TokenUsage // non-nil only in the final chunk
}
