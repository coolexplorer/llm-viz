package port

import "github.com/kimseunghwan/llm-viz/backend/internal/domain"

// EventBroadcaster is the outbound port for real-time usage event delivery via SSE.
type EventBroadcaster interface {
	// Subscribe creates a buffered channel for the given session.
	// The caller must call Unsubscribe when done to prevent goroutine leaks.
	Subscribe(sessionID string) <-chan domain.UsageEvent

	// Unsubscribe closes and removes the channel for the given session.
	Unsubscribe(sessionID string)

	// Publish sends an event to the subscriber with the matching sessionID.
	// Non-blocking: events are dropped if the subscriber buffer is full.
	Publish(event domain.UsageEvent)
}
