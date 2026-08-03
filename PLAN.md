### REVISED PLAN: Tabbed Browsing Architecture with Per-Window Isolation

## Core Model: Each Tab = Separate Window Instance

Instead of modifying the existing single-window architecture, the browser wraps **multiple independent toolkit instances** (one per tab/window). Each window has:
- Isolated DOM tree and layout cache
- Isolated Goja JS runtime
- Isolated event loop and input routing
- Isolated storage (cookies, localStorage, sessionStorage)

A **browser shell** manages the tabbed UI, tab switching, and resource sharing at the network layer.

---

## Phase 0: Refactor for Multiple Window Instances

Enable the toolkit to run as multiple independent, non-interfering instances.

* **Extract Window Context:** Move all state (DOM root, Goja VM, layout cache, event registry, viewport dimensions, focus state) into a `WindowContext` struct. Currently global state becomes instance fields.
* **Decouple Rendering Backend:** Abstract mustard rendering so multiple windows can render to different display regions (tabbed UI) or even separate OS windows.
* **Shared Network Layer:** Create a global HTTP client with origin-keyed cookie jar, TLS session cache, and connection pooling; all windows share this, but cookies are partitioned by origin.
* **Build Browser Shell:** Create a top-level coordinator that spawns/destroys `WindowContext` instances on tab create/close, routes input to active tab only, manages tab switching and viewport layout.
* **Implement Origin Tracking:** Parse and store origin (scheme + domain + port) on each window at load time; use origin to partition cookies and enforce SOP on all resource loads.

### Phase 0 Completion Checklist
- [x] State moved from global to `WindowContext` struct with no regressions
- [ ] Two or more simultaneous windows run without memory interference
- [ ] Mustard rendering supports multiple viewports (tabbed layout)
- [ ] Browser shell routes mouse/keyboard input to active window only
- [ ] Tab creation and destruction leak no memory
- [ ] Cookie jar partitions by origin across all windows
- [ ] TLS session cache shared; cookies isolated per origin
- [ ] Origin parsing handles IPv6, non-standard ports
- [ ] Tab switching is smooth with no layout artifacts
- [ ] Inactive windows do not execute JS or recalculate layout

---

## Phase 1: Isolate and Refactor the DOM and Tree Structure

Establish a clean, spec-compliant DOM foundation **per window** with robust tree traversal and mutation support.

* **Replace HTML Parser:** Complete removal of custom `ketchup` parser; integrate `golang.org/x/net/html` for W3C-compliant HTML5 parsing.
* **Define Core Node Structure:** Create unified node type supporting parent, first-child, next-sibling, previous-sibling pointers. Support element nodes, text nodes, comment nodes, document fragments. **Each node includes reference to owning `WindowContext`.**
* **Implement Standard DOM Methods:** Expose `appendChild()`, `removeChild()`, `insertBefore()`, `cloneNode()`, `replaceChild()`. **Add origin validation to prevent cross-origin DOM injection.**
* **Integrate CSS Selector Engine:** Embed `andybalholm/cascadia` for `querySelector()`, `querySelectorAll()`, `getElementById()`, `getElementsByClassName()`, `getElementsByTagName()`.
* **Define Text Node Handling:** Separate text node type with proper whitespace handling.
* **Add DOM Query Caching:** Implement memoization for selector results; invalidate on origin boundary changes.
* **Implement Origin-Aware Node Access:** Prevent JavaScript from one origin accessing DOM nodes from another origin through iframe boundaries.

### Phase 1 Completion Checklist
- [ ] HTML parser replaces ketchup across all test pages
- [ ] Node structure supports full tree traversal
- [ ] All selector methods working with cascadia
- [ ] Text nodes render correctly with proper whitespace
- [ ] Each node carries owning WindowContext reference
- [ ] Cross-origin DOM access correctly blocked
- [ ] Iframes with sandbox attribute create isolated DOM subtrees

---

## Phase 2: Decouple CSS Parsing from Layout and Integrate Flexbox

Separate style parsing from layout calculation using a mature flexbox solver.

