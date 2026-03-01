package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
)

// ---------------------------------------------------------------------------
// handleSSE — validation
// ---------------------------------------------------------------------------

func TestHandleSSE_MethodNotAllowed(t *testing.T) {
	srv, _, _ := defaultTestServer()
	for _, method := range []string{"POST", "PUT", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(method, "/api/sse?session_id=s1", nil)
			srv.handleSSE(rr, req)
			assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
		})
	}
}

func TestHandleSSE_MissingSessionID(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/sse", nil) // no session_id
	srv.handleSSE(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "session_id is required")
}

// Note: httptest.ResponseRecorder implements http.Flusher in Go 1.20+,
// so the "streaming not supported" path is not reachable in tests.
// The SSE streaming path is tested via flushableRecorder below.

// ---------------------------------------------------------------------------
// handleSSE — with flusher
// ---------------------------------------------------------------------------

func TestHandleSSE_InitialPing(t *testing.T) {
	srv, _, _ := defaultTestServer()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/api/sse?session_id=my-session", nil)
	req = req.WithContext(ctx)
	rr := newFlushableRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.handleSSE(rr, req)
	}()

	// Give the goroutine time to write the initial ping.
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done

	body := rr.Body.String()
	assert.Contains(t, body, "event: ping")
	assert.Contains(t, body, "my-session")
}

func TestHandleSSE_SSEHeaders(t *testing.T) {
	srv, _, _ := defaultTestServer()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/api/sse?session_id=hdr-session", nil)
	req = req.WithContext(ctx)
	rr := newFlushableRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.handleSSE(rr, req)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done

	assert.Equal(t, "text/event-stream", rr.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", rr.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", rr.Header().Get("Connection"))
}

func TestHandleSSE_EventDelivery(t *testing.T) {
	srv, _, broadcaster := defaultTestServer()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/api/sse?session_id=event-session", nil)
	req = req.WithContext(ctx)
	rr := newFlushableRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.handleSSE(rr, req)
	}()

	// Wait for the initial ping to be written before publishing.
	time.Sleep(20 * time.Millisecond)

	// Publish an event.
	broadcaster.Publish(domain.UsageEvent{
		SessionID: "event-session",
		Usage: domain.NormalizedUsage{
			ID:       "usage-1",
			Provider: domain.ProviderOpenAI,
			Model:    "gpt-4o",
		},
	})

	// Wait for the event to be written.
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done

	body := rr.Body.String()
	assert.Contains(t, body, "event: ping")
	assert.Contains(t, body, "event: usage")
	assert.Contains(t, body, "usage-1")
}

func TestHandleSSE_ContextCancellation(t *testing.T) {
	srv, _, _ := defaultTestServer()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/api/sse?session_id=cancel-test", nil)
	req = req.WithContext(ctx)
	rr := newFlushableRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.handleSSE(rr, req)
	}()

	// Cancel immediately after connection.
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Handler should return promptly after context cancellation.
	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not return after context cancellation")
	}
}

func TestHandleSSE_Flushed(t *testing.T) {
	srv, _, _ := defaultTestServer()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/api/sse?session_id=flush-test", nil)
	req = req.WithContext(ctx)
	rr := newFlushableRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.handleSSE(rr, req)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done

	// Flush should have been called at least once for the initial ping.
	assert.Greater(t, rr.flushed, 0)
}

// ---------------------------------------------------------------------------
// SSE response format
// ---------------------------------------------------------------------------

func TestSSEFormat_PingEvent(t *testing.T) {
	srv, _, _ := defaultTestServer()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/api/sse?session_id=fmt-test", nil)
	req = req.WithContext(ctx)
	rr := newFlushableRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.handleSSE(rr, req)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done

	body := rr.Body.String()
	lines := strings.Split(body, "\n")
	// SSE event format: "event: ping\ndata: {...}\n\n"
	hasEventLine := false
	hasDataLine := false
	for _, line := range lines {
		if line == "event: ping" {
			hasEventLine = true
		}
		if strings.HasPrefix(line, "data:") {
			hasDataLine = true
		}
	}
	assert.True(t, hasEventLine, "missing 'event: ping' line")
	assert.True(t, hasDataLine, "missing 'data:' line")
}

// ---------------------------------------------------------------------------
// Full routing for SSE
// ---------------------------------------------------------------------------

func TestRouting_SSE_MissingSessionID(t *testing.T) {
	srv, _, _ := defaultTestServer()
	rr := serveRequest(srv, "GET", "/api/sse", "")
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
