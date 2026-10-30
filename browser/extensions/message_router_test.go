package extensions

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMessageRouterCreation(t *testing.T) {
	r := NewMessageRouter()
	require.NotNil(t, r)
	require.Equal(t, 0, r.RouteCount())
}

func TestMessageRouterRegisterAndRoute(t *testing.T) {
	r := NewMessageRouter()

	var received []Message
	r.RegisterHandler("ext1", func(msg Message) error {
		received = append(received, msg)
		return nil
	})

	// Same-extension delivery works.
	err := r.Route(Message{
		Sender:  "ext1",
		Target:  "ext1",
		Type:    "background",
		Payload: map[string]interface{}{"hello": "world"},
	})
	require.NoError(t, err)

	require.Len(t, received, 1)
	require.Equal(t, "ext1", received[0].Sender)
	require.Equal(t, "ext1", received[0].Target)
	require.Equal(t, "world", received[0].Payload.(map[string]interface{})["hello"])

	// Cross-extension delivery is blocked.
	err = r.Route(Message{
		Sender:  "ext2",
		Target:  "ext1",
		Type:    "background",
		Payload: map[string]interface{}{"hello": "world"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cross-extension messaging blocked")
	require.Len(t, received, 1) // no additional delivery
}

func TestMessageRouterRouteEmptySender(t *testing.T) {
	r := NewMessageRouter()

	err := r.Route(Message{
		Target:  "ext1",
		Type:    "background",
		Payload: nil,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "sender cannot be empty")
}

func TestMessageRouterRouteUnknownTarget(t *testing.T) {
	r := NewMessageRouter()

	// Same-extension message to unknown target within same extension.
	err := r.Route(Message{
		Sender:  "ext1",
		Target:  "ext1",
		Type:    "background",
		Payload: nil,
	})
	require.NoError(t, err)

	// Cross-extension messaging is blocked.
	err = r.Route(Message{
		Sender:  "ext1",
		Target:  "ext2",
		Type:    "background",
		Payload: nil,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cross-extension messaging blocked")
}

func TestMessageRouterBroadcast(t *testing.T) {
	r := NewMessageRouter()

	var received1, received2, received3 []Message
	r.RegisterHandler("ext1", func(msg Message) error {
		received1 = append(received1, msg)
		return nil
	})
	r.RegisterHandler("ext2", func(msg Message) error {
		received2 = append(received2, msg)
		return nil
	})
	r.RegisterHandler("ext3", func(msg Message) error {
		received3 = append(received3, msg)
		return nil
	})

	// Broadcast from ext3 only delivers to ext3's own handlers.
	err := r.Route(Message{
		Sender:  "ext3",
		Type:    "broadcast",
		Payload: map[string]interface{}{"msg": "hi"},
	})
	require.NoError(t, err)

	require.Empty(t, received1)
	require.Empty(t, received2)
	require.Len(t, received3, 1)
	require.Equal(t, "ext3", received3[0].Sender)
	require.Equal(t, "hi", received3[0].Payload.(map[string]interface{})["msg"])
}

func TestMessageRouterValidateOrigin(t *testing.T) {
	// Valid origin.
	err := ValidateOrigin(Message{Sender: "ext1"}, "ext1")
	require.NoError(t, err)

	// Invalid origin.
	err = ValidateOrigin(Message{Sender: "ext2"}, "ext1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not match")
}

func TestMessageRouterRegisterSandbox(t *testing.T) {
	r := NewMessageRouter()
	sandbox := NewContentScriptSandbox("ext1")

	r.RegisterSandbox("ext1", sandbox)
	// Registration doesn't add routes.
	require.Equal(t, 0, r.RouteCount())

	// Unregister.
	r.UnregisterSandbox("ext1")
	// Should not panic.
}

func TestMessageRouterDeliverToSandbox(t *testing.T) {
	r := NewMessageRouter()
	sandbox := NewContentScriptSandbox("ext1")

	// Register onMessage handler in sandbox.
	_, err := sandbox.Run(`
		var received = null;
		chrome.runtime.onMessage = function(msg) { received = msg; };
		function getReceived() { return received; }
	`)
	require.NoError(t, err)

	r.RegisterSandbox("ext1", sandbox)

	err = r.DeliverToSandbox("ext1", Message{
		Sender:  "background",
		Target:  "ext1",
		Type:    "background",
		Payload: map[string]interface{}{"hello": "world"},
	})
	require.NoError(t, err)

	val, err := sandbox.Run("getReceived()")
	require.NoError(t, err)
	exported := val.Export().(map[string]interface{})
	require.Equal(t, "world", exported["hello"])
}

func TestMessageRouterDeliverToUnknownSandbox(t *testing.T) {
	r := NewMessageRouter()

	err := r.DeliverToSandbox("nonexistent", Message{
		Sender: "background",
		Payload: nil,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no sandbox for extension")
}

func TestMessageRouterRoutesRecorded(t *testing.T) {
	r := NewMessageRouter()
	r.RegisterHandler("ext1", func(msg Message) error { return nil })

	err := r.Route(Message{
		Sender:  "ext1",
		Target:  "ext1",
		Type:    "background",
		Payload: nil,
	})
	require.NoError(t, err)

	require.Equal(t, 1, r.RouteCount())
	routes := r.Routes()
	require.Len(t, routes, 1)
	require.Equal(t, "ext1", routes[0].From)
	require.Equal(t, "ext1", routes[0].To)
}

func TestMessageRouterMultipleHandlers(t *testing.T) {
	r := NewMessageRouter()

	var count1, count2 int
	r.RegisterHandler("ext1", func(msg Message) error {
		count1++
		return nil
	})
	r.RegisterHandler("ext1", func(msg Message) error {
		count2++
		return nil
	})

	err := r.Route(Message{
		Sender:  "ext1",
		Target:  "ext1",
		Type:    "background",
		Payload: nil,
	})
	require.NoError(t, err)

	require.Equal(t, 1, count1)
	require.Equal(t, 1, count2)
}

func TestMessageRouterHandlerError(t *testing.T) {
	r := NewMessageRouter()
	handlerErr := errors.New("handler failed")

	r.RegisterHandler("ext1", func(msg Message) error {
		return handlerErr
	})

	err := r.Route(Message{
		Sender:  "ext1",
		Target:  "ext1",
		Type:    "background",
		Payload: nil,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "handler failed")
}
