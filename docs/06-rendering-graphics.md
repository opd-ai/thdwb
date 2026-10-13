# Chapter 6: Rendering & Graphics — gg + mustard

> **In this chapter**: How thdwb draws pixels on screen — the 2D graphics library (gg), text shaping, and the UI toolkit/window management (mustard). This is the "painter" and "house frame" of the browser.

---

## 6.1 What Is Rendering?

After layout calculates **where** everything goes (x, y, width, height), **rendering** (painting) draws the actual pixels: backgrounds, borders, text, images, shadows.

**Analogy**: Layout is the architect's blueprint with measurements. Rendering is the construction crew actually building and painting the house.

```
LAYOUT OUTPUT                    RENDERING (gg)                    SCREEN PIXELS
──────────────                   ─────────────                    ─────────────
div: x=10, y=10,           1. Draw background              ┌─────────────────┐
  w=200, h=100                 2. Draw border                   │ ████████████  │
  bg=blue                      3. Draw text                     │ █ Hello ████  │
  border=2px black             4. Draw children                 │ █ World ████  │
  "Hello World"                5. Composite layers              │ ████████████  │
                                  │                              └─────────────────┘
```

---

## 6.2 Key Terms (Defined in [Glossary](../glossary.md))

- **Graphics Context** — Drawing surface + state (color, font, transform)
- **Rasterization** — Converting vector shapes to pixels
- **Text Shaping** — Converting characters to positioned glyphs
- **Glyph** — Visual representation of a character
- **Font Face** — Specific font file + size + style
- **Compositing** — Combining multiple layers into final image
- **Double Buffering** — Draw to offscreen buffer, then swap to screen
- **HiDPI** — High pixel density displays (Retina, etc.)

---

## 6.3 The gg Package: 2D Graphics Library

thdwb uses **gg** (a pure Go 2D graphics library, forked from fogleman/gg) for all drawing.

### File Overview

```
gg/
├── context.go        # Main Context: drawing API, state management
├── path.go           # Path construction (lines, curves, shapes)
├── matrix.go         # Transformation matrix (translate, rotate, scale)
├── gradient.go       # Linear/radial gradients
├── pattern.go        # Patterns (solid, gradient, image)
├── bezier.go         # Bézier curve math
├── point.go          # Point math
├── wrap.go           # Text wrapping
├── util.go           # Utilities
├── fonts.go          # Font loading (in assets/)
├── context_test.go   # Tests
└── *.go              # Asset files (fonts, icons)
```

---

## 6.4 Context: The Drawing Surface

```go
// Context holds all state for drawing operations.
// Like HTML5 CanvasRenderingContext2D but in Go.
type Context struct {
    width         int           // Canvas width in pixels
    height        int           // Canvas height in pixels
    rasterizer    *raster.Rasterizer  // Converts paths to pixels
    im            *image.RGBA   // Pixel buffer (destination)
    mask          *image.Alpha  // Alpha mask for clipping
    color         color.Color   // Current fill/stroke color
    fillPattern   Pattern       // Fill style (solid, gradient, image)
    strokePattern Pattern       // Stroke style
    lineWidth     float64       // Stroke width
    lineCap       LineCap       // End cap style (round, butt, square)
    lineJoin      LineJoin      // Corner join style (round, bevel)
    fillRule      FillRule      // Winding vs even-odd
    fontFace      font.Face     // Current font
    fontHeight    float64       // Font height in pixels
    matrix        Matrix        // Current transform
    stack         []*Context    // Saved states (Push/Pop)
}
```

### Creating a Context

```go
// NewContext creates a new RGBA image and drawing context.
func NewContext(width, height int) *Context {
    return NewContextForRGBA(image.NewRGBA(image.Rect(0, 0, width, height)))
}

// NewContextForImage copies an image and creates a context for drawing on it.
func NewContextForImage(im image.Image) *Context

// NewContextForRGBA uses an existing RGBA buffer (no copy).
func NewContextForRGBA(im *image.RGBA) *Context
```

