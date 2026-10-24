# Chapter 8: UI Toolkit & Application — mustard/app

> **In this chapter**: How thdwb builds the browser UI using mustard widgets, and how the application manages multiple windows/tabs. This is the "house frame and furniture" of the browser.

---

## 8.1 What Is the Browser UI?

The **browser UI** (also called **chrome** — not to be confused with Google Chrome) is everything *around* the web content: windows, frames, buttons, labels, inputs, canvases, and event handling.

**Analogy**: If the web page is a painting, the browser UI is the frame, the wall hooks, the gallery labels, and the lighting — all the stuff that lets you view and navigate between paintings.

```
┌─────────────────────────────────────────────────────────────────┐
│                    BROWSER WINDOW (mustard.Window)               │
├─────────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ ROOT FRAME (VerticalFrame or HorizontalFrame)             │  │
│  │  ┌─────────────────────────────────────────────────────┐  │  │
│  │  │ CHILD WIDGETS (Button, Label, Canvas, Input, etc.)  │  │  │
│  │  └─────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 8.2 Key Terms (Defined in [Glossary](../glossary.md))

- **Chrome (Browser UI)** — The browser's own interface (not the web page)
- **Widget** — Reusable UI component (button, input, label, canvas, etc.)
- **Frame** — Container widget that holds other widgets (vertical/horizontal layout)
- **Layout** — Algorithm for positioning widgets (vertical, horizontal)
- **Event Loop** — Main loop processing input, updating, rendering
- **Hit Testing** — Finding which widget is under the mouse
- **Focus** — Which widget receives keyboard input
- **Tab** — A web content container with its own WindowContext

---

## 8.3 The mustard Package: UI Toolkit

### File Overview

```
mustard/
├── app.go              # Application: CreateNewApp, AddWindow, ActivateWindow, DestroyWindow
├── window.go           # Window creation, OpenGL, event loop, widget management
├── frame.go            # Frame tree (widget hierarchy, CreateFrame)
├── widget.go           # Base Widget interface and baseWidget implementation
├── button.go           # Button widget (CreateButtonWidget)
├── label.go            # Label widget (CreateLabelWidget)
├── input.go            # Text input widget (CreateInputWidget)
├── canvas.go           # Canvas widget (CreateCanvasWidget)
├── scrollbar.go        # Scrollbar widget (CreateScrollBarWidget)
├── text.go             # Text widget (CreateTextWidget)
├── image.go            # Image widget (CreateImageWidget)
├── tree.go             # Tree widget (CreateTreeWidget)
├── events.go           # Event processing (ProcessKeyboardInput, ProcessPointerClick, etc.)
├── contextMenu.go      # Context menu (EnableContextMenus, AddContextMenuEntry, etc.)
├── types.go            # Shared types (Widget, Window, Frame, etc.)
├── gl.go               # OpenGL shader setup
├── shaders.go          # Shader sources
├── mui.go              # Material UI helpers
└── *_test.go
```

---

## 8.4 Application Entry Point: app.go

### Main Function

```go
// main is the entry point.
func main() {
    // 1. Initialize GLFW hints
    mustard.SetGLFWHints()

    // 2. Create application
    app := mustard.CreateNewApp("thdwb")

    // 3. Create a window
    window := mustard.CreateNewWindow("thdwb", 900, 600, true)

    // 4. Create root frame and set it on the window
    rootFrame := mustard.CreateFrame(mustard.VerticalFrame)
    window.SetRootFrame(rootFrame)

    // 5. Add window to app
    app.AddWindow(window)

    // 6. Activate window (enables tabbed mode)
    app.ActivateWindow(window)

    // 7. Show window
    window.Show()

    // 8. Run application (blocks)
    app.Run(func() {
        // Per-frame callback
    })
}
```

### App Struct

```go
type App struct {
    name         string
    windows      []*Window
    activeWindow *Window
    tabbedMode   bool
}
```

### App Methods

| Method | Description |
|--------|-------------|
| `CreateNewApp(name string) *App` | Creates a new application with the given name |
| `(app *App) Run(callback func())` | Runs the main event loop (blocks forever) |
| `(app *App) AddWindow(window *Window)` | Adds a window to the app |
| `(app *App) ActivateWindow(window *Window) error` | Activates a window (enables tabbed mode) |
| `(app *App) Windows() []*Window` | Returns all managed windows |
| `(app *App) DestroyWindow(window *Window)` | Destroys a window and removes it from the app |

### Example: Creating an App with a Window

```go
app := mustard.CreateNewApp("my-browser")
window := mustard.CreateNewWindow("My Browser", 900, 600, true)
rootFrame := mustard.CreateFrame(mustard.VerticalFrame)
window.SetRootFrame(rootFrame)
app.AddWindow(window)
app.ActivateWindow(window)
window.Show()
app.Run(func() {
    // Per-frame callback
})
```

---

## 8.5 Window

### Window Struct

```go
type Window struct {
    title  string
    width  int
    height int
    hiDPI  bool
    active bool
    // ... internal fields for GLFW, OpenGL, widgets, events
}
```

### Window Methods

| Method | Description |
|--------|-------------|
| `CreateNewWindow(title string, width, height int, hiDPI bool) *Window` | Creates a new GLFW/OpenGL window |
| `(window *Window) Show()` | Shows the window and marks it for reflow |
| `(window *Window) AttachActiveChangeListener(callback func(bool))` | Registers a callback for active state changes |
| `(window *Window) IsActive() bool` | Returns whether the window is active |
| `(window *Window) SetRootFrame(frame *Frame)` | Sets the root frame for the window |
| `(window *Window) GetSize() (int, int)` | Returns the window dimensions |
| `(window *Window) RequestReflow()` | Requests a layout reflow |
| `(window *Window) SetTitle(title string)` | Sets the window title |
| `(window *Window) RecreateContext()` | Recreates the gg.Context |
| `(window *Window) GetContext() *gg.Context` | Returns the gg.Context for drawing |
| `(window *Window) GetCursorPosition() (float64, float64)` | Returns the current cursor position |
| `(window *Window) RegisterTree(tree *TreeWidget)` | Registers a tree widget for event handling |
| `(window *Window) RegisterButton(button *ButtonWidget, callback func())` | Registers a button with its click callback |
| `(window *Window) UnregisterButton(button *ButtonWidget)` | Unregisters a button |
| `(window *Window) RegisterInput(input *InputWidget)` | Registers an input widget |
| `(window *Window) UnregisterInput(input *InputWidget)` | Unregisters an input widget |
| `(window *Window) AttachPointerPositionEventListener(callback func(float64, float64))` | Registers a pointer position listener |
| `(window *Window) AttachScrollEventListener(callback func(int))` | Registers a scroll listener |
| `(window *Window) AttachClickEventListener(callback func(MustardKey))` | Registers a click listener |
| `(window *Window) AttachPointerClickEventListener(callback func(float64, float64, MustardKey))` | Registers a pointer click listener |
| `(window *Window) AttachKeyboardInputEventListener(callback func(KeyboardInput))` | Registers a keyboard input listener |
| `(window *Window) AttachTextInputEventListener(callback func(rune))` | Registers a text input listener |
| `(window *Window) SetCursor(cursorType cursorType)` | Sets the cursor type |
| `(window *Window) AddOverlay(overlay *Overlay)` | Adds an active overlay |
| `(window *Window) RemoveOverlay(overlay *Overlay)` | Removes an active overlay |
| `(window *Window) AddStaticOverlay(overlay *Overlay)` | Adds a static overlay |
| `(window *Window) RemoveStaticOverlay(ref string)` | Removes a static overlay by ref |

### Example: Creating a Window with a Button

```go
window := mustard.CreateNewWindow("My Window", 800, 600, false)
rootFrame := mustard.CreateFrame(mustard.VerticalFrame)
window.SetRootFrame(rootFrame)

