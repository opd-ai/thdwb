package extensions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	goja "github.com/dop251/goja"
)

// SandboxMode defines the execution world for a content script.
type SandboxMode int

const (
	// IsolatedWorld runs the content script in an isolated JavaScript
	// environment that cannot access the page's JavaScript objects or
	// globals. This is the default and most secure mode.
	IsolatedWorld SandboxMode = iota
	// MainWorld runs the content script in the same JavaScript world as
	// the page, sharing globals and objects. Less secure but required for
	// some extension use cases.
	MainWorld
)

// Message represents a message passed between content scripts and background
// scripts, or between content scripts and the browser shell.
type Message struct {
	// Sender is the extension ID of the message sender.
	Sender string `json:"sender"`

	// Target is the intended recipient extension ID, or empty for broadcast.
	Target string `json:"target,omitempty"`

	// Type is the message type (e.g. "content-script", "background").
	Type string `json:"type"`

	// Payload is the message data.
	Payload interface{} `json:"payload"`

	// TabID identifies the tab the message originated from, if any.
	TabID int `json:"tab_id,omitempty"`
}

// ContentScriptSandbox provides an isolated Goja VM for executing content
// scripts. It restricts access to dangerous globals (eval, Function,
// setTimeout, etc.) and exposes only a minimal, safe API surface including
// chrome.runtime messaging.
type ContentScriptSandbox struct {
	mu          sync.Mutex
	runtime     *goja.Runtime
	extID       string
	mode        SandboxMode
	messages    chan Message
	globalsInit bool
}

// NewContentScriptSandbox creates a new isolated sandbox for the given
// extension. The sandbox runs in IsolatedWorld mode by default.
func NewContentScriptSandbox(extID string) *ContentScriptSandbox {
	return &ContentScriptSandbox{
		extID:    extID,
		mode:     IsolatedWorld,
		runtime:  goja.New(),
		messages: make(chan Message, 128),
	}
}
func (s *ContentScriptSandbox) SetMode(mode SandboxMode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = mode
}

// Mode returns the current execution world.
func (s *ContentScriptSandbox) Mode() SandboxMode {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mode
}

// ExtensionID returns the extension ID this sandbox belongs to.
func (s *ContentScriptSandbox) ExtensionID() string {
	return s.extID
}

// initGlobals configures the sandbox's global object with a restricted,
// safe API surface. Dangerous functions like eval and Function are set to
// undefined so they cannot be called from content scripts.
// initGlobals configures the sandbox's global object with a restricted,
// safe API surface. Dangerous functions like eval and Function are set to
// undefined so they cannot be called from content scripts.
// Callers must hold s.mu.
func (s *ContentScriptSandbox) initGlobals() error {
	if s.globalsInit {
		return nil
	}

	rt := s.runtime

	// Neutralize dangerous globals by setting them to undefined.
	_ = rt.Set("eval", goja.Undefined())
	_ = rt.Set("Function", goja.Undefined())
	_ = rt.Set("setTimeout", goja.Undefined())
	_ = rt.Set("setInterval", goja.Undefined())
	_ = rt.Set("setImmediate", goja.Undefined())

	// Expose a minimal console object.
	console := rt.NewObject()
	_ = console.Set("log", func(call goja.FunctionCall) goja.Value {
		args := make([]string, len(call.Arguments))
		for i, a := range call.Arguments {
			args[i] = fmt.Sprintf("%v", a)
		}
		fmt.Printf("[content-script:%s] %s\n", s.extID, strings.Join(args, " "))
		return goja.Undefined()
	})
	_ = console.Set("error", func(call goja.FunctionCall) goja.Value {
		args := make([]string, len(call.Arguments))
		for i, a := range call.Arguments {
			args[i] = fmt.Sprintf("%v", a)
		}
		fmt.Printf("[content-script:%s] ERROR: %s\n", s.extID, strings.Join(args, " "))
		return goja.Undefined()
	})
	_ = rt.Set("console", console)

	// Expose chrome.runtime messaging API.
	chrome := rt.NewObject()
	runtimeObj := rt.NewObject()
	_ = runtimeObj.Set("sendMessage", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(rt.NewGoError(fmt.Errorf("sendMessage requires a message argument")))
		}
		msg := Message{
			Sender: s.extID,
			Type:   "content-script",
			Payload: call.Argument(0).Export(),
		}
		select {
		case s.messages <- msg:
		default:
			// Channel full; drop the message.
		}
		return goja.Undefined()
	})
	// onMessage is a settable property that content scripts can assign to
	// receive messages from the background script.
	_ = runtimeObj.Set("onMessage", goja.Undefined())
	_ = runtimeObj.Set("id", s.extID)
	_ = chrome.Set("runtime", runtimeObj)
	_ = rt.Set("chrome", chrome)

	// Expose fetch stub for content scripts.
	_ = rt.Set("fetch", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		url := call.Argument(0).String()
		fmt.Printf("[content-script:%s] fetch(%s)\n", s.extID, url)
		return goja.Undefined()
	})

	// Expose XMLHttpRequest constructor stub.
	_ = rt.Set("XMLHttpRequest", func(call goja.ConstructorCall) *goja.Object {
		obj := call.This
		_ = obj.Set("open", func(call goja.FunctionCall) goja.Value {
			return goja.Undefined()
		})
		_ = obj.Set("send", func(call goja.FunctionCall) goja.Value {
			return goja.Undefined()
		})
		_ = obj.Set("setRequestHeader", func(call goja.FunctionCall) goja.Value {
			return goja.Undefined()
		})
		_ = obj.Set("addEventListener", func(call goja.FunctionCall) goja.Value {
			return goja.Undefined()
		})
		return obj
	})

	s.globalsInit = true
	return nil
}