---

## 6.5 Drawing Primitives

### Rectangles

```go
// DrawRectangle adds a rectangle to the current path.
func (c *Context) DrawRectangle(x, y, w, h float64)

// Fill fills the current path with fill pattern.
func (c *Context) Fill()

// Stroke strokes the current path with stroke pattern.
func (c *Context) Stroke()

// FillPreserve / StrokePreserve keep the path after fill/stroke.
func (c *Context) FillPreserve()
func (c *Context) StrokePreserve()
```

### Paths (Lines, Curves)

```go
// MoveTo starts a new subpath at (x, y).
func (c *Context) MoveTo(x, y float64)

// LineTo adds a straight line to (x, y).
func (c *Context) LineTo(x, y float64)

// CurveTo adds a cubic Bézier curve.
func (c *Context) CurveTo(x1, y1, x2, y2, x3, y3 float64)

// QuadraticTo adds a quadratic Bézier curve.
func (c *Context) QuadraticTo(x1, y1, x2, y2 float64)

// ClosePath closes the current subpath (line to start).
func (c *Context) ClosePath()
```

### Circles & Arcs

```go
// DrawCircle adds a circle to the path.
func (c *Context) DrawCircle(x, y, r float64)

// DrawArc adds an arc (angle in radians).
func (c *Context) DrawArc(x, y, r, a1, a2 float64)
```

### Example: Drawing a Rounded Rectangle

```go
func drawRoundedRect(c *gg.Context, x, y, w, h, r float64) {
    c.MoveTo(x+r, y)
    c.LineTo(x+w-r, y)
    c.QuadraticTo(x+w, y, x+w, y+r)
    c.LineTo(x+w, y+h-r)
    c.QuadraticTo(x+w, y+h, x+w-r, y+h)
    c.LineTo(x+r, y+h)
    c.QuadraticTo(x, y+h, x, y+h-r)
    c.LineTo(x, y+r)
    c.QuadraticTo(x, y, x+r, y)
    c.ClosePath()
}
```

---

## 6.6 Colors & Patterns

### Solid Colors

```go
// SetRGB sets fill color (0-1 range).
func (c *Context) SetRGB(r, g, b float64)

// SetRGBA sets fill color with alpha.
func (c *Context) SetRGBA(r, g, b, a float64)

// SetHexColor sets fill color from hex string "#RRGGBB" or "#RRGGBBAA".
func (c *Context) SetHexColor(hex string)

// SetColor sets fill color from any color.Color.
func (c *Context) SetColor(c color.Color)
```

### Gradients

```go
// NewLinearGradient creates a linear gradient pattern.
func NewLinearGradient(x0, y0, x1, y1 float64) *Gradient

// NewRadialGradient creates a radial gradient pattern.
func NewRadialGradient(x0, y0, r0, x1, y1, r1 float64) *Gradient

// Gradient.AddColorStop adds a color at offset (0-1).
func (g *Gradient) AddColorStop(offset float64, c color.Color)

// SetFillPattern sets the fill to a pattern.
func (c *Context) SetFillPattern(p Pattern)
```

### Image Patterns

```go
// NewImagePattern creates a pattern from an image.
func NewImagePattern(im image.Image, repeat bool) *ImagePattern
```

---

## 6.7 Text Rendering: The Hard Part

Text rendering is **surprisingly complex**:
1. **Font loading** — Parse TTF/OTF files
2. **Text shaping** — Characters → glyphs with positions (ligatures, kerning, RTL)
3. **Glyph rasterization** — Glyph outlines → pixels
4. **Layout** — Line breaking, alignment, wrapping

### Font Loading

```go
// LoadFontFace loads a TTF font from bytes at given size (points).
func LoadFontFace(fontBytes []byte, points float64) (font.Face, error)

// LoadFontFaceFromFile loads a TTF font from file path.
func LoadFontFaceFromFile(path string, points float64) (font.Face, error)
```