button := mustard.CreateButtonWidget("Click Me", nil)
button.SetWidth(100)
button.SetHeight(40)
button.SetContent("Click Me")

rootFrame.AttachWidget(button)
window.Show()
```

---

## 8.6 Widgets

### Widget Interface

```go
type Widget interface {
    Buffer() *image.RGBA
    SetBuffer(*image.RGBA)
    SetNeedsRepaint(bool)
    NeedsRepaint() bool
    Widgets() []Widget
    ComputedBox() *box
    SetWindow(*Window)
    BaseWidget() *baseWidget
    draw()
}
```

### baseWidget

The `baseWidget` struct provides common functionality for all widgets:

| Method | Description |
|--------|-------------|
| `RequestReflow()` | Requests a reflow on the associated window |
| `RequestRepaint()` | Marks the widget as needing repaint |
| `GetRect() (float64, float64, float64, float64)` | Returns (top, left, width, height) |
| `GetTop() float64` | Returns the top position |
| `GetLeft() float64` | Returns the left position |
| `GetWidth() float64` | Returns the width |
| `GetHeight() float64` | Returns the height |
| `ComputedBox() *box` | Returns the computed bounding box |
| `SetBuffer(buffer *image.RGBA)` | Sets the render buffer |
| `SetWindow(window *Window)` | Associates the widget with a window |
| `Buffer() *image.RGBA` | Returns the render buffer |
| `Widgets() []Widget` | Returns child widgets |
| `AttachWidget(wd Widget)` | Attaches a child widget |
| `DetachWidget(wd Widget) Widget` | Detaches a child widget |
| `BaseWidget() *baseWidget` | Returns the base widget |
| `GetWindow() *Window` | Returns the associated window |
| `NeedsRepaint() bool` | Returns whether repaint is needed |
| `SetNeedsRepaint(value bool)` | Sets the repaint flag |
| `IsPointInside(x, y float64) bool` | Checks if a point is inside the widget |
| `SetVisible(visible bool)` | Sets visibility |
| `GetVisible() bool` | Returns visibility |

### Button Widget

```go
type ButtonWidget struct {
    baseWidget
    content   string
    icon      image.Image
    fontSize  float64
    fontColor string
    selected  bool
    padding   float64
    onClick   func()
}
```

| Method | Description |
|--------|-------------|
| `CreateButtonWidget(label string, asset []byte) *ButtonWidget` | Creates a new button widget |
| `(button *ButtonWidget) SetWidth(width float64)` | Sets width (marks as fixed) |
| `(button *ButtonWidget) SetHeight(height float64)` | Sets height (marks as fixed) |
| `(button *ButtonWidget) SetFontSize(fontSize float64)` | Sets font size |
| `(button *ButtonWidget) SetPadding(padding float64)` | Sets padding |
| `(button *ButtonWidget) SetContent(content string)` | Sets button text |
| `(button *ButtonWidget) Click()` | Calls the onClick callback |
| `(button *ButtonWidget) GetContent() string` | Returns button text |
| `(button *ButtonWidget) SetFontColor(fontColor string)` | Sets font color (must start with #) |
| `(button *ButtonWidget) SetBackgroundColor(backgroundColor string)` | Sets background color (must start with #) |

### Label Widget

```go
type LabelWidget struct {
    baseWidget
    content   string
    fontSize  float64
    fontColor string
}
```

| Method | Description |
|--------|-------------|
| `CreateLabelWidget(content string) *LabelWidget` | Creates a new label widget |
| `(label *LabelWidget) SetWidth(width float64)` | Sets width (marks as fixed) |
| `(label *LabelWidget) SetHeight(height float64)` | Sets height (marks as fixed) |
| `(label *LabelWidget) SetFontSize(fontSize float64)` | Sets font size |
| `(label *LabelWidget) SetContent(content string)` | Sets label text |
| `(label *LabelWidget) GetContent() string` | Returns label text |
| `(label *LabelWidget) SetFontColor(fontColor string)` | Sets font color (must start with #) |
| `(label *LabelWidget) SetBackgroundColor(backgroundColor string)` | Sets background color (must start with #) |

### Input Widget

```go
type InputWidget struct {
    baseWidget
    value           string
    selected        bool
    active          bool
    padding         float64
    fontSize        float64
    fontColor       string
    cursorPosition  int
    returnCallback  func()
}
```

| Method | Description |
|--------|-------------|
| `CreateInputWidget() *InputWidget` | Creates a new input widget |
| `(input *InputWidget) SetWidth(width float64)` | Sets width (marks as fixed) |
| `(input *InputWidget) SetHeight(height float64)` | Sets height (marks as fixed) |
| `(input *InputWidget) SetFontSize(fontSize float64)` | Sets font size |
| `(input *InputWidget) SetReturnCallback(returnCallback func())` | Sets the Enter key callback |
| `(input *InputWidget) SetFontColor(fontColor string)` | Sets font color (must start with #) |
| `(input *InputWidget) SetValue(value string)` | Sets input value |
| `(input *InputWidget) GetValue() string` | Returns input value |
| `(input *InputWidget) GetCursorPos() int` | Returns cursor position |
| `(input *InputWidget) SetBackgroundColor(backgroundColor string)` | Sets background color (must start with #) |

### Canvas Widget

```go
type CanvasWidget struct {
    baseWidget
    context        *gg.Context
    drawingContext *gg.Context
    renderer       func(*CanvasWidget)
    scrollable     bool
    offset         int
    drawingRepaint bool
}
```

| Method | Description |
|--------|-------------|
| `CreateCanvasWidget(renderer func(*CanvasWidget)) *CanvasWidget` | Creates a new canvas widget |
| `(canvas *CanvasWidget) SetWidth(width float64)` | Sets width (marks as fixed) |
| `(canvas *CanvasWidget) SetHeight(height float64)` | Sets height (marks as fixed) |
| `(canvas *CanvasWidget) EnableScrolling()` | Enables scrolling |
| `(canvas *CanvasWidget) DisableScrolling()` | Disables scrolling and resets offset |
| `(canvas *CanvasWidget) SetOffset(offset int)` | Sets scroll offset |
| `(canvas *CanvasWidget) GetOffset() int` | Returns scroll offset |
| `(canvas *CanvasWidget) GetContext() *gg.Context` | Returns the drawing context |
| `(canvas *CanvasWidget) SetContext(ctx *gg.Context)` | Sets the drawing context |
| `(canvas *CanvasWidget) SetDrawingRepaint(repaint bool)` | Sets the drawing repaint flag |

### Text Widget

```go
type TextWidget struct {
    baseWidget
    content   string
    fontSize  float64
    fontColor string
}
```

| Method | Description |
|--------|-------------|
| `CreateTextWidget(content string) *TextWidget` | Creates a new text widget |
| `(text *TextWidget) SetWidth(width float64)` | Sets width (marks as fixed) |
| `(text *TextWidget) SetHeight(height float64)` | Sets height (marks as fixed) |
| `(text *TextWidget) SetFontSize(fontSize float64)` | Sets font size |
| `(text *TextWidget) SetContent(content string)` | Sets text content |
| `(text *TextWidget) GetContent() string` | Returns text content |
| `(text *TextWidget) SetFontColor(fontColor string)` | Sets font color (must start with #) |
| `(text *TextWidget) SetBackgroundColor(backgroundColor string)` | Sets background color (must start with #) |

### Image Widget

```go
type ImageWidget struct {
    baseWidget
    path string
    img  image.Image
}
```

| Method | Description |
|--------|-------------|
| `CreateImageWidget(path []byte) *ImageWidget` | Creates a new image widget |
| `(imageWidget *ImageWidget) SetWidth(width float64)` | Sets width (marks as fixed) |
| `(imageWidget *ImageWidget) SetHeight(height float64)` | Sets height (marks as fixed) |

### Tree Widget

```go
type TreeWidget struct {
    baseWidget
    fontSize   float64
    fontColor  string
    nodes      []*TreeWidgetNode
    openIcon   image.Image
    closeIcon  image.Image
    selectCallback func(*TreeWidgetNode)
}
```

| Method | Description |
|--------|-------------|
| `CreateTreeWidget() *TreeWidget` | Creates a new tree widget |
| `(tree *TreeWidget) Click()` | Handles tree click at cursor position |
| `(tree *TreeWidget) SetSelectCallback(selectCallback func(*TreeWidgetNode))` | Sets node selection callback |
| `(tree *TreeWidget) SelectNode(node *TreeWidgetNode)` | Selects a specific node |
| `(tree *TreeWidget) SelectNodeByValue(value string)` | Selects node by value |
| `(tree *TreeWidget) SetWidth(width float64)` | Sets width (marks as fixed) |
| `(tree *TreeWidget) SetHeight(height float64)` | Sets height (marks as fixed) |
| `(tree *TreeWidget) SetFontSize(fontSize float64)` | Sets font size |
| `(tree *TreeWidget) SetFontColor(fontColor string)` | Sets font color (must start with #) |
| `(tree *TreeWidget) SetBackgroundColor(backgroundColor string)` | Sets background color (must start with #) |

### Tree Widget Node

```go
type TreeWidgetNode struct {
    Key      string
    Value    string
    Parent   *TreeWidgetNode
    Children []*TreeWidgetNode
    isOpen     bool
    isSelected bool
    index      int
    box        box
}
```

| Method | Description |
|--------|-------------|
| `CreateTreeWidgetNode(key, value string) *TreeWidgetNode` | Creates a new tree node |
| `(node *TreeWidgetNode) Toggle()` | Toggles open/closed state |
| `(node *TreeWidgetNode) Close()` | Closes the node |
| `(node *TreeWidgetNode) Open()` | Opens the node |
| `(node *TreeWidgetNode) AddNode(childNode *TreeWidgetNode)` | Adds a child node |
| `(node *TreeWidgetNode) NextSibling() *TreeWidgetNode` | Returns next sibling |
| `(node *TreeWidgetNode) PreviousSibling() *TreeWidgetNode` | Returns previous sibling |

### ScrollBar Widget

```go
type ScrollBarWidget struct {
    baseWidget
    orientation ScrollBarOrientation
    selected    bool
    thumbSize   float64
    thumbColor  string
    scrollerSize   float64
    scrollerOffset float64
}
```

| Method | Description |
|--------|-------------|
| `CreateScrollBarWidget(orientation ScrollBarOrientation) *ScrollBarWidget` | Creates a new scrollbar widget |
| `(scrollBar *ScrollBarWidget) SetWidth(width float64)` | Sets width (marks as fixed) |
| `(scrollBar *ScrollBarWidget) SetHeight(height float64)` | Sets height (marks as fixed) |
| `(scrollBar *ScrollBarWidget) SetTrackColor(backgroundColor string)` | Sets track color (must start with #) |
| `(scrollBar *ScrollBarWidget) SetScrollerSize(scrollerSize float64)` | Sets scroller size |
| `(scrollBar *ScrollBarWidget) SetThumbSize(thumbSize float64)` | Sets thumb size |
| `(scrollBar *ScrollBarWidget) SetThumbColor(thumbColor string)` | Sets thumb color |
| `(scrollBar *ScrollBarWidget) SetScrollerOffset(scrollerOffset float64)` | Sets scroller offset |

### Frame Widget

```go
type Frame struct {
    baseWidget
    orientation FrameOrientation
}
```

| Method | Description |
|--------|-------------|
| `CreateFrame(orientation FrameOrientation) *Frame` | Creates a new frame widget |
| `(frame *Frame) SetBackgroundColor(backgroundColor string)` | Sets background color (must start with #) |
| `(frame *Frame) SetWidth(width float64)` | Sets width (marks as fixed) |
| `(frame *Frame) SetHeight(height float64)` | Sets height (marks as fixed) |
| `(frame *Frame) GetHeight() float64` | Returns frame height |
| `(frame *Frame) SetVisible(visible bool)` | Sets visibility |
| `(frame *Frame) RemoveWidget(index int)` | Removes child widget at index |

---

## 8.7 Overlays

### Overlay Struct

```go
type Overlay struct {
    ref    string
    active bool
    top    float64
    left   float64
    width  float64
    height float64
    position image.Point
    backgroundBuffer *image.RGBA
    buffer           *image.RGBA
}
```

| Function | Description |
|----------|-------------|
| `CreateStaticOverlay(ref string, ctx *gg.Context, position image.Point) *Overlay` | Creates a static overlay from a gg.Context |

### Window Overlay Methods

| Method | Description |
|--------|-------------|
| `(window *Window) AddOverlay(overlay *Overlay)` | Adds an active overlay |
| `(window *Window) RemoveOverlay(overlay *Overlay)` | Removes an active overlay |
| `(window *Window) AddStaticOverlay(overlay *Overlay)` | Adds a static overlay |
| `(window *Window) RemoveStaticOverlay(ref string)` | Removes a static overlay by ref |

---

## 8.8 Context Menu

### Window Context Menu Methods

| Method | Description |
|--------|-------------|
| `(window *Window) EnableContextMenus()` | Initializes the context menu system |
| `(window *Window) AddContextMenuEntry(entryText string, action func())` | Adds a menu entry |
| `(window *Window) DestroyContextMenu()` | Destroys the context menu |
| `(window *Window) DrawContextMenu()` | Draws the context menu at cursor position |
| `(window *Window) SetContextMenuOverlay(overlay *Overlay)` | Sets the context menu overlay |
| `(window *Window) SelectEntry(entry *menuEntry)` | Selects a menu entry |
| `(window *Window) DeselectEntries()` | Deselects all entries |

### menuEntry Struct

```go
type menuEntry struct {
    entryText string
    action    func()
    top  float64
    left float64
    width  float64
    height float64
}
```

| Method | Description |
|--------|-------------|
| `(entry *menuEntry) PointIntersects(x, y float64) bool` | Checks if a point intersects the entry |
| `(entry *menuEntry) getCoords() (float64, float64, float64, float64)` | Returns entry coordinates |
| `(entry *menuEntry) setCoords(top, left, width, height float64)` | Sets entry coordinates |

---

## 8.9 Event Processing

### Window Event Methods

| Method | Description |
|--------|-------------|
| `(window *Window) ProcessKeyboardInput(input KeyboardInput)` | Dispatches keyboard input to listeners |
| `(window *Window) ProcessTextInput(char rune)` | Dispatches text input to listeners |
| `(window *Window) ProcessPointerPosition()` | Processes pointer position events |
| `(window *Window) ProcessPointerClick(button glfw.MouseButton)` | Processes pointer click events |
| `(window *Window) ProcessScroll(x, y float64)` | Processes scroll events |
| `(window *Window) ProcessReturnKey()` | Processes Return key for active input |
| `(window *Window) ProcessArrowKeys(arrowKey string)` | Processes arrow key input |
| `(window *Window) ProcessTrees()` | Processes tree widget hover/selection |
| `(window *Window) ProcessButtons()` | Processes button hover/selection |
| `(window *Window) ProcessInputs()` | Processes input widget hover/selection |
| `(window *Window) ProcessInputActivation()` | Activates selected input widget |
| `(window *Window) ProcessButtonClick()` | Clicks selected button |
| `(window *Window) ProcessTreeClick()` | Clicks selected tree node |
| `(window *Window) ProcessContextMenu()` | Processes context menu hover |
| `(window *Window) ProcessContextMenuClick()` | Processes context menu click |

### Event Listener Methods

| Method | Description |
|--------|-------------|
| `(window *Window) AttachPointerPositionEventListener(callback func(float64, float64))` | Registers pointer position listener |
| `(window *Window) AttachScrollEventListener(callback func(int))` | Registers scroll listener |
| `(window *Window) AttachClickEventListener(callback func(MustardKey))` | Registers click listener |
| `(window *Window) AttachPointerClickEventListener(callback func(float64, float64, MustardKey))` | Registers pointer click listener |
| `(window *Window) AttachKeyboardInputEventListener(callback func(KeyboardInput))` | Registers keyboard input listener |
| `(window *Window) AttachTextInputEventListener(callback func(rune))` | Registers text input listener |
| `(window *Window) AttachActiveChangeListener(callback func(bool))` | Registers active state change listener |

---

## 8.10 Types & Constants

### MustardKey

```go
type MustardKey int

