// Package http — API key HTTP handler tests (Red phase, TDD).
//
// Tests use serveRequest (full router) so they compile without referencing
// non-existent handler methods. They fail at runtime with 404 until the
// key-management routes are registered in NewServer (Task #2).
//
// Routes expected after implementation:
//
//	POST   /api/keys           → save a new API key, returns masked metadata
//	GET    /api/keys           → list keys (optional ?provider= filter)
//	DELETE /api/keys/{id}      → delete key by ID
//
// Request/response shapes are defined by these tests and must be honoured by
// the implementation.
package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// POST /api/keys — save key
// ---------------------------------------------------------------------------

func TestHandleKeys_SaveKey_Success(t *testing.T) {
	srv, _, _ := defaultTestServer()
	body := `{"provider":"openai","name":"My GPT Key","key":"sk-proj-abc123456789"}`

	rr := serveRequest(srv, "POST", "/api/keys", body)

	require.Equal(t, http.StatusCreated, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.NotEmpty(t, resp["id"])
	assert.Equal(t, "openai", resp["provider"])
	assert.Equal(t, "My GPT Key", resp["name"])
	assert.NotEmpty(t, resp["masked_key"])
	assert.NotEmpty(t, resp["created_at"])
	// Raw key must never appear in response.
	assert.Nil(t, resp["key"], "raw key must not be returned")
}

func TestHandleKeys_SaveKey_AnthropicProvider(t *testing.T) {
	srv, _, _ := defaultTestServer()
	body := `{"provider":"anthropic","name":"Claude Key","key":"sk-ant-api03-test123"}`

	rr := serveRequest(srv, "POST", "/api/keys", body)

	require.Equal(t, http.StatusCreated, rr.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "anthropic", resp["provider"])
	maskedKey, ok := resp["masked_key"].(string)
	require.True(t, ok)
	assert.Contains(t, maskedKey, "***", "masked key must contain ***")
	assert.False(t, strings.Contains(maskedKey, "sk-ant-api03-test123"), "masked key must not reveal full value")
}

func TestHandleKeys_SaveKey_MethodNotAllowed(t *testing.T) {
	srv, _, _ := defaultTestServer()
	for _, method := range []string{"PUT", "PATCH"} {
		t.Run(method, func(t *testing.T) {
			rr := serveRequest(srv, method, "/api/keys", "")
			assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
		})
	}
}

func TestHandleKeys_SaveKey_InvalidJSON(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := serveRequest(srv, "POST", "/api/keys", `{invalid json`)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleKeys_SaveKey_MissingProvider(t *testing.T) {
	srv, _, _ := defaultTestServer()
	body := `{"name":"test","key":"sk-proj-abc"}`
	rr := serveRequest(srv, "POST", "/api/keys", body)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "provider")
}

func TestHandleKeys_SaveKey_MissingKey(t *testing.T) {
	srv, _, _ := defaultTestServer()
	body := `{"provider":"openai","name":"test"}`
	rr := serveRequest(srv, "POST", "/api/keys", body)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "key")
}

func TestHandleKeys_SaveKey_MissingName(t *testing.T) {
	srv, _, _ := defaultTestServer()
	body := `{"provider":"openai","key":"sk-proj-abc"}`
	rr := serveRequest(srv, "POST", "/api/keys", body)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "name")
}

