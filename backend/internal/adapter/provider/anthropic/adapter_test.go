package anthropic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	sdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/testutil"
)

// newTestAdapter builds an Adapter pointed at the given test server URL.
// Being in package anthropic, we can set unexported fields directly.
func newTestAdapter(t *testing.T, handler http.HandlerFunc) (*Adapter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	a := &Adapter{
		client: sdk.NewClient(option.WithAPIKey("test-key"), option.WithBaseURL(srv.URL)),
		models: supportedModels(),
	}
	return a, srv
}

// ---------------------------------------------------------------------------
// convertMessages
// ---------------------------------------------------------------------------

func TestConvertMessages_UserAndAssistant(t *testing.T) {
	msgs := []domain.Message{
		{Role: "system", Content: "Be helpful"}, // excluded — handled separately
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi"},
	}
	result := convertMessages(msgs)
	// system is excluded; user + assistant = 2
	assert.Len(t, result, 2)
}

func TestConvertMessages_OnlySystem(t *testing.T) {
	msgs := []domain.Message{
		{Role: "system", Content: "Only system"},
	}
	result := convertMessages(msgs)
	assert.Empty(t, result) // system messages are not included
}

func TestConvertMessages_UnknownRoleIgnored(t *testing.T) {
	msgs := []domain.Message{
		{Role: "user", Content: "Hello"},
		{Role: "tool", Content: "tool result"},
		{Role: "function", Content: "fn result"},
	}
	result := convertMessages(msgs)
	assert.Len(t, result, 1) // only user
}

func TestConvertMessages_Empty(t *testing.T) {
	result := convertMessages(nil)
	assert.Empty(t, result)
}

// ---------------------------------------------------------------------------
// extractSystem
// ---------------------------------------------------------------------------

func TestExtractSystem_Found(t *testing.T) {
	msgs := []domain.Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Hello"},
	}
	sys := extractSystem(msgs)
	assert.Equal(t, "You are a helpful assistant", sys)
}

func TestExtractSystem_NotFound(t *testing.T) {
	msgs := []domain.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi"},
	}
	sys := extractSystem(msgs)
	assert.Equal(t, "", sys)
}

func TestExtractSystem_ReturnsFirst(t *testing.T) {
	msgs := []domain.Message{
		{Role: "system", Content: "First system"},
		{Role: "system", Content: "Second system"},
	}
	sys := extractSystem(msgs)
	assert.Equal(t, "First system", sys) // only first is returned
}

func TestExtractSystem_Empty(t *testing.T) {
	sys := extractSystem(nil)
	assert.Equal(t, "", sys)
}

// ---------------------------------------------------------------------------
// ProviderID / SupportedModels
// ---------------------------------------------------------------------------

func TestProviderID(t *testing.T) {
	a := &Adapter{}
	assert.Equal(t, domain.ProviderAnthropic, a.ProviderID())
}

func TestSupportedModels(t *testing.T) {
	a := &Adapter{models: supportedModels()}
	models := a.SupportedModels()
	require.NotEmpty(t, models)
	for _, m := range models {
		assert.Equal(t, domain.ProviderAnthropic, m.Provider)
		assert.NotEmpty(t, m.ID)
		assert.Greater(t, m.ContextWindow, int64(0))
	}
}

func TestSupportedModels_ContainsExpectedModels(t *testing.T) {
	models := supportedModels()
	ids := make(map[string]bool)
	for _, m := range models {
		ids[m.ID] = true
	}
	assert.True(t, ids["claude-opus-4-6"])
	assert.True(t, ids["claude-sonnet-4-6"])
	assert.True(t, ids["claude-haiku-4-5"])
}

// ---------------------------------------------------------------------------
// Complete — success path
// ---------------------------------------------------------------------------

func TestComplete_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 200, testutil.AnthropicSuccessResponse(
			"msg-123", "Hello!", 100, 50, 20, 10,
		))
	})
	a, _ := newTestAdapter(t, handler)

	result, err := a.Complete(context.Background(), domain.CompletionRequest{
		Model:    "claude-opus-4-6",
		Messages: []domain.Message{{Role: "user", Content: "Hi"}},
	})

	require.NoError(t, err)
	assert.Equal(t, "msg-123", result.ID)
	assert.Equal(t, "Hello!", result.Content)
	assert.Equal(t, int64(100), result.Usage.InputTokens)
	assert.Equal(t, int64(50), result.Usage.OutputTokens)
	assert.Equal(t, int64(150), result.Usage.TotalTokens) // computed
	assert.Equal(t, int64(20), result.Usage.CacheReadTokens)
	assert.Equal(t, int64(10), result.Usage.CacheWriteTokens)
}

func TestComplete_WithSystemMessage(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 200, testutil.AnthropicSuccessResponse(
			"msg-sys", "Got it", 50, 20, 0, 0,
		))
	})
	a, _ := newTestAdapter(t, handler)

	result, err := a.Complete(context.Background(), domain.CompletionRequest{
		Model: "claude-opus-4-6",
		Messages: []domain.Message{
			{Role: "system", Content: "You are a helpful assistant"},
			{Role: "user", Content: "What time is it?"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "Got it", result.Content)
}

func TestComplete_DefaultMaxTokens(t *testing.T) {
	// MaxTokens <= 0 should default to 1024 in the adapter.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 200, testutil.AnthropicSuccessResponse(
			"msg-default", "Default!", 10, 5, 0, 0,
		))
	})
	a, _ := newTestAdapter(t, handler)

	result, err := a.Complete(context.Background(), domain.CompletionRequest{
		Model:     "claude-opus-4-6",
		Messages:  []domain.Message{{Role: "user", Content: "Hi"}},
		MaxTokens: 0, // should default to 1024
	})

	require.NoError(t, err)
	assert.Equal(t, "Default!", result.Content)
}

