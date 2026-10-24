# Chapter 7: JavaScript Runtime — hotdog/js_bindings

> **In this chapter**: How thdwb runs JavaScript using the Goja engine, binds DOM APIs (document, window, console), handles events, and implements Fetch/XHR. This is the "brain" of the browser.

---

## 7.1 What Is a JavaScript Engine?

A **JavaScript engine** reads JavaScript code and executes it. It's the "brain" that makes web pages interactive.

**Analogy**: HTML is the skeleton, CSS is the skin, JavaScript is the muscles and nerves. The JS engine is the nervous system — it receives signals (events), processes them, and sends commands (DOM changes, network requests).

```
JAVASCRIPT CODE                    GOJA ENGINE                    BROWSER ACTION
──────────────────                 ─────────────                  ──────────────
document.getElementById      1. Parse (AST)                  2. Call Go function
  ("btn").addEventListener         │                              │
  ("click", () => {              ▼                              ▼
    fetch("/api/data")      3. Execute bytecode          4. Native Go code runs:
      .then(r => r.json())       │                          - DOM mutation
      .then(d => console.log(d)) ▼                          - Network request
  });                      5. Return value               - Console output
```

---

## 7.2 Key Terms (Defined in [Glossary](../glossary.md))

- **JS Engine** — Program that executes JavaScript (V8, SpiderMonkey, Goja)
- **Goja** — Pure Go JavaScript engine (ES5.1+)
- **VM (Virtual Machine)** — Isolated JS execution environment
- **Binding** — Connecting JS functions to Go functions
- **DOM API** — JavaScript interface to read/change HTML
- **Event Loop** — Queue of tasks (macrotasks, microtasks)
- **Promise** — Async result placeholder
- **Fetch/XHR** — APIs for network requests
- **Same-Origin Policy** — Security: scripts can't cross origins
- **CSP** — Content Security Policy (restricts what JS can do)

---

## 7.3 The hotdog Package: JS Runtime

### File Overview

```
hotdog/
├── js_bindings.go      # Main: Goja VM setup, DOM bindings, APIs
├── node.go             # NodeDOM (DOM tree) - also used by parser
├── window_context.go   # WindowContext (per-tab isolation)
├── structs.go          # Shared structs (Document, RenderBox, etc.)
├── settings.go         # Browser settings
├── xpath.go            # XPath support
├── storage.go          # localStorage/sessionStorage
├── window_context_test.go
└── *_test.go
```

---

## 7.4 Goja: Pure Go JavaScript Engine

thdwb uses **Goja** (dop251/goja) — a pure Go implementation of ECMAScript 5.1+.

### Why Goja?
- **Pure Go** — No CGO, cross-compiles easily
- **Embeddable** — Designed for embedding in Go apps
- **ES5.1+** — Supports modern JS (Promises, classes, modules partially)
- **No JIT** — Interpreter only (slower than V8 but simpler)