func TestHandleKeys_SaveKey_InvalidProvider(t *testing.T) {
	srv, _, _ := defaultTestServer()
	body := `{"provider":"fakeprovider","name":"x","key":"apikey-abc"}`
	rr := serveRequest(srv, "POST", "/api/keys", body)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleKeys_SaveKey_EmptyKey(t *testing.T) {
	srv, _, _ := defaultTestServer()
	body := `{"provider":"openai","name":"test","key":""}`
	rr := serveRequest(srv, "POST", "/api/keys", body)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// ---------------------------------------------------------------------------
// GET /api/keys — list keys
// ---------------------------------------------------------------------------

func TestHandleKeys_ListKeys_Success_Empty(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := serveRequest(srv, "GET", "/api/keys", "")

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	_, hasKeys := resp["keys"]
	assert.True(t, hasKeys, "response must contain 'keys' field")
	_, hasCount := resp["count"]
	assert.True(t, hasCount, "response must contain 'count' field")
}

func TestHandleKeys_ListKeys_MethodNotAllowed(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := serveRequest(srv, "PUT", "/api/keys", "")
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestHandleKeys_ListKeys_WithProviderFilter(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := serveRequest(srv, "GET", "/api/keys?provider=openai", "")

	// Route must be registered and respond (not 404).
	assert.NotEqual(t, http.StatusNotFound, rr.Code)
	if rr.Code == http.StatusOK {
		var resp map[string]any
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		_, hasKeys := resp["keys"]
		assert.True(t, hasKeys)
	}
}

func TestHandleKeys_ListKeys_NeverReturnsRawKey(t *testing.T) {
	// After saving a key, list must not expose the raw value.
	srv, _, _ := defaultTestServer()

	// Save a key first.
	saveBody := `{"provider":"openai","name":"secret","key":"sk-proj-very-secret-value-1234"}`
	serveRequest(srv, "POST", "/api/keys", saveBody)

	rr := serveRequest(srv, "GET", "/api/keys", "")
	require.Equal(t, http.StatusOK, rr.Code)
	// The raw key must not appear anywhere in the response body.
	assert.False(t, strings.Contains(rr.Body.String(), "sk-proj-very-secret-value-1234"),
		"raw key must never appear in list response")
}

func TestHandleKeys_ListKeys_ResponseContainsMaskedKey(t *testing.T) {
	srv, _, _ := defaultTestServer()

	saveBody := `{"provider":"anthropic","name":"antkey","key":"sk-ant-api03-testmasked"}`
	saveRr := serveRequest(srv, "POST", "/api/keys", saveBody)
	if saveRr.Code != http.StatusCreated {
		t.Skip("Save endpoint not yet implemented")
	}

	listRr := serveRequest(srv, "GET", "/api/keys", "")
	require.Equal(t, http.StatusOK, listRr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(listRr.Body).Decode(&resp))
	keys, ok := resp["keys"].([]any)
	require.True(t, ok)
	require.Len(t, keys, 1)

	keyObj, ok := keys[0].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, keyObj["masked_key"])
	assert.Contains(t, keyObj["masked_key"].(string), "***")
}

// ---------------------------------------------------------------------------
// DELETE /api/keys/{id} — delete key
// ---------------------------------------------------------------------------

func TestHandleKeys_DeleteKey_Success(t *testing.T) {
	srv, _, _ := defaultTestServer()

	// Save then delete.
	saveRr := serveRequest(srv, "POST", "/api/keys", `{"provider":"openai","name":"d","key":"sk-proj-todelete"}`)
	if saveRr.Code != http.StatusCreated {
		t.Skip("Save endpoint not yet implemented")
	}
	var saved map[string]any
	require.NoError(t, json.NewDecoder(saveRr.Body).Decode(&saved))
	id := saved["id"].(string)

	delRr := serveRequest(srv, "DELETE", "/api/keys/"+id, "")
	assert.Equal(t, http.StatusNoContent, delRr.Code)
}

func TestHandleKeys_DeleteKey_MethodNotAllowed(t *testing.T) {
	srv, _, _ := defaultTestServer()
	// PATCH on a key ID path must not be allowed.
	rr := serveRequest(srv, "PATCH", "/api/keys/some-id", "")
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestHandleKeys_DeleteKey_NotFound(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := serveRequest(srv, "DELETE", "/api/keys/ghost-id-does-not-exist", "")
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandleKeys_DeleteKey_EmptyID(t *testing.T) {
	srv, _, _ := defaultTestServer()
	// DELETE /api/keys/ with empty segment should be treated as list route → method not allowed.
	rr := serveRequest(srv, "DELETE", "/api/keys/", "")
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

// ---------------------------------------------------------------------------
// Route registration sanity — all routes must return something other than 404
// ---------------------------------------------------------------------------

func TestHandleKeys_RoutesAreRegistered(t *testing.T) {
	srv, _, _ := defaultTestServer()

	routes := []struct {
		method string
		path   string
	}{
		{"POST", "/api/keys"},
		{"GET", "/api/keys"},
		{"DELETE", "/api/keys/some-id"},
	}

	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			rr := serveRequest(srv, r.method, r.path, "")
			assert.NotEqual(t, http.StatusNotFound, rr.Code,
				"route %s %s must be registered (got 404 — not implemented yet)", r.method, r.path)
		})
	}
}
