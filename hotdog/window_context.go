package hotdog

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sort"
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

	// Layout cache and pending invalidations
	layoutCache    map[string]interface{}
	reflowMu       sync.Mutex
	pendingReflows map[*NodeDOM]struct{}

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
	animationFrameTimer   *time.Timer
	active                bool
	destroyed             bool

	// JS runtime (Goja VM) - per window isolation
	jsRuntimeMu sync.Mutex
	jsRuntime   *goja.Runtime

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
		pendingReflows:    make(map[*NodeDOM]struct{}),
		inputState:        make(map[string]interface{}),
		eventRegistry:     make(map[string][]eventHandler),
		nodeEventRegistry: make(map[*NodeDOM]map[string][]eventHandler),
		nodeWrappers:      make(map[*NodeDOM]*goja.Object),
		mutationObservers: make(map[*MutationObserver]struct{}),
		animationFrames:   make(map[uint64]func()),
		sessionStorage:    NewSessionStorageManager(),
		active:            true,
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
	if !wc.IsActive() {
		return
	}
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
	if wc.destroyed {
		wc.animationFrameMu.Unlock()
		return 0
	}
	wc.animationFrameCounter++
	id := wc.animationFrameCounter
	wc.animationFrames[id] = callback
	wc.scheduleAnimationFrameLocked()
	wc.animationFrameMu.Unlock()
	return id
}

func (wc *WindowContext) scheduleAnimationFrameLocked() {
	if wc.active && wc.animationFrameTimer == nil && len(wc.animationFrames) > 0 {
		wc.animationFrameTimer = time.AfterFunc(16*time.Millisecond, wc.runAnimationFrame)
	}
}

func (wc *WindowContext) runAnimationFrame() {
	wc.jsRuntimeMu.Lock()
	defer wc.jsRuntimeMu.Unlock()
	if !wc.beginAnimationFrame() {
		return
	}
	for _, id := range wc.animationFrameIDs() {
		if !wc.runAnimationFrameCallback(id) {
			return
		}
	}
}

func (wc *WindowContext) beginAnimationFrame() bool {
	wc.animationFrameMu.Lock()
	defer wc.animationFrameMu.Unlock()
	if !wc.active || wc.jsRuntime == nil {
		wc.animationFrameTimer = nil
		return false
	}
	wc.animationFrameTimer = nil
	return true
}

