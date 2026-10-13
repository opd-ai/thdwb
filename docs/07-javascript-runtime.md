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
    VM           *goja.Runtime        // JavaScript virtual machine
    Document     *Document            // DOM document
    Origin       string               // Scheme://host:port
    EventLoop    *EventLoop           // JS event loop
    Storage      *StoragePartition    // localStorage/sessionStorage
    HTTPClient   *http.Client         // Isolated network client
    // ... other window-specific state
}
```

**Why isolate?**
- Tab A: `example.com` — logs in, sets cookies, runs JS
- Tab B: `evil.com` — CANNOT access Tab A's VM, DOM, cookies, localStorage
- This is **Container Tabs** / **Site Isolation** (Firefox/Chrome feature)

**Reference**: [Firefox Site Isolation](https://searchfox.org/mozilla-central/source/dom/security) | [MDN - Same-Origin Policy](https://developer.mozilla.org/en-US/docs/Web/Security/Same-origin_policy)

---

## 7.6 Creating a VM and Global Objects

### NewWindowContext: VM Setup

```go
// NewWindowContext creates a new isolated window context with its own Goja VM.
func NewWindowContext() *WindowContext {
    ctx := &WindowContext{
        VM:        goja.New(),           // NEW VM per window!
        EventLoop: NewEventLoop(),
        Storage:   NewStoragePartition(),
    }

    // Set up global objects: window, document, console, etc.
    ctx.setupGlobals()

    return ctx
}
```

### Setting Up Globals

```go
func (ctx *WindowContext) setupGlobals() {
    vm := ctx.VM

    // 1. console object
    consoleObj := vm.NewObject()
    consoleObj.Set("log", ctx.consoleLog)
    consoleObj.Set("error", ctx.consoleError)
    consoleObj.Set("warn", ctx.consoleWarn)
    vm.Set("console", consoleObj)

    // 2. window object (globalThis)
    windowObj := vm.NewObject()
    windowObj.Set("document", ctx.Document.toJSValue(vm))
    windowObj.Set("location", ctx.locationToJS(vm))
    windowObj.Set("navigator", ctx.navigatorToJS(vm))
    windowObj.Set("localStorage", ctx.Storage.LocalStorage.toJSValue(vm))
    windowObj.Set("sessionStorage", ctx.Storage.SessionStorage.toJSValue(vm))
    windowObj.Set("fetch", ctx.fetch)
    windowObj.Set("XMLHttpRequest", ctx.XMLHttpRequestConstructor)
    windowObj.Set("setTimeout", ctx.setTimeout)
    windowObj.Set("setInterval", ctx.setInterval)
    windowObj.Set("clearTimeout", ctx.clearTimeout)
    windowObj.Set("clearInterval", ctx.clearInterval)
    windowObj.Set("addEventListener", ctx.addEventListener)
    windowObj.Set("removeEventListener", ctx.removeEventListener)
    windowObj.Set("postMessage", ctx.postMessage)
    // ... more

    vm.Set("window", windowObj)
    vm.Set("self", windowObj)  // window === self
    vm.Set("globalThis", windowObj)
}
```

---

## 7.7 DOM Bindings: document, Element, Node

### Document Binding

```go
func (doc *Document) toJSValue(vm *goja.Runtime) goja.Value {
    obj := vm.NewObject()

    // Properties
    obj.Set("title", doc.Title)
    obj.Set("URL", doc.URL)
    obj.Set("readyState", "complete")

    // Methods
    obj.Set("getElementById", func(id string) goja.Value {
        node := doc.GetElementById(id)
        if node == nil {
            return goja.Null()
        }
        return node.toJSValue(vm)
    })

    obj.Set("querySelector", func(selector string) goja.Value {
        node, _ := doc.DOM.QuerySelector(selector)
        if node == nil {
            return goja.Null()
        }
        return node.toJSValue(vm)
    })

    obj.Set("querySelectorAll", func(selector string) goja.Value {
        nodes, _ := doc.DOM.QuerySelectorAll(selector)
        arr := vm.NewArray()
        for i, n := range nodes {
            arr.Set(strconv.Itoa(i), n.toJSValue(vm))
        }
        return arr
    })

    obj.Set("createElement", func(tagName string) goja.Value {
        node := &NodeDOM{
            Element: tagName,
            Type:    NodeTypeElement,
            Document: doc,
            Style:   mayo.GetElementStylesheet(tagName, nil),
            RenderBox: &RenderBox{},
        }
        return node.toJSValue(vm)
    })

    obj.Set("createTextNode", func(text string) goja.Value {
        node := &NodeDOM{
            Element: "html:text",
            Type:    NodeTypeText,
            Content: text,
            Document: doc,
        }
        return node.toJSValue(vm)
    })

    return obj
}
```

### Element/Node Binding

```go
func (node *NodeDOM) toJSValue(vm *goja.Runtime) goja.Value {
    obj := vm.NewObject()

    // Properties
    obj.Set("tagName", strings.ToUpper(node.Element))
    obj.Set("id", node.GetAttribute("id"))
    obj.Set("className", node.GetAttribute("class"))
    obj.Set("textContent", node.TextContent())
    obj.Set("innerHTML", node.InnerHTML())

    // Style property (CSSStyleDeclaration-like)
    styleObj := vm.NewObject()
    for prop, val := range node.Style.Properties {
        styleObj.Set(prop, val)
    }
    obj.Set("style", styleObj)

    // Methods
    obj.Set("setAttribute", func(name, value string) {
        node.SetAttribute(name, value)
        node.MarkDirty()  // Trigger relayout
    })

    obj.Set("getAttribute", func(name string) string {
        return node.GetAttribute(name)
    })

    obj.Set("removeAttribute", func(name string) {
        node.RemoveAttribute(name)
        node.MarkDirty()
    })

    obj.Set("appendChild", func(child goja.Value) goja.Value {
        childNode := child.ToObject(ctx).Export().(*NodeDOM)
        node.AppendChild(childNode)
        node.MarkDirty()
        return child
    })

    obj.Set("removeChild", func(child goja.Value) goja.Value {
        childNode := child.ToObject(ctx).Export().(*NodeDOM)
        node.RemoveChild(childNode)
        node.MarkDirty()
        return child
    })

    obj.Set("addEventListener", node.addEventListener)
    obj.Set("removeEventListener", node.removeEventListener)
    obj.Set("dispatchEvent", node.dispatchEvent)

    // Query selector on element
    obj.Set("querySelector", func(selector string) goja.Value {
        found, _ := node.QuerySelector(selector)
        if found == nil {
            return goja.Null()
        }
        return found.toJSValue(vm)
    })

    obj.Set("querySelectorAll", func(selector string) goja.Value {
        nodes, _ := node.QuerySelectorAll(selector)
        arr := vm.NewArray()
        for i, n := range nodes {
            arr.Set(strconv.Itoa(i), n.toJSValue(vm))
        }
        return arr
    })

    return obj
}
```

---

## 7.8 Console API: Logging from JS

```go
func (ctx *WindowContext) consoleLog(call goja.FunctionCall) goja.Value {
    args := make([]string, len(call.Arguments))
    for i, arg := range call.Arguments {
        args[i] = arg.String()
    }
    // Log with origin attribution
    log.Printf("[JS:%s] %s", ctx.Origin, strings.Join(args, " "))
    return goja.Undefined()
}

