package extensions

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRouteReentrantHandlerNoDeadlock (audit C-26) verifies a handler that
// replies through Route on the same call stack does not deadlock. The
// registered onMessage handler runs background JS; the canonical "ack a
// content-script message" pattern calls chrome.runtime.sendMessage from
// inside it. Route used to hold its plain mutex across the synchronous
// handler invocation, so that re-entry blocked forever (probe: DEADLOCK
// CONFIRMED).
func TestRouteReentrantHandlerNoDeadlock(t *testing.T) {
	r := NewMessageRouter()

	var innerDelivered int32
	r.RegisterHandler("ext-inner", func(msg Message) error {
		atomic.AddInt32(&innerDelivered, 1)
		return nil
	})
	r.RegisterHandler("ext-1", func(msg Message) error {
		// Re-enter Route from inside the handler -- the sendMessage shape.
		return r.Route(Message{Sender: "ext-inner", Target: "ext-inner", Payload: "ack"})
	})

	done := make(chan error, 1)
	go func() {
		done <- r.Route(Message{Sender: "ext-1", Target: "ext-1", Payload: "ping"})
	}()

	select {
	case err := <-done:
		require.NoError(t, err, "re-entrant Route must complete, not deadlock")
	case <-time.After(2 * time.Second):
		t.Fatal("Route self-deadlocked on handler re-entry (audit C-26)")
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&innerDelivered),
		"inner re-entrant message must be delivered")
}

// TestRouteDeliversToHandlers verifies the unlocked delivery still runs every
// registered handler and propagates handler errors.
func TestRouteDeliversToHandlers(t *testing.T) {
	r := NewMessageRouter()

	var got []string
	r.RegisterHandler("ext-a", func(msg Message) error {
		got = append(got, "h1")
		return nil
	})
	r.RegisterHandler("ext-a", func(msg Message) error {
		got = append(got, "h2")
		return nil
	})
	require.NoError(t, r.Route(Message{Sender: "ext-a", Payload: "x"}))
	assert.Equal(t, []string{"h1", "h2"}, got)

	// Handler error propagates.
	r.RegisterHandler("ext-b", func(msg Message) error {
		return assertError
	})
	err := r.Route(Message{Sender: "ext-b", Target: "ext-b"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deliver to ext-b")

	// Empty sender still rejected.
	require.Error(t, r.Route(Message{Sender: "", Target: "ext-a"}))

	// Cross-extension blocked.
	require.Error(t, r.Route(Message{Sender: "ext-a", Target: "ext-c"}))

	// Unknown target with empty Target: broadcast path, no handlers -> nil.
	require.NoError(t, r.Route(Message{Sender: "ext-z"}))
}

var assertError = errString("handler failed")

type errString string

func (e errString) Error() string { return string(e) }
