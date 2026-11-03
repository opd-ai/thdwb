# THDWB Browser Engine Audit Report

**Audit date:** 2026-11-03
**HEAD audited:** `384313d` (branch `master`, `github.com/opd-ai/thdwb` local clone at `~/go/src/github.com/danfragoso/thdwb`)
**Method:** three parallel debugging passes (memory/DOM/rendering · off-by-one/boundaries · concurrency/JS/error-handling) over 255 Go files / ~61k LOC, with empirical probes against the pinned `goja` (v0.0.0-20261026132650) and `cascadia` (v1.3.5) versions in `go.sum`.

## Executive Summary

| Severity | Count |
|---|---|
| Critical | 1 |
| High | 18 |
| Medium | 13 |
| Low | 3 |
| **Total** | **35** |

**Rendering/execution stability status:**

- **The live render path is reachable and partially wired** (shell `loadTabContent` → per-tab `WindowContext.LoadURL` → ketchup re-parse → `bun` layout into the web-content canvas), but **layout crashes deterministically on common inputs**: any node with the default style (`FontWeight: 0`) dereferences a nil font face (`bun/fonts.go` only maps 300/400/600/700/800 → `gg.Context` holds a nil face → `MeasureString` panics); `RenderDocument` and `createFlexNodes` nil-deref on fresh documents (`RenderBox`/`Style` unset).
- **The concurrency model is unsound for multi-window/multi-extension operation.** The one Critical finding is `window.postMessage` delivering into another window's goja VM **without that window's runtime lock**; `chrome.alarms` and the `WebRequest` observer list have the same class of bug. A custom reentrant mutex keyed on goroutine-ID can deadlock permanently if the owner goroutine dies, and `removeEventListener` is a verified silent no-op (closure code-pointers are per-instance in modern Go).
- **Long-session memory growth is guaranteed**, not possible: per-navigation `Documents`/`nodeWrappers`/window-listener accumulation is never released, tab close never calls `WindowContext.Destroy()`, the global `ResourceCache`/`ImgCache` have no eviction **and no lock** (data race under concurrent fetches), and every `CanvasWidget` allocates a 12000px-tall drawing buffer (~92 MB at 1080p width).
- **Error handling converts failures into silent success or crashes:** fetch/XHR classify transport errors by `strings.Contains(body, "error")` (any page mentioning "error" fails), JS exceptions in event listeners / rAF / alarm callbacks are discarded with `_, _ =` (and a *non-catchable* Go panic in a timer/network goroutine kills the process — verified against goja's `runWrapped`), and the error page interpolates raw error strings into HTML.
- Already fixed since the 2026-11-01 architecture audit (verified at this HEAD): double-wrapped HTTPS transport, `Code=0`-on-error swallowing (`Code=-1` now, `LoadURL` rejects `<= 0`), `document.title` accessor binding, first-frame `Frame.draw` index-out-of-range, and the `VerticalFrame`/`HorizontalFrame` orientation swap.

## Findings by Subsystem

### Memory, DOM & Rendering Issues

| # | File:Line:Function | Severity | Root cause | Fix |
|---|---|---|---|---|
| 1.1 | `hotdog/window_context.go:315:LoadHTML` (+897-899 `Destroy`) | High | Each navigation appends to `wc.Documents` and re-binds the persistent goja realm, but old DOM trees, `nodeWrappers`, `nodeEventRegistry` and window-level `eventRegistry` entries are never released, so every page load leaks the full previous DOM + JS closures. | Reset per-document state on navigation (clear wrappers/events referencing the old root, or create a fresh goja runtime per document): `func (wc *WindowContext) loadDocument(doc *Document) { wc.clearJSBindingsFor(wc.ActiveDocument); wc.Documents = wc.Documents[:len(wc.Documents)-1]; ... }` |
| 1.2 | `browser/shell/shell.go:508-511:loadTabContent` | High | Per-tab `WindowContext` is created in `tabWindowContexts[tab.ID]` but no close path deletes the map entry or calls `wc.Destroy()`, so the whole engine state (goja VM, DOM, cookie jar, HTTP client, rAF timer) of every tab ever opened is retained, and pending fetch/XHR goroutines keep it alive. | Wire tab close to `delete(bs.tabWindowContexts, tab.ID); wc.Destroy()` and cancel in-flight network work (see 3.3). |
| 1.3 | `hotdog/js_bindings.go:1575,1791:fetch/XMLHttpRequest.send` + `sauce/sauce.go:796` (global client) | High | `go func()` network workers use `http.Client`s with **no `Timeout`** and no cancellation context, so a stalled peer leaks a goroutine per request and leaves the JS promise/XHR object unsettled forever, even after the window is destroyed (`ScheduleAnimationFrame` then silently drops the callback). | Set a timeout and propagate cancellation: `wc.httpClient = &http.Client{Jar: ..., Timeout: 60 * time.Second}`; build requests with `http.NewRequestWithContext(wc.ctx, ...)` and `wc.ctx, wc.cancel = context.WithCancel(context.Background())` invoked in `Destroy()`. |
| 1.4 | `hotdog/js_bindings.go:1132:appendChildNode` (+`node.go:264 MarkDirty`, `xpath.go:6 getXPath`) | High | DOM mutation never checks whether the inserted node is an ancestor of the target, so `el.appendChild(el.parentElement)` creates a Parent/Children cycle that overflows the stack in `MarkDirty`/`Print`/`getXPath` (infinite recursion) and corrupts every tree walk. | Reject cycles before linking: `func (node *NodeDOM) appendChildNode(child *NodeDOM) { for a := node; a != nil; a = a.Parent { if a == child { return } } ... }` |
| 1.5 | `hotdog/node.go:1003:CloneNode` | Medium | The clone copies the `Style *Stylesheet` **pointer**, so a cloned node aliases the original's stylesheet; later cascade merges (`mergeStylesheet` rewrites fields in place) change the original element's computed style. | Deep-copy the stylesheet: `if node.Style != nil { s := *node.Style; clone.Style = &s }` (also clone `ColorRGBA` sub-pointers). |
| 1.6 | `hotdog/node.go:851:SetTextContent` | Medium | `SetTextContent` orphans children without the `nodeWrappers` cleanup that `SetInnerHTML`/`removeChildNode` perform, so removed nodes (and their goja wrapper objects) stay reachable in the wrapper map forever. | Reuse the existing cleanup: `if node.WindowCtx != nil && node.WindowCtx.nodeWrappers != nil { delete(node.WindowCtx.nodeWrappers, child) }` in the removal loop. |
| 1.7 | `mustard/canvas.go:170:ensureContexts` | High | Every canvas resize allocates `gg.NewContext(width, 12000)` — a hard-coded 12000px-tall offscreen buffer (~92 MB per widget at 1920px width) for the scrollable drawing context, which multiplies across tabs/canvases. | Size the drawing buffer to real content: `h := max(viewportHeight, contentHeight)` (or `viewportHeight * 2` as an upper bound) instead of the constant `12000`. |
| 1.8 | `bun/listItem.go:18:paintListItemElement` | Medium | Only `Style.BackgroundColor` is nil-checked before `node.Style.Color.R` is dereferenced, so a `<li>` without an explicit color panics the paint pass. | `color := node.Style.Color; if color == nil { color = &hotdog.ColorRGBA{R: 0, G: 0, B: 0, A: 1} }`. |
| 1.9 | `bun/inline.go:134-140:calculateInlineLayout` | Medium | The fallback `im, _, _ = image.Decode(assets.ErrorImage())` ignores the decode error, so on failure `im` is nil and `im.Bounds()` panics inside layout for a broken `<img>`. | `if im, _, err = image.Decode(bytes.NewReader(assets.ErrorImage())); err != nil { return }`. |
| 1.10 | `gg/context.go:116-119:SetImage` | Medium | `SetImage` swaps the backing `*image.RGBA` and width/height but keeps the old `raster.NewRasterizer(w,h)`, so all later Fill/Stroke rasterize into stale scanline bounds (edges silently unpainted, `Clear()` fills the new image) — a paint-buffer/bounds mismatch (latent: mustard recreates contexts instead). | Recreate the rasterizer: `dc.rasterizer = raster.NewRasterizer(dc.width, dc.height)`. |
| 1.11 | `hotdog/window_context.go:632:SetLayoutCache` | Medium | `layoutCache` has no size limit, no TTL, and is not cleared on navigation (only `Destroy`), so a long-lived tab grows it without bound with per-element layout entries. | Bound it (e.g., LRU of N entries) and `ClearLayoutCache()` inside the navigation reset of 1.1. |
| 1.12 | `sauce/sauce.go:799-800,822-834,851-857` (global `cache`/`imageCache`) | Medium | The process-global `ResourceCache`/`ImgCache` append on every GET/image with no eviction, LRU, or dedup (`AddImage` re-appends existing keys), so memory grows for the life of the process on any real browsing session. | Replace the slice with a size-bounded map (LRU or max-count with eviction); dedup by key on add. |

### Off-by-One & Boundary Issues

| # | File:Line:Function | Severity | Root cause | Fix |
|---|---|---|---|---|
| 2.1 | `hotdog/structs.go:82-87:History.PopNext` **and** `sauce/sauce.go:698-703:History.PopNext` | High | `PopNext` moves the last forward page to the back stack and then sets `nextPages = nil`, discarding the entire remaining forward history, so forward navigation after multiple backs is destroyed. | `h.previousPages = append(h.previousPages, h.nextPages[len(h.nextPages)-1]); h.nextPages = h.nextPages[:len(h.nextPages)-1]` (identical fix in both copies; better: delete the duplicate `sauce.History` and use one type). |
| 2.2 | `hotdog/window_context.go:377-386:functionKey` + `js_bindings.go:530,1380:removeEventListener` | High | `functionKey` uses `reflect.Value.Pointer()` on the closure returned by `goja.AssertFunction`; in the current toolchain every closure instance gets a distinct code pointer (empirically verified: register key `4782464` vs remove key `4782432` for the same JS function), so the removal key can never match the registration key and `removeEventListener` is a **silent no-op** — listeners accumulate and double-fire across re-registrations. | Key listeners by goja object identity instead of Go function pointer: `nodeWrapperSymbol`-style symbol on the listener, or keep `map[goja.Value]func` and compare the JS object: `func (w *JSDOMWrapper) removeEventListener(...) { if w.listenerRegistry[eventType][listener] { ... } }`. |
| 2.3 | `mayo/mayo.go:265:parseInlineStylesheet` | High | `strings.ReplaceAll(styleString, " ", "")` deletes **all** whitespace before splitting declarations, so multi-value inline shorthand collapses: `style="margin: 10px 20px"` becomes `margin:10px20px` and parses as a single value `10px` — every space-bearing inline value (margin/padding/flex-basis/box-shadow) is corrupted. | Split on `;` first, then `strings.SplitN(decl, ":", 2)` per declaration, trimming only the outer whitespace: `for _, decl := range strings.Split(styleString, ";") { if p, v, ok := strings.Cut(decl, ":"); ok { mapPropToStylesheet(sheet, []string{strings.TrimSpace(p), strings.TrimSpace(v)}) } }`. |
| 2.4 | `mayo/mayo.go:84-112:mapSizeValue/mapIntValue` | Medium | The number regexes `[0-9]+(\.[0-9]+)?` match only unsigned digits, so negative values (`margin-top: -5px`, `z-index: -1`) are silently made positive, and units (`100%`, `2em`) are silently reinterpreted as raw px. | Parse with sign and record units: `f, err := strconv.ParseFloat(strings.TrimSuffix(num, "%"), 64)` then resolve `%`/`em` against the parent box (or, minimum, preserve the sign and reject unknown units explicitly). |
| 2.5 | `mayo/css.go:31-35:ParseStylesheet` | High | `cascadia.Parse` **rejects comma group selectors** (`Parse("a, b")` → "3 bytes left over", verified), so any rule like `#nav a, .foo {…}` is stored with zero specificity `[0 0 0]` while `cascadia.Compile` in `ApplyStylesheets` still matches it — the group's highest-specificity arm loses to later low-specificity rules (cascade precedence miscalculated). | On parse failure, fall back to `cascadia.Compile` and compute specificity as the max over the group's members, or split on top-level commas and parse each arm: `parts := splitTopLevel(sel, ','); maxSp = max(parse(part).Specificity())`. |
| 2.6 | `hotdog/node.go:706-709:GetTextContent` | High | For non-text nodes the method returns `node.Content`, which is empty for elements, so `element.textContent` is **always `""`** instead of the concatenated descendant text (violates the DOM spec and breaks page JS that reads text). | Recurse: `func (node *NodeDOM) GetTextContent() string { if node.Type == NodeTypeText { return collapse(node.Content) }; var b strings.Builder; for _, c := range node.Children { b.WriteString(c.GetTextContent()) }; return b.String() }`. |
| 2.7 | `sauce/sauce.go:1106-1112:GetImage` | High | Data-URL support is hard-coded to the 22-byte prefix `data:image/png;base64,` and decodes with `base64.RawStdEncoding`, so (a) any other `data:` image (`jpeg`, `gif`…) falls through to `http.NewRequest("GET", "data:...")` and fails, and (b) **padded** base64 (the standard, spec-conformant encoding) fails `RawStdEncoding` decode — typical `<img src="data:image/png;base64,iVBOR…=>">` renders the error stand-in. | Detect `strings.HasPrefix(imgUrl, "data:")`, extract the MIME, and decode permissively: `b64 := strings.TrimRight(s, "="); raw, err := base64.StdEncoding.DecodeString(s); if err != nil { raw, err = base64.RawStdEncoding.DecodeString(b64) }`. |
| 2.8 | `gg/wrap.go:12-26:splitOnSpace` | Low | The `i > 0` guard skips the first space-run boundary, so a line **starting with whitespace** yields a field slice whose word/separator pairing is shifted — the leading space is measured and rendered as part of the first "word". | Strip leading whitespace per line before pairing (`line = strings.TrimLeftFunc(line, unicode.IsSpace)`), or build explicit word/sep pairs. |
| 2.9 | `bun/listItem.go:30:calculateListItemLayout` | Low | `node.RenderBox.Width = node.Parent.RenderBox.Width - 30` is not clamped, so narrow parents produce **negative** box widths that flow into `DrawRectangle`/`MeasureStringWrapped` as degenerate geometry. | `if w := node.Parent.RenderBox.Width - 30; w > 0 { node.RenderBox.Width = w } else { node.RenderBox.Width = 0 }`. |
| 2.10 | `gg/context.go:569-571:SetPixel` | Medium | `SetPixel` calls `dc.im.Set(x, y, …)` with no bounds check; `image.RGBA.Set` **panics** on out-of-range points, so any renderer emitting a negative/oversized coordinate (focus-ring offset `box.Left-2`, off-screen nodes) crashes the paint pass. | `if x < 0 || y < 0 || x >= dc.width || y >= dc.height { return }; dc.im.Set(x, y, dc.color)`. |
| 2.11 | `gg/util.go:89-108:parseHexColor` | Low | Unsupported hex lengths (4/5/7 digits) and malformed input hit no branch (the `Sscanf` errors are discarded), silently yielding `{0,0,0,255}` — any bad color string paints **opaque black** instead of failing loudly. | Validate `len(x) ∈ {3,6,8}` and propagate/ignore-with-log; on `Sscanf` failure fall back to a documented default or return an error to `SetHexColor`. |

### Concurrency, JavaScript & Error Handling Issues

| # | File:Line:Function | Severity | Root cause | Fix |
|---|---|---|---|---|
| 3.1 | `hotdog/js_bindings.go:1432-1440:postMessage` | **Critical** | After `eventObj` is built on `targetWC`'s goja runtime, `targetWC.EmitEvent("message", eventObj)` runs the target's JS listeners **without acquiring `targetWC.jsRuntimeMu`**, so two windows posting to each other (or a post racing any target-side RunJS/rAF) drive two goja VMs concurrently — goja is not thread-safe → data race / heap corruption (the in-code comment "Goja is internally synchronized" is false). | Serialize on the target's lock, mirroring `dispatchMouseEventAt`: `targetWC.jsRuntimeMu.lock(); defer targetWC.jsRuntimeMu.unlock(); targetWC.EmitEvent("message", eventObj)` (the lock is reentrant, so same-window posts keep working). |
| 3.2 | `hotdog/window_context.go:23-82:reentrantMutex` | High | The mutex owner is the parsed **goroutine ID**: if the owner goroutine dies while holding (an unrecovered panic in a JS-bound function re-panics — see 3.7), `owner` is stuck non-zero and every other goroutine blocks in `cond.Wait()` forever; additionally Go **reuses goroutine IDs**, so a future goroutine can be misidentified as the owner; a parse failure returns ID `0`, which is the "unowned" sentinel. | Do not hand-roll reentrancy on goroutine IDs: either (a) route all goja access through one per-window dispatcher goroutine (mailbox) so a plain `sync.Mutex` suffices, or (b) keep a `sync.Mutex` + `owner uintptr` token checked in a `defer func(){ if owner==token { broadcast } }()` that also unlocks on panic. |
| 3.3 | `hotdog/window_context.go:881-929:Destroy` (vs `468:EmitEvent`, `871:SetInputState`) | High | `Destroy` nils `eventRegistry`/`nodeEventRegistry`/`inputState` **without holding `eventMu`** (and `inputState` has no mutex at all), racing any concurrent `EmitEvent`/`RegisterEvent`/`SetInputState` — concurrent map read/write panics or lost events during tab close. | Take the locks during teardown: `wc.eventMu.Lock(); wc.eventRegistry = nil; wc.nodeEventRegistry = nil; wc.eventMu.Unlock()`; add an `inputMu` for `inputState`; set `destroyed` first under `animationFrameMu` and gate all map writes on it. |
| 3.4 | `hotdog/window_context.go:434-444,463-474:EmitNodeEvent/EmitEvent` | High | Handlers are read as a slice **header** (shared backing array) and iterated outside the lock, while `RemoveEventByKey`/`RemoveNodeEventByKey` filter in place via `handlers[:0]` — a concurrent removal (or a handler that removes itself) mutates the array mid-iteration: data race, skipped/duplicated handler calls. | Snapshot: `handlers := make([]eventHandler, len(reg[eventName])); copy(handlers, reg[eventName])` before unlocking, and iterate the copy. |
| 3.5 | `hotdog/js_bindings.go:1655-1667:XMLHttpRequest.updateReadyState` (+`1761:send`) | High | `open()` calls `updateReadyState(OPENED)`, but the "readyState only increases" clamp maps it back to the current state, so after **any** finished or failed request (readyState ≥ DONE) the object can never return to OPENED and the next `send()` throws `InvalidStateError` — every XHR object is single-use, breaking standard retry/reuse patterns. | Reset in `open()` instead of clamping downward transitions: `readyState = OPENED; status = 0; responseText = ""; ...` (keep the monotonic guard only *within* a single send lifecycle). |
| 3.6 | `hotdog/js_bindings.go:1588,1830:fetch/XHR error detection` | High | Transport failure is detected by `resp.Code == 0 && strings.Contains(resp.Body, "error")`, so any **legitimate** response body containing the substring "error" (error pages, docs, search results) is misclassified as a transport error and rejected. | Use the typed signal instead of body sniffing: `if resp.Code < 0 { scheduleFetchError(...) }` (`sauce` already sets `Code = -1` on transport/read errors). |
| 3.7 | `hotdog/js_bindings.go:524,1240,1374,1786` (+`browser/extensions/chrome_api.go:2790`) | Medium | Every JS callback invoked from a Go goroutine (event listeners, rAF callbacks, window listeners, XHR listeners, alarm listeners) discards the returned error with `_, _ =`, so JS exceptions in page scripts are **silently swallowed** (no console.error, no report); worse, goja's `runWrapped` **re-panics** non-catchable Go panics, so a nil-deref in any bound Go function running from a timer/network goroutine (outside the VM call stack) kills the whole process. | Route errors to the page and contain panics: `if _, err := listener(...); err != nil { c.error(err) /* console.error */ }; defer func(){ if r := recover(); r != nil { c.error(fmt.Errorf("uncaught in %s: %v", hook, r)) } }()`. |
| 3.8 | `browser/extensions/chrome_api.go:2783-2793:FireAlarm` (+`manager.go:140` `AfterFunc`) | High | `chrome.alarms` fires the extension's `onAlarm` listeners directly on the timer goroutine — `cb(...)` and `c.vm.ToValue(...)` touch the goja runtime with no `jsRuntimeMu` — racing any concurrent page/extension JS execution on the same VM. | Dispatch through the owning window's serialized path: `wc.jsRuntimeMu.lock(); defer wc.jsRuntimeMu.unlock()` around the listener loop (or queue the fire onto the window's frame queue). |
| 3.9 | `sauce/sauce.go:758-790:webRequestObservers` | High | The global `webRequestObservers` slice is appended (extension registration), nil-slotted (unregister) and ranged (inside every `RoundTrip`) with **no synchronization**, and unregister never shrinks the slice — a data race under concurrent extension install + fetch, plus unbounded growth of dead slots. | Guard with `sync.RWMutex` (copy under RLock for iteration) and actually remove: `observers = append(observers[:id], observers[id+1:]...)`. |
| 3.10 | `sauce/sauce.go:723-735:loadErrorPage` | Medium | The error string is interpolated raw into `<h1>` without escaping; `http.Client`/`url.Parse` error text embeds the **attacker-controlled URL**, so a crafted load (`http://x?m="<img src=x onerror=…>"`) injects markup into the browser's own error page (self-XSS on the internal origin). | `"<h1>" + html.EscapeString(err) + "</h1>"` (`html` = `net/url`-style `html.EscapeString` from `golang.org/x/net/html` or a local escaper). |
| 3.11 | `mustard/frame.go:70-108:Frame.draw` (ebiten frame loop) | Medium | The per-frame draw path (and everything it calls: `bun` layout/paint, `gg` operations) runs with **no panic recovery** anywhere in the UI loop — the only `recover()` in the tree is in `browser/extensions/crash/recovery.go` — so any rendering bug (nil font, nil `Style`, negative box, out-of-bounds pixel — see 1.8/1.9/2.10) becomes a process crash instead of a dropped frame. | Wrap the frame body: `defer func(){ if r := recover(); r != nil { logRenderPanic(r) } }()` in the ebiten `Layout`/`Update`/`Draw` entry, and report to the browser's error logger. |
| 3.12 | `hotdog/window_context.go:318,144-288:LoadHTML/InitJSRuntime` | Medium | The goja realm is created once per window and **reused across navigations**: `window`-level event listeners, globals and content-script state registered by the previous page survive into the new page (real browsers install a fresh realm per document), so stale handlers fire against the new DOM and state accumulates per navigation. | On navigation, either reset the realm (`wc.jsRuntime = goja.New()` + rebind) or, minimally, clear `eventRegistry`/window listeners and re-run `InitJSRuntime` only after detaching old document references. |

## Critical Findings

### C-1 · `postMessage` cross-window delivery without the target VM lock (heap corruption / crash)
- **Subsystem:** Concurrency / JavaScript — `hotdog/js_bindings.go:1432-1440` (`postMessage`)
- **Why critical:** two windows posting to each other (or a post racing a target-side `requestAnimationFrame`/`RunJS`) execute goja on two goroutines against the same (or two) VMs. Goja is explicitly not thread-safe; the observable failures are data races, corrupted VM state, and process crashes — nondeterministic, hard to reproduce, and reachable from any page pair (same-origin or `*`).
- **Fix:**
```go
// postMessage, per-target delivery:
targetWC.jsRuntimeMu.lock()
defer targetWC.jsRuntimeMu.unlock()
targetWC.EmitEvent("message", eventObj)
```
  (The lock is reentrant, so the same-window case — where the sender already holds its own lock — remains safe. Delete the incorrect "Goja is internally synchronized" comment at line 1437.)

### C-2 · Deterministic layout crashes on the live render path (nil font / nil Style / nil RenderBox)
- **Subsystem:** Memory, DOM & Rendering — `bun/bun.go:18` (`RenderDocument`), `bun/layout.go:15,40` (`createFlexNodes`/`createRenderTree`), `bun/fonts.go:9-15` (`sansSerif`), `gg/context.go:702-705,853-859` (`SetFont`/`MeasureString`)
- **Why critical:** the shipping shell now renders every tab through `bun`. Three independent nil paths panic on ordinary pages: (a) `RenderDocument` writes `document.DOM.RenderBox.Width` on a freshly parsed document whose `RenderBox` is nil; (b) `createFlexNodes`/`createRenderTree` dereference `root.Style.Display` on any node without a stylesheet; (c) `calculateNode` assigns the default style `FontWeight: 0`, and `sansSerif[0]` is a **nil** `font.Face` (only 300/400/600/700/800 are mapped) — `gg.Context` then holds a nil face and `MeasureString`/`drawString` dereference it. With no `recover()` in the ebiten frame loop (3.11), each is a process death.
- **Fix:**
```go
// bun/bun.go — guard the root box:
if document.DOM.RenderBox == nil { document.DOM.RenderBox = &hotdog.RenderBox{} }
document.DOM.RenderBox.Width = float64(ctx.Width())

// bun/layout.go — nil-style guard at both entry points:
if root == nil || root.Style == nil { return nil }

// bun/fonts.go — total map: fall back to 400, never nil:
face := sansSerif[weight]
if face == nil { face = sansSerif[400] }
return face
```

### C-3 · Guaranteed long-session memory growth (navigation leak + tab leak + unbounded caches)
- **Subsystem:** Memory — `hotdog/window_context.go:315` (`LoadHTML`), `browser/shell/shell.go:508-511` (`loadTabContent`), `sauce/sauce.go:799-800` (`cache`/`imageCache`), `mustard/canvas.go:170` (`ensureContexts`)
- **Why critical:** RSS grows monotonically with use, independent of any bug in page code: every navigation leaks the previous DOM + JS wrapper set (1.1/3.12), every tab ever opened leaks its whole engine (1.2), every GET/image accumulates in unbounded global caches (1.12, and racy — 3.9's neighbors), and each canvas widget holds a ~92 MB buffer (1.7). A 30-minute session exhausts RAM.
- **Fix (minimal viable set):**
```go
// shell tab close:
func (bs *BrowserShell) closeTab(tab *Tab) {
    if wc := bs.tabWindowContexts[tab.ID]; wc != nil {
        wc.Destroy()
        delete(bs.tabWindowContexts, tab.ID)
    }
    ...
}
// navigation reset (inside LoadHTML):
wc.eventMu.Lock()
wc.nodeWrappers = make(map[*NodeDOM]*goja.Object)
wc.nodeEventRegistry = make(map[*NodeDOM]map[string][]eventHandler)
wc.eventRegistry = make(map[string][]eventHandler)
wc.eventMu.Unlock()
wc.ClearLayoutCache()

// sauce caches: replace unbounded slices with a bounded LRU (max ~256 resources, ~128 images)
// mustard canvas: drawingContext = gg.NewContext(width, max(viewportHeight, contentHeight))
```

### C-4 · `removeEventListener` is a silent no-op (listeners accumulate and double-fire)
- **Subsystem:** Off-by-One & Boundaries / JS — `hotdog/window_context.go:377-386` (`functionKey`), `js_bindings.go:530,1380`
- **Why critical (for a browser):** verified against the pinned goja: `AssertFunction` returns a fresh Go closure per call and `reflect.Value.Pointer()` yields a **distinct value per closure instance** (probe: register `4782464` vs remove `4782432` for the same JS function). The removal key can never equal the registration key, so no listener is ever removed — memory grows per add/remove cycle and stale listeners fire on later events.
- **Fix:**
```go
// Register keyed by the JS object itself:
func (w *JSDOMWrapper) addEventListener(call goja.FunctionCall) goja.Value {
    ...
    w.windowCtx.registerNodeEventJS(w.node, eventType, listener, wrappedHandler) // map[goja.Value]
}
func (w *JSDOMWrapper) removeEventListener(call goja.FunctionCall) goja.Value {
    ...
    w.windowCtx.RemoveNodeEventByJSValue(w.node, eventType, call.Arguments[1])
}
```

### C-5 · JS errors are invisible and can kill the process
- **Subsystem:** Concurrency / Error handling — `hotdog/js_bindings.go:524,1240,1374,1786`; goja `runWrapped` (verified: re-panics non-catchable panics when the VM call stack is empty, i.e. for all timer/network-goroutine callbacks)
- **Why critical:** the debugging surface is broken at both ends — normal JS exceptions in listeners/rAF/fetch/XHR/alarm callbacks vanish (`, _ =`), while *any* Go-level fault in those same paths (the nil-font and nil-Style bugs in C-2 are exactly this shape) is an unrecovered panic in a background goroutine → the entire browser process dies with no report.
- **Fix:**
```go
func runJSListener(name string, log consoleLogger, fn func() error) {
    defer func() {
        if r := recover(); r != nil {
            log.error(fmt.Sprintf("uncaught in %s: %v", name, r))
        }
    }()
    if err := fn(); err != nil {
        log.error(err) // e.g. "Uncaught TypeError: …"
    }
}
```
  plus a `recover()` in the ebiten frame entry (3.11) so render faults drop a frame instead of the process.

---

## Appendix · Verified-as-fixed since the 2026-11-01 architecture audit
For completeness, the following previously-known defects were re-verified as **fixed at this HEAD** and are not reported above: double-wrapped `originCookieTransport` (sauce.go:957-963 guard), transport-error swallowing (`Code = -1` at sauce.go:978/996/1004 + `LoadURL` reject at window_context.go:354), `document.title` bound as a function (js_bindings.go:188 now uses `DefineAccessorProperty`), first-frame `Frame.draw` index-out-of-range (visible-children snapshot at frame.go:88-100), and the `VerticalFrame`/`HorizontalFrame` orientation swap (widget.go:46-86 now matches shell usage: root column = `VerticalFrame` stacking top→bottom).

## Appendix · Method notes
- Parallel subagent passes were executed over the same HEAD; findings were cross-checked across passes (e.g., the fetch goroutines appear in both memory and concurrency findings and are listed once each per class).
- Empirical probes (scratch dir, throwaway): goja `Callable` closure-pointer behavior, `cascadia.Parse` on group/`:has`/`@media` selectors. Both matched the pinned versions in `go.sum`.
- Severity scale: **Critical** = data race/crash/corruption reachable from normal use; **High** = crash, guaranteed leak, or spec-breaking functional failure on common inputs; **Medium** = leak/race/incorrect behavior on less common inputs or latent API bugs; **Low** = cosmetic/edge degradation.