thdwb bundles **Open Sans** font (in `assets/fonts.go` and `assets/open_sans.go`).

### Setting the Font

```go
// SetFontFace sets the current font for text operations.
func (c *Context) SetFontFace(face font.Face)

// FontHeight returns the current font's height in pixels.
func (c *Context) FontHeight() float64
```

### Drawing Text

```go
// DrawString draws text at (x, y) with alignment.
// Anchor: 0=left, 0.5=center, 1=right (horizontal)
//         0=top, 0.5=middle, 1=bottom (vertical)
func (c *Context) DrawString(s string, x, y float64)

// DrawStringAnchored draws with explicit anchor points.
func (c *Context) DrawStringAnchored(s string, x, y, ax, ay float64)

// MeasureString returns width and height of text.
func (c *Context) MeasureString(s string) (w, h float64)
```

### Text Wrapping

```go
// DrawStringWrapped draws text wrapped to max width.
// Returns number of lines drawn.
func (c *Context) DrawStringWrapped(
    s string, x, y, ax, ay, width float64,
    lineSpacing float64, align Align,
) int
```

**Alignment constants:**
```go
const (
    AlignLeft   Align = iota
    AlignCenter
    AlignRight
)
```

---

## 6.8 Transformations: Translate, Rotate, Scale

```go
// Translate moves the origin.
func (c *Context) Translate(x, y float64)

// Rotate rotates around origin (radians).
func (c *Context) Rotate(angle float64)

// Scale scales the coordinate system.
func (c *Context) Scale(x, y float64)

// Shear shears the coordinate system.
func (c *Context) Shear(x, y float64)

// Identity resets transform to identity.
func (c *Context) Identity()

// Push saves current state (transform, color, font, etc.).
func (c *Context) Push()

// Pop restores saved state.
func (c *Context) Pop()
```

**Transform Order Matters:**
```go
// Translate then Rotate: rotates around new origin
c.Translate(100, 100)
c.Rotate(45 * math.Pi / 180)

// Rotate then Translate: rotates around (0,0), then moves
c.Rotate(45 * math.Pi / 180)
c.Translate(100, 100)
```

---

## 6.9 Clipping: Restricting Draw Area

```go
// Clip sets the clipping region to the current path.
// Subsequent drawing only affects inside the clip.
func (c *Context) Clip()

// ClipPreserve clips but keeps the path.
func (c *Context) ClipPreserve()

// ResetClip clears the clipping region.
func (c *Context) ResetClip()
```

**Use case**: Scrollable areas — clip content to viewport.

---

## 6.10 Image Operations

```go
// DrawImage draws an image at (x, y).
func (c *Context) DrawImage(im image.Image, x, y int)

// DrawImageAnchored draws with anchor.
func (c *Context) DrawImageAnchored(im image.Image, x, y int, ax, ay float64)

// DrawImageRect draws a sub-rectangle of an image.
func (c *Context) DrawImageRect(im image.Image, sx, sy, sw, sh, dx, dy, dw, dh float64)
```

---

## 6.11 Saving Output

```go
// SavePNG saves the context as PNG.
func (c *Context) SavePNG(filename string) error

// SaveJPG saves as JPEG with quality (1-100).
func (c *Context) SaveJPG(filename string, quality int) error

// Image returns the underlying image.RGBA.
func (c *Context) Image() *image.RGBA
```

---

## 6.12 The mustard Package: UI Toolkit & Windows

While gg draws pixels, **mustard** manages:
- **Windows** (GLFW + OpenGL)
- **Widgets** (buttons, inputs, labels, scrollbars)
- **Event loop** (mouse, keyboard, resize)
- **Frame tree** (widget hierarchy)

### File Overview