* **Retain Mayo CSS Parser:** Keep string parsing and style extraction from `<style>` blocks and `style` attributes.
* **Standardize Style Representation:** Unified Go struct for CSS properties (display, position, flexbox, sizing, margins, padding, borders, colors, fonts).
* **Initialize Flexbox Nodes:** Every DOM element allocates reference to flexbox node; use `kjk/flex` (pure Go Yoga port) as solver.
* **Build Style-to-Layout Translator:** Convert parsed mayo CSS values into flexbox node setter calls.
* **Implement Cascade and Inheritance:** Support CSS inheritance rules, specificity calculation, author/user/browser stylesheet priority.
* **Handle Computed Styles:** Resolve final computed styles after cascade application.
* **Replace Layout Calculation:** Invoke `flex.CalculateLayout()` on root node to compute all descendant positions in single pass.
* **Add Layout Validation:** Verify solved layouts conform to flexbox spec.

### Phase 2 Completion Checklist
- [ ] Mayo parser outputs clean style structs with no regressions
- [ ] Every DOM element has live flexbox node
- [ ] Style translation produces correct flex node configurations
- [ ] CSS inheritance rules apply correctly
- [ ] Cascade and specificity tests pass
- [ ] Layout solver produces identical output to existing mayo
- [ ] Complex nested flexbox layouts render without regression

---

## Phase 3: Implement JavaScript Runtime and DOM Bindings

Introduce interactive scripting with per-window Goja isolation and security sandboxing.

* **Initialize Per-Window Goja VM:** Set up pure Go JS runtime **per `WindowContext`** using `dop251/goja`; never share runtimes between windows.
* **Build DOM Wrapper Functions:** Create Go-to-JS bridge functions for DOM manipulation. **All wrappers enforce origin checks and throw SecurityError on violations.**
* **Register Global Objects:** Populate `document`, `window`, `console` objects. **Prevent access to DOM nodes from other origins.**
* **Implement querySelector Methods:** Expose cascadia queries; **selectors respect sandbox boundaries and cannot cross iframe borders.**
* **Support DOM Properties:** Map element properties (id, className, innerHTML, textContent, style). **Validate all writes against CSP and origin restrictions.**
* **Implement Event Objects:** Define JS event objects bridging to Go. **Prevent event bubbling across security boundaries.**
* **Add Console API:** Implement `console.log()`, `console.error()`, `console.warn()`. **Log output attributed to origin and window.**
* **Implement Storage APIs:** Expose `localStorage` and `sessionStorage` backed by window storage partition. **Storage keyed by origin, not window; session storage cleared on tab close.**
* **Add Inter-Window Messaging:** Implement `window.postMessage(data, targetOrigin)`. **Validate targetOrigin and prevent unsolicited messages.**
* **Create Fetch API Bridge:** Implement XMLHttpRequest and Fetch API with enforced CORS preflight, origin header validation, cookie jar isolation, and header filtering.
* **Prevent Dangerous APIs:** Block `eval()`, `Function()` constructor unless permitted by CSP `script-src 'unsafe-eval'`.

### Phase 3 Completion Checklist
- [ ] Each window has isolated Goja VM with no memory leaks
- [ ] Cross-window runtime variable access impossible
- [ ] DOM operations from JS mutate Go DOM nodes within same origin
- [ ] Cross-origin DOM access attempts throw SecurityError
- [ ] querySelector respects iframe boundaries
- [ ] Element properties work correctly with CSP checks
- [ ] console output logged with origin attribution
- [ ] Storage partitioned by origin, not window
- [ ] postMessage validates targetOrigin
- [ ] CORS preflight works for cross-origin requests
- [ ] eval() blocked unless permitted by CSP
- [ ] Fetch/XMLHttpRequest uses correct cookie jar partition
- [ ] No cross-window variable access leakage

---

## Phase 4: Implement Event System, Mutation Observation, and Reactive Layout

Enable interactive UIs with event handling, mutation tracking, and continuous layout recalculation **per window**.

* **Build Event Registry:** Store callback references on DOM nodes per event type. **Events delivered only to nodes within same window context.**
* **Expose addEventListener API:** Implement `element.addEventListener(type, listener)`. **Listeners isolated per VM.**
* **Implement removeEventListener:** Allow cleanup of registered event listeners.
* **Create Dirty Flags:** Add `dirty` boolean flag to each DOM node; set on text, attribute, class, or style changes.
* **Build Invalidation Queue:** Maintain queue of dirty nodes requiring layout recalculation. **Each window has its own invalidation queue.**
* **Implement requestAnimationFrame:** Expose frame scheduling for script callbacks. **Callbacks per-window and fire in isolation.**
* **Create Mutation Observer Interface:** Build simple mutation observer. **Mutations observable only within window.**
* **Integrate Layout Recalculation Loop:** After JS mutations, walk dirty nodes and trigger layout solver. **Caching per-window.**
* **Implement Viewport Resize Handling:** Capture viewport changes and mark root node dirty. **Viewport size can differ per window.**

