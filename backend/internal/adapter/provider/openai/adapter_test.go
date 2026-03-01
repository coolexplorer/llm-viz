package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	sdk "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/testutil"
)

// newTestAdapter builds an Adapter pointed at the given test server URL.
// Being in package openai, we can set the unexported client field directly.
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

func TestConvertMessages_RolesMapping(t *testing.T) {
	msgs := []domain.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi"},
		{Role: "system", Content: "You are helpful"},
	}
	result := convertMessages(msgs)
	assert.Len(t, result, 3)
}

func TestConvertMessages_UnknownRoleIgnored(t *testing.T) {
	msgs := []domain.Message{
		{Role: "user", Content: "Hello"},
		{Role: "unknown", Content: "ignored"},
		{Role: "tool", Content: "also ignored"},
	}
	result := convertMessages(msgs)
	assert.Len(t, result, 1) // only the user message
}

func TestConvertMessages_Empty(t *testing.T) {
	result := convertMessages(nil)
	assert.Empty(t, result)
}

func TestConvertMessages_OnlySystem(t *testing.T) {
	msgs := []domain.Message{{Role: "system", Content: "System prompt"}}
	result := convertMessages(msgs)
	assert.Len(t, result, 1)
}

// ---------------------------------------------------------------------------
// supportedModels / ProviderID / SupportedModels
// ---------------------------------------------------------------------------

func TestProviderID(t *testing.T) {
	a := &Adapter{}
	assert.Equal(t, domain.ProviderOpenAI, a.ProviderID())
}

func TestSupportedModels(t *testing.T) {
	a := &Adapter{models: supportedModels()}
	models := a.SupportedModels()
	require.NotEmpty(t, models)
	for _, m := range models {
		assert.Equal(t, domain.ProviderOpenAI, m.Provider)
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
	assert.True(t, ids["gpt-4o"])
	assert.True(t, ids["gpt-4o-mini"])
}

// ---------------------------------------------------------------------------
// Complete — success path
// ---------------------------------------------------------------------------

func TestComplete_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 200, testutil.OpenAISuccessResponse("cmpl-123", "Hello!", 100, 50, 20))
	})
	a, _ := newTestAdapter(t, handler)

	result, err := a.Complete(context.Background(), domain.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []domain.Message{{Role: "user", Content: "Hi"}},
	})

	require.NoError(t, err)
	assert.Equal(t, "cmpl-123", result.ID)
	assert.Equal(t, "Hello!", result.Content)
	assert.Equal(t, int64(100), result.Usage.InputTokens)
	assert.Equal(t, int64(50), result.Usage.OutputTokens)
	assert.Equal(t, int64(150), result.Usage.TotalTokens)
	assert.Equal(t, int64(20), result.Usage.CacheReadTokens)
}

func TestComplete_WithMaxTokens(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 200, testutil.OpenAISuccessResponse("cmpl-456", "Bounded!", 10, 5, 0))
	})
	a, _ := newTestAdapter(t, handler)

	result, err := a.Complete(context.Background(), domain.CompletionRequest{
		Model:     "gpt-4o",
		Messages:  []domain.Message{{Role: "user", Content: "Hi"}},
		MaxTokens: 512,
	})

	require.NoError(t, err)
	assert.Equal(t, "cmpl-456", result.ID)
	assert.Equal(t, "Bounded!", result.Content)
}

func TestComplete_EmptyChoices(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 200, map[string]any{
			"id":      "cmpl-empty",
			"object":  "chat.completion",
			"choices": []any{},
			"usage": map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 0,
				"total_tokens":      10,
			},
		})
	})
	a, _ := newTestAdapter(t, handler)

	result, err := a.Complete(context.Background(), domain.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []domain.Message{{Role: "user", Content: "Hi"}},
	})

	require.NoError(t, err)
	assert.Equal(t, "", result.Content) // empty when no choices
}

// ---------------------------------------------------------------------------
// mapError — tested via Complete with mock HTTP errors
// ---------------------------------------------------------------------------

func TestComplete_MapError_Unauthorized(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 401, testutil.OpenAIErrorResponse("authentication_error", "Invalid API key"))
	})
	a, _ := newTestAdapter(t, handler)

	_, err := a.Complete(context.Background(), domain.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []domain.Message{{Role: "user", Content: "Hi"}},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidAPIKey)
}

func TestComplete_MapError_RateLimited(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 429, testutil.OpenAIErrorResponse("rate_limit_error", "Rate limit exceeded"))
	})
	a, _ := newTestAdapter(t, handler)

	_, err := a.Complete(context.Background(), domain.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []domain.Message{{Role: "user", Content: "Hi"}},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRateLimited)
}

func TestComplete_MapError_ServiceUnavailable(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 503, testutil.OpenAIErrorResponse("service_unavailable", "Service temporarily unavailable"))
	})
	a, _ := newTestAdapter(t, handler)

	_, err := a.Complete(context.Background(), domain.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []domain.Message{{Role: "user", Content: "Hi"}},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrProviderUnavailable)
}

func TestComplete_MapError_InternalServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 500, testutil.OpenAIErrorResponse("internal_server_error", "Server error"))
	})
	a, _ := newTestAdapter(t, handler)

	_, err := a.Complete(context.Background(), domain.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []domain.Message{{Role: "user", Content: "Hi"}},
	})

	require.Error(t, err)
	// 500 is not a domain sentinel, wrapped as generic openai error
	assert.NotErrorIs(t, err, domain.ErrInvalidAPIKey)
	assert.NotErrorIs(t, err, domain.ErrRateLimited)
	assert.NotErrorIs(t, err, domain.ErrProviderUnavailable)
}

// ---------------------------------------------------------------------------
// Stream
// ---------------------------------------------------------------------------

func TestStream_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 200, testutil.OpenAISuccessResponse("cmpl-789", "Stream!", 100, 50, 0))
	})
	a, _ := newTestAdapter(t, handler)

	var chunks []domain.StreamChunk
	err := a.Stream(context.Background(), domain.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []domain.Message{{Role: "user", Content: "Hi"}},
	}, func(chunk domain.StreamChunk) error {
		chunks = append(chunks, chunk)
		return nil
	})

	require.NoError(t, err)
	require.Len(t, chunks, 1)
	assert.True(t, chunks[0].IsFinal)
	assert.Equal(t, "Stream!", chunks[0].Delta)
	require.NotNil(t, chunks[0].Usage)
	assert.Equal(t, int64(100), chunks[0].Usage.InputTokens)
}

func TestStream_PropagatesError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.RespondJSON(w, 401, testutil.OpenAIErrorResponse("authentication_error", "Invalid key"))
	})
	a, _ := newTestAdapter(t, handler)

	err := a.Stream(context.Background(), domain.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []domain.Message{{Role: "user", Content: "Hi"}},
	}, func(_ domain.StreamChunk) error { return nil })

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidAPIKey)
}
