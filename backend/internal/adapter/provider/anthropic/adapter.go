package anthropic

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/port"
)

// Adapter wraps the Anthropic SDK to implement port.LLMProvider.
type Adapter struct {
	client sdk.Client // value type, not pointer
	models []domain.ModelInfo
}

// New creates an Anthropic adapter with the provided API key.
func New(apiKey string) *Adapter {
	return &Adapter{
		client: sdk.NewClient(option.WithAPIKey(apiKey)),
		models: supportedModels(),
	}
}

func (a *Adapter) ProviderID() domain.ProviderID      { return domain.ProviderAnthropic }
func (a *Adapter) SupportedModels() []domain.ModelInfo { return a.models }

// Complete sends a non-streaming completion request to Anthropic.
func (a *Adapter) Complete(ctx context.Context, req domain.CompletionRequest) (*domain.CompletionResult, error) {
	maxTokens := int64(req.MaxTokens)
	if maxTokens <= 0 {
		maxTokens = 1024
	}

	params := sdk.MessageNewParams{
		Model:     sdk.Model(req.Model),
		MaxTokens: maxTokens,
		Messages:  convertMessages(req.Messages),
	}

	// Extract system message and pass separately (Anthropic API requirement).
	if sys := extractSystem(req.Messages); sys != "" {
		params.System = []sdk.TextBlockParam{{Text: sys}}
	}

	resp, err := a.client.Messages.New(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	content := ""
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &domain.CompletionResult{
		ID:      resp.ID,
		Content: content,
		Usage: domain.TokenUsage{
			InputTokens:      resp.Usage.InputTokens,
			OutputTokens:     resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
			CacheWriteTokens: resp.Usage.CacheCreationInputTokens,
			CacheReadTokens:  resp.Usage.CacheReadInputTokens,
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

// mapError translates Anthropic SDK errors to domain-level sentinel errors.
func mapError(err error) error {
	var apiErr *sdk.Error
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 401:
			return fmt.Errorf("%w: anthropic", domain.ErrInvalidAPIKey)
		case 429:
			return fmt.Errorf("%w: anthropic", domain.ErrRateLimited)
		case 400:
			return fmt.Errorf("anthropic bad request: %w", err)
		case 503:
			return fmt.Errorf("%w: anthropic", domain.ErrProviderUnavailable)
		}
	}
	return fmt.Errorf("anthropic: %w", err)
}

// convertMessages converts domain messages to Anthropic SDK message params.
// System messages are excluded here — use extractSystem instead.
func convertMessages(msgs []domain.Message) []sdk.MessageParam {
	out := make([]sdk.MessageParam, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "user":
			out = append(out, sdk.NewUserMessage(sdk.NewTextBlock(m.Content)))
		case "assistant":
			out = append(out, sdk.NewAssistantMessage(sdk.NewTextBlock(m.Content)))
		}
	}
	return out
}

// extractSystem returns the content of the first system message, or empty string.
func extractSystem(msgs []domain.Message) string {
	for _, m := range msgs {
		if m.Role == "system" {
			return m.Content
		}
	}
	return ""
}

func supportedModels() []domain.ModelInfo {
	return []domain.ModelInfo{
		{
			ID:              "claude-opus-4-6",
			DisplayName:     "Claude Opus 4.6",
			Provider:        domain.ProviderAnthropic,
			ContextWindow:   200_000,
			MaxOutputTokens: 128_000,
		},
		{
			ID:              "claude-sonnet-4-6",
			DisplayName:     "Claude Sonnet 4.6",
			Provider:        domain.ProviderAnthropic,
			ContextWindow:   200_000,
			MaxOutputTokens: 64_000,
		},
		{
			ID:              "claude-haiku-4-5",
			DisplayName:     "Claude Haiku 4.5",
			Provider:        domain.ProviderAnthropic,
			ContextWindow:   200_000,
			MaxOutputTokens: 64_000,
		},
	}
}