// LoadScript reads a JavaScript file from the extension directory and
// executes it in the sandbox. Relative paths are resolved against the
// extension's base path.
func (s *ContentScriptSandbox) LoadScript(extPath, jsPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.initGlobals(); err != nil {
		return fmt.Errorf("init globals: %w", err)
	}

	absPath := jsPath
	if !filepath.IsAbs(jsPath) {
		absPath = filepath.Join(extPath, jsPath)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read content script %s: %w", jsPath, err)
	}

	if _, err := s.runtime.RunString(string(data)); err != nil {
		return fmt.Errorf("execute content script %s: %w", jsPath, err)
	}

	return nil
}

// PopMessage retrieves the next message from the sandbox's outbound channel,
// or returns false if no message is available.
func (s *ContentScriptSandbox) PopMessage() (Message, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case msg := <-s.messages:
		return msg, true
	default:
		return Message{}, false
	}
}

// PostMessage delivers a message to the sandbox's content script, if the
// sandbox has registered a message listener via chrome.runtime.onMessage.
func (s *ContentScriptSandbox) PostMessage(msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rt := s.runtime
	chromeVal := rt.Get("chrome")
	if chromeVal == nil {
		return fmt.Errorf("no onMessage listener registered")
	}
	chromeObj := chromeVal.ToObject(rt)
	if chromeObj == nil {
		return fmt.Errorf("no onMessage listener registered")
	}
	runtimeVal := chromeObj.Get("runtime")
	if runtimeVal == nil {
		return fmt.Errorf("no onMessage listener registered")
	}
	runtimeObj := runtimeVal.ToObject(rt)
	if runtimeObj == nil {
		return fmt.Errorf("no onMessage listener registered")
	}
	onMessage := runtimeObj.Get("onMessage")
	if onMessage == nil {
		return fmt.Errorf("no onMessage listener registered")
	}

	cb, ok := goja.AssertFunction(onMessage)
	if !ok {
		return fmt.Errorf("onMessage is not a function")
	}

	_, err := cb(goja.Undefined(), rt.ToValue(msg.Payload))
	return err
}

// Run executes an arbitrary script string in the sandbox context.
func (s *ContentScriptSandbox) Run(script string) (goja.Value, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.initGlobals(); err != nil {
		return nil, fmt.Errorf("init globals: %w", err)
	}

	return s.runtime.RunString(script)
}

// Close releases the sandbox's resources.
func (s *ContentScriptSandbox) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	close(s.messages)
	return nil
}