func (wc *WindowContext) animationFrameIDs() []uint64 {
	wc.animationFrameMu.Lock()
	defer wc.animationFrameMu.Unlock()
	ids := make([]uint64, 0, len(wc.animationFrames))
	for id := range wc.animationFrames {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func (wc *WindowContext) runAnimationFrameCallback(id uint64) bool {
	wc.animationFrameMu.Lock()
	if !wc.active {
		wc.animationFrameMu.Unlock()
		return false
	}
	callback := wc.animationFrames[id]
	delete(wc.animationFrames, id)
	wc.animationFrameMu.Unlock()
	if callback != nil {
		callback()
	}
	return true
}

// IsActive reports whether this window is allowed to run scripts and drain layout work.
func (wc *WindowContext) IsActive() bool {
	if wc == nil {
		return false
	}
	wc.animationFrameMu.Lock()
	defer wc.animationFrameMu.Unlock()
	return wc.active
}

// SetActive pauses or resumes script callbacks and layout queue draining for this window.
func (wc *WindowContext) SetActive(active bool) {
	if wc == nil {
		return
	}
	wc.animationFrameMu.Lock()
	defer wc.animationFrameMu.Unlock()
	if wc.destroyed {
		return
	}
	if wc.active == active {
		return
	}
	wc.active = active
	if !active {
		if wc.animationFrameTimer != nil {
			wc.animationFrameTimer.Stop()
			wc.animationFrameTimer = nil
		}
		return
	}
	wc.scheduleAnimationFrameLocked()
}

// CancelAnimationFrame removes a queued frame callback for this window.
func (wc *WindowContext) CancelAnimationFrame(id uint64) bool {
	wc.animationFrameMu.Lock()
	defer wc.animationFrameMu.Unlock()
	if _, ok := wc.animationFrames[id]; !ok {
		return false
	}
	delete(wc.animationFrames, id)
	if len(wc.animationFrames) == 0 && wc.animationFrameTimer != nil {
		wc.animationFrameTimer.Stop()
		wc.animationFrameTimer = nil
	}
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

func (wc *WindowContext) queueReflow(node *NodeDOM) (*NodeDOM, bool) {
	if node == nil {
		return nil, false
	}
	root := node
	for root.Parent != nil {
		root = root.Parent
	}
	wc.reflowMu.Lock()
	defer wc.reflowMu.Unlock()
	if wc.pendingReflows == nil {
		wc.pendingReflows = make(map[*NodeDOM]struct{})
	}
	if _, exists := wc.pendingReflows[root]; exists {
		return root, false
	}
	wc.pendingReflows[root] = struct{}{}
	return root, true
}

// DrainReflowQueue returns unique DOM roots invalidated since the previous drain.
func (wc *WindowContext) DrainReflowQueue() []*NodeDOM {
	wc.animationFrameMu.Lock()
	defer wc.animationFrameMu.Unlock()
	if !wc.active {
		return nil
	}
	wc.reflowMu.Lock()
	defer wc.reflowMu.Unlock()
	roots := make([]*NodeDOM, 0, len(wc.pendingReflows))
	for root := range wc.pendingReflows {
		roots = append(roots, root)
	}
	wc.pendingReflows = make(map[*NodeDOM]struct{})
	return roots
}

// GetJSRuntime returns the Goja runtime for this window context.
// The returned runtime must not be accessed concurrently; use RunJS and SetJSValue.
func (wc *WindowContext) GetJSRuntime() *goja.Runtime {
	return wc.jsRuntime
}

// SetJSValue sets a global while serializing access to this window's Goja runtime.
func (wc *WindowContext) SetJSValue(name string, value interface{}) error {
	wc.jsRuntimeMu.Lock()
	defer wc.jsRuntimeMu.Unlock()
	if wc.jsRuntime == nil {
		return fmt.Errorf("no JavaScript runtime available")
	}
	return wc.jsRuntime.Set(name, value)
}

// RunJS executes a script while serializing access to this window's Goja runtime.
func (wc *WindowContext) RunJS(script string) (goja.Value, error) {
	wc.jsRuntimeMu.Lock()
	defer wc.jsRuntimeMu.Unlock()
	if !wc.IsActive() {
		return nil, fmt.Errorf("window context is inactive")
	}
	if wc.jsRuntime == nil {
		return nil, fmt.Errorf("no JavaScript runtime available")
	}
	return wc.jsRuntime.RunString(script)
}

// BindWindowInput routes pointer clicks from a mustard window to this window context.
func (wc *WindowContext) BindWindowInput(window *mustard.Window) {
	if wc == nil || window == nil {
		return
	}
	wc.Window = window
	window.AttachPointerClickEventListener(func(x, y float64, button mustard.MustardKey) {
		wc.DispatchMouseClick(x, y, button)
	})
	window.AttachKeyboardInputEventListener(func(input mustard.KeyboardInput) {
		wc.DispatchKeyboardInput(input)
	})
	window.AttachTextInputEventListener(func(char rune) {
		wc.DispatchTextInput(char)
	})
}

// DispatchMouseClick hit-tests this window’s active document and dispatches a DOM mouse event.
func (wc *WindowContext) DispatchMouseClick(x, y float64, button mustard.MustardKey) *NodeDOM {
	if wc == nil || !wc.IsActive() || wc.ActiveDocument == nil || wc.ActiveDocument.DOM == nil {
		return nil
	}
	eventType, buttonCode, ok := mouseClickDetails(button)
	if !ok {
		return nil
	}
	target := wc.ActiveDocument.DOM.CalcPointIntersection(x, y)
	if target != nil {
		wc.dispatchMouseEventAt(target, eventType, buttonCode, x, y)
	}
	return target
}

func mouseClickDetails(button mustard.MustardKey) (string, int, bool) {
	switch button {
	case mustard.MouseLeft:
		return "click", 0, true
	case mustard.MouseRight:
		return "contextmenu", 2, true
	default:
		return "", 0, false
	}
}

func (wc *WindowContext) dispatchMouseEventAt(target *NodeDOM, eventType string, button int, x, y float64) {
	wc.jsRuntimeMu.Lock()
	defer wc.jsRuntimeMu.Unlock()
	if wc.jsRuntime == nil || !wc.IsActive() {
		return
	}
	wrapper := &JSDOMWrapper{node: target, runtime: wc.jsRuntime, windowCtx: wc}
	event := newJSEvent(wc.jsRuntime, goja.ConstructorCall{Arguments: []goja.Value{wc.jsRuntime.ToValue(eventType)}})
	event.Set("button", button)
	event.Set("clientX", x)
	event.Set("clientY", y)
	wrapper.dispatchDOMEvent(event)
}

func (wc *WindowContext) DispatchKeyboardInput(input mustard.KeyboardInput) *NodeDOM {
	target := wc.focusedDOMNode()
	if target == nil {
		return nil
	}
	eventType := "keydown"
	if input.Action == mustard.KeyboardRelease {
		eventType = "keyup"
	} else if input.Action != mustard.KeyboardPress && input.Action != mustard.KeyboardRepeat {
		return nil
	}
	properties := map[string]interface{}{
		"key": input.Key, "repeat": input.Action == mustard.KeyboardRepeat,
		"ctrlKey": input.Control, "shiftKey": input.Shift, "altKey": input.Alt, "metaKey": input.Super,
	}
	wc.dispatchKeyboardEventAt(target, eventType, properties)
	return target
}

func (wc *WindowContext) DispatchTextInput(char rune) *NodeDOM {
	target := wc.focusedDOMNode()
	if target == nil {
		return nil
	}
	data := string(char)
	wc.dispatchKeyboardEventAt(target, "keypress", map[string]interface{}{"key": data, "charCode": int(char)})
	wc.dispatchKeyboardEventAt(target, "input", map[string]interface{}{"data": data, "inputType": "insertText"})
	return target
}

func (wc *WindowContext) focusedDOMNode() *NodeDOM {
	if wc == nil || wc.focusedElement == nil || wc.focusedElement.WindowCtx != wc {
		return nil
	}
	return wc.focusedElement
}

func (wc *WindowContext) dispatchKeyboardEventAt(target *NodeDOM, eventType string, properties map[string]interface{}) {
	wc.jsRuntimeMu.Lock()
	defer wc.jsRuntimeMu.Unlock()
	if wc.jsRuntime == nil || !wc.IsActive() {
		return
	}
	options := wc.jsRuntime.NewObject()
	options.Set("bubbles", true)
	options.Set("cancelable", true)
	event := newJSEvent(wc.jsRuntime, goja.ConstructorCall{Arguments: []goja.Value{wc.jsRuntime.ToValue(eventType), options}})
	for name, value := range properties {
		event.Set(name, value)
	}
	(&JSDOMWrapper{node: target, runtime: wc.jsRuntime, windowCtx: wc}).dispatchDOMEvent(event)
}

// SetFocusedElement sets the currently focused DOM element.
func (wc *WindowContext) SetFocusedElement(el *NodeDOM) {
	if wc.focusedElement == el {
		return
	}
	if wc.focusedElement != nil {
		wc.focusedElement.RequestRepaint()
	}
	wc.focusedElement = el
	if el != nil {
		el.RequestRepaint()
	}
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
	wc.jsRuntimeMu.Lock()
	defer wc.jsRuntimeMu.Unlock()
	wc.ActiveDocument = nil
	wc.Documents = nil
	wc.Viewport = nil
	wc.StatusLabel = nil
	wc.History = nil
	wc.Window = nil
	wc.Profiler = nil
	wc.Origin = nil
	wc.ClearLayoutCache()
	wc.reflowMu.Lock()
	wc.pendingReflows = nil
	wc.reflowMu.Unlock()
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
	wc.destroyed = true
	wc.active = false
	if wc.animationFrameTimer != nil {
		wc.animationFrameTimer.Stop()
		wc.animationFrameTimer = nil
	}
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
