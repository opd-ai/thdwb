package hotdog

import (
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"time"

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

type eventHandler struct {
	key     uintptr
	handler func(interface{})
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
	eventRegistry     map[string][]eventHandler
	nodeEventRegistry map[*NodeDOM]map[string][]eventHandler
	nodeWrappers      map[*NodeDOM]*goja.Object

	// Mutation observers for this window.
	mutationObserverMu sync.Mutex
	mutationObservers  map[*MutationObserver]struct{}

	// Animation frame callbacks for this window, stored per-window for isolation.
	animationFrameMu      sync.Mutex
	animationFrameCounter uint64
	animationFrames       map[uint64]func()

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
		ActiveDocument:    &Document{},
		Documents:         make([]*Document, 0),
		History:           &History{},
		Profiler:          profiler,
		Settings:          settings,
		BuildInfo:         buildInfo,
		layoutCache:       make(map[string]interface{}),
		inputState:        make(map[string]interface{}),
		eventRegistry:     make(map[string][]eventHandler),
		nodeEventRegistry: make(map[*NodeDOM]map[string][]eventHandler),
		nodeWrappers:      make(map[*NodeDOM]*goja.Object),
		mutationObservers: make(map[*MutationObserver]struct{}),
		animationFrames:   make(map[uint64]func()),
		sessionStorage:    NewSessionStorageManager(),
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

func functionKey(fn interface{}) uintptr {
	if fn == nil {
		return 0
	}
	value := reflect.ValueOf(fn)
	if !value.IsValid() || value.Kind() != reflect.Func {
		return 0
	}
	return value.Pointer()
}

// RegisterEvent registers an event handler for this window context.
func (wc *WindowContext) RegisterEvent(eventName string, handler func(interface{})) {
	wc.registerEvent(eventName, functionKey(handler), handler)
}

func (wc *WindowContext) registerEvent(eventName string, key uintptr, handler func(interface{})) {
	wc.eventRegistry[eventName] = append(wc.eventRegistry[eventName], eventHandler{key: key, handler: handler})
}

func (wc *WindowContext) registerNodeEvent(node *NodeDOM, eventName string, key uintptr, handler func(interface{})) {
	if node == nil {
		return
	}
	if wc.nodeEventRegistry[node] == nil {
		wc.nodeEventRegistry[node] = make(map[string][]eventHandler)
	}
	wc.nodeEventRegistry[node][eventName] = append(wc.nodeEventRegistry[node][eventName], eventHandler{key: key, handler: handler})
}

func (wc *WindowContext) RemoveNodeEventByKey(node *NodeDOM, eventName string, key uintptr) {
	handlers := wc.nodeEventRegistry[node][eventName]
	filtered := handlers[:0]
	for _, handler := range handlers {
		if handler.key != key {
			filtered = append(filtered, handler)
		}
	}
	if len(filtered) == 0 {
		delete(wc.nodeEventRegistry[node], eventName)
	} else {
		wc.nodeEventRegistry[node][eventName] = filtered
	}
	if len(wc.nodeEventRegistry[node]) == 0 {
		delete(wc.nodeEventRegistry, node)
	}
}

func (wc *WindowContext) EmitNodeEvent(node *NodeDOM, eventName string, data interface{}) {
	for _, handler := range wc.nodeEventRegistry[node][eventName] {
		handler.handler(data)
	}
}

func (wc *WindowContext) RemoveEventByKey(eventName string, key uintptr) {
	handlers := wc.eventRegistry[eventName]
	filtered := handlers[:0]
	for _, handler := range handlers {
		if handler.key != key {
			filtered = append(filtered, handler)
		}
	}
	if len(filtered) == 0 {
		delete(wc.eventRegistry, eventName)
		return
	}
	wc.eventRegistry[eventName] = filtered
}

// EmitEvent emits an event to all registered handlers for this window context.
func (wc *WindowContext) EmitEvent(eventName string, data interface{}) {
	if handlers, ok := wc.eventRegistry[eventName]; ok {
		for _, handler := range handlers {
			handler.handler(data)
		}
	}
}

func (wc *WindowContext) RegisterMutationObserver(observer *MutationObserver) {
	if observer == nil {
		return
	}
	wc.mutationObserverMu.Lock()
	defer wc.mutationObserverMu.Unlock()
	wc.mutationObservers[observer] = struct{}{}
}

func (wc *WindowContext) UnregisterMutationObserver(observer *MutationObserver) {
	if observer == nil {
		return
	}
	wc.mutationObserverMu.Lock()
	defer wc.mutationObserverMu.Unlock()
	delete(wc.mutationObservers, observer)
}

func (wc *WindowContext) NotifyMutationObservers(target *NodeDOM, mutationType string) {
	if target == nil {
		return
	}
	wc.mutationObserverMu.Lock()
	observers := make([]*MutationObserver, 0, len(wc.mutationObservers))
	for observer := range wc.mutationObservers {
		if _, ok := observer.observed[target]; ok {
			observers = append(observers, observer)
		}
	}
	wc.mutationObserverMu.Unlock()

	for _, observer := range observers {
		observer.Trigger(target, mutationType)
	}
}

// ScheduleAnimationFrame queues a callback to run on the next animation frame for this window.
func (wc *WindowContext) ScheduleAnimationFrame(callback func()) uint64 {
	wc.animationFrameMu.Lock()
	wc.animationFrameCounter++
	id := wc.animationFrameCounter
	wc.animationFrames[id] = callback
	wc.animationFrameMu.Unlock()

	time.AfterFunc(16*time.Millisecond, func() {
		wc.animationFrameMu.Lock()
		cb, ok := wc.animationFrames[id]
		if ok {
			delete(wc.animationFrames, id)
		}
		wc.animationFrameMu.Unlock()
		if ok {
			cb()
		}
	})
	return id
}

// CancelAnimationFrame removes a queued frame callback for this window.
func (wc *WindowContext) CancelAnimationFrame(id uint64) bool {
	wc.animationFrameMu.Lock()
	defer wc.animationFrameMu.Unlock()
	if _, ok := wc.animationFrames[id]; !ok {
		return false
	}
	delete(wc.animationFrames, id)
	return true
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
	wc.nodeEventRegistry = nil
	wc.nodeWrappers = nil
	wc.mutationObserverMu.Lock()
	for observer := range wc.mutationObservers {
		observer.disconnect()
	}
	wc.mutationObservers = nil
	wc.mutationObserverMu.Unlock()
	wc.animationFrameMu.Lock()
	for id := range wc.animationFrames {
		delete(wc.animationFrames, id)
	}
	wc.animationFrames = nil
	wc.animationFrameMu.Unlock()
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