const (
    MouseLeft  MustardKey = iota  // 0
    MouseRight                    // 1
)
```

### KeyboardAction

```go
type KeyboardAction int

const (
    KeyboardPress   KeyboardAction = iota  // 0
    KeyboardRepeat                         // 1
    KeyboardRelease                        // 2
)
```

### KeyboardInput

```go
type KeyboardInput struct {
    Key     string
    Action  KeyboardAction
    Control bool
    Shift   bool
    Alt     bool
    Super   bool
}
```

### FrameOrientation

```go
type FrameOrientation int

const (
    VerticalFrame   FrameOrientation = iota  // 0
    HorizontalFrame                          // 1
)
```

### ScrollBarOrientation

```go
type ScrollBarOrientation int

const (
    VerticalScrollBar   ScrollBarOrientation = iota  // 0
    HorizontalScrollBar                              // 1
)
```

### cursorType

```go
type cursorType int

const (
    DefaultCursor cursorType = iota  // 0
    PointerCursor                    // 1
)
```

### widgetType

```go
type widgetType int

const (
    buttonWidget   widgetType = iota  // 0
    canvasWidget                      // 1
    frameWidget                       // 2
    imageWidget                       // 3
    inputWidget                       // 4
    labelWidget                       // 5
    treeWidget                        // 6
    scrollbarWidget                   // 7
    textWidget                        // 8
)
```

### widgetPosition

```go
type widgetPosition int

const (
    PositionRelative widgetPosition = iota  // 0
    PositionAbsolute                        // 1
)
```

---

## 8.11 Layout Calculation Helpers

| Function | Description |
|----------|-------------|
| `calculateChildrenWidgetsLayout(children []*baseWidget, top, left, width, height float64, orientation FrameOrientation) []*baseWidget` | Calculates layout for child widgets |
| `getFixedWidthElements(elements []*baseWidget) ([]*baseWidget, []*baseWidget)` | Separates fixed vs volatile width elements |
| `getFixedHeightElements(elements []*baseWidget) ([]*baseWidget, []*baseWidget)` | Separates fixed vs volatile height elements |
| `calculateFlexibleWidth(avaiableWidth float64, elements []*baseWidget) float64` | Calculates remaining width for flexible elements |
| `calculateFlexibleHeight(avaiableHeight float64, elements []*baseWidget) float64` | Calculates remaining height for flexible elements |
| `getCoreWidgets(widgets []Widget) []*baseWidget` | Returns only visible widgets |
| `copyWidgetToBuffer(widget Widget, src image.Image)` | Copies widget content to buffer |
| `compositeWidget(buffer *image.RGBA, widget Widget)` | Composites a widget to buffer |
| `compositeAll(buffer *image.RGBA, widget Widget)` | Recursively composites all widgets |

---

## 8.12 Example: Complete UI Setup

```go
package main