func (ctx *WindowContext) consoleError(call goja.FunctionCall) goja.Value {
    args := make([]string, len(call.Arguments))
    for i, arg := range call.Arguments {
        args[i] = arg.String()
    }
    log.Printf("[JS ERROR:%s] %s", ctx.Origin, strings.Join(args, " "))
    return goja.Undefined()
}

func (ctx *WindowContext) consoleWarn(call goja.FunctionCall) goja.Value {
    args := make([]string, len(call.Arguments))
    for i, arg := range call.Arguments {
        args[i] = arg.String()
    }
    log.Printf("[JS WARN:%s] %s", ctx.Origin, strings.Join(args, " "))
    return goja.Undefined()
}
```

**Output example:**
```
[JS:https://example.com] Page loaded
[JS ERROR:https://example.com] Failed to fetch: network error
```

---

## 7.9 Event System: addEventListener

### EventTarget Interface

```go
// EventTarget is implemented by NodeDOM, WindowContext, etc.
type EventTarget interface {
    AddEventListener(eventType string, listener goja.Value)
    RemoveEventListener(eventType string, listener goja.Value)
    DispatchEvent(event *Event) bool
}
```

### Adding Listeners

```go
func (node *NodeDOM) addEventListener(call goja.FunctionCall) goja.Value {
    eventType := call.Argument(0).String()
    listener := call.Argument(1)

    // Validate: listener must be a function
    if goja.IsCallable(listener) == false {
        panic(vm.NewTypeError("listener must be a function"))
    }

    // Store in node's event listener map
    if node.EventListeners == nil {
        node.EventListeners = make(map[string][]goja.Value)
    }
    node.EventListeners[eventType] = append(node.EventListeners[eventType], listener)

    return goja.Undefined()
}
```

### Dispatching Events

```go
func (node *NodeDOM) dispatchEvent(call goja.FunctionCall) goja.Value {
    eventObj := call.Argument(0).ToObject(vm)
    event := &Event{
        Type:      eventObj.Get("type").String(),
        Target:    node,
        CurrentTarget: node,
        Bubbles:   eventObj.Get("bubbles").ToBoolean(),
        Cancelable: eventObj.Get("cancelable").ToBoolean(),
    }

    // 1. Capture phase (top-down) - not fully implemented
    // 2. Target phase
    node.fireListeners(event)

    // 3. Bubble phase (bottom-up)
    if event.Bubbles && !event.PropagationStopped {
        node.bubbleEvent(event)
    }

    return vm.ToValue(!event.DefaultPrevented)
}

func (node *NodeDOM) fireListeners(event *Event) {
    listeners := node.EventListeners[event.Type]
    for _, listener := range listeners {
        fn := listener.(goja.Callable)
        fn(goja.Undefined(), vm.ToValue(event))
    }
}

func (node *NodeDOM) bubbleEvent(event *Event) {
    parent := node.Parent
    for parent != nil {
        event.CurrentTarget = parent
        parent.fireListeners(event)
        if event.PropagationStopped {
            break
        }
        parent = parent.Parent
    }
}
```

**Reference**: [MDN - EventTarget](https://developer.mozilla.org/en-US/docs/Web/API/EventTarget) | [Firefox EventDispatcher](https://searchfox.org/mozilla-central/source/dom/events/EventDispatcher.cpp)

---

## 7.10 Timer APIs: setTimeout, setInterval

```go
func (ctx *WindowContext) setTimeout(call goja.FunctionCall) goja.Value {
    fn := call.Argument(0)
    delay := call.Argument(1).ToInteger()

    if !goja.IsCallable(fn) {
        panic(vm.NewTypeError("callback must be a function"))
    }

    id := ctx.EventLoop.AddTimer(time.Duration(delay)*time.Millisecond, false, fn)
    return vm.ToValue(id)
}

func (ctx *WindowContext) setInterval(call goja.FunctionCall) goja.Value {
    fn := call.Argument(0)
    delay := call.Argument(1).ToInteger()

    if !goja.IsCallable(fn) {
        panic(vm.NewTypeError("callback must be a function"))
    }

    id := ctx.EventLoop.AddTimer(time.Duration(delay)*time.Millisecond, true, fn)
    return vm.ToValue(id)
}

func (ctx *WindowContext) clearTimeout(call goja.FunctionCall) goja.Value {
    id := call.Argument(0).ToInteger()
    ctx.EventLoop.RemoveTimer(int(id))
    return goja.Undefined()
}

func (ctx *WindowContext) clearInterval(call goja.FunctionCall) goja.Value {
    id := call.Argument(0).ToInteger()
    ctx.EventLoop.RemoveTimer(int(id))
    return goja.Undefined()
}
```

### Event Loop Implementation

```go
type EventLoop struct {
    mu       sync.Mutex
    timers   map[int]*Timer
    nextID   int
    microtasks []func()
}

type Timer struct {
    ID        int
    Delay     time.Duration
    Repeat    bool
    Callback  goja.Value
    Deadline  time.Time
}

func (el *EventLoop) Run() {
    for {
        el.mu.Lock()
        now := time.Now()

        // Run expired timers
        for _, timer := range el.timers {
            if now.After(timer.Deadline) {
                // Execute callback in VM
                go func(cb goja.Value) {
                    cb(goja.Undefined())
                }(timer.Callback)

                if timer.Repeat {
                    timer.Deadline = now.Add(timer.Delay)
                } else {
                    delete(el.timers, timer.ID)
                }
            }
        }

        // Run microtasks (Promise callbacks)
        for len(el.microtasks) > 0 {
            task := el.microtasks[0]
            el.microtasks = el.microtasks[1:]
            task()
        }

        el.mu.Unlock()
        time.Sleep(1 * time.Millisecond)
    }
}
```

---

## 7.11 Fetch API: Network Requests from JS

```go
func (ctx *WindowContext) fetch(call goja.FunctionCall) goja.Value {
    url := call.Argument(0).String()
    options := call.Argument(1)

    // Parse options (method, headers, body, credentials, etc.)
    req, err := ctx.buildFetchRequest(url, options)
    if err != nil {
        return vm.NewPromiseReject(err)
    }

    // Create Promise
    promise, resolve, reject := vm.NewPromise()

    // Execute asynchronously
    go func() {
        // Use window's isolated HTTP client (with its cookie jar)
        resp, err := ctx.HTTPClient.Do(req)
        if err != nil {
            reject(vm.ToValue(err.Error()))
            return
        }

        // Build Response object
        jsResp := ctx.buildJSResponse(vm, resp)
        resolve(jsResp)
    }()

    return promise
}
```

### Response Object

```go
func (ctx *WindowContext) buildJSResponse(vm *goja.Runtime, resp *http.Response) goja.Value {
    obj := vm.NewObject()

    obj.Set("ok", resp.StatusCode >= 200 && resp.StatusCode < 300)
    obj.Set("status", resp.StatusCode)
    obj.Set("statusText", http.StatusText(resp.StatusCode))

    // headers
    headersObj := vm.NewObject()
    for k, v := range resp.Header {
        headersObj.Set(k, strings.Join(v, ", "))
    }
    obj.Set("headers", headersObj)

    // .json() method
    obj.Set("json", func(call goja.FunctionCall) goja.Value {
        body, _ := io.ReadAll(resp.Body)
        var data interface{}
        json.Unmarshal(body, &data)
        return vm.ToValue(data)
    })

    // .text() method
    obj.Set("text", func(call goja.FunctionCall) goja.Value {
        body, _ := io.ReadAll(resp.Body)
        return vm.ToValue(string(body))
    })

    // .blob(), .arrayBuffer(), etc.

    return obj
}
```

**Reference**: [MDN - Fetch API](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API) | [Firefox Fetch](https://searchfox.org/mozilla-central/source/dom/fetch/Fetch.cpp)

---

## 7.12 XMLHttpRequest: Legacy API

```go
func (ctx *WindowContext) XMLHttpRequestConstructor(call goja.ConstructorCall) *goja.Object {
    xhr := &XMLHttpRequest{
        VM:       ctx.VM,
        WindowCtx: ctx,
        State:    XHR_UNSENT,
    }

    obj := call.This
    obj.Set("open", xhr.open)
    obj.Set("send", xhr.send)
    obj.Set("setRequestHeader", xhr.setRequestHeader)
    obj.Set("abort", xhr.abort)
    obj.Set("onreadystatechange", goja.Null())
    obj.Set("readyState", xhr.State)
    obj.Set("responseText", "")
    obj.Set("status", 0)
    obj.Set("statusText", "")

    return obj
}
```

---

## 7.13 Storage: localStorage & sessionStorage

### StoragePartition: Per-Origin Isolation

```go
// StoragePartition isolates storage by origin.
// Each origin gets its own localStorage and sessionStorage.
type StoragePartition struct {
    mu           sync.RWMutex
    localStorage  map[string]*StorageArea  // key: origin
    sessionStorage map[string]*StorageArea
}

type StorageArea struct {
    mu    sync.RWMutex
    data  map[string]string
    origin string
}
```

### localStorage Binding

```go
func (sa *StorageArea) toJSValue(vm *goja.Runtime) goja.Value {
    obj := vm.NewObject()

    obj.Set("getItem", func(key string) goja.Value {
        sa.mu.RLock()
        val, ok := sa.data[key]
        sa.mu.RUnlock()
        if !ok {
            return goja.Null()
        }
        return vm.ToValue(val)
    })

    obj.Set("setItem", func(key, value string) {
        sa.mu.Lock()
        sa.data[key] = value
        sa.mu.Unlock()
        sa.persist()  // Save to disk
    })

    obj.Set("removeItem", func(key string) {
        sa.mu.Lock()
        delete(sa.data, key)
        sa.mu.Unlock()
        sa.persist()
    })

    obj.Set("clear", func() {
        sa.mu.Lock()
        sa.data = make(map[string]string)
        sa.mu.Unlock()
        sa.persist()
    })

    obj.Set("key", func(index int) goja.Value {
        sa.mu.RLock()
        defer sa.mu.RUnlock()
        keys := make([]string, 0, len(sa.data))
        for k := range sa.data {
            keys = append(keys, k)
        }
        if index < 0 || index >= len(keys) {
            return goja.Null()
        }
        return vm.ToValue(keys[index])
    })

    obj.Set("length", len(sa.data))

    return obj
}
```

**Reference**: [MDN - Web Storage](https://developer.mozilla.org/en-US/docs/Web/API/Web_Storage_API) | [Firefox Storage](https://searchfox.org/mozilla-central/source/dom/storage/Storage.cpp)

---

## 7.14 postMessage: Cross-Window Communication

```go
func (ctx *WindowContext) postMessage(call goja.FunctionCall) goja.Value {
    message := call.Argument(0).Export()
    targetOrigin := call.Argument(1).String()

    // Security: validate targetOrigin
    if targetOrigin != "*" && targetOrigin != ctx.Origin {
        // Find target window by origin
        targets := hotdog.FindWindowsByOrigin(targetOrigin)
        for _, target := range targets {
            target.VM.RunScript(fmt.Sprintf(`
                window.dispatchEvent(new MessageEvent("message", {
                    data: %s,
                    origin: "%s",
                    source: window
                }));
            `, toJSON(message), ctx.Origin))
        }
    } else if targetOrigin == "*" {
        // Broadcast to all windows (use with caution!)
        hotdog.ForEachWindow(func(w *WindowContext) {
            if w != ctx {
                w.VM.RunScript(fmt.Sprintf(`
                    window.dispatchEvent(new MessageEvent("message", {
                        data: %s,
                        origin: "%s",
                        source: window
                    }));
                `, toJSON(message), ctx.Origin))
            }
        })
    }

    return goja.Undefined()
}
```

**Reference**: [MDN - postMessage](https://developer.mozilla.org/en-US/docs/Web/API/Window/postMessage) | [Firefox PostMessage](https://searchfox.org/mozilla-central/source/dom/base/Window.cpp#3000)

---

## 7.15 Security: Same-Origin Policy & CSP

### Origin Checking

```go
// checkSameOrigin verifies that a JS operation is allowed.
// Throws SecurityError if cross-origin access attempted.
func (ctx *WindowContext) checkSameOrigin(targetOrigin string) error {
    if targetOrigin == ctx.Origin {
        return nil
    }
    // Check if CORS allows it (for fetch/XHR)
    // For DOM access: always deny cross-origin
    return fmt.Errorf("SecurityError: Blocked cross-origin access from %s to %s", ctx.Origin, targetOrigin)
}
```

### CSP Enforcement

```go
// ContentSecurityPolicy parses and enforces CSP headers.
type ContentSecurityPolicy struct {
    directives map[string][]string
}

func (csp *ContentSecurityPolicy) Allows(scriptSrc string) bool {
    // Check script-src directive
    allowed := csp.directives["script-src"]
    for _, source := range allowed {
        if matchSource(source, scriptSrc) {
            return true
        }
    }
    return false
}

// Blocks eval() unless 'unsafe-eval' in script-src
func (ctx *WindowContext) eval(call goja.FunctionCall) goja.Value {
    if !ctx.CSP.Allows("'unsafe-eval'") {
        panic(vm.NewError(vm.NewGoError(errors.New("eval() blocked by CSP"))))
    }
    // ... execute eval
}
```

**Reference**: [MDN - CSP](https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP) | [Firefox CSP](https://searchfox.org/mozilla-central/source/dom/security/ContentSecurityPolicy.cpp)

---

## 7.16 Module System (Experimental)

Goja supports ES modules partially:

```go
// Enable modules
vm.Set("import", func(specifier string) goja.Value {
    // Resolve specifier to URL
    // Fetch module code
    // Parse and execute in module context
    // Return module namespace object
})
```

---

## 7.17 Cross-References

| Concept | Chapter |
|---------|---------|
| How network layer isolates cookies | [Chapter 2: Network Layer](../02-network-layer.md) |
| How DOM is structured | [Chapter 3: HTML Parsing](../03-html-parsing-dom.md) |
| How styles affect JS-computed styles | [Chapter 4: CSS Styling](../04-css-styling.md) |
| How layout triggers from JS | [Chapter 5: Layout Engine](../05-layout-engine.md) |
| How Canvas API draws | [Chapter 6: Rendering](../06-rendering-graphics.md) |
| Event system details | [Glossary: Event](../glossary.md#event), [Glossary: Event Listener](../glossary.md#event-listener) |

---

## 7.18 Example Test: Running JavaScript

```go
// Example_GojaVM demonstrates running JavaScript in a window context.
func Example_GojaVM() {
    ctx := hotdog.NewWindowContext()
    defer ctx.Close()

    // Run simple script
    result, err := ctx.VM.RunScript(`
        var x = 10;
        var y = 20;
        x + y;
    `)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Result:", result.ToInteger())  // 30

    // Access DOM
    _, err = ctx.VM.RunScript(`
        document.title = "New Title";
        console.log("Title changed to:", document.title);
    `)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Document title:", ctx.Document.Title)

    // Output:
    // Result: 30
    // [JS:about:blank] Title changed to: New Title
    // Document title: New Title
}
```

Run with: `go test -run Example_GojaVM ./hotdog`

---

## 7.19 Example Test: Fetch from JS

```go
// Example_FetchAPI demonstrates the Fetch API from JavaScript.
func Example_FetchAPI() {
    ctx := hotdog.NewWindowContext()
    defer ctx.Close()

    // Mock HTTP server for testing
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"message": "Hello from server!"}`))
    }))
    defer server.Close()

    // Run fetch in JS
    _, err := ctx.VM.RunScript(fmt.Sprintf(`
        fetch("%s")
            .then(r => r.json())
            .then(data => console.log("Got data:", data.message))
            .catch(e => console.error("Error:", e));
    `, server.URL))
    if err != nil {
        log.Fatal(err)
    }

    // Give event loop time to process
    time.Sleep(100 * time.Millisecond)

    // Output:
    // [JS:about:blank] Got data: Hello from server!
}
```

Run with: `go test -run Example_FetchAPI ./hotdog`

---

## 7.20 Example Test: Event Listeners

```go
// Example_AddEventListener demonstrates DOM event handling from JS.
func Example_AddEventListener() {
    ctx := hotdog.NewWindowContext()
    defer ctx.Close()

    // Create a button in DOM
    _, err := ctx.VM.RunScript(`
        var btn = document.createElement("button");
        btn.textContent = "Click Me";
        btn.id = "myBtn";
        document.body.appendChild(btn);

        btn.addEventListener("click", function() {
            console.log("Button clicked!");
            this.textContent = "Clicked!";
        });
    `)
    if err != nil {
        log.Fatal(err)
    }

    // Simulate click (would normally come from UI)
    btn := ctx.Document.GetElementById("myBtn")
    if btn != nil {
        event := &hotdog.Event{Type: "click", Target: btn, Bubbles: true}
        btn.DispatchEvent(event)
    }

    // Output:
    // [JS:about:blank] Button clicked!
}
```

Run with: `go test -run Example_AddEventListener ./hotdog`

---

## 7.21 Summary

- **Goja** = Pure Go JS engine (ES5.1+), one VM per window
- **WindowContext** = Isolated environment: VM, DOM, storage, network
- **Global objects**: window, document, console, fetch, localStorage, etc.
- **DOM bindings**: Element methods (querySelector, appendChild, style, etc.)
- **Event system**: addEventListener, dispatchEvent, capture/bubble phases
- **Timers**: setTimeout, setInterval with event loop
- **Fetch/XHR**: Network requests using window's isolated HTTP client
- **Storage**: localStorage/sessionStorage partitioned by origin
- **postMessage**: Cross-window communication with origin validation
- **Security**: Same-origin policy, CSP, eval() blocking

---

## 7.22 Next Chapter

👉 **[Chapter 8: UI Toolkit & Application — mustard/app](../08-ui-toolkit.md)**

Learn how the browser UI (tabs, address bar, buttons) is built with mustard widgets, and how the application manages multiple windows.

---

*Key terms from this chapter: [CSP](../glossary.md#csp), [Event Loop](../glossary.md#event-loop), [Fetch](../glossary.md#fetch), [Goja](../glossary.md#goja), [JS Engine](../glossary.md#js-engine), [localStorage](../glossary.md#localstorage), [Microtask](../glossary.md#microtask), [Promise](../glossary.md#promise), [Same-Origin Policy](../glossary.md#same-origin-policy), [sessionStorage](../glossary.md#sessionstorage), [VM](../glossary.md#vm), [XHR](../glossary.md#xhr)*