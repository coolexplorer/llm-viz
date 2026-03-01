// Package testutil provides shared HTTP mock helpers and JSON fixture builders
// for use across backend test packages.
package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// NewMockServer creates a test HTTP server that responds with the given handler.
// The server is automatically closed when the test completes.
func NewMockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// RespondJSON writes a JSON-encoded body with the given status code.
func RespondJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// OpenAISuccessResponse builds a minimal OpenAI chat completion response body.
func OpenAISuccessResponse(id, content string, promptTokens, completionTokens, cachedTokens int64) map[string]any {
	return map[string]any{
		"id":     id,
		"object": "chat.completion",
		"model":  "gpt-4o",
		"choices": []map[string]any{
			{
				"index":         0,
				"message":       map[string]any{"role": "assistant", "content": content},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
			"total_tokens":      promptTokens + completionTokens,
			"prompt_tokens_details": map[string]any{
				"cached_tokens": cachedTokens,
				"audio_tokens":  0,
			},
			"completion_tokens_details": map[string]any{
				"reasoning_tokens": 0,
			},
		},
	}
}

// OpenAIErrorResponse builds an OpenAI-format error response body.
func OpenAIErrorResponse(errType, message string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"type":    errType,
			"message": message,
			"code":    errType,
		},
	}
}

// AnthropicSuccessResponse builds a minimal Anthropic messages response body.
func AnthropicSuccessResponse(id, content string, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens int64) map[string]any {
	return map[string]any{
		"id":    id,
		"type":  "message",
		"role":  "assistant",
		"model": "claude-opus-4-6",
		"content": []map[string]any{
			{"type": "text", "text": content},
		},
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":                inputTokens,
			"output_tokens":               outputTokens,
			"cache_creation_input_tokens": cacheWriteTokens,
			"cache_read_input_tokens":     cacheReadTokens,
		},
	}
}

// AnthropicErrorResponse builds an Anthropic-format error response body.
func AnthropicErrorResponse(errType, message string) map[string]any {
	return map[string]any{
		"type": "error",
		"error": map[string]any{
			"type":    errType,
			"message": message,
		},
	}
}