import (
    "github.com/danfragoso/thdwb/mustard"
)

func main() {
    mustard.SetGLFWHints()

    app := mustard.CreateNewApp("thdwb")

    window := mustard.CreateNewWindow("thdwb", 900, 600, true)
    rootFrame := mustard.CreateFrame(mustard.VerticalFrame)
    window.SetRootFrame(rootFrame)

    // Add a label
    label := mustard.CreateLabelWidget("Hello, thdwb!")
    label.SetWidth(200)
    label.SetHeight(40)
    label.SetFontColor("#000000")
    rootFrame.AttachWidget(label)

    // Add a button
    button := mustard.CreateButtonWidget("Click Me", nil)
    button.SetWidth(100)
    button.SetHeight(40)
    button.SetContent("Click Me")
    window.RegisterButton(button, func() {
        println("Button clicked!")
    })
    rootFrame.AttachWidget(button)

    app.AddWindow(window)
    app.ActivateWindow(window)
    window.Show()

    app.Run(func() {
        // Per-frame callback
    })
}
```

---

## 8.13 Cross-References

| Concept | Chapter |
|---------|---------|
| How widgets draw with gg | [Chapter 6: Rendering](../06-rendering-graphics.md) |
| How tabs isolate JS/Network | [Chapter 7: JavaScript Runtime](../07-javascript-runtime.md) |
| How network loads pages | [Chapter 2: Network Layer](../02-network-layer.md) |
| How layout works | [Chapter 5: Layout Engine](../05-layout-engine.md) |
| Event handling basics | [Glossary: Event](../glossary.md#event) |

---

## 8.14 Summary

- **mustard** = UI toolkit: windows, widgets, event loop, OpenGL
- **App** = Manages multiple windows/tabs (`CreateNewApp`, `AddWindow`, `ActivateWindow`, `DestroyWindow`)
- **Window** = GLFW window + OpenGL + widget tree + event loop (`CreateNewWindow`, `SetRootFrame`, `Show`)
- **Widget** = UI component interface with `Buffer`, `SetBuffer`, `NeedsRepaint`, `Widgets`, `ComputedBox`, `SetWindow`, `BaseWidget`, `draw`
- **baseWidget** = Common widget implementation with visibility, rect, hit testing, attach/detach
- **Button/Label/Input/Canvas/Text/Image/Tree/ScrollBar** = Concrete widget types with type-specific setters
- **Frame** = Container widget with vertical/horizontal orientation
- **Overlay** = Floating UI layer (active or static)
- **Context Menu** = Right-click menu with entries and actions
- **Event Processing** = Keyboard, text, pointer, scroll, click, tree, button, input handling
- **Constants** = MustardKey, KeyboardAction, FrameOrientation, ScrollBarOrientation, cursorType, widgetType, widgetPosition

---

## 8.15 Next Steps

You've now completed the textbook! The chapters cover:

1. **Introduction** — Browser architecture overview
2. **Network Layer** — HTTP, cookies, cache, certificates
3. **HTML Parsing & DOM** — Tokenization, tree construction
4. **CSS Styling** — Parsing, cascade, inheritance, computed styles
5. **Layout Engine** — Block, inline, flexbox layout
6. **Rendering & Graphics** — gg 2D graphics, mustard UI toolkit
7. **JavaScript Runtime** — Goja VM, DOM bindings, Fetch, events
8. **UI Toolkit & Application** — Windows, widgets, event handling, overlays, context menus

---

## 8.16 Further Reading

- **Mozilla Developer Network (MDN)**: https://developer.mozilla.org/
- **Firefox Source Code (Searchfox)**: https://searchfox.org/mozilla-central/
- **HTML Spec (WHATWG)**: https://html.spec.whatwg.org/
- **CSS Specs (W3C)**: https://www.w3.org/Style/CSS/
- **JavaScript Spec (ECMAScript)**: https://tc39.es/ecma262/
- **Goja (JavaScript engine)**: https://github.com/dop251/goja
- **gg (2D graphics)**: https://github.com/fogleman/gg