### Phase 4 Completion Checklist
- [ ] Event listeners registered and fired correctly
- [ ] removeEventListener successfully removes callbacks
- [ ] Dirty flag system marks nodes correctly on mutation
- [ ] Layout recalculation happens after DOM mutations
- [ ] requestAnimationFrame callbacks execute in correct order and isolation
- [ ] Mutation observer reacts to DOM changes
- [ ] Interactive components respond to events
- [ ] No layout thrashing
- [ ] Viewport resize triggers relayout without artifacts
- [ ] Each window maintains independent event loop and animation frame
- [ ] Event bubbling stops at security boundaries (iframe sandbox)

---

## Phase 5: Connect Layout Output to Rendering and Input Mapping

Bind solved layout coordinates to the visual render pipeline and map hardware input events back to DOM targets **per window**.

* **Query Flexbox Layout Results:** Extract computed x, y, width, height, padding, border from flexbox nodes.
* **Pass Geometry to Mustard:** Feed layout-computed bounding boxes to rendering layer. **Each window renders to its own viewport.**
* **Implement Paint Validation:** Detect changed layout/style; mark only affected subtrees for repaint. **Painting independent per window.**
* **Add Text Rendering Integration:** Wire text node content and computed font properties; respect flexbox dimensions.
* **Implement Hit Testing:** Build spatial index/tree-walk to find topmost DOM node at (x, y) coordinate. **Hit testing searches only active window's DOM tree.**
* **Map Input to Events:** Capture mouse clicks, key presses, text input from GLFW/OpenGL backend; resolve to target elements; fire JS event callbacks. **Input routed only to active window.**
* **Implement Focus Management:** Track and update focused element; fire `focus` and `blur` events. **Focus per-window; one element focused per window context.**
* **Handle Scroll Events:** Implement scrolling for overflow elements and emit `scroll` events. **Scroll position per-window.**
* **Build Fullscreen/Modal Support:** Handle `element.focus()` and z-index stacking. **Modals confined to their window.**
* **Implement Tab Switching Logic:** Pause layout/event loop in inactive windows; resume in active window. Implement efficient viewport swapping.

### Phase 5 Completion Checklist
- [ ] Layout coordinates flow from flexbox solver to mustard renderer
- [ ] Paint applied only to changed regions
- [ ] Text renders within flexbox-computed boundaries
- [ ] Hit testing identifies target elements correctly
- [ ] Mouse clicks fire to correct elements in correct window
- [ ] Keyboard input routes to focused element in active window
- [ ] Focus ring renders and updates correctly per window
- [ ] Scrollable content works without artifacts
- [ ] No excessive paint/layout cycles per window
- [ ] Inactive windows do not execute JS or recalculate layout
- [ ] Tab switching is smooth with no layout stalls
- [ ] Each window can have different viewport dimensions

---

## Phase 6: Network, Cookie, and Storage Security

Implement secure network handling, cookie isolation, and storage partitioning per origin.

* **Implement Secure Cookie Jar:** Build `http.CookieJar` keyed by origin (scheme + domain + port). Enforce:
  - **Secure flag:** Cookies with `Secure` flag only sent over HTTPS.
  - **HttpOnly flag:** Cookies marked `HttpOnly` not accessible from JavaScript.
  - **SameSite enforcement:** Support `SameSite=Strict`, `SameSite=Lax`, `SameSite=None`.
  - **Domain and Path matching:** Ensure cookies sent only to matching request origins.
* **Build HTTPS Enforcement:** Require HTTPS for all resource loads by default. Implement HSTS support (read/store HSTS headers, enforce future HTTPS on matching domains).
* **Implement Certificate Validation:** Validate TLS certificates against OS trusted CA store; reject self-signed/expired unless explicitly pinned.
* **Create Origin-Keyed Storage Partitions:**
  - **localStorage:** Persistent per-origin key-value store, accessible only to scripts from that origin.
  - **sessionStorage:** Per-window, per-origin key-value store, cleared when window is closed.
* **Implement Content Security Policy (CSP):** Parse CSP headers on page load; validate inline scripts, style injection, external resource loads against `script-src`, `style-src`, `img-src` directives. Block violations and log to console.
* **Build CORS Validator:** Implement checks for `Access-Control-Allow-*` headers and preflight handling for cross-origin requests.
* **Implement Same-Origin Policy:** At page load and resource requests (images, stylesheets, scripts, XHR/Fetch), verify request origin matches window's loaded origin. Block mismatches unless CORS headers permit.

