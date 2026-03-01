package sse_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/transport/sse"
)

func TestNewBroadcaster(t *testing.T) {
	b := sse.NewBroadcaster()
	require.NotNil(t, b)
}

func TestBroadcaster_SubscribeReceivesEvents(t *testing.T) {
	b := sse.NewBroadcaster()
	ch := b.Subscribe("s1")
	require.NotNil(t, ch)

	event := domain.UsageEvent{SessionID: "s1", Usage: domain.NormalizedUsage{ID: "u1"}}
	b.Publish(event)

	select {
	case got := <-ch:
		assert.Equal(t, "s1", got.SessionID)
		assert.Equal(t, "u1", got.Usage.ID)
	case <-time.After(time.Second):
		t.Fatal("expected event not received")
	}
}

func TestBroadcaster_UnsubscribeClosesChannel(t *testing.T) {
	b := sse.NewBroadcaster()
	ch := b.Subscribe("s1")
	b.Unsubscribe("s1")

	// Channel should be closed; reading from it should not block.
	select {
	case _, open := <-ch:
		assert.False(t, open, "channel should be closed after Unsubscribe")
	case <-time.After(time.Second):
		t.Fatal("channel was not closed after Unsubscribe")
	}
}

func TestBroadcaster_PublishNonExistentSession(t *testing.T) {
	b := sse.NewBroadcaster()
	// Publishing to a session with no subscriber should not panic.
	b.Publish(domain.UsageEvent{SessionID: "ghost"})
}

func TestBroadcaster_MultipleSubscribers(t *testing.T) {
	b := sse.NewBroadcaster()
	ch1 := b.Subscribe("session-A")
	ch2 := b.Subscribe("session-B")

	b.Publish(domain.UsageEvent{SessionID: "session-A", Usage: domain.NormalizedUsage{ID: "a"}})
	b.Publish(domain.UsageEvent{SessionID: "session-B", Usage: domain.NormalizedUsage{ID: "b"}})

	var got1, got2 domain.UsageEvent
	select {
	case got1 = <-ch1:
	case <-time.After(time.Second):
		t.Fatal("session-A event not received")
	}
	select {
	case got2 = <-ch2:
	case <-time.After(time.Second):
		t.Fatal("session-B event not received")
	}

	assert.Equal(t, "a", got1.Usage.ID)
	assert.Equal(t, "b", got2.Usage.ID)
}

func TestBroadcaster_PublishOnlyToMatchingSession(t *testing.T) {
	b := sse.NewBroadcaster()
	ch1 := b.Subscribe("s1")
	ch2 := b.Subscribe("s2")

	// Publish only to s1.
	b.Publish(domain.UsageEvent{SessionID: "s1"})

	select {
	case <-ch1:
		// correct
	case <-time.After(time.Second):
		t.Fatal("s1 event not received")
	}

	// ch2 should NOT receive anything.
	select {
	case <-ch2:
		t.Fatal("s2 received an event it shouldn't have")
	case <-time.After(10 * time.Millisecond):
		// expected: nothing received
	}

	b.Unsubscribe("s1")
	b.Unsubscribe("s2")
}

func TestBroadcaster_PublishDoesNotBlockOnFullBuffer(t *testing.T) {
	b := sse.NewBroadcaster()
	ch := b.Subscribe("s1")

	// Publish more events than the buffer (16) without reading.
	for i := 0; i < 20; i++ {
		b.Publish(domain.UsageEvent{SessionID: "s1"})
	}

	// Drain what arrived.
	var count int
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	// Should have received at most 16 (buffer capacity), rest dropped.
	assert.LessOrEqual(t, count, 16)
	b.Unsubscribe("s1")
}

func TestBroadcaster_UnsubscribeNonExistent(t *testing.T) {
	b := sse.NewBroadcaster()
	// Should not panic when unsubscribing a session that was never subscribed.
	b.Unsubscribe("nobody")
}

func TestBroadcaster_ConcurrentPublishSubscribe(t *testing.T) {
	b := sse.NewBroadcaster()
	const sessions = 5
	const events = 10

	var wg sync.WaitGroup
	channels := make([]<-chan domain.UsageEvent, sessions)
	for i := 0; i < sessions; i++ {
		sessionID := domain.ProviderID(string(rune('A' + i)))
		channels[i] = b.Subscribe(string(sessionID))
	}

	// Concurrent publishers.
	wg.Add(sessions)
	for i := 0; i < sessions; i++ {
		go func(idx int) {
			defer wg.Done()
			sessionID := string(rune('A' + idx))
			for j := 0; j < events; j++ {
				b.Publish(domain.UsageEvent{SessionID: sessionID})
			}
		}(i)
	}

	wg.Wait()
	// No panic, no data race — success.
	for i := 0; i < sessions; i++ {
		b.Unsubscribe(string(rune('A' + i)))
	}
}