func TestComplete_MultipleTextBlocks(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 200, map[string]any{
			"id":    "msg-multi",
			"type":  "message",
			"role":  "assistant",
			"model": "claude-opus-4-6",
			"content": []map[string]any{
				{"type": "text", "text": "Hello "},
				{"type": "text", "text": "World!"},
			},
			"stop_reason": "end_turn",
			"usage": map[string]any{
				"input_tokens":                int64(10),
				"output_tokens":               int64(5),
				"cache_creation_input_tokens": int64(0),
				"cache_read_input_tokens":     int64(0),
			},
		})
	})
	a, _ := newTestAdapter(t, handler)

	result, err := a.Complete(context.Background(), domain.CompletionRequest{
		Model:    "claude-opus-4-6",
		Messages: []domain.Message{{Role: "user", Content: "Hi"}},
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello World!", result.Content) // concatenated
}

// ---------------------------------------------------------------------------
// mapError — tested via Complete with mock HTTP errors
// ---------------------------------------------------------------------------

func TestComplete_MapError_Unauthorized(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 401, testutil.AnthropicErrorResponse("authentication_error", "Invalid API key"))
	})
	a, _ := newTestAdapter(t, handler)

	_, err := a.Complete(context.Background(), domain.CompletionRequest{
		Model:    "claude-opus-4-6",
		Messages: []domain.Message{{Role: "user", Content: "Hi"}},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidAPIKey)
}

func TestComplete_MapError_RateLimited(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 429, testutil.AnthropicErrorResponse("rate_limit_error", "Rate limit exceeded"))
	})
	a, _ := newTestAdapter(t, handler)

	_, err := a.Complete(context.Background(), domain.CompletionRequest{
		Model:    "claude-opus-4-6",
		Messages: []domain.Message{{Role: "user", Content: "Hi"}},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRateLimited)
}

func TestComplete_MapError_ServiceUnavailable(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 503, testutil.AnthropicErrorResponse("overloaded_error", "Service overloaded"))
	})
	a, _ := newTestAdapter(t, handler)

	_, err := a.Complete(context.Background(), domain.CompletionRequest{
		Model:    "claude-opus-4-6",
		Messages: []domain.Message{{Role: "user", Content: "Hi"}},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrProviderUnavailable)
}

func TestComplete_MapError_BadRequest(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 400, testutil.AnthropicErrorResponse("invalid_request_error", "Bad request"))
	})
	a, _ := newTestAdapter(t, handler)

	_, err := a.Complete(context.Background(), domain.CompletionRequest{
		Model:    "claude-opus-4-6",
		Messages: []domain.Message{{Role: "user", Content: "Hi"}},
	})

	require.Error(t, err)
	// 400 is wrapped as "anthropic bad request" — not a domain sentinel
	assert.NotErrorIs(t, err, domain.ErrInvalidAPIKey)
	assert.NotErrorIs(t, err, domain.ErrRateLimited)
}

func TestComplete_MapError_InternalServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 500, testutil.AnthropicErrorResponse("api_error", "Server error"))
	})
	a, _ := newTestAdapter(t, handler)

	_, err := a.Complete(context.Background(), domain.CompletionRequest{
		Model:    "claude-opus-4-6",
		Messages: []domain.Message{{Role: "user", Content: "Hi"}},
	})

	require.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrInvalidAPIKey)
	assert.NotErrorIs(t, err, domain.ErrRateLimited)
}

// ---------------------------------------------------------------------------
// Stream
// ---------------------------------------------------------------------------

func TestStream_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 200, testutil.AnthropicSuccessResponse(
			"msg-stream", "Streamed!", 80, 30, 5, 2,
		))
	})
	a, _ := newTestAdapter(t, handler)

	var chunks []domain.StreamChunk
	err := a.Stream(context.Background(), domain.CompletionRequest{
		Model:    "claude-opus-4-6",
		Messages: []domain.Message{{Role: "user", Content: "Hi"}},
	}, func(chunk domain.StreamChunk) error {
		chunks = append(chunks, chunk)
		return nil
	})

	require.NoError(t, err)
	require.Len(t, chunks, 1)
	assert.True(t, chunks[0].IsFinal)
	assert.Equal(t, "Streamed!", chunks[0].Delta)
	require.NotNil(t, chunks[0].Usage)
	assert.Equal(t, int64(80), chunks[0].Usage.InputTokens)
	assert.Equal(t, int64(5), chunks[0].Usage.CacheReadTokens)
}

func TestStream_PropagatesError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 429, testutil.AnthropicErrorResponse("rate_limit_error", "Slow down"))
	})
	a, _ := newTestAdapter(t, handler)

	err := a.Stream(context.Background(), domain.CompletionRequest{
		Model:    "claude-opus-4-6",
		Messages: []domain.Message{{Role: "user", Content: "Hi"}},
	}, func(_ domain.StreamChunk) error { return nil })

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRateLimited)
}
