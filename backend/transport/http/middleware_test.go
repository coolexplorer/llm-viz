package http

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// corsMiddleware
// ---------------------------------------------------------------------------

func TestCORSMiddleware_AllowAll(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), "*")

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// When allowedOrigin == "*" and Origin header is present, we echo the origin.
	assert.Equal(t, "https://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, OPTIONS", rr.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, Authorization", rr.Header().Get("Access-Control-Allow-Headers"))
}

func TestCORSMiddleware_AllowAll_NoOriginHeader(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), "*")

	req := httptest.NewRequest("GET", "/", nil) // no Origin header
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// Without Origin header, set * directly
	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_SpecificOrigin_Matching(t *testing.T) {
	allowed := "https://dashboard.example.com"
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), allowed)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", allowed)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, allowed, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_SpecificOrigin_NotMatching(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), "https://allowed.com")

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Non-matching origin should not set ACAO header.
	assert.Equal(t, "", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should NOT be called for OPTIONS.
		w.WriteHeader(http.StatusOK)
	}), "*")

	req := httptest.NewRequest("OPTIONS", "/api/complete", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// OPTIONS preflight returns 204 No Content, not 200.
	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestCORSMiddleware_Credentials(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), "*")

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))
}

// ---------------------------------------------------------------------------
// loggingMiddleware
// ---------------------------------------------------------------------------

func TestLoggingMiddleware_PassesThrough(t *testing.T) {
	logger := slog.Default()
	handler := loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}), logger)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
}

func TestLoggingMiddleware_DefaultStatus200(t *testing.T) {
	logger := slog.Default()
	handler := loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write body without explicit WriteHeader → defaults to 200
		_, _ = w.Write([]byte("ok"))
	}), logger)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// ---------------------------------------------------------------------------
// responseWriter
// ---------------------------------------------------------------------------

func TestResponseWriter_CapturesStatusCode(t *testing.T) {
	base := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: base, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, rw.statusCode)
	assert.Equal(t, http.StatusNotFound, base.Code)
}

func TestResponseWriter_DefaultStatus(t *testing.T) {
	base := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: base, statusCode: http.StatusOK}
	// No WriteHeader call — status should remain the default.
	assert.Equal(t, http.StatusOK, rw.statusCode)
}

func TestResponseWriter_WriteDelegates(t *testing.T) {
	base := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: base, statusCode: http.StatusOK}

	_, err := rw.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, "hello", base.Body.String())
}

// ---------------------------------------------------------------------------
// CORS + logging middleware stack (integration)
// ---------------------------------------------------------------------------

func TestMiddlewareStack(t *testing.T) {
	srv, _, _ := defaultTestServer()

	req := httptest.NewRequest("GET", "/api/health", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rr := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)

	// Both CORS and logging are applied.
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NotEmpty(t, rr.Header().Get("Access-Control-Allow-Origin"))
}