**Reference**: [Goja GitHub](https://github.com/dop251/goja) | [MDN - JS Engine](https://developer.mozilla.org/en-US/docs/Glossary/JavaScript_engine)

---

## 7.5 Per-Window VM Isolation

**Critical security feature**: Each browser window/tab gets its **own Goja VM**.

```go
// WindowContext holds all state for a single browser window/tab instance.
// This enables multiple independent windows to run without memory interference.
// Each window has its own:
// - Goja VM (JavaScript runtime)
// - DOM tree
// - Event loop
// - Storage (cookies, localStorage, sessionStorage)
// - Network client (isolated cookie jar)
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
```

### Why isolate?
- Tab A: `example.com` — logs in, sets cookies, runs JS
- Tab B: `evil.com` — CANNOT access Tab A's VM, DOM, cookies, localStorage
- This is **Container Tabs** / **Site Isolation** (Firefox/Chrome feature)

**Reference**: [Firefox Site Isolation](https://searchfox.org/mozilla-central/source/dom/security) | [MDN - Same-Origin Policy](https://developer.mozilla.org/en-US/docs/Web/Security/Same-origin_policy)

---

## 7.6 Creating a VM and Global Objects

### NewWindowContext: VM Setup

```go
func NewWindowContext(settings *Settings, buildInfo *assets.BuildInfo, profiler *profiler.Profiler) *WindowContext
```

Creates a new isolated window context with its own Goja VM, cookie jar, HTTP client, session storage, and event registry.

### InitJSRuntime: Setting Up Globals

```go
func (wc *WindowContext) InitJSRuntime() error
```

Initializes the JavaScript runtime with DOM bindings. Sets up the following global objects:

| Global | Description |
|--------|-------------|
| `Event` | Event constructor for creating JS events |
| `console` | Object with `log()`, `error()`, `warn()` methods |
| `document` | DOM document with querySelector, getElementById, createElement, etc. |
| `window` | Window object with document, console, location, addEventListener, etc. |
| `self` | Alias for `window` (window.self === window) |
| `localStorage` | Origin-partitioned persistent storage |
| `sessionStorage` | Per-window, per-origin session storage |
| `fetch` | Fetch API bridge |
| `XMLHttpRequest` | XHR constructor |
| `requestAnimationFrame` | Frame scheduling callback |
| `cancelAnimationFrame` | Cancel scheduled frame |
| `MutationObserver` | DOM mutation observer constructor |
| `postMessage` | Inter-window messaging |
| `eval` | Blocked unless CSP allows 'unsafe-eval' |
| `Function` | Blocked unless CSP allows 'unsafe-eval' |

### Example: Initializing the JS Runtime

```go
wc := hotdog.NewWindowContext(settings, buildInfo, profiler)
wc.SetOrigin("https://example.com")
if err := wc.InitJSRuntime(); err != nil {
    log.Fatal(err)
}

// Run JavaScript
result, err := wc.RunJS("console.log('Hello from JS!'); document.title = 'Updated';")
if err != nil {
    log.Fatal(err)
}
```

---

## 7.7 DOM Bindings: document, Element, Node

### document Object

The `document` object exposes the following methods:

| Method | Description |
|--------|-------------|
| `querySelector(selector)` | Returns first matching element using cascadia selector |
| `querySelectorAll(selector)` | Returns all matching elements |
| `getElementById(id)` | Returns element by ID |
| `getElementsByClassName(name)` | Returns elements by class name |
| `getElementsByTagName(name)` | Returns elements by tag name |
| `createElement(tagName)` | Creates a new element node |
| `createTextNode(text)` | Creates a new text node |
| `body` | Accessor property for the body element |
| `documentElement` | Accessor property for the root element |
| `title` | Accessor property for the document title |

### Element/Node Binding

DOM nodes are wrapped in `JSDOMWrapper` objects that expose:

| Property/Method | Description |
|-----------------|-------------|
| `id` | Element ID attribute |
| `className` | Element class attribute |
| `innerHTML` | Element inner HTML |
| `textContent` | Element text content |
| `style` | Element style properties |
| `addEventListener(type, listener)` | Registers an event listener |
| `removeEventListener(type, listener)` | Removes an event listener |

All DOM operations enforce origin checks and throw `SecurityError` on cross-origin violations.

---

## 7.8 Window Object

The `window` object exposes:

| Property/Method | Description |
|-----------------|-------------|
| `document` | Reference to the document object |
| `console` | Reference to the console object |
| `location` | Location accessor (getter) |
| `addEventListener(type, listener)` | Registers a window-level event listener |
| `removeEventListener(type, listener)` | Removes a window-level event listener |
| `requestAnimationFrame(callback)` | Schedules an animation frame callback |
| `cancelAnimationFrame(id)` | Cancels a scheduled animation frame |
| `Event` | Event constructor |
| `MutationObserver` | MutationObserver constructor |
| `postMessage(data, targetOrigin)` | Sends a message to another window |
| `fetch(input, init)` | Fetch API |
| `XMLHttpRequest` | XHR constructor |
| `localStorage` | Origin-partitioned storage |
| `sessionStorage` | Per-window session storage |

---

## 7.9 Console API

```go
type JSConsoleWrapper struct {
    windowCtx *WindowContext
    runtime   *goja.Runtime
}
```

| Method | Description |
|--------|-------------|
| `console.log(...args)` | Logs output with origin attribution |
| `console.error(...args)` | Logs error output with origin attribution |
| `console.warn(...args)` | Logs warning output with origin attribution |

Output format: `[console.log] [origin: <origin>] <args>`

---

## 7.10 Event System

### Event Object

```go
func newJSEvent(runtime *goja.Runtime, call goja.ConstructorCall) *goja.Object
```

Creates a JS event object with:
- `type` — Event type string
- `bubbles` — Whether event bubbles (default: false)
- `cancelable` — Whether event is cancelable (default: false)
- `defaultPrevented` — Whether preventDefault was called (default: false)
- `cancelBubble` — Whether stopPropagation was called (default: false)
- `preventDefault()` — Prevents default action if cancelable
- `stopPropagation()` — Stops event bubbling

### Event Listener Registration

| Method | Description |
|--------|-------------|
| `(wc *WindowContext) RegisterEvent(eventName string, handler func(interface{}))` | Registers a window-level event handler |
| `(wc *WindowContext) RemoveEventByKey(eventName string, key uintptr)` | Removes a window-level event handler |
| `(wc *WindowContext) EmitEvent(eventName string, data interface{})` | Emits an event to all registered handlers |
| `(wc *WindowContext) registerNodeEvent(node, eventName, key, handler)` | Registers a node-level event handler |
| `(wc *WindowContext) RemoveNodeEventByKey(node, eventName, key)` | Removes a node-level event handler |
| `(wc *WindowContext) EmitNodeEvent(node, eventName, data)` | Emits an event to node handlers |

Events are delivered only to nodes within the same window context.

---

## 7.11 Animation Frames

| Method | Description |
|--------|-------------|
| `(wc *WindowContext) ScheduleAnimationFrame(callback func()) uint64` | Schedules an animation frame callback |
| `(wc *WindowContext) CancelAnimationFrame(id uint64) bool` | Cancels a scheduled animation frame |

Animation frames are per-window and fire in isolation. Each window maintains its own animation frame timer.

---

## 7.12 Mutation Observers

```go
type MutationObserver struct {
    runtime   *goja.Runtime
    windowCtx *WindowContext
    callback  goja.Callable
    observed  map[*NodeDOM]struct{}
}
```

| Method | Description |
|--------|-------------|
| `observe(target, options)` | Starts observing a DOM node for mutations |
| `disconnect()` | Stops observing all nodes |
| `takeRecords()` | Returns mutation records (empty array) |

Mutations are observable only within the same window context.

---

## 7.13 Storage APIs

### localStorage

Persistent per-origin key-value store, accessible only to scripts from that origin. Survives across window reloads.

### sessionStorage

Per-window, per-origin key-value store. Cleared when the window is closed.

Both are exposed as global objects on `window` and as standalone globals.

---

## 7.14 Inter-Window Messaging

```go
(window *WindowWrapper) postMessage(data, targetOrigin)
```

Sends a message to another window. Validates `targetOrigin` and prevents unsolicited messages. Uses the global window registry to find matching windows by origin.

---

## 7.15 Fetch API Bridge

```go
(window *WindowWrapper) fetch(input, init)
(window *WindowWrapper) XMLHttpRequest
```

Implements Fetch API and XMLHttpRequest with:
- Enforced CORS preflight
- Origin header validation
- Cookie jar isolation (per-window cookie jar)
- Header filtering

---

## 7.16 Security: Dangerous APIs

| API | Behavior |
|-----|----------|
| `eval()` | Blocked unless CSP allows `script-src 'unsafe-eval'` |
| `Function()` constructor | Blocked unless CSP allows `script-src 'unsafe-eval'` |

Both throw `SecurityError` when blocked.

---

## 7.17 WindowContext Methods

| Method | Description |
|--------|-------------|
| `SetOrigin(rawURL string) error` | Sets the origin for this window context |
| `GetOrigin() *url.URL` | Returns the origin |
| `RegisterEvent(eventName string, handler func(interface{}))` | Registers a window event handler |
| `RemoveEventByKey(eventName string, key uintptr)` | Removes a window event handler |
| `EmitEvent(eventName string, data interface{})` | Emits a window event |
| `RegisterMutationObserver(observer *MutationObserver)` | Registers a mutation observer |
| `UnregisterMutationObserver(observer *MutationObserver)` | Unregisters a mutation observer |
| `NotifyMutationObservers(target *NodeDOM, mutationType string)` | Notifies observers of a mutation |
| `ScheduleAnimationFrame(callback func()) uint64` | Schedules an animation frame |
| `CancelAnimationFrame(id uint64) bool` | Cancels an animation frame |
| `IsActive() bool` | Returns whether the window is active |
| `SetActive(active bool)` | Sets the window active state |
| `GetLayoutCache(key string) (interface{}, bool)` | Gets a layout cache entry |
| `SetLayoutCache(key string, value interface{})` | Sets a layout cache entry |
| `ClearLayoutCache()` | Clears the layout cache |
| `GetJSRuntime() *goja.Runtime` | Returns the Goja VM |
| `SetJSValue(name string, value interface{}) error` | Sets a global JS value |
| `RunJS(script string) (goja.Value, error)` | Runs JavaScript code |
| `BindWindowInput(window *mustard.Window)` | Binds window input events |
| `DispatchMouseClick(x, y float64, button mustard.MustardKey) *NodeDOM` | Dispatches a mouse click event |
| `DispatchKeyboardInput(input mustard.KeyboardInput) *NodeDOM` | Dispatches a keyboard input event |
| `DispatchTextInput(char rune) *NodeDOM` | Dispatches a text input event |
| `SetFocusedElement(el *NodeDOM)` | Sets the focused DOM element |
| `GetFocusedElement() *NodeDOM` | Returns the focused DOM element |
| `SetInputState(key string, value interface{})` | Sets input state |
| `GetInputState(key string) (interface{}, bool)` | Gets input state |
| `Destroy()` | Destroys the window context |
| `GetSessionStorageManager() *SessionStorageManager` | Returns session storage manager |
| `GetCookieJar() *sauce.OriginCookieJar` | Returns the cookie jar |
| `GetHTTPClient() *http.Client` | Returns the HTTP client |

---

## 7.18 Example: Running JavaScript with DOM Access

```go
wc := hotdog.NewWindowContext(settings, buildInfo, profiler)
wc.SetOrigin("https://example.com")
wc.InitJSRuntime()

// Create a div element via JS
_, err := wc.RunJS(`
    var div = document.createElement('div');
    div.id = 'my-div';
    div.textContent = 'Hello from JavaScript';
    console.log('Created div with id:', div.id);
`)
if err != nil {
    log.Fatal(err)
}

// Set up an event listener
wc.RegisterEvent("click", func(data interface{}) {
    println("Click event received:", data)
})

// Run JS that triggers the event
wc.RunJS("window.dispatchEvent(new Event('click'));")
```

---

## 7.19 Cross-References

| Concept | Chapter |
|---------|---------|
| DOM tree structure | [Chapter 3: HTML Parsing & DOM](../03-html-parsing-dom.md) |
| CSS styling | [Chapter 4: CSS Styling](../04-css-styling.md) |
| Layout engine | [Chapter 5: Layout Engine](../05-layout-engine.md) |
| UI toolkit | [Chapter 8: UI Toolkit & Application](../08-ui-toolkit.md) |
| Network & cookies | [Chapter 2: Network Layer](../02-network-layer.md) |
| Security (SOP, CORS, CSP) | [Chapter 6: Rendering & Graphics](../06-rendering-graphics.md) |

---

## 7.20 Summary

- **Goja** = Pure Go JavaScript engine (ES5.1+)
- **WindowContext** = Per-window isolation with own VM, DOM, storage, network
- **InitJSRuntime** = Sets up global objects: Event, console, document, window, localStorage, sessionStorage, fetch, XMLHttpRequest, requestAnimationFrame, MutationObserver, postMessage
- **DOM bindings** = querySelector, getElementById, createElement, createTextNode, element properties (id, className, innerHTML, textContent, style)
- **Events** = Event constructor, addEventListener/removeEventListener, preventDefault, stopPropagation
- **Animation frames** = requestAnimationFrame/cancelAnimationFrame per window
- **Mutation observers** = Observe DOM changes within window
- **Storage** = localStorage (persistent per-origin), sessionStorage (per-window per-origin)
- **postMessage** = Inter-window messaging with origin validation
- **Fetch/XHR** = CORS-enforced network requests with cookie jar isolation
- **Security** = eval() and Function() blocked unless CSP allows 'unsafe-eval'; cross-origin DOM access throws SecurityError

---

## 7.21 Further Reading

- **Goja (JavaScript engine)**: https://github.com/dop251/goja
- **MDN - JavaScript**: https://developer.mozilla.org/en-US/docs/Web/JavaScript
- **MDN - DOM**: https://developer.mozilla.org/en-US/docs/Web/API/Document_Object_Model
- **MDN - Fetch API**: https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API
- **MDN - Same-Origin Policy**: https://developer.mozilla.org/en-US/docs/Web/Security/Same-origin_policy
- **MDN - Content Security Policy**: https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP
