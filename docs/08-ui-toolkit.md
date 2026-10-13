# Chapter 8: UI Toolkit & Application — mustard/app

> **In this chapter**: How thdwb builds the browser UI (tabs, address bar, buttons) using mustard widgets, and how the application manages multiple windows/tabs. This is the "house frame and furniture" of the browser.

---

## 8.1 What Is the Browser UI?

The **browser UI** (also called **chrome** — not to be confused with Google Chrome) is everything *around* the web content: tabs, address bar, back/forward buttons, bookmarks, menus, settings.

**Analogy**: If the web page is a painting, the browser UI is the frame, the wall hooks, the gallery labels, and the lighting — all the stuff that lets you view and navigate between paintings.

```
┌─────────────────────────────────────────────────────────────────┐
│                    BROWSER WINDOW (mustard.Window)               │
├─────────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ TAB BAR (Frame with LayoutHorizontal)                      │  │
│  │  [Tab 1: Google]  [Tab 2: GitHub]  [Tab 3: MDN]  [+]     │  │
│  └───────────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ TOOLBAR (Frame)                                            │  │
│  │  [←] [→] [⟳] [🏠] [_______________________________] [🔍]  │  │
│  │   Back  Forward  Reload  Home      Address Bar        Search │
│  └───────────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ WEB CONTENT AREA (Canvas widget)                           │  │
│  │                                                           │  │
│  │        Rendered web page (gg.Context)                     │  │
│  │                                                           │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 8.2 Key Terms (Defined in [Glossary](../glossary.md))

- **Chrome (Browser UI)** — The browser's own interface (not the web page)
- **Widget** — Reusable UI component (button, input, label, etc.)
- **Frame** — Container widget that holds other widgets
- **Layout** — Algorithm for positioning widgets (vertical, horizontal, flex)
- **Event Loop** — Main loop processing input, updating, rendering
- **Hit Testing** — Finding which widget is under the mouse
- **Focus** — Which widget receives keyboard input
- **Tab** — A web content container with its own WindowContext

---

## 8.3 The mustard Package: UI Toolkit

### File Overview (Recap)

```
mustard/
├── window.go           # Window creation, OpenGL, event loop
├── app.go              # Application: tab management, main entry
├── frame.go            # Frame tree (widget hierarchy)
├── widget.go           # Base Widget interface
├── button.go           # Button widget
├── label.go            # Label widget
├── input.go            # Text input widget
├── canvas.go           # Canvas widget (web content)
├── scrollbar.go        # Scrollbar widget
├── text.go             # Text rendering helpers
├── image.go            # Image widget
├── events.go           # Event types and handling
├── contextMenu.go      # Context menu
├── tree.go             # Tree widget
├── types.go            # Shared types
├── gl.go               # OpenGL shader setup
├── shaders.go          # Shader sources
├── mui.go              # Material UI helpers
└── *_test.go
```

---

## 8.4 Application Entry Point: app.go

### Main Function

```go
// main is the entry point (in mustard/app.go or separate main.go).
func main() {
    // 1. Initialize GLFW
    mustard.SetGLFWHints()

    // 2. Create application
    app := mustard.NewApp()

    // 3. Create first window with a tab
    win := app.NewWindow("https://example.com")

    // 4. Run application (blocks)
    app.Run()
}
```

### App Struct: Tab Manager

```go
// App manages multiple windows and tabs.
type App struct {
    Windows     []*Window
    ActiveWindow *Window
    Settings    *Settings
}

// NewApp creates the application.
func NewApp() *App {
    return &App{
        Windows: make([]*Window, 0),
        Settings: LoadSettings(),
    }
}

// NewWindow creates a new browser window with initial URL.
func (app *App) NewWindow(initialURL string) *Window {
    win := mustard.CreateNewWindow("thdwb", 1024, 768, true)

    // Create tab with web content
    tab := NewTab(initialURL)
    win.SetContent(tab)

    app.Windows = append(app.Windows, win)
    app.ActiveWindow = win

    return win
}
```

---

## 8.5 Tab: Web Content Container

Each tab holds a **WindowContext** (from hotdog) which has its own DOM, JS VM, network client, etc.

```go
// Tab represents a single browser tab with web content.
type Tab struct {
    WindowCtx   *hotdog.WindowContext  // Isolated JS + DOM + Network
    Canvas      *mustard.Canvas        // Widget for rendering
    URL         string
    Title       string
    Favicon     image.Image
    Loading     bool
    History     []string
    HistoryIndex int
}

// NewTab creates a tab and starts loading the URL.
func NewTab(url string) *Tab {
    ctx := hotdog.NewWindowContext()

    canvas := mustard.NewCanvas(800, 600)
    canvas.SetDocument(ctx.Document)

    tab := &Tab{
        WindowCtx: ctx,
        Canvas:    canvas,
        URL:       url,
    }

    // Start loading in background
    go tab.LoadURL(url)

    return tab
}

// LoadURL navigates the tab to a new URL.
func (tab *Tab) LoadURL(url string) {
    tab.Loading = true
    tab.Canvas.SetLoading(true)

    // Fetch via window's isolated network client
    resource, err := tab.WindowCtx.Fetch(url)
    if err != nil {
        tab.LoadError(err)
        return
    }

    // Parse HTML
    doc := ketchup.ParseHTML(string(resource.Body), tab.WindowCtx)
    tab.WindowCtx.Document = doc

    // Update title
    tab.Title = bun.GetPageTitle(doc.DOM)

    // Render to canvas
    dc := gg.NewContext(tab.Canvas.Width, tab.Canvas.Height)
    bun.RenderDocument(dc, doc, false)
    tab.Canvas.SetContext(dc)

    tab.Loading = false
    tab.Canvas.SetLoading(false)
    tab.Canvas.MarkRepaint()
}
```

---

## 8.6 Tab Bar: Tab Management UI

### Tab Bar Widget

```go
// TabBar is a horizontal frame holding tab buttons.
type TabBar struct {
    *Frame
    App        *App
    TabButtons map[*Tab]*Button
    NewTabBtn  *Button
}

func NewTabBar(app *App) *TabBar {
    tb := &TabBar{
        Frame:      NewFrame(),
        App:        app,
        TabButtons: make(map[*Tab]*Button),
    }
    tb.Layout = LayoutHorizontal

    // New tab button
    tb.NewTabBtn = NewButton("+")
    tb.NewTabBtn.OnClick = func() {
        app.NewWindow("about:blank")
    }
    tb.AddChild(tb.NewTabBtn)

    return tb
}

func (tb *TabBar) AddTab(tab *Tab) {
    btn := NewButton(tab.Title)
    btn.OnClick = func() {
        tb.App.SwitchToTab(tab)
    }
    btn.OnRightClick = func() {
        tb.ShowTabContextMenu(tab)
    }
    tb.TabButtons[tab] = btn
    tb.InsertChildBefore(btn, tb.NewTabBtn)
    tb.MarkReflow()
}

func (tb *TabBar) RemoveTab(tab *Tab) {
    if btn, ok := tb.TabButtons[tab]; ok {
        tb.RemoveChild(btn)
        delete(tb.TabButtons, tab)
        tb.MarkReflow()
    }
}
```

---

## 8.7 Toolbar: Navigation Controls

```go
// Toolbar holds navigation buttons and address bar.
type Toolbar struct {
    *Frame
    Tab        *Tab
    BackBtn    *Button
    ForwardBtn *Button
    ReloadBtn  *Button
    HomeBtn    *Button
    AddressBar *Input
    GoBtn      *Button
}

func NewToolbar(tab *Tab) *Toolbar {
    tb := &Toolbar{
        Frame:  NewFrame(),
        Tab:    tab,
    }
    tb.Layout = LayoutHorizontal

    // Back button
    tb.BackBtn = NewButton("←")
    tb.BackBtn.OnClick = func() { tab.GoBack() }
    tb.AddChild(tb.BackBtn)

    // Forward button
    tb.ForwardBtn = NewButton("→")
    tb.ForwardBtn.OnClick = func() { tab.GoForward() }
    tb.AddChild(tb.ForwardBtn)

    // Reload button
    tb.ReloadBtn = NewButton("⟳")
    tb.ReloadBtn.OnClick = func() { tab.Reload() }
    tb.AddChild(tb.ReloadBtn)

    // Home button
    tb.HomeBtn = NewButton("🏠")
    tb.HomeBtn.OnClick = func() { tab.LoadURL("https://example.com") }
    tb.AddChild(tb.HomeBtn)

    // Address bar (flex: takes remaining space)
    tb.AddressBar = NewInput()
    tb.AddressBar.Placeholder = "Enter URL or search..."
    tb.AddressBar.OnEnter = func(url string) {
        tab.LoadURL(normalizeURL(url))
    }
    tb.AddressBar.FlexGrow = 1
    tb.AddChild(tb.AddressBar)

    // Go/Search button
    tb.GoBtn = NewButton("🔍")
    tb.GoBtn.OnClick = func() {
        tab.LoadURL(normalizeURL(tb.AddressBar.Text))
    }
    tb.AddChild(tb.GoBtn)

    return tb
}
```

---

## 8.8 Address Bar: URL Input & Display

```go
// normalizeURL converts user input to a valid URL.
func normalizeURL(input string) string {
    input = strings.TrimSpace(input)

    // Already a URL?
    if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
        return input
    }

    // Has scheme?
    if strings.Contains(input, "://") {
        return input
    }

    // Looks like a domain?
    if strings.Contains(input, ".") && !strings.Contains(input, " ") {
        return "https://" + input
    }

    // Otherwise treat as search query
    return "https://www.google.com/search?q=" + url.QueryEscape(input)
}
```

### Updating Address Bar on Navigation

```go
func (tab *Tab) onNavigation(url string) {
    tab.URL = url
    tab.Toolbar.AddressBar.SetText(url)
    tab.Toolbar.AddressBar.CursorPos = len(url)
}
```

---

## 8.9 Window Content: Assembling the UI

```go
func (win *Window) SetContent(tab *Tab) {
    // Create root frame (vertical layout)
    root := NewFrame()
    root.Layout = LayoutVertical

    // 1. Tab bar
    tabBar := NewTabBar(win.App)
    tabBar.AddTab(tab)
    root.AddChild(tabBar)

    // 2. Toolbar
    toolbar := NewToolbar(tab)
    tab.Toolbar = toolbar
    root.AddChild(toolbar)

    // 3. Web content canvas (flex: takes remaining space)
    tab.Canvas.FlexGrow = 1
    root.AddChild(tab.Canvas)

    win.rootFrame = root
    win.needsReflow = true
}
```

---

## 8.10 Event Loop: The Heartbeat

```go
func (win *Window) Run() {
    for !win.glfwWindowShouldClose() {
        // 1. Process OS events (mouse, keyboard, resize)
        win.processEvents()

        // 2. Run JS microtasks (Promises, queueMicrotask)
        win.ActiveTab.WindowCtx.EventLoop.RunMicrotasks()

        // 3. Reflow if needed (layout changes)
        if win.needsReflow {
            win.reflow()
        }

        // 4. Repaint if needed (visual changes)
        if win.needsRepaint {
            win.repaint()
        }

        // 5. Swap buffers (show rendered frame)
        win.swapBuffers()

        // 6. Sleep briefly to not hog CPU
        glfw.PollEvents()
        time.Sleep(1 * time.Millisecond)
    }
}
```

### Reflow: Layout Widgets

```go
func (win *Window) reflow() {
    win.needsReflow = false

    // Calculate layout for entire widget tree
    win.rootFrame.Layout(0, 0, float64(win.width), float64(win.height))

    // Also reflow web content if needed
    if win.ActiveTab != nil && win.ActiveTab.WindowCtx.NeedsReflow {
        win.ActiveTab.WindowCtx.Reflow()
    }
}
```

### Repaint: Draw Everything

```go
func (win *Window) repaint() {
    win.needsRepaint = false

    // 1. Clear gg context
    win.context.SetRGB(1, 1, 1)
    win.context.Clear()

    // 2. Draw widget tree (browser chrome)
    win.rootFrame.Draw(win.context)

    // 3. Draw web content (already rendered to tab's canvas)
    if win.ActiveTab != nil {
        tab := win.ActiveTab
        win.context.DrawImage(tab.Canvas.Context.Image(),
            int(tab.Canvas.X), int(tab.Canvas.Y))
    }

    // 4. Upload to OpenGL texture
    win.backend.UploadTexture(win.context.Image())

    // 5. Draw full-screen quad
    win.backend.DrawQuad()
}
```

---

## 8.11 Input Handling: Mouse & Keyboard

### Mouse Events

```go
func (win *Window) processEvents() {
    for _, event := range win.pendingEvents {
        switch event.Type {
        case EventMouseMove:
            win.handleMouseMove(event.X, event.Y)
        case EventMouseDown:
            win.handleMouseDown(event.X, event.Y, event.Button)
        case EventMouseUp:
            win.handleMouseUp(event.X, event.Y, event.Button)
        case EventMouseScroll:
            win.handleScroll(event.X, event.Y, event.ScrollY)
        }
    }
}

func (win *Window) handleMouseDown(x, y float64, button glfw.MouseButton) {
    // 1. Hit test: find widget under cursor
    target := win.hitTest(x, y)

    // 2. Focus the target
    if target != nil {
        win.setFocus(target)
    }

    // 3. Dispatch to widget
    if target != nil {
        target.HandleEvent(Event{Type: EventMouseDown, X: x, Y: y, Button: button})
    }

    // 4. If web content area, forward to JS
    if win.ActiveTab != nil && win.ActiveTab.Canvas.Contains(x, y) {
        win.ActiveTab.WindowCtx.DispatchMouseEvent(x, y, button, "down")
    }
}
```

### Keyboard Events

```go
func (win *Window) handleKeyDown(key glfw.Key, char rune, mods glfw.ModifierKey) {
    // 1. If focused widget is input, send to it
    if win.focusedWidget != nil {
        win.focusedWidget.HandleEvent(Event{
            Type: EventKeyDown, Key: key, Char: char, Modifiers: mods,
        })
        return
    }

    // 2. Global shortcuts
    if mods&glfw.ModControl != 0 {
        switch key {
        case glfw.KeyT:
            win.App.NewWindow("about:blank")  // Ctrl+T = new tab
        case glfw.KeyW:
            win.CloseTab(win.ActiveTab)       // Ctrl+W = close tab
        case glfw.KeyL:
            win.ActiveTab.Toolbar.AddressBar.Focus()  // Ctrl+L = address bar
        case glfw.KeyR:
            win.ActiveTab.Reload()            // Ctrl+R = reload
        }
    }

    // 3. Forward to web content if focused
    if win.ActiveTab != nil && win.ActiveTab.Canvas.Focused {
        win.ActiveTab.WindowCtx.DispatchKeyEvent(key, char, mods, "down")
    }
}
```

---

## 8.12 Focus Management

```go
func (win *Window) setFocus(widget Widget) {
    // Blur previous
    if win.focusedWidget != nil {
        win.focusedWidget.HandleEvent(Event{Type: EventBlur})
    }

    // Focus new
    win.focusedWidget = widget
    if widget != nil {
        widget.HandleEvent(Event{Type: EventFocus})
    }
}

// Widget interface includes focus handling
type Widget interface {
    // ...
    Focus() bool      // Can this widget receive focus?
    Blur()            // Called when losing focus
}
```

---

## 8.13 Context Menu: Right-Click Menu

```go
// ContextMenu shows on right-click.
type ContextMenu struct {
    *Frame
    Items []*MenuItem
}

type MenuItem struct {
    Label    string
    Shortcut string
    Action   func()
    Enabled  bool
}

func (cm *ContextMenu) Show(x, y float64) {
    cm.SetBounds(x, y, 200, float64(len(cm.Items)*25))
    cm.Visible = true
    cm.MarkRepaint()
}

func (cm *ContextMenu) Hide() {
    cm.Visible = false
}
```

### Tab Context Menu

```go
func (tb *TabBar) ShowTabContextMenu(tab *Tab) {
    menu := NewContextMenu()
    menu.Items = []*MenuItem{
        {Label: "Close Tab", Action: func() { tb.App.CloseTab(tab) }},
        {Label: "Close Other Tabs", Action: func() { tb.App.CloseOtherTabs(tab) }},
        {Label: "Close Tabs to Right", Action: func() { tb.App.CloseTabsToRight(tab) }},
        {Label: "Reload", Action: func() { tab.Reload() }},
        {Label: "Duplicate", Action: func() { tb.App.NewWindow(tab.URL) }},
        {Label: "Pin Tab", Action: func() { tab.Pinned = !tab.Pinned }},
    }
    menu.Show(mouseX, mouseY)
}
```

---

## 8.14 History Navigation

```go
func (tab *Tab) GoBack() {
    if tab.HistoryIndex > 0 {
        tab.HistoryIndex--
        tab.LoadURL(tab.History[tab.HistoryIndex])
    }
}

func (tab *Tab) GoForward() {
    if tab.HistoryIndex < len(tab.History)-1 {
        tab.HistoryIndex++
        tab.LoadURL(tab.History[tab.HistoryIndex])
    }
}

func (tab *Tab) addToHistory(url string) {
    // Truncate forward history
    tab.History = tab.History[:tab.HistoryIndex+1]
    tab.History = append(tab.History, url)
    tab.HistoryIndex = len(tab.History) - 1
}
```

---

## 8.15 Settings & Persistence

```go
// Settings holds user preferences.
type Settings struct {
    HomePage       string
    SearchEngine   string
    Theme          string  // "light", "dark", "system"
    FontSize       int
    EnableJS       bool
    BlockPopups    bool
    // ...
}

func LoadSettings() *Settings {
    // Load from ~/.config/thdwb/settings.json
    // Return defaults if not found
}

func (s *Settings) Save() {
    // Write to ~/.config/thdwb/settings.json
}
```

---

## 8.16 Cross-References

| Concept | Chapter |
|---------|---------|
| How widgets draw with gg | [Chapter 6: Rendering](../06-rendering-graphics.md) |
| How tabs isolate JS/Network | [Chapter 7: JavaScript Runtime](../07-javascript-runtime.md) |
| How network loads pages | [Chapter 2: Network Layer](../02-network-layer.md) |
| How layout works | [Chapter 5: Layout Engine](../05-layout-engine.md) |
| Event handling basics | [Glossary: Event](../glossary.md#event) |

---

## 8.17 Example Test: Creating a Tab

```go
// Example_NewTab demonstrates creating a tab with web content.
func Example_NewTab() {
    // This test shows the structure without actually showing a window
    ctx := hotdog.NewWindowContext()
    defer ctx.Close()

    tab := mustard.NewTab("https://example.com")

    fmt.Printf("Tab URL: %s\n", tab.URL)
    fmt.Printf("Tab Title: %s\n", tab.Title)
    fmt.Printf("Has WindowContext: %v\n", tab.WindowCtx != nil)
    fmt.Printf("Has Canvas: %v\n", tab.Canvas != nil)

    // Output:
    // Tab URL: https://example.com
    // Tab Title: 
    // Has WindowContext: true
    // Has Canvas: true
}
```

Run with: `go test -run Example_NewTab ./mustard`

---

## 8.18 Example Test: Toolbar

```go
// Example_Toolbar demonstrates creating a toolbar with navigation controls.
func Example_Toolbar() {
    ctx := hotdog.NewWindowContext()
    defer ctx.Close()

    tab := mustard.NewTab("https://example.com")
    toolbar := mustard.NewToolbar(tab)

    fmt.Printf("Toolbar children: %d\n", len(toolbar.Children))
    fmt.Printf("Has BackBtn: %v\n", toolbar.BackBtn != nil)
    fmt.Printf("Has AddressBar: %v\n", toolbar.AddressBar != nil)
    fmt.Printf("AddressBar placeholder: %s\n", toolbar.AddressBar.Placeholder)

    // Output:
    // Toolbar children: 6
    // Has BackBtn: true
    // Has AddressBar: true
    // AddressBar placeholder: Enter URL or search...
}
```

Run with: `go test -run Example_Toolbar ./mustard`

---

## 8.19 Summary

- **mustard** = UI toolkit: windows, widgets, event loop, OpenGL
- **App** = Manages multiple windows/tabs
- **Tab** = Web content container with isolated WindowContext
- **TabBar** = Horizontal frame with tab buttons + new tab button
- **Toolbar** = Navigation buttons + address bar + search
- **Window** = GLFW window + OpenGL + widget tree + event loop
- **Event loop** = Process events → JS microtasks → Reflow → Repaint → Swap
- **Focus management** = Keyboard input routing
- **Context menus** = Right-click actions
- **History** = Back/forward navigation per tab

---

## 8.20 Next Steps

You've now completed the textbook! The chapters cover:

1. **Introduction** — Browser architecture overview
2. **Network Layer** — HTTP, cookies, cache, certificates
3. **HTML Parsing & DOM** — Tokenization, tree construction
4. **CSS Styling** — Parsing, cascade, inheritance, computed styles
5. **Layout Engine** — Block, inline, flexbox layout
6. **Rendering & Graphics** — gg 2D graphics, mustard UI toolkit
7. **JavaScript Runtime** — Goja VM, DOM bindings, Fetch, events
8. **UI Toolkit & Application** — Tabs, toolbar, window management

---

## 8.21 Further Reading

- **Mozilla Developer Network (MDN)**: https://developer.mozilla.org/
- **Firefox Source Code (Searchfox)**: https://searchfox.org/mozilla-central/
- **HTML Spec (WHATWG)**: https://html.spec.whatwg.org/
- **CSS Specs (W3C)**: https://www.w3.org/Style/CSS/
- **JavaScript Spec (ECMAScript)**: https://tc39.es/ecma262/
- **Goja Engine**: https://github.com/dop251/goja
- **gg Graphics**: https://github.com/fogleman/gg
- **kjk/flex (Yoga port)**: https://github.com/kjk/flex

---

*Key terms from this chapter: [Chrome](../glossary.md#chrome), [Focus](../glossary.md#focus), [Frame](../glossary.md#frame), [Hit Testing](../glossary.md#hit-testing), [Layout](../glossary.md#layout), [Tab](../glossary.md#tab), [Widget](../glossary.md#widget), [Window](../glossary.md#window)*