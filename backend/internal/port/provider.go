package port

import (
	"context"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
)

// StreamHandler is called for each streaming chunk from a provider.
// Return an error to abort the stream.
type StreamHandler func(chunk domain.StreamChunk) error

// LLMProvider is the outbound port for any LLM provider.
// Each provider adapter must implement this interface.
type LLMProvider interface {
	// Complete sends a non-streaming completion request.
	Complete(ctx context.Context, req domain.CompletionRequest) (*domain.CompletionResult, error)

	// Stream sends a streaming completion request, invoking handler for each chunk.
	// The final chunk carries Usage; earlier chunks have nil Usage.
	Stream(ctx context.Context, req domain.CompletionRequest, handler StreamHandler) error

	// ProviderID returns the canonical identifier for this provider.
	ProviderID() domain.ProviderID

	// SupportedModels returns the list of models available from this provider.
	SupportedModels() []domain.ModelInfo
}
