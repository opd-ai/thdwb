// Package debug provides a remote debugging protocol for extensions, similar
// to the Chrome DevTools Protocol. It allows inspecting extension VMs,
// executing code in extension context, and monitoring extension execution.
package debug

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"

	goja "github.com/dop251/goja"
)

// DebugProtocol is a remote debugging protocol server for extensions. It
// exposes extension VMs for inspection and control over a TCP connection.
type DebugProtocol struct {
	mu       sync.Mutex
	listener net.Listener
	clients  map[*client]struct{}
	vms      map[string]*goja.Runtime // extension ID -> VM
	running  bool
}

// client represents a connected debugging client.
type client struct {
	conn   net.Conn
	enc    *json.Encoder
	dec    *json.Decoder
	mu     sync.Mutex
	closed bool
}

// NewDebugProtocol creates a new DebugProtocol server.
func NewDebugProtocol() *DebugProtocol {
	return &DebugProtocol{
		clients: make(map[*client]struct{}),
		vms:     make(map[string]*goja.Runtime),
	}
}

// RegisterVM registers an extension VM for debugging.
func (dp *DebugProtocol) RegisterVM(extID string, vm *goja.Runtime) {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	dp.vms[extID] = vm
}

// UnregisterVM removes an extension VM from debugging.
func (dp *DebugProtocol) UnregisterVM(extID string) {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	delete(dp.vms, extID)
}

// Listen starts the debug protocol server on the given address.
func (dp *DebugProtocol) Listen(addr string) error {
	dp.mu.Lock()
	if dp.running {
		dp.mu.Unlock()
		return fmt.Errorf("debug protocol already listening")
	}
	dp.running = true
	dp.mu.Unlock()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		dp.mu.Lock()
		dp.running = false
		dp.mu.Unlock()
		return fmt.Errorf("listen: %w", err)
	}
	dp.listener = ln

	go dp.acceptLoop()
	return nil
}

// Close stops the debug protocol server and disconnects all clients.
func (dp *DebugProtocol) Close() error {
	dp.mu.Lock()
	if !dp.running {
		dp.mu.Unlock()
		return nil
	}
	dp.running = false
	ln := dp.listener
	dp.mu.Unlock()

	if ln != nil {
		if err := ln.Close(); err != nil {
			return err
		}
	}

	dp.mu.Lock()
	for c := range dp.clients {
		c.close()
	}
	dp.clients = make(map[*client]struct{})
	dp.mu.Unlock()
	return nil
}

// acceptLoop accepts incoming connections and handles each client.
func (dp *DebugProtocol) acceptLoop() {
	for {
		conn, err := dp.listener.Accept()
		if err != nil {
			return
		}
		c := &client{
			conn: conn,
			enc:  json.NewEncoder(conn),
			dec:  json.NewDecoder(conn),
		}
		dp.mu.Lock()
		dp.clients[c] = struct{}{}
		dp.mu.Unlock()
		go dp.handleClient(c)
	}
}

// handleClient reads messages from a client and dispatches them.
func (dp *DebugProtocol) handleClient(c *client) {
	defer c.close()

	for {
		var msg Message
		if err := c.dec.Decode(&msg); err != nil {
			return // client disconnected
		}
		dp.dispatch(c, msg)
	}
}

// Message is a debug protocol message.
type Message struct {
	ID     int                    `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// Response is a debug protocol response.
type Response struct {
	ID     int         `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  *Error      `json:"error,omitempty"`
}

// Error is a debug protocol error.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	errCodeInvalidParams = -32602
	errCodeMethodNotFound = -32601
	errCodeInternalError  = -32603
	errCodeVMNotFound     = -32000
)

// dispatch routes a message to the appropriate handler.
func (dp *DebugProtocol) dispatch(c *client, msg Message) {
	switch msg.Method {
	case "Debug.listExtensions":
		dp.handleListExtensions(c, msg)
	case "Debug.getExtensionInfo":
		dp.handleGetExtensionInfo(c, msg)
	case "Debug.evaluate":
		dp.handleEvaluate(c, msg)
	case "Debug.getVMState":
		dp.handleGetVMState(c, msg)
	default:
		c.send(&Response{ID: msg.ID, Error: &Error{Code: errCodeMethodNotFound, Message: "method not found"}})
	}
}

