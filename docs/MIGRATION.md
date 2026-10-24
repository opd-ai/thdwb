# Migration Guide: Transitioning to the New thdwb Toolkit API

This guide documents the transition from the existing thdwb codebase to the new modular toolkit API. It covers breaking changes, API mappings, and migration steps.

---

## Overview

thdwb has been refactored into a modular architecture with separate packages for each concern:

| Package | Purpose |
|---------|---------|
| `mustard` | UI toolkit (windows, widgets, event loop, OpenGL) |
| `hotdog` | JavaScript runtime, DOM, per-window isolation |
| `bun` | Layout engine, rendering |
| `mayo` | CSS parsing and styling |
| `ketchup` | HTML parsing |
| `sauce` | Network layer, cookies, certificates |
| `gg` | 2D graphics library |
| `profiler` | Performance profiling |
| `assets` | Embedded assets (fonts, icons, pages) |

---

## Breaking Changes

### 1. Application Entry Point

**Before (old monolithic structure):**
```go
// Old: Single main.go with everything
func main() {
    // Initialize everything inline
    window := createWindow()
    app := createApp()
    app.Run()
}
```

**After (new modular structure):**
```go
// New: Modular with explicit package imports
import (
    "github.com/danfragoso/thdwb/mustard"
)

func main() {
    mustard.SetGLFWHints()
    app := mustard.CreateNewApp("thdwb")
    window := mustard.CreateNewWindow("thdwb", 900, 600, true)
    rootFrame := mustard.CreateFrame(mustard.VerticalFrame)
    window.SetRootFrame(rootFrame)
    app.AddWindow(window)
    app.ActivateWindow(window)
    window.Show()
    app.Run(func() {
        // Per-frame callback
    })
}
```

### 2. Window Creation

**Before:**
```go
// Old: Window creation embedded in app logic
win := createWindow("title", 800, 600)
```

**After:**
```go
// New: Explicit mustard package
window := mustard.CreateNewWindow("title", 800, 600, true) // hiDPI parameter added
```

The `hiDPI` parameter is now required. Pass `true` for high-DPI displays, `false` otherwise.

### 3. Widget Creation

**Before:**
```go
// Old: Widget constructors varied by type
btn := createButton("Click Me")
lbl := createLabel("Hello")
inp := createInput()
```

**After:**
```go
// New: Consistent Create*Widget pattern
btn := mustard.CreateButtonWidget("Click Me", nil)
lbl := mustard.CreateLabelWidget("Hello")
inp := mustard.CreateInputWidget()
canvas := mustard.CreateCanvasWidget(func(c *mustard.CanvasWidget) {})
tree := mustard.CreateTreeWidget()
scrollbar := mustard.CreateScrollBarWidget(mustard.VerticalScrollBar)
text := mustard.CreateTextWidget("Hello")
image := mustard.CreateImageWidget(nil)
```

### 4. Frame Creation

**Before:**
```go
// Old: Frame creation without explicit orientation
frame := createFrame()
```

**After:**
```go
// New: Orientation is required
frame := mustard.CreateFrame(mustard.VerticalFrame)
// or
frame := mustard.CreateFrame(mustard.HorizontalFrame)
```

### 5. App/Window Management

**Before:**
```go
// Old: App and window management combined
app := createApp()
win := app.newWindow("url")
app.run()
```

**After:**
```go
// New: Separate App and Window concepts
app := mustard.CreateNewApp("thdwb")
window := mustard.CreateNewWindow("thdwb", 900, 600, true)
rootFrame := mustard.CreateFrame(mustard.VerticalFrame)
window.SetRootFrame(rootFrame)
app.AddWindow(window)
app.ActivateWindow(window) // Enables tabbed mode
window.Show()
app.Run(func() {})
```

Key changes:
- `CreateNewApp(name string)` replaces `createApp()`
- `AddWindow(window)` replaces `app.newWindow()`
- `ActivateWindow(window)` is required for tabbed mode
- `SetRootFrame(frame)` must be called before showing the window

### 6. Widget Property Setters

**Before:**
```go
// Old: Inconsistent setter patterns
btn.setWidth(100)
btn.setText("Click Me")
lbl.setColor("#ff0000")
```

**After:**
```go
// New: Consistent Set* pattern with validation
btn.SetWidth(100)
btn.SetContent("Click Me")
lbl.SetFontColor("#FF0000") // Must start with #
lbl.SetBackgroundColor("#00FF00") // Must start with #
```

Color setters now validate that the value starts with `#`. Invalid colors are silently ignored.

### 7. JavaScript Runtime (hotdog)

**Before:**
```go
// Old: JS runtime setup varied
ctx := createJSContext()
ctx.runScript("console.log('hi')")
```

