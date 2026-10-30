package extensions

import (
	"fmt"
	"sync"
)

// MessageRoute defines the direction of a message.
type MessageRoute int

const (
	// RouteContentToBackground is a message from a content script to a background script.
	RouteContentToBackground MessageRoute = iota
	// RouteBackgroundToContent is a message from a background script to a content script.
	RouteBackgroundToContent
	// RouteBroadcast is a message broadcast to all content scripts of an extension.
	RouteBroadcast
)

// MessageHandler is a callback invoked when a message is received.
type MessageHandler func(msg Message) error

// MessageRouter routes messages between content scripts, background scripts,
// and the browser shell. It validates message origins and enforces that
// messages can only be sent by the extension that owns the sandbox.
type MessageRouter struct {
	mu          sync.Mutex
	handlers    map[string][]MessageHandler // keyed by extension ID
	routes      []RouteRecord
	sandboxes   map[string]*ContentScriptSandbox
}

// RouteRecord records a routed message for auditing.
type RouteRecord struct {
	Route  MessageRoute
	From   string
	To     string
	Msg    Message
}

// NewMessageRouter creates a new message router.
func NewMessageRouter() *MessageRouter {
	return &MessageRouter{
		handlers:  make(map[string][]MessageHandler),
		routes:    make([]RouteRecord, 0),
		sandboxes: make(map[string]*ContentScriptSandbox),
	}
}

// RegisterSandbox associates a content script sandbox with the router.
func (r *MessageRouter) RegisterSandbox(extID string, sandbox *ContentScriptSandbox) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sandboxes[extID] = sandbox
}

// UnregisterSandbox removes a content script sandbox from the router.
func (r *MessageRouter) UnregisterSandbox(extID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sandboxes, extID)
}

// RegisterHandler registers a message handler for the given extension ID.
func (r *MessageRouter) RegisterHandler(extID string, handler MessageHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[extID] = append(r.handlers[extID], handler)
}

// Route sends a message through the router, validating the sender and
// delivering to the appropriate recipients. Cross-extension messaging is
// blocked: messages can only be delivered within the same extension.
func (r *MessageRouter) Route(msg Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate sender: must be non-empty.
	if msg.Sender == "" {
		return fmt.Errorf("message sender cannot be empty")
	}

	// Record the route.
	record := RouteRecord{
		Route: RouteBroadcast,
		From:  msg.Sender,
		To:    msg.Target,
		Msg:   msg,
	}

	// Determine route type.
	if msg.Target != "" {
		if msg.Target == msg.Sender {
			record.Route = RouteBackgroundToContent
		} else {
			record.Route = RouteContentToBackground
		}
	}

	r.routes = append(r.routes, record)

	// Cross-extension messaging is blocked: a message can only be delivered
	// to handlers registered for the sender's own extension. When Target is
	// empty (broadcast from chrome.runtime.sendMessage), deliver only to the
	// sender's extension handlers. When Target is set and differs from Sender,
	// reject the message as cross-extension.
	target := msg.Target
	if target == "" {
		// Broadcast: deliver only to handlers of the sender's extension.
		handlers, ok := r.handlers[msg.Sender]
		if !ok {
			return nil // no handlers for this extension; nothing to deliver
		}
		for _, h := range handlers {
			if err := h(msg); err != nil {
				return fmt.Errorf("deliver to %s: %w", msg.Sender, err)
			}
		}
		return nil
	}

	if target != msg.Sender {
		return fmt.Errorf("cross-extension messaging blocked: sender %q cannot target %q", msg.Sender, target)
	}

	// Deliver to handlers for the specific target (same extension).
	// If no handlers are registered, silently succeed (nothing to deliver).
	handlers, ok := r.handlers[target]
	if !ok {
		return nil
	}
	for _, h := range handlers {
		if err := h(msg); err != nil {
			return fmt.Errorf("deliver to %s: %w", target, err)
		}
	}

	return nil
}

// DeliverToSandbox delivers a message to a content script sandbox via
// chrome.runtime.onMessage.
func (r *MessageRouter) DeliverToSandbox(extID string, msg Message) error {
	r.mu.Lock()
	sandbox, ok := r.sandboxes[extID]
	r.mu.Unlock()

	if !ok {
		return fmt.Errorf("no sandbox for extension %s", extID)
	}

	return sandbox.PostMessage(msg)
}

// Routes returns a copy of all recorded routes.
func (r *MessageRouter) Routes() []RouteRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]RouteRecord, len(r.routes))
	copy(out, r.routes)
	return out
}

// RouteCount returns the number of recorded routes.
func (r *MessageRouter) RouteCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.routes)
}

// ValidateOrigin checks that the message sender matches the expected
// extension ID, preventing cross-extension message spoofing.
func ValidateOrigin(msg Message, expectedExtID string) error {
	if msg.Sender != expectedExtID {
		return fmt.Errorf("message sender %q does not match expected extension %q", msg.Sender, expectedExtID)
	}
	return nil
}
