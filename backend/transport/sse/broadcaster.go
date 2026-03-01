package sse

import (
	"sync"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
)

// Broadcaster routes usage events to active SSE client sessions.
// Goroutine-safe; channels are buffered to prevent publish from blocking.
type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[string]chan domain.UsageEvent
}

// NewBroadcaster creates a ready-to-use SSE broadcaster.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[string]chan domain.UsageEvent),
	}
}

// Subscribe creates a buffered channel for the given session.
// The caller must call Unsubscribe when the SSE connection closes.
func (b *Broadcaster) Subscribe(sessionID string) <-chan domain.UsageEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan domain.UsageEvent, 16) // buffered: handles burst without blocking
	b.subscribers[sessionID] = ch
	return ch
}

// Unsubscribe closes and removes the channel for the given session.
func (b *Broadcaster) Unsubscribe(sessionID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ch, ok := b.subscribers[sessionID]; ok {
		close(ch)
		delete(b.subscribers, sessionID)
	}
}

// Publish sends an event to the subscriber matching the event's SessionID.
// Non-blocking: events are dropped if the subscriber buffer is full.
func (b *Broadcaster) Publish(event domain.UsageEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	ch, ok := b.subscribers[event.SessionID]
	if !ok {
		return
	}
	select {
	case ch <- event: // deliver
	default:          // drop if buffer full — prevents blocking the caller
	}
}