**After:**
```go
// New: Per-window isolation with explicit setup
import (
    hotdog "github.com/danfragoso/thdwb/hotdog"
    "github.com/danfragoso/thdwb/mustard"
)

wc := hotdog.NewWindowContext(settings, buildInfo, profiler)
wc.SetOrigin("https://example.com")
wc.InitJSRuntime()
result, err := wc.RunJS("console.log('hi')")
```

Key changes:
- `NewWindowContext` now requires `settings`, `buildInfo`, and `profiler` parameters
- `SetOrigin()` must be called before `InitJSRuntime()`
- `InitJSRuntime()` sets up all global objects (window, document, console, etc.)
- `RunJS()` returns `(goja.Value, error)`

### 8. Event Handling

**Before:**
```go
// Old: Event handling varied by widget type
btn.onClick = func() { ... }
input.onKeyPress = func(key) { ... }
```

**After:**
```go
// New: Consistent listener registration on Window
window.AttachClickEventListener(func(key mustard.MustardKey) { ... })
window.AttachKeyboardInputEventListener(func(input mustard.KeyboardInput) { ... })
window.AttachPointerPositionEventListener(func(x, y float64) { ... })
window.AttachScrollEventListener(func(direction int) { ... })
window.AttachTextInputEventListener(func(char rune) { ... })
window.AttachActiveChangeListener(func(active bool) { ... })
```

### 9. DOM Bindings (JavaScript)

**Before:**
```js
// Old: DOM API varied
var el = document.getElementById("my-id");
el.innerHTML = "Hello";
```

**After:**
```js
// New: Standard DOM API via Goja bindings
var el = document.getElementById("my-id");
el.textContent = "Hello";
el.innerHTML = "Hello";
el.id = "new-id";
el.className = "my-class";
el.style.color = "red";

el.addEventListener("click", function() { ... });
el.removeEventListener("click", handler);
```

Available document methods:
- `querySelector(selector)` - First matching element
- `querySelectorAll(selector)` - All matching elements
- `getElementById(id)` - Element by ID
- `getElementsByClassName(name)` - Elements by class
- `getElementsByTagName(name)` - Elements by tag
- `createElement(tagName)` - Create element
- `createTextNode(text)` - Create text node

### 10. Storage APIs

**Before:**
```js
// Old: Storage varied
localStorage.setItem("key", "value");
sessionStorage.setItem("key", "value");
```

**After:**
```js
// New: Origin-partitioned storage
localStorage.setItem("key", "value");
localStorage.getItem("key");
localStorage.removeItem("key");
localStorage.clear();

sessionStorage.setItem("key", "value");
sessionStorage.getItem("key");
sessionStorage.removeItem("key");
sessionStorage.clear();
```

- `localStorage`: Persistent per-origin, survives across window reloads
- `sessionStorage`: Per-window, per-origin, cleared on window close

### 11. Network Layer (sauce)

**Before:**
```go
// Old: Network calls varied
resp := fetch(url)
resp := getResource(url)
```

**After:**
```go
// New: Consistent sauce package API
import sauce "github.com/danfragoso/thdwb/sauce"

resource := sauce.GetResource(urlString)
resource := sauce.FetchExternalPageWithOptions(url, method, body, headers)
resource := sauce.MakeCORSRequest(url, origin, method, body, headers)
imageData, err := sauce.GetImage(url)
```

### 12. CSS Styling (mayo)

**Before:**
```go
// Old: CSS parsing varied
styles := parseCSS(cssString)
```

**After:**
```go
// New: mayo package
import mayo "github.com/danfragoso/thdwb/mayo"

stylesheets := mayo.ParseStylesheet(cssString)
mayo.ApplyStylesheets(document)
mayo.ApplyInheritance(root)
```

### 13. HTML Parsing (ketchup)

**Before:**
```go
// Old: HTML parsing varied
doc := parseHTML(htmlString)
```

**After:**
```go
// New: ketchup package
import ketchup "github.com/danfragoso/thdwb/ketchup"
import hotdog "github.com/danfragoso/thdwb/hotdog"

wc := hotdog.NewWindowContext(settings, buildInfo, profiler)
doc := ketchup.ParseHTML(htmlString, wc)
doc := ketchup.ParsePlainText(text, wc)
```

### 14. Layout Engine (bun)

**Before:**
```go
// Old: Layout embedded in render
renderDocument(ctx, doc)
```

**After:**
```go
// New: bun package with explicit layout mode
import bun "github.com/danfragoso/thdwb/bun"

err := bun.RenderDocument(ctx, document, false) // false = legacy layout
// or
err := bun.RenderDocument(ctx, document, true)  // true = experimental flexbox
```

### 15. Graphics (gg)

**Before:**
```go
// Old: gg usage varied
ctx := gg.NewContext(800, 600)
```

