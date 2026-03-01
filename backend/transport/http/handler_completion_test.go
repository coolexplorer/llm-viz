package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
)

// ---------------------------------------------------------------------------
// handleCompletion — validation
// ---------------------------------------------------------------------------

func TestHandleCompletion_MethodNotAllowed(t *testing.T) {
	srv, _, _ := defaultTestServer()
	for _, method := range []string{"GET", "PUT", "DELETE", "PATCH"} {
		t.Run(method, func(t *testing.T) {
			rr := callHandler(srv, srv.handleCompletion, method, "/api/complete", "")
			assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
		})
	}
}

func TestHandleCompletion_InvalidJSON(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := callHandler(srv, srv.handleCompletion, "POST", "/api/complete", `{invalid json`)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid JSON body")
}

func TestHandleCompletion_MissingProvider(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := callHandler(srv, srv.handleCompletion, "POST", "/api/complete", `{
		"model": "gpt-4o",
		"messages": [{"role": "user", "content": "hi"}]
	}`)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "provider is required")
}

func TestHandleCompletion_MissingModel(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := callHandler(srv, srv.handleCompletion, "POST", "/api/complete", `{
		"provider": "openai",
		"messages": [{"role": "user", "content": "hi"}]
	}`)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "model is required")
}

func TestHandleCompletion_MissingMessages(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := callHandler(srv, srv.handleCompletion, "POST", "/api/complete", `{
		"provider": "openai",
		"model": "gpt-4o",
		"messages": []
	}`)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "messages is required")
}

// ---------------------------------------------------------------------------
// handleCompletion — domain error → HTTP status mapping
// ---------------------------------------------------------------------------

func TestHandleCompletion_UnknownProvider(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := callHandler(srv, srv.handleCompletion, "POST", "/api/complete", `{
		"provider": "nonexistent",
		"model": "gpt-4o",
		"messages": [{"role": "user", "content": "hi"}]
	}`)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleCompletion_InvalidAPIKey(t *testing.T) {
	repo := &mockRepo{}
	broadcaster := newMockBroadcaster()
	provider := &mockProvider{
		id:  "mock",
		err: fmt.Errorf("%w: mock", domain.ErrInvalidAPIKey),
	}
	srv := newTestServer(provider, repo, broadcaster)

	rr := callHandler(srv, srv.handleCompletion, "POST", "/api/complete", `{
		"provider": "mock",
		"model": "mock-model",
		"messages": [{"role": "user", "content": "hi"}]
	}`)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestHandleCompletion_RateLimited(t *testing.T) {
	repo := &mockRepo{}
	broadcaster := newMockBroadcaster()
	provider := &mockProvider{
		id:  "mock",
		err: fmt.Errorf("%w: mock", domain.ErrRateLimited),
	}
	srv := newTestServer(provider, repo, broadcaster)

	rr := callHandler(srv, srv.handleCompletion, "POST", "/api/complete", `{
		"provider": "mock",
		"model": "mock-model",
		"messages": [{"role": "user", "content": "hi"}]
	}`)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

func TestHandleCompletion_ProviderUnavailable(t *testing.T) {
	repo := &mockRepo{}
	broadcaster := newMockBroadcaster()
	provider := &mockProvider{
		id:  "mock",
		err: fmt.Errorf("%w: mock", domain.ErrProviderUnavailable),
	}
	srv := newTestServer(provider, repo, broadcaster)

	rr := callHandler(srv, srv.handleCompletion, "POST", "/api/complete", `{
		"provider": "mock",
		"model": "mock-model",
		"messages": [{"role": "user", "content": "hi"}]
	}`)
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
}

func TestHandleCompletion_ContextExceeded(t *testing.T) {
	repo := &mockRepo{}
	broadcaster := newMockBroadcaster()
	provider := &mockProvider{
		id:  "mock",
		err: fmt.Errorf("%w: mock", domain.ErrContextExceeded),
	}
	srv := newTestServer(provider, repo, broadcaster)

	rr := callHandler(srv, srv.handleCompletion, "POST", "/api/complete", `{
		"provider": "mock",
		"model": "mock-model",
		"messages": [{"role": "user", "content": "hi"}]
	}`)
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestHandleCompletion_ModelNotFound(t *testing.T) {
	repo := &mockRepo{}
	broadcaster := newMockBroadcaster()
	provider := &mockProvider{
		id:  "mock",
		err: fmt.Errorf("%w: gpt-x", domain.ErrModelNotFound),
	}
	srv := newTestServer(provider, repo, broadcaster)

	rr := callHandler(srv, srv.handleCompletion, "POST", "/api/complete", `{
		"provider": "mock",
		"model": "gpt-x",
		"messages": [{"role": "user", "content": "hi"}]
	}`)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// ---------------------------------------------------------------------------
// handleCompletion — success
// ---------------------------------------------------------------------------

func TestHandleCompletion_Success(t *testing.T) {
	srv, repo, broadcaster := defaultTestServer()

	rr := callHandler(srv, srv.handleCompletion, "POST", "/api/complete", `{
		"provider": "mock",
		"model": "mock-model",
		"session_id": "sess-1",
		"messages": [{"role": "user", "content": "Hello!"}]
	}`)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp completionResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.NotEmpty(t, resp.ID)
	assert.Equal(t, "mock", resp.Provider)
	assert.Equal(t, "mock-model", resp.Model)
	assert.Equal(t, "sess-1", resp.SessionID)

	// Verify persistence and broadcast
	assert.Len(t, repo.saved, 1)
	assert.Len(t, broadcaster.published, 1)
}

func TestHandleCompletion_Success_ResponseFields(t *testing.T) {
	srv, _, _ := defaultTestServer()

	rr := callHandler(srv, srv.handleCompletion, "POST", "/api/complete", `{
		"provider": "mock",
		"model": "my-model",
		"session_id": "my-session",
		"project_tag": "proj-x",
		"messages": [{"role": "user", "content": "Hi"}]
	}`)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp completionResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, int64(100), resp.Usage.Usage.InputTokens)
	assert.Equal(t, int64(50), resp.Usage.Usage.OutputTokens)
}

// ---------------------------------------------------------------------------
// writeError — status code mapping
// ---------------------------------------------------------------------------

func TestWriteError_InternalServerError(t *testing.T) {
	rr := httptest.NewRecorder()
	writeError(rr, fmt.Errorf("some generic error"))
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestWriteError_AllSentinels(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{domain.ErrInvalidAPIKey, http.StatusUnauthorized},
		{domain.ErrRateLimited, http.StatusTooManyRequests},
		{domain.ErrUnknownProvider, http.StatusBadRequest},
		{domain.ErrModelNotFound, http.StatusBadRequest},
		{domain.ErrContextExceeded, http.StatusUnprocessableEntity},
		{domain.ErrProviderUnavailable, http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			rr := httptest.NewRecorder()
			writeError(rr, tt.err)
			assert.Equal(t, tt.status, rr.Code)
		})
	}
}
