package hotdog

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/danfragoso/thdwb/assets"
	mustard "github.com/danfragoso/thdwb/mustard"
	profiler "github.com/danfragoso/thdwb/profiler"
	sauce "github.com/danfragoso/thdwb/sauce"
	goja "github.com/dop251/goja"
)

// goroutineID returns the ID of the calling goroutine.
// Used for reentrant locking of the per-window Goja runtime.
func goroutineID() uint64 {
	buf := make([]byte, 64)
	buf = buf[:runtime.Stack(buf, false)]
	// Format: "goroutine N ["
	buf = buf[len("goroutine "):]
	if i := bytes.IndexByte(buf, ' '); i >= 0 {
		buf = buf[:i]
	}
	id, err := strconv.ParseUint(string(buf), 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// reentrantMutex is a mutex that can be locked multiple times by the same
// goroutine. Different goroutines block until the owner releases it.
// This prevents deadlocks when JS callbacks (run under jsRuntimeMu) re-enter
// RunJS/SetJSValue on the same goroutine, while still serializing access
// across goroutines (Goja is not thread-safe).
type reentrantMutex struct {
	mu    sync.Mutex
	cond  *sync.Cond
	owner uint64
	depth int
}

func newReentrantMutex() *reentrantMutex {
	m := &reentrantMutex{}
	m.cond = sync.NewCond(&m.mu)
	return m
}

func (m *reentrantMutex) lock() {
	id := goroutineID()
	m.mu.Lock()
	for m.owner != 0 && m.owner != id {
		m.cond.Wait()
	}
	m.owner = id
	m.depth++
	m.mu.Unlock()
}

func (m *reentrantMutex) unlock() {
	id := goroutineID()
	m.mu.Lock()
	if m.owner != id {
		m.mu.Unlock()
		return
	}
	m.depth--
	if m.depth == 0 {
		m.owner = 0
		m.cond.Broadcast()
	}
	m.mu.Unlock()
}

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
	Window *mustard.WindowBase

	// Profiling
	Profiler *profiler.Profiler

	// Settings and build info (shared reference)
	Settings  *Settings
	BuildInfo *assets.BuildInfo

	// Origin tracking for cookie partitioning and SOP
	Origin *url.URL

	// ContainerID identifies the container (origin isolation group) this
	// window belongs to. An empty value means the default (non-container)
	// context.
	ContainerID string

	// Layout cache and pending invalidations
	layoutCache    map[string]interface{}
	reflowMu       sync.Mutex
	pendingReflows map[*NodeDOM]struct{}

	// Focus and input state
	focusedElement *NodeDOM
	inputState     map[string]interface{}

	// Event registry for this window
	eventMu           sync.Mutex
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
	// reentrant so JS callbacks can call RunJS/SetJSValue without deadlocking.
	jsRuntimeMu *reentrantMutex
	jsRuntime   *goja.Runtime

	// Session storage manager (per-window, origin-partitioned)
	sessionStorage *SessionStorageManager

	// Cookie jar for this window (per-container, per-origin cookie isolation)
	cookieJar *sauce.ContainerCookieJar

	// HTTP client for this window (with per-window cookie jar)
	httpClient *http.Client

	// Debug state
	DebugFlag   bool
	DebugWindow *mustard.WindowBase
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
		eventMu:           sync.Mutex{},
		eventRegistry:     make(map[string][]eventHandler),
		nodeEventRegistry: make(map[*NodeDOM]map[string][]eventHandler),
		nodeWrappers:      make(map[*NodeDOM]*goja.Object),
		mutationObservers: make(map[*MutationObserver]struct{}),
		animationFrames:   make(map[uint64]func()),
		sessionStorage:    NewSessionStorageManager(),
		active:            true,
	}

	// Initialize per-window Goja VM for JavaScript execution
	wc.jsRuntimeMu = newReentrantMutex()
	wc.jsRuntime = goja.New()

	// Initialize per-window cookie jar and HTTP client for cookie isolation
	wc.cookieJar = sauce.NewContainerCookieJar()
	wc.httpClient = &http.Client{
		Jar: sauce.NewDefaultContainerCookieJar(wc.cookieJar),
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

// LoadHTML parses the given HTML string and sets it as the active document
// for this window context. It sets the document origin from baseURL and
// initializes the JS runtime with the document's DOM.
func (wc *WindowContext) LoadHTML(html, baseURL string) error {
	if wc == nil {
		return fmt.Errorf("window context is nil")
	}
	if wc.destroyed {
		return fmt.Errorf("window context is destroyed")
	}

	doc := &Document{
		Title: "",
		URL:   &url.URL{},
		DOM:   &NodeDOM{Type: NodeTypeElement, Element: "html", Children: make([]*NodeDOM, 0)},
	}

	if baseURL != "" {
		u, err := url.Parse(baseURL)
		if err == nil {
			doc.URL = u
			wc.SetOrigin(baseURL)
		}
	}

	nodes, err := parseHTMLFragment(html, doc)
	if err != nil {
		return fmt.Errorf("parse HTML: %w", err)
	}

	for _, node := range nodes {
		doc.DOM.appendChildNode(node)
	}

	wc.ActiveDocument = doc
	wc.Documents = append(wc.Documents, doc)

	// Initialize the JS runtime with the window object
	if err := wc.InitJSRuntime(); err != nil {
		return fmt.Errorf("init JS runtime: %w", err)
	}

	return nil
}

// LoadURL fetches the given URL and loads its content into this window context.
// It uses the window's HTTP client for cookie isolation.
func (wc *WindowContext) LoadURL(rawURL string) error {
	if wc == nil {
		return fmt.Errorf("window context is nil")
	}
	if wc.destroyed {
		return fmt.Errorf("window context is destroyed")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse URL: %w", err)
	}

	resource := sauce.FetchExternalPageWithOptions(u, "GET", nil, nil)
	if resource.Code >= 400 {
		return fmt.Errorf("HTTP %d: %s", resource.Code, resource.Body)
	}

	return wc.LoadHTML(resource.Body, resource.URL.String())
}

// GetOrigin returns the origin for this window context.
func (wc *WindowContext) GetOrigin() *url.URL {
	if wc.Origin == nil {
		return &url.URL{}
	}
	return wc.Origin
}

// GetContainerID returns the container ID for this window context.
func (wc *WindowContext) GetContainerID() string {
	return wc.ContainerID
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
	wc.eventMu.Lock()
	defer wc.eventMu.Unlock()
	wc.eventRegistry[eventName] = append(wc.eventRegistry[eventName], eventHandler{key: key, handler: handler})
}

func (wc *WindowContext) registerNodeEvent(node *NodeDOM, eventName string, key uintptr, handler func(interface{})) {
	if node == nil {
		return
	}
	wc.eventMu.Lock()
	defer wc.eventMu.Unlock()
	if wc.nodeEventRegistry[node] == nil {
		wc.nodeEventRegistry[node] = make(map[string][]eventHandler)
	}
	wc.nodeEventRegistry[node][eventName] = append(wc.nodeEventRegistry[node][eventName], eventHandler{key: key, handler: handler})
}

func (wc *WindowContext) RemoveNodeEventByKey(node *NodeDOM, eventName string, key uintptr) {
	if node == nil {
		return
	}
	wc.eventMu.Lock()
	defer wc.eventMu.Unlock()
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
	if !wc.IsActive() {
		return
	}
	wc.eventMu.Lock()
	handlers := wc.nodeEventRegistry[node][eventName]
	wc.eventMu.Unlock()
	for _, handler := range handlers {
		handler.handler(data)
	}
}

func (wc *WindowContext) RemoveEventByKey(eventName string, key uintptr) {
	wc.eventMu.Lock()
	defer wc.eventMu.Unlock()
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
	wc.eventMu.Lock()
	handlers := wc.eventRegistry[eventName]
	wc.eventMu.Unlock()
	for _, handler := range handlers {
		handler.handler(data)
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
	wc.jsRuntimeMu.lock()
	defer wc.jsRuntimeMu.unlock()

	wc.animationFrameMu.Lock()
	if !wc.active || wc.jsRuntime == nil {
		wc.animationFrameTimer = nil
		wc.animationFrameMu.Unlock()
		return
	}
	wc.animationFrameTimer = nil
	wc.animationFrameMu.Unlock()

	// Run callbacks one at a time, re-checking active after each so that
	// SetActive(false) during a callback defers remaining callbacks.
	// jsRuntimeMu is held during callback execution because Goja is not
	// thread-safe. The mutex is reentrant, so callbacks that call RunJS or
	// SetJSValue do not deadlock.
	for {
		wc.animationFrameMu.Lock()
		if !wc.active || len(wc.animationFrames) == 0 {
			wc.animationFrameMu.Unlock()
			break
		}
		// Get the lowest ID to maintain insertion order.
		var minID uint64
		first := true
		for id := range wc.animationFrames {
			if first || id < minID {
				minID = id
				first = false
			}
		}
		callback := wc.animationFrames[minID]
		delete(wc.animationFrames, minID)
		wc.animationFrameMu.Unlock()

		if callback != nil {
			callback()
		}
	}
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
	wc.reflowMu.Lock()
	defer wc.reflowMu.Unlock()
	if !wc.active {
		return nil
	}
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
// jsRuntimeMu is held during script execution because Goja is not thread-safe.
// The mutex is reentrant, so JS callbacks that call SetJSValue from within
// RunJS do not deadlock.
func (wc *WindowContext) SetJSValue(name string, value interface{}) error {
	wc.jsRuntimeMu.lock()
	defer wc.jsRuntimeMu.unlock()
	if !wc.IsActive() {
		return fmt.Errorf("window context is inactive")
	}
	if wc.jsRuntime == nil {
		return fmt.Errorf("no JavaScript runtime available")
	}
	return wc.jsRuntime.Set(name, value)
}

// RunJS executes a script while serializing access to this window's Goja runtime.
// jsRuntimeMu is held during script execution because Goja is not thread-safe.
// The mutex is reentrant, so JS callbacks that call RunJS from within RunJS
// (e.g. postMessage handlers) do not deadlock.
func (wc *WindowContext) RunJS(script string) (goja.Value, error) {
	wc.jsRuntimeMu.lock()
	defer wc.jsRuntimeMu.unlock()
	if !wc.IsActive() {
		return nil, fmt.Errorf("window context is inactive")
	}
	if wc.jsRuntime == nil {
		return nil, fmt.Errorf("no JavaScript runtime available")
	}
	return wc.jsRuntime.RunString(script)
}

// BindWindowInput routes pointer clicks from a mustard window to this window context.
func (wc *WindowContext) BindWindowInput(window *mustard.WindowBase) {
	if wc == nil || window == nil {
		return
	}
	wc.Window = window
	window.AttachActiveChangeListener(wc.SetActive)
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
	wc.jsRuntimeMu.lock()
	defer wc.jsRuntimeMu.unlock()
	if wc.jsRuntime == nil || !wc.IsActive() {
		return
	}
	runtime := wc.jsRuntime
	wrapper := &JSDOMWrapper{node: target, runtime: runtime, windowCtx: wc}
	event := newJSEvent(runtime, goja.ConstructorCall{Arguments: []goja.Value{runtime.ToValue(eventType)}})
	_ = event.Set("button", button)
	_ = event.Set("clientX", x)
	_ = event.Set("clientY", y)
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
	wc.jsRuntimeMu.lock()
	defer wc.jsRuntimeMu.unlock()
	if wc.jsRuntime == nil || !wc.IsActive() {
		return
	}
	runtime := wc.jsRuntime
	options := runtime.NewObject()
	_ = options.Set("bubbles", true)
	_ = options.Set("cancelable", true)
	event := newJSEvent(runtime, goja.ConstructorCall{Arguments: []goja.Value{runtime.ToValue(eventType), options}})
	for name, value := range properties {
		_ = event.Set(name, value)
	}
	(&JSDOMWrapper{node: target, runtime: runtime, windowCtx: wc}).dispatchDOMEvent(event)
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
	wc.jsRuntimeMu.lock()
	defer wc.jsRuntimeMu.unlock()
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
func (wc *WindowContext) GetCookieJar() *sauce.ContainerCookieJar {
	return wc.cookieJar
}

// GetHTTPClient returns the HTTP client for this window context.
func (wc *WindowContext) GetHTTPClient() *http.Client {
	return wc.httpClient
}

// TransferStorage moves all storage (cookies, localStorage, sessionStorage)
// for the given origin from one container to another. This is used when a tab
// is moved to a different container.
func (wc *WindowContext) TransferStorage(fromContainer, toContainer string, origin *url.URL) error {
	if wc.cookieJar != nil {
		wc.cookieJar.TransferOrigin(fromContainer, toContainer, origin)
	}
	manager, err := getLocalStorageManager()
	if err != nil {
		return err
	}
	if err := manager.TransferOrigin(fromContainer, toContainer, origin); err != nil {
		return err
	}
	wc.sessionStorage.TransferOrigin(fromContainer, toContainer, origin)
	return nil
}

// TransferAllStorage moves all storage (cookies, localStorage, sessionStorage)
// from one container to another across all origins. This is used when a
// container is removed and its data needs to be migrated.
func (wc *WindowContext) TransferAllStorage(fromContainer, toContainer string) error {
	if wc.cookieJar != nil {
		wc.cookieJar.TransferContainer(fromContainer, toContainer)
	}
	manager, err := getLocalStorageManager()
	if err != nil {
		return err
	}
	if err := manager.TransferContainer(fromContainer, toContainer); err != nil {
		return err
	}
	wc.sessionStorage.TransferContainer(fromContainer, toContainer)
	return nil
}