// handleListExtensions returns the list of debuggable extensions.
func (dp *DebugProtocol) handleListExtensions(c *client, msg Message) {
	dp.mu.Lock()
	ids := make([]string, 0, len(dp.vms))
	for id := range dp.vms {
		ids = append(ids, id)
	}
	dp.mu.Unlock()

	c.send(&Response{ID: msg.ID, Result: map[string]interface{}{"extensions": ids}})
}

// handleGetExtensionInfo returns info about a specific extension.
func (dp *DebugProtocol) handleGetExtensionInfo(c *client, msg Message) {
	extID, ok := msg.Params["extensionId"].(string)
	if !ok || extID == "" {
		c.send(&Response{ID: msg.ID, Error: &Error{Code: errCodeInvalidParams, Message: "missing extensionId"}})
		return
	}

	dp.mu.Lock()
	vm, ok := dp.vms[extID]
	dp.mu.Unlock()

	if !ok {
		c.send(&Response{ID: msg.ID, Error: &Error{Code: errCodeVMNotFound, Message: "extension VM not found"}})
		return
	}

	info := map[string]interface{}{
		"extensionId": extID,
		"vm":          "goja",
		"hasVM":       vm != nil,
	}
	c.send(&Response{ID: msg.ID, Result: info})
}

// handleEvaluate evaluates JavaScript in an extension's VM.
func (dp *DebugProtocol) handleEvaluate(c *client, msg Message) {
	extID, ok := msg.Params["extensionId"].(string)
	if !ok || extID == "" {
		c.send(&Response{ID: msg.ID, Error: &Error{Code: errCodeInvalidParams, Message: "missing extensionId or expression"}})
		return
	}
	source, ok := msg.Params["expression"].(string)
	if !ok || source == "" {
		c.send(&Response{ID: msg.ID, Error: &Error{Code: errCodeInvalidParams, Message: "missing extensionId or expression"}})
		return
	}

	dp.mu.Lock()
	vm, ok := dp.vms[extID]
	dp.mu.Unlock()

	if !ok {
		c.send(&Response{ID: msg.ID, Error: &Error{Code: errCodeVMNotFound, Message: "extension VM not found"}})
		return
	}

	result, err := vm.RunString(source)
	if err != nil {
		c.send(&Response{ID: msg.ID, Error: &Error{Code: errCodeInternalError, Message: err.Error()}})
		return
	}

	c.send(&Response{ID: msg.ID, Result: map[string]interface{}{"result": resultToString(vm, result)}})
}

// handleGetVMState returns the state of an extension's VM.
func (dp *DebugProtocol) handleGetVMState(c *client, msg Message) {
	extID, ok := msg.Params["extensionId"].(string)
	if !ok || extID == "" {
		c.send(&Response{ID: msg.ID, Error: &Error{Code: errCodeInvalidParams, Message: "missing extensionId"}})
		return
	}

	dp.mu.Lock()
	vm, ok := dp.vms[extID]
	dp.mu.Unlock()

	if !ok {
		c.send(&Response{ID: msg.ID, Error: &Error{Code: errCodeVMNotFound, Message: "extension VM not found"}})
		return
	}

	c.send(&Response{ID: msg.ID, Result: map[string]interface{}{
		"extensionId": extID,
		"hasVM":       vm != nil,
	}})
}

// resultToString converts a goja Value to a string representation.
func resultToString(vm *goja.Runtime, val goja.Value) string {
	if val == nil {
		return "null"
	}
	return val.String()
}

// send sends a response to a client.
func (c *client) send(resp *Response) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	_ = c.enc.Encode(resp)
}

// close closes the client connection.
func (c *client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	_ = c.conn.Close()
}