### Phase 6 Completion Checklist
- [ ] Cookies enforce Secure, HttpOnly, SameSite, and domain/path matching
- [ ] HTTPS required; HSTS headers cached and enforced
- [ ] Certificate validation blocks self-signed and expired certs
- [ ] localStorage persists per-origin across windows
- [ ] sessionStorage is per-window and cleared on tab close
- [ ] CSP violations block inline scripts and log to console
- [ ] CORS preflight requests work for cross-origin XHR/Fetch
- [ ] SOP blocks cross-origin resource loads by default
- [ ] No cross-window cookie leaks in test suite

---

## Phase 7: Testing, Documentation, and Optimization

Build comprehensive test coverage, user-facing documentation, and performance optimization.

* **Create Unit Test Suite:** Exhaustive tests for DOM manipulation, CSS parsing, layout calculation, event dispatch, JS integration, security boundaries.
* **Build Integration Tests:** Multi-phase workflows (HTML parse → CSS resolve → layout → render → event handling).
* **Create Example Applications:** Reference UI components (buttons, forms, modals, navigation, dashboards) and tabbed browser example.
* **Write API Documentation:** Document all public types, methods, functions with examples.
* **Build Rendering Benchmarks:** Profile paint, layout, event handling; identify and optimize hotspots.
* **Implement Instrumentation:** Add optional debug logging, performance counters, flame graph integration.
* **Create Migration Guide:** Document transition from existing thdwb code to new toolkit API.
* **Add Type Safety Checks:** Implement go vet, staticcheck rules.

### Phase 7 Completion Checklist
- [ ] 80%+ line coverage for core DOM, layout, event, security modules
- [ ] All example components render and interact correctly
- [ ] Tabbed browser example functional with multiple windows
- [ ] API docs complete and accurate
- [ ] Benchmark suite established with baseline metrics
- [ ] Debug instrumentation functional and non-intrusive
- [ ] Migration guide covers all breaking API changes
- [ ] No static analysis warnings in main codebase
- [ ] Example gallery builds and runs successfully
- [ ] Security tests verify SOP, CORS, CSP, cookie isolation

---

## Phase 8: Stretch Goal – CGo-Free Hardware-Accelerated Rendering Path

Decouple from all CGo dependencies while maintaining or improving hardware acceleration.

* **Audit Current CGo Dependencies:** Document all CGo bindings in mustard (GLFW, OpenGL, FreeType, image libraries).
* **Replace GLFW:** Migrate window creation/input to pure Go bindings (syscall on Windows/macOS, X11/Wayland on Linux) or `ebitengine/purego`.
* **Implement Direct GPU Access:** Use platform-native GPU APIs:
  - **Windows:** Direct3D 12 via `golang.org/x/sys/windows`.
  - **macOS:** Metal via syscall bindings.
  - **Linux:** Vulkan via `vulkan-go` or pure Go bindings.
* **Port Font Rendering:** Replace FreeType with pure Go font rasterizer using `golang.org/x/image/font` and `golang.org/x/image/vector` for TrueType/OpenType glyph rendering.
* **Replace Image Codecs:** Swap CGo image libraries for pure Go implementations (`image/png`, `image/jpeg`, `golang.org/x/image/webp`, `kolesa-team/go-avif`).
* **Build Platform Abstraction Layer:** Create thin interface layer (`platform.Renderer`, `platform.Window`, `platform.Input`) with native implementations per OS.
* **Implement GPU Command Buffer Abstraction:** Define platform-agnostic command buffer format; translate into native GPU APIs at runtime.
* **Add Software Fallback Renderer:** Implement pure Go CPU-based renderer using `golang.org/x/image/draw` as fallback for headless/GPU-restricted environments.
* **Create Build Tags:** Use conditional compilation (`// +build cgo,nocgo`) allowing users to choose CGo or pure Go rendering at compile time; default to pure Go.
* **Ensure Cross-Platform Testing:** Validate pure Go renderer on Windows, macOS, Linux; test headless rendering and virtualized GPU scenarios.

### Phase 8 Completion Checklist
- [ ] All CGo dependencies catalogued and replacement plan documented
- [ ] Window creation and event loop work without GLFW on all platforms
- [ ] GPU rendering functional via Direct3D 12, Metal, or Vulkan
- [ ] Font rendering produces correct glyphs and metrics