package hotdog

import (
	"net/http"
	"net/url"
	"sync"

	"github.com/danfragoso/thdwb/assets"
	mustard "github.com/danfragoso/thdwb/mustard"
	profiler "github.com/danfragoso/thdwb/profiler"
	sauce "github.com/danfragoso/thdwb/sauce"
	goja "github.com/dop251/goja"
)

// WindowRegistry manages all window contexts for cross-window communication.
type WindowRegistry struct {
	mu      sync.RWMutex
	windows map[*WindowContext]struct{}
}

// Global window registry for cross-window messaging.
var globalWindowRegistry = &WindowRegistry{
	windows: make(map[*WindowContext]struct{}),
}

// RegisterWindow adds a window context to the global registry.
func RegisterWindow(wc *WindowContext) {
	globalWindowRegistry.mu.Lock()
	defer globalWindowRegistry.mu.Unlock()
	globalWindowRegistry.windows[wc] = struct{}{}
}

// UnregisterWindow removes a window context from the global registry.
func UnregisterWindow(wc *WindowContext) {
	globalWindowRegistry.mu.Lock()
	defer globalWindowRegistry.mu.Unlock()
	delete(globalWindowRegistry.windows, wc)
}

// FindWindowsByOrigin returns all window contexts matching the given origin.
// If targetOrigin is "*", all windows are returned.
func FindWindowsByOrigin(targetOrigin string) []*WindowContext {
	globalWindowRegistry.mu.RLock()
	defer globalWindowRegistry.mu.RUnlock()

	var result []*WindowContext
	for wc := range globalWindowRegistry.windows {
		if targetOrigin == "*" {
			result = append(result, wc)
			continue
		}
		wcOrigin := wc.GetOrigin().String()
		if wcOrigin == targetOrigin {
			result = append(result, wc)
		}
	}
	return result
}

// WindowContext holds all state for a single browser window/tab instance.
// This enables multiple independent windows to run without memory interference.
type WindowContext struct {
	// DOM and document state
	ActiveDocument *Document
	Documents      []*Document

	// Rendering and viewport
	Viewport    *mustard.CanvasWidget
	StatusLabel *mustard.LabelWidget

	// Navigation history
	History *History

	// Mustard window reference
	Window *mustard.Window

	// Profiling
	Profiler *profiler.Profiler

	// Settings and build info (shared reference)
	Settings  *Settings
	BuildInfo *assets.BuildInfo

	// Origin tracking for cookie partitioning and SOP
	Origin *url.URL

	// Layout cache
	layoutCache map[string]interface{}

	// Focus and input state
	focusedElement *NodeDOM
	inputState     map[string]interface{}

	// Event registry for this window
	eventRegistry map[string][]func(interface{})

	// JS runtime (Goja VM) - per window isolation
	jsRuntime *goja.Runtime

	// Session storage manager (per-window, origin-partitioned)
	sessionStorage *SessionStorageManager

	// Cookie jar for this window (per-window cookie isolation)
	cookieJar *sauce.OriginCookieJar

	// HTTP client for this window (with per-window cookie jar)
	httpClient *http.Client

	// Debug state
	DebugFlag   bool
	DebugWindow *mustard.Window
	DebugTree   *mustard.TreeWidget
}

// NewWindowContext creates a new isolated window context.
func NewWindowContext(settings *Settings, buildInfo *assets.BuildInfo, profiler *profiler.Profiler) *WindowContext {
	wc := &WindowContext{
		ActiveDocument: &Document{},
		Documents:      make([]*Document, 0),
		History:        &History{},
		Profiler:       profiler,
		Settings:       settings,
		BuildInfo:      buildInfo,
		layoutCache:    make(map[string]interface{}),
		inputState:     make(map[string]interface{}),
		eventRegistry:  make(map[string][]func(interface{})),
		sessionStorage: NewSessionStorageManager(),
	}

	// Initialize per-window Goja VM for JavaScript execution
	wc.jsRuntime = goja.New()

	// Initialize per-window cookie jar and HTTP client for cookie isolation
	wc.cookieJar = sauce.NewOriginCookieJar()
	wc.httpClient = &http.Client{
		Jar: wc.cookieJar,
	}

	// Register this window for cross-window communication
	RegisterWindow(wc)

	return wc
}

// SetOrigin sets the origin for this window context (scheme + host + port).
func (wc *WindowContext) SetOrigin(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	wc.Origin = &url.URL{
		Scheme: parsed.Scheme,
		Host:   parsed.Host,
	}
	return nil
}

// GetOrigin returns the origin for this window context.
func (wc *WindowContext) GetOrigin() *url.URL {
	if wc.Origin == nil {
		return &url.URL{}
	}
	return wc.Origin
}

// RegisterEvent registers an event handler for this window context.
func (wc *WindowContext) RegisterEvent(eventName string, handler func(interface{})) {
	wc.eventRegistry[eventName] = append(wc.eventRegistry[eventName], handler)
}

// EmitEvent emits an event to all registered handlers for this window context.
func (wc *WindowContext) EmitEvent(eventName string, data interface{}) {
	if handlers, ok := wc.eventRegistry[eventName]; ok {
		for _, handler := range handlers {
			handler(data)
		}
	}
}

// GetLayoutCache retrieves a cached layout value.
func (wc *WindowContext) GetLayoutCache(key string) (interface{}, bool) {
	val, ok := wc.layoutCache[key]
	return val, ok
}

// SetLayoutCache stores a layout value in the cache.
func (wc *WindowContext) SetLayoutCache(key string, value interface{}) {
	wc.layoutCache[key] = value
}

// ClearLayoutCache clears the layout cache.
func (wc *WindowContext) ClearLayoutCache() {
	wc.layoutCache = make(map[string]interface{})
}

// GetJSRuntime returns the Goja runtime for this window context.
// This allows controlled access to the JS runtime for DOM bindings.
func (wc *WindowContext) GetJSRuntime() *goja.Runtime {
	return wc.jsRuntime
}

// SetFocusedElement sets the currently focused DOM element.
func (wc *WindowContext) SetFocusedElement(el *NodeDOM) {
	wc.focusedElement = el
}

// GetFocusedElement returns the currently focused DOM element.
func (wc *WindowContext) GetFocusedElement() *NodeDOM {
	return wc.focusedElement
}

// SetInputState sets a value in the input state map.
func (wc *WindowContext) SetInputState(key string, value interface{}) {
	wc.inputState[key] = value
}

// GetInputState retrieves a value from the input state map.
func (wc *WindowContext) GetInputState(key string) (interface{}, bool) {
	val, ok := wc.inputState[key]
	return val, ok
}

// Destroy cleans up the window context resources.
func (wc *WindowContext) Destroy() {
	wc.ActiveDocument = nil
	wc.Documents = nil
	wc.Viewport = nil
	wc.StatusLabel = nil
	wc.History = nil
	wc.Window = nil
	wc.Profiler = nil
	wc.Origin = nil
	wc.ClearLayoutCache()
	wc.inputState = nil
	wc.eventRegistry = nil
	wc.focusedElement = nil
	wc.jsRuntime = nil // Allow Goja VM to be garbage collected
	wc.sessionStorage.ClearAllSessionStorage()
	wc.sessionStorage = nil
	wc.cookieJar = nil
	wc.httpClient = nil
	wc.DebugWindow = nil
	wc.DebugTree = nil

	// Unregister this window from cross-window communication
	UnregisterWindow(wc)
}

// GetSessionStorageManager returns the session storage manager for this window.
func (wc *WindowContext) GetSessionStorageManager() *SessionStorageManager {
	return wc.sessionStorage
}

// GetCookieJar returns the cookie jar for this window context.
func (wc *WindowContext) GetCookieJar() *sauce.OriginCookieJar {
	return wc.cookieJar
}

// GetHTTPClient returns the HTTP client for this window context.
func (wc *WindowContext) GetHTTPClient() *http.Client {
	return wc.httpClient
}