```
mustard/
├── window.go           # Window creation, OpenGL context, event loop
├── app.go              # Application entry point, tab management
├── frame.go            # Frame tree (widget hierarchy)
├── widget.go           # Base Widget interface
├── button.go           # Button widget
├── label.go            # Label widget
├── input.go            # Text input widget
├── canvas.go           # Canvas widget (gg.Context wrapper)
├── scrollbar.go        # Scrollbar widget
├── text.go             # Text rendering helpers
├── image.go            # Image widget
├── events.go           # Event types and handling
├── contextMenu.go      # Context menu
├── tree.go             # Tree widget (for bookmarks, etc.)
├── types.go            # Shared types
├── gl.go               # OpenGL shader setup
├── shaders.go          # Shader sources
├── mui.go              # Material UI helpers
└── *_test.go           # Tests
```

---

## 6.13 Window: The Top-Level Container

### Creating a Window

```go
// CreateNewWindow creates a GLFW window with OpenGL context.
// hiDPI enables high-DPI support (Retina displays).
func CreateNewWindow(title string, width, height int, hiDPI bool) *Window
```

### Window Struct

```go
type Window struct {
    title       string
    width       int
    height      int
    hiDPI       bool
    glw         *glfw.Window      // GLFW window handle
    active      bool              // Has focus
    visible     bool              // Is shown

    // Rendering
    context     *gg.Context       // Drawing context
    backend     *GLBackend        // OpenGL backend
    frameBuffer *FrameBuffer      // Offscreen buffer

    // Input
    defaultCursor *glfw.Cursor
    pointerCursor *glfw.Cursor
    registeredButtons map[string]*Button
    registeredInputs  map[string]*Input
    activeInput       *Input

    // Widget tree
    rootFrame   *Frame

    // Events
    activeChangeListeners []func(bool)
    needsReflow           bool
    needsRepaint          bool
}
```

### Window Lifecycle

```go
// Show makes the window visible and starts event loop.
func (w *Window) Show() {
    w.needsReflow = true
    w.visible = true
    w.glw.Show()
}

// Run starts the main event loop (blocks until window closes).
func (w *Window) Run() {
    for !w.glfwWindowShouldClose() {
        w.processEvents()
        if w.needsReflow {
            w.reflow()
        }
        if w.needsRepaint {
            w.repaint()
        }
        w.swapBuffers()
        glfw.PollEvents()
    }
}
```

---

## 6.14 Frame Tree: Widget Hierarchy

Widgets are organized in a **tree** (like DOM but for UI chrome):

```go
// Frame is a container widget that holds children.
type Frame struct {
    Widget
    Children []Widget
    Layout   LayoutFunc  // Function to position children
}

// Widget is the base interface for all UI elements.
type Widget interface {
    Draw(ctx *gg.Context)
    Layout(x, y, w, h float64)
    HandleEvent(event Event) bool
    GetBounds() (x, y, w, h float64)
    SetBounds(x, y, w, h float64)
}
```

### Layout Functions

```go
// LayoutVertical stacks children top-to-bottom.
func LayoutVertical(children []Widget, x, y, w, h float64)

// LayoutHorizontal stacks children left-to-right.
func LayoutHorizontal(children []Widget, x, y, w, h float64)

// LayoutFlex arranges children with flexbox-like logic.
func LayoutFlex(children []Widget, x, y, w, h float64, direction string)
```

---

## 6.15 Widgets: Building Blocks

### Button

```go
type Button struct {
    WidgetBase
    Text     string
    OnClick  func()
    Style    ButtonStyle
}

func (b *Button) Draw(ctx *gg.Context) {
    // Draw background
    ctx.SetColor(b.Style.Background)
    ctx.DrawRectangle(b.X, b.Y, b.Width, b.Height)
    ctx.Fill()

    // Draw border
    ctx.SetColor(b.Style.Border)
    ctx.SetLineWidth(b.Style.BorderWidth)
    ctx.Stroke()

    // Draw text
    ctx.SetFontFace(b.Style.Font)
    ctx.SetColor(b.Style.TextColor)
    ctx.DrawStringAnchored(b.Text, b.X+b.Width/2, b.Y+b.Height/2, 0.5, 0.5)
}
```

### Label

```go
type Label struct {
    WidgetBase
    Text  string
    Style LabelStyle
}
```

### Input (Text Field)

```go
type Input struct {
    WidgetBase
    Text       string
    Placeholder string
    OnChange   func(string)
    CursorPos  int
    Selection  [2]int
}
```

### Canvas (Web Content Area)

```go
type Canvas struct {
    WidgetBase
    Context *gg.Context  // The gg context for web content
    Document *hotdog.Document  // Rendered document
}

func (c *Canvas) Draw(ctx *gg.Context) {
    // Draw the web content (already rendered to c.Context)
    ctx.DrawImage(c.Context.Image(), int(c.X), int(c.Y))
}
```

---

## 6.16 Event Handling

### Event Types

```go
type EventType int

const (
    EventMouseMove EventType = iota
    EventMouseDown
    EventMouseUp
    EventMouseScroll
    EventKeyDown
    EventKeyUp
    EventChar
    EventResize
    EventFocus
    EventBlur
)

type Event struct {
    Type      EventType
    X, Y      float64      // Mouse position
    Button    glfw.MouseButton
    Key       glfw.Key
    Char      rune
    Modifiers glfw.ModifierKey
    Width     int          // For resize
    Height    int
}
```

### Event Dispatch

```go
func (w *Window) processEvents() {
    for _, event := range w.pendingEvents {
        // 1. Find target widget (hit test)
        target := w.hitTest(event.X, event.Y)

        // 2. Dispatch to target
        if target != nil {
            target.HandleEvent(event)
        }

        // 3. Handle window-level events
        w.handleWindowEvent(event)
    }
}
```

---

## 6.17 OpenGL Backend: GPU Acceleration

thdwb uses **OpenGL** via `go-gl/gl` for fast rendering.

### GLBackend

```go
type GLBackend struct {
    program     uint32  // Shader program
    vao         uint32  // Vertex array object
    vbo         uint32  // Vertex buffer
    texture     uint32  // Texture for frame buffer
    width       int
    height      int
}

// createGLBackend compiles shaders, sets up buffers.
func createGLBackend() *GLBackend
```

### Shaders

**Vertex Shader** (passes position + UV):
```glsl
#version 150
in vec2 position;
in vec2 texCoord;
out vec2 vTexCoord;
void main() {
    vTexCoord = texCoord;
    gl_Position = vec4(position, 0.0, 1.0);
}
```

**Fragment Shader** (samples texture):
```glsl
#version 150
in vec2 vTexCoord;
out vec4 fragColor;
uniform sampler2D texture;
void main() {
    fragColor = texture(texture, vTexCoord);
}
```

### Rendering Pipeline

```
1. gg.Context draws to image.RGBA (CPU)
2. Upload image.RGBA to OpenGL texture (GPU)
3. Draw full-screen quad with texture
4. Swap buffers (double buffering)
```

---

## 6.18 HiDPI Support

```go
func CreateNewWindow(title string, width, height int, hiDPI bool) *Window {
    glw, _ := glfw.CreateWindow(width, height, title, nil, nil)

    xscale, yscale := float32(1), float32(1)
    if hiDPI {
        xscale, yscale = glw.GetContentScale()  // e.g., 2.0 on Retina
    }

    window := &Window{
        width:  int(float32(width) / xscale),
        height: int(float32(height) / yscale),
        hiDPI:  hiDPI,
        // ...
    }

    // Create gg.Context at LOGICAL size
    // OpenGL framebuffer at PHYSICAL size (logical * scale)
}
```

**Key insight**: Logical pixels (CSS pixels) vs Physical pixels (device pixels). gg works in logical; OpenGL framebuffer is physical.

---

## 6.19 Cross-References

| Concept | Chapter |
|---------|---------|
| How layout provides boxes to paint | [Chapter 5: Layout Engine](../05-layout-engine.md) |
| How JS Canvas API uses gg | [Chapter 7: JavaScript Runtime](../07-javascript-runtime.md) |
| How network loads images | [Chapter 2: Network Layer](../02-network-layer.md) |
| Text shaping details | [Glossary: Text Rendering](../glossary.md#text-rendering) |
| OpenGL basics | [MDN - WebGL](https://developer.mozilla.org/en-US/docs/Web/API/WebGL_API) |

---

## 6.20 Example Test: Drawing with gg

```go
// Example_ggDrawing demonstrates basic gg drawing operations.
func Example_ggDrawing() {
    // Create 400x300 context
    dc := gg.NewContext(400, 300)

    // White background
    dc.SetRGB(1, 1, 1)
    dc.Clear()

    // Draw a blue rectangle
    dc.SetHexColor("#1a73e8")
    dc.DrawRectangle(50, 50, 150, 100)
    dc.Fill()

    // Draw red border
    dc.SetHexColor("#ea4335")
    dc.SetLineWidth(3)
    dc.Stroke()

    // Draw text
    dc.SetFontFace(basicfont.Face7x13)
    dc.SetHexColor("#000000")
    dc.DrawStringAnchored("Hello, gg!", 200, 200, 0.5, 0.5)

    // Save
    dc.SavePNG("gg_output.png")
    fmt.Println("Saved gg_output.png")
}
```

Run with: `go test -run Example_ggDrawing ./gg`

---

## 6.21 Example Test: Creating a Window

```go
// Example_CreateWindow demonstrates creating a mustard window.
// Note: This requires a display (use xvfb-run on headless systems).
func Example_CreateWindow() {
    // Set GLFW hints (OpenGL 3.2 Core)
    mustard.SetGLFWHints()

    // Create window
    win := mustard.CreateNewWindow("thdwb Test", 800, 600, true)
    defer win.Destroy()

    // Add a button
    btn := mustard.NewButton("Click Me")
    btn.OnClick = func() {
        fmt.Println("Button clicked!")
    }
    win.rootFrame.AddChild(btn)

    // Layout
    win.rootFrame.Layout = mustard.LayoutVertical

    // Show and run (blocks)
    // win.Show()
    // win.Run()

    fmt.Println("Window created (not shown in test)")
    // Output: Window created (not shown in test)
}
```

Run with: `go test -run Example_CreateWindow ./mustard`

---

## 6.22 Summary

- **gg** = Pure Go 2D graphics (paths, text, images, transforms)
- **Context** = Drawing surface + state stack (Push/Pop)
- **Text rendering** = Font loading → shaping → rasterization
- **mustard** = UI toolkit: windows, widgets, event loop, OpenGL
- **Window** = GLFW window + OpenGL context + gg.Context
- **Frame tree** = Widget hierarchy with layout functions
- **HiDPI** = Logical vs physical pixels, content scale
- **OpenGL backend** = GPU texture upload + quad draw

---

## 6.23 Next Chapter

👉 **[Chapter 7: JavaScript Runtime — hotdog/js_bindings](../07-javascript-runtime.md)**

Learn how thdwb runs JavaScript with Goja, binds DOM APIs, handles events, and implements Fetch/XHR.

---

*Key terms from this chapter: [Compositing](../glossary.md#compositing), [Double Buffering](../glossary.md#double-buffering), [Font Face](../glossary.md#font-face), [Glyph](../glossary.md#glyph), [Graphics Context](../glossary.md#graphics-context), [HiDPI](../glossary.md#hidpi), [Rasterization](../glossary.md#rasterization), [Text Shaping](../glossary.md#text-shaping)*