package openai

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/port"
)

// Adapter wraps the OpenAI SDK to implement port.LLMProvider.
type Adapter struct {
	client sdk.Client // value type, not pointer
	models []domain.ModelInfo
}

// New creates an OpenAI adapter with the provided API key.
func New(apiKey string) *Adapter {
	return &Adapter{
		client: sdk.NewClient(option.WithAPIKey(apiKey)),
		models: supportedModels(),
	}
}

func (a *Adapter) ProviderID() domain.ProviderID      { return domain.ProviderOpenAI }
func (a *Adapter) SupportedModels() []domain.ModelInfo { return a.models }

// Complete sends a non-streaming completion request to OpenAI.
func (a *Adapter) Complete(ctx context.Context, req domain.CompletionRequest) (*domain.CompletionResult, error) {
	params := sdk.ChatCompletionNewParams{
		Model:    req.Model,
		Messages: convertMessages(req.Messages),
	}
	if req.MaxTokens > 0 {
		params.MaxTokens = sdk.Int(int64(req.MaxTokens))
	}

	resp, err := a.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	content := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
	}

	return &domain.CompletionResult{
		ID:      resp.ID,
		Content: content,
		Usage: domain.TokenUsage{
			InputTokens:     resp.Usage.PromptTokens,
			OutputTokens:    resp.Usage.CompletionTokens,
			TotalTokens:     resp.Usage.TotalTokens,
			CacheReadTokens: resp.Usage.PromptTokensDetails.CachedTokens,
			ReasoningTokens: resp.Usage.CompletionTokensDetails.ReasoningTokens,
		},
	}, nil
}

// Stream calls Complete and emits a single final chunk (Phase 1 implementation).
func (a *Adapter) Stream(ctx context.Context, req domain.CompletionRequest, handler port.StreamHandler) error {
	result, err := a.Complete(ctx, req)
	if err != nil {
		return err
	}
	return handler(domain.StreamChunk{
		Delta:   result.Content,
		IsFinal: true,
		Usage:   &result.Usage,
	})
}

// mapError translates OpenAI SDK errors to domain-level sentinel errors.
func mapError(err error) error {
	var apiErr *sdk.Error
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 401:
			return fmt.Errorf("%w: openai", domain.ErrInvalidAPIKey)
		case 429:
			return fmt.Errorf("%w: openai", domain.ErrRateLimited)
		case 503:
			return fmt.Errorf("%w: openai", domain.ErrProviderUnavailable)
		}
	}
	return fmt.Errorf("openai: %w", err)
}

// convertMessages converts domain messages to OpenAI SDK message params.
func convertMessages(msgs []domain.Message) []sdk.ChatCompletionMessageParamUnion {
	out := make([]sdk.ChatCompletionMessageParamUnion, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "user":
			out = append(out, sdk.UserMessage(m.Content))
		case "assistant":
			out = append(out, sdk.AssistantMessage(m.Content))
		case "system":
			out = append(out, sdk.SystemMessage(m.Content))
		}
	}
	return out
}

func supportedModels() []domain.ModelInfo {
	return []domain.ModelInfo{
		{
			ID:              "gpt-4o",
			DisplayName:     "GPT-4o",
			Provider:        domain.ProviderOpenAI,
			ContextWindow:   128_000,
			MaxOutputTokens: 16_384,
		},
		{
			ID:              "gpt-4o-mini",
			DisplayName:     "GPT-4o mini",
			Provider:        domain.ProviderOpenAI,
			ContextWindow:   128_000,
			MaxOutputTokens: 16_384,
		},
		{
			ID:              "o1",
			DisplayName:     "OpenAI o1",
			Provider:        domain.ProviderOpenAI,
			ContextWindow:   200_000,
			MaxOutputTokens: 100_000,
		},
		{
			ID:              "o3-mini",
			DisplayName:     "OpenAI o3-mini",
			Provider:        domain.ProviderOpenAI,
			ContextWindow:   200_000,
			MaxOutputTokens: 100_000,
		},
	}
}