**After:**
```go
// New: Same gg API, now in its own package
import gg "github.com/danfragoso/thdwb/gg"

ctx := gg.NewContext(800, 600)
ctx.SetHexColor("#ff0000")
ctx.DrawRectangle(0, 0, 100, 100)
ctx.Fill()
```

---

## Migration Checklist

- [ ] Update imports to use new package paths
- [ ] Replace `createApp()` with `mustard.CreateNewApp(name)`
- [ ] Replace `createWindow()` with `mustard.CreateNewWindow(title, width, height, hiDPI)`
- [ ] Add `window.SetRootFrame(frame)` before `window.Show()`
- [ ] Add `app.ActivateWindow(window)` for tabbed mode
- [ ] Replace widget constructors with `Create*Widget` pattern
- [ ] Replace `setXxx()` with `SetXxx()` (capitalized)
- [ ] Add `#` prefix to color values
- [ ] Update event handlers to use `Attach*EventListener` pattern
- [ ] Update JS runtime setup to use `NewWindowContext(settings, buildInfo, profiler)`
- [ ] Call `wc.SetOrigin()` before `wc.InitJSRuntime()`
- [ ] Update DOM JS code to use standard API (textContent, querySelector, etc.)
- [ ] Update storage code to use localStorage/sessionStorage
- [ ] Update network calls to use sauce package functions
- [ ] Update CSS parsing to use mayo.ParseStylesheet
- [ ] Update HTML parsing to use ketchup.ParseHTML
- [ ] Update layout calls to use bun.RenderDocument with explicit mode

---

## Example: Complete Migration

### Before (Old Monolithic Code)

```go
package main

import (
    "fmt"
    "github.com/go-gl/glfw/v3.3/glfw"
)

func main() {
    // Old: Everything inline
    if err := glfw.Init(); err != nil {
        panic(err)
    }
    defer glfw.Terminate()

    window, err := glfw.CreateWindow(800, 600, "My Browser", nil, nil)
    if err != nil {
        panic(err)
    }
    defer window.Destroy()

    window.MakeContextCurrent()

    // Old: Inline widget creation
    btn := createButton("Click Me")
    btn.onClick = func() {
        fmt.Println("Clicked!")
    }

    // Old: Inline event loop
    for !window.ShouldClose() {
        glfw.PollEvents()
        // render...
        window.SwapBuffers()
    }
}
```

### After (New Modular Code)

```go
package main

import (
    "fmt"
    "github.com/danfragoso/thdwb/mustard"
)

func main() {
    mustard.SetGLFWHints()

    app := mustard.CreateNewApp("my-browser")

    window := mustard.CreateNewWindow("My Browser", 800, 600, true)
    rootFrame := mustard.CreateFrame(mustard.VerticalFrame)
    window.SetRootFrame(rootFrame)

    btn := mustard.CreateButtonWidget("Click Me", nil)
    btn.SetWidth(100)
    btn.SetHeight(40)
    btn.SetContent("Click Me")
    window.RegisterButton(btn, func() {
        fmt.Println("Clicked!")
    })
    rootFrame.AttachWidget(btn)

    app.AddWindow(window)
    app.ActivateWindow(window)
    window.Show()

    app.Run(func() {
        // Per-frame callback
    })
}
```

---

## Compatibility Notes

| Feature | Old | New | Notes |
|---------|-----|-----|-------|
| Window creation | `createWindow()` | `mustard.CreateNewWindow()` | Added `hiDPI` param |
| App creation | `createApp()` | `mustard.CreateNewApp()` | Added `name` param |
| Frame creation | `createFrame()` | `mustard.CreateFrame()` | Added `orientation` param |
| Button | `createButton()` | `mustard.CreateButtonWidget()` | Added `asset` param |
| Label | `createLabel()` | `mustard.CreateLabelWidget()` | Same |
| Input | `createInput()` | `mustard.CreateInputWidget()` | Same |
| Canvas | `createCanvas()` | `mustard.CreateCanvasWidget()` | Added `renderer` param |
| Color setters | `setColor()` | `SetFontColor()`/`SetBackgroundColor()` | Must start with `#` |
| Event listeners | `onClick =` | `window.RegisterButton()` | More explicit |
| JS runtime | `createJSContext()` | `hotdog.NewWindowContext()` | Requires settings/buildInfo/profiler |
| DOM API | Varies | Standard (querySelector, etc.) | More spec-compliant |
| Storage | Varies | localStorage/sessionStorage | Origin-partitioned |

---

## Getting Help

- **API Documentation**: See `docs/07-javascript-runtime.md` and `docs/08-ui-toolkit.md`
- **Examples**: See `examples/` directory for working code
- **Tests**: See `*_test.go` files in each package for usage patterns
- **Issues**: File issues at https://github.com/danfragoso/thdwb/issues