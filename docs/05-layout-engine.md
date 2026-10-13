# Chapter 5: Layout Engine — The bun Package

> **In this chapter**: How thdwb calculates the exact position and size of every element. This is the "architect" of the browser — turning styled elements into positioned boxes.

---

## 5.1 What Is Layout?

After HTML parsing (DOM tree) and CSS styling (computed styles), the browser knows **what** elements exist and **how** they should look. But it doesn't know **where** they go on screen.

**Layout** (also called **reflow**) is the process of calculating:
- **Position**: x, y coordinates (top-left corner)
- **Size**: width, height
- For **every element** in the DOM tree

**Analogy**: The architect takes the blueprint (DOM) and decorating plan (CSS) and produces a detailed construction plan with exact measurements for every wall, window, and door.

```
COMPUTED STYLES                    LAYOUT ENGINE                    RENDER BOXES
─────────────────                  ──────────────                   ────────────
div {                              1. Start at root                div: x=0, y=0, w=800, h=600
  width: 800px;                       (viewport)                     │
  height: 600px;                    2. Calculate children           ├─ header: x=0, y=0, w=800, h=100
  display: block;                      recursively                    │
}                                     3. Handle display types         ├─ main: x=0, y=100, w=800, h=400
h1 {                                  (block, inline, flex)           │  ├─ article: x=20, y=120, w=760, h=360
  margin: 20px;                       4. Resolve percentages          │  │  └─ h1: x=40, y=140, w=720, h=40
  font-size: 32px;                    5. Handle flexbox               │  └─ aside: x=20, y=120, w=200, h=360
}                                        (kjk/flex)                   └─ footer: x=0, y=500, w=800, h=100
```

---

## 5.2 Key Terms (Defined in [Glossary](../glossary.md))

- **Layout/Reflow** — Calculating positions and sizes
- **Box Model** — Content, padding, border, margin layers
- **Render Box** — Computed rectangle: x, y, width, height
- **Block Layout** — Vertical stacking (div, p, h1)
- **Inline Layout** — Horizontal flow (span, a, text)
- **Flexbox** — Flexible row/column layout
- **Viewport** — Visible area of the page
- **Containing Block** — Ancestor that establishes coordinate system

---

## 5.3 The bun Package: File Overview

```
bun/
├── bun.go            # Main entry: RenderDocument, RenderDocumentToCanvas
├── layout.go         # Core layout algorithms (block, inline, flex)
├── block.go          # Block layout implementation
├── inline.go         # Inline layout implementation
├── listItem.go       # List item layout
├── fonts.go          # Font metrics for text measurement
├── bun_test.go       # Tests
└── bun.go            # (also has CanvasContext interface)
```

---

## 5.4 The Two Layout Modes

thdwb supports **two layout engines** (controlled by `experimentalLayout` flag):

| Mode | Description | Status |
|------|-------------|--------|
| **Legacy** (`experimentalLayout=false`) | Custom block/inline layout in `layout.go` | Stable, default |
| **Experimental** (`experimentalLayout=true`) | Flexbox-based using `kjk/flex` library | In development |

We'll cover both, starting with the legacy (simpler) mode.

---

## 5.5 Legacy Layout: Block & Inline

### Entry Point: `RenderDocument`

```go
// RenderDocument lays out and paints a document to a gg.Context.
// If experimentalLayout=false, uses legacy block/inline layout.
// If experimentalLayout=true, uses flexbox-based layout.
func RenderDocument(ctx *gg.Context, document *hotdog.Document, experimentalLayout bool) error
```

### Legacy Flow

```go
func RenderDocument(ctx *gg.Context, document *hotdog.Document, experimentalLayout bool) error {
    if !experimentalLayout {
        // 1. Find <body> (fallback to <html> if no <body>)
        body, err := document.DOM.FindChildByName("body")
        if err != nil || body == nil {
            body = document.DOM
        }

        // 2. Set root render box to viewport size
        document.DOM.RenderBox.Width = float64(ctx.Width())
        document.DOM.RenderBox.Height = float64(ctx.Height())

        // 3. Clear background (white)
        ctx.SetRGB(1, 1, 1)
        ctx.Clear()

        // 4. LAYOUT: Recursively calculate positions/sizes
        layoutDOM(ctx, body, 0)

        // 5. PAINT: Recursively draw
        paintTree(ctx, body)
    }
    // ... experimental path
}
```

### `layoutDOM`: Recursive Layout

```go
// layoutDOM recursively calculates layout for a node and its children.
// Parameters:
//   ctx - drawing context (for font measurement)
//   node - DOM node to layout
//   childIndex - index among siblings (for list-item counters)
func layoutDOM(ctx *gg.Context, node *hotdog.NodeDOM, childIndex int) {
    nodeChildren := getNodeChildren(node)

    // 1. Initialize render box
    node.RenderBox = &hotdog.RenderBox{}

    // 2. Calculate THIS node's layout (position, size)
    calculateNode(ctx, node, childIndex)

    // 3. Recursively layout children
    for i := 0; i < len(nodeChildren); i++ {
        layoutDOM(ctx, nodeChildren[i], i)

        // 4. Accumulate height (block layout stacks vertically)
        node.RenderBox.Height += nodeChildren[i].RenderBox.Height
    }
}
```

### `calculateNode`: Dispatch by Display Type

```go
func calculateNode(ctx *gg.Context, node *hotdog.NodeDOM, position int) {
    switch node.Style.Display {
    case "block":
        calculateBlockLayout(ctx, node, position)
    case "inline":
        calculateInlineLayout(ctx, node, position)
    case "list-item":
        calculateListItemLayout(ctx, node, position)
    }
}
```

---

## 5.6 Block Layout: Vertical Stacking

**Block elements** (div, p, h1, section, etc.) stack **vertically**, each taking full width.

### Block Layout Algorithm

```go
func calculateBlockLayout(ctx *gg.Context, node *hotdog.NodeDOM, position int) {
    box := node.RenderBox
    parentBox := node.Parent.RenderBox

    // 1. Width: fill containing block (minus margins)
    box.Width = parentBox.Width
    if node.Style.MarginLeft != "" {
        box.Width -= parseLength(node.Style.MarginLeft)
    }
    if node.Style.MarginRight != "" {
        box.Width -= parseLength(node.Style.MarginRight)
    }

    // 2. X position: left margin from parent's left
    box.Left = parentBox.Left
    if node.Style.MarginLeft != "" {
        box.Left += parseLength(node.Style.MarginLeft)
    }

    // 3. Y position: after previous sibling (or parent's top)
    if position == 0 {
        box.Top = parentBox.Top
    } else {
        // After previous sibling's bottom + margins
        prevSibling := node.Parent.Children[position-1]
        box.Top = prevSibling.RenderBox.Top + prevSibling.RenderBox.Height
        if node.Style.MarginTop != "" {
            box.Top += parseLength(node.Style.MarginTop)
        }
    }

    // 4. Height: auto = sum of children (calculated in layoutDOM)
    //    Explicit height = that value
    if node.Style.Height != "" && node.Style.Height != "auto" {
        box.Height = parseLength(node.Style.Height)
    }
    // Otherwise height accumulates from children in layoutDOM
}
```

### Visual: Block Layout

```
HTML:                          Layout Result:
─────────────────────────────────────────────────────────────
<body>                         ┌─────────────────────────────┐
  <header>        ──────►      │ header: x=0, y=0, w=800, h=100│
  </header>                    ├─────────────────────────────┤
  <main>                       │ main: x=0, y=100, w=800, h=400│
    <article>                  │   ├─ article: x=20, y=120...  │
    </article>                 │   └─ aside: x=20, y=120...    │
    <aside>                    ├─────────────────────────────┤
    </aside>                   │ footer: x=0, y=500, w=800, h=100│
  </main>                      └─────────────────────────────┘
  <footer>
  </footer>
</body>
```

**Key insight**: Block layout is **simple vertical stacking**. Each block sits below the previous one. Width comes from parent. Height comes from children (or explicit value).

---

## 5.7 Inline Layout: Horizontal Flow

**Inline elements** (span, a, strong, text nodes) flow **horizontally**, wrapping at line ends.

### Inline Layout Algorithm (Simplified)

```go
func calculateInlineLayout(ctx *gg.Context, node *hotdog.NodeDOM, position int) {
    box := node.RenderBox
    parentBox := node.Parent.RenderBox

    // Inline elements don't create new lines themselves.
    // They're positioned by their INLINE CONTAINER (nearest block ancestor).
    // The container tracks: currentX, currentY, lineHeight

    // For text nodes: measure text width
    if node.Type == hotdog.NodeTypeText {
        box.Width = measureTextWidth(ctx, node.Content, node.Style)
        box.Height = node.Style.FontSize * node.Style.LineHeight
    }

    // Position determined by inline container's line layout
    // (handled in parent's block layout during child iteration)
}
```

**Actually**, in thdwb's legacy layout, inline layout is handled **during the parent's block layout** — the block container lays out its inline children in lines.

### Line Box Model

```
Inline Container (block element):
┌────────────────────────────────────────────────────────────┐
│ Line 1: "Hello " (span)  "world" (text)  "!" (span)       │
│         x=10                    x=70              x=120    │
│         ────────────────────────────────────────────────   │
│ Line 2: "This wraps" (text)  "to next line" (text)        │
│         x=10                    x=100                      │
└────────────────────────────────────────────────────────────┘
```

**Reference**: [MDN - Inline Layout](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Flow_Layout/Block_and_Inline_Layout_in_Normal_Flow) | [Firefox nsInlineFrame](https://searchfox.org/mozilla-central/source/layout/generic/nsInlineFrame.cpp)

---

## 5.8 List Item Layout

**List items** (`<li>`) are block-level with a **marker** (bullet/number).

```go
func calculateListItemLayout(ctx *gg.Context, node *hotdog.NodeDOM, position int) {
    // 1. Do block layout for the content
    calculateBlockLayout(ctx, node, position)

    // 2. Add space for marker (bullet/number) on the left
    box := node.RenderBox
    markerWidth := 20.0  // Approximate marker width
    box.Left += markerWidth
    box.Width -= markerWidth

    // 3. Marker position (outside the content box)
    //    Rendered separately in paintListItemElement
}
```

**Reference**: [MDN - List Items](https://developer.mozilla.org/en-US/docs/Web/CSS/list-style) | [Firefox nsListBoxFrame](https://searchfox.org/mozilla-central/source/layout/generic/nsListBoxFrame.cpp)

---

## 5.9 The RenderBox: Layout Output

```go
// RenderBox holds the computed layout for a node.
// This is what the painter uses to draw.
type RenderBox struct {
    // Position (relative to viewport origin)
    Left   float64
    Top    float64

    // Size
    Width  float64
    Height float64

    // Reference back to DOM node (for painting focus rings, etc.)
    Node *NodeDOM

    // For flexbox layout (experimental mode)
    FlexNode interface{}
}
```

**Every element with `Display != "none"` gets a RenderBox.**

---

## 5.10 Experimental Layout: Flexbox (kjk/flex)

When `experimentalLayout=true`, thdwb uses the **kjk/flex** library (Go port of Facebook's Yoga).

### Flexbox Flow

```go
func RenderDocument(ctx *gg.Context, document *hotdog.Document, experimentalLayout bool) error {
    if experimentalLayout {
        // 1. Find <html> root
        html, _ := document.DOM.FindChildByName("html")

        // 2. Create flex nodes for entire DOM tree
        createFlexNodes(html)

        // 3. Build render tree (only visible nodes)
        renderTree := createRenderTree(html)

        // 4. Set viewport size on root
        renderTree.RenderBox.Width = float64(ctx.Width())
        renderTree.RenderBox.Height = float64(ctx.Height())

        // 5. Clear background
        ctx.SetRGB(1, 1, 1)
        ctx.Clear()

        // 6. LAYOUT: Run flexbox algorithm
        layoutNode(ctx, renderTree)

        // 7. PAINT
        paintTree(ctx, renderTree)

        // 8. Debug: print tree
        renderTree.Print(0)
    }
}
```

### Creating Flex Nodes

```go
// createFlexNodes walks the DOM and creates a flex.Node for each element.
// Uses kjk/flex library (Yoga port).
func createFlexNodes(node *hotdog.NodeDOM) *flex.Node {
    // Map CSS flex properties to flex.Node properties
    flexNode := flex.NewNode()

    // Display: flex/inline-flex → flex.Node
    if node.Style.Display == "flex" || node.Style.Display == "inline-flex" {
        flexNode.SetFlexDirection(mapFlexDirection(node.Style.FlexDirection))
        flexNode.SetFlexWrap(mapFlexWrap(node.Style.FlexWrap))
        flexNode.SetJustifyContent(mapJustifyContent(node.Style.JustifyContent))
        flexNode.SetAlignItems(mapAlignItems(node.Style.AlignItems))
        flexNode.SetAlignContent(mapAlignContent(node.Style.AlignContent))
    }

    // Flex item properties
    flexNode.SetFlexGrow(node.Style.FlexGrow)
    flexNode.SetFlexShrink(node.Style.FlexShrink)
    flexNode.SetFlexBasis(mapFlexBasis(node.Style.FlexBasis))

    // Size constraints
    if node.Style.Width != "" && node.Style.Width != "auto" {
        flexNode.SetWidth(parseLength(node.Style.Width))
    }
    if node.Style.Height != "" && node.Style.Height != "auto" {
        flexNode.SetHeight(parseLength(node.Style.Height))
    }
    // ... min/max width/height, margins, padding, border

    // Store reference on DOM node
    node.FlexNode = flexNode

    // Recurse for children
    for _, child := range node.Children {
        childFlex := createFlexNodes(child)
        flexNode.AddChild(childFlex)
    }

    return flexNode
}
```

### Running Flexbox Layout

```go
// layoutNode runs the flexbox algorithm and extracts results.
func layoutNode(ctx *gg.Context, node *RenderNode) {
    flexNode := node.FlexNode.(*flex.Node)

    // 1. Run Yoga layout algorithm
    flexNode.CalculateLayout(node.RenderBox.Width, node.RenderBox.Height, flex.DirectionLTR)

    // 2. Extract computed layout
    node.RenderBox.Left = flexNode.LayoutGetLeft()
    node.RenderBox.Top = flexNode.LayoutGetTop()
    node.RenderBox.Width = flexNode.LayoutGetWidth()
    node.RenderBox.Height = flexNode.LayoutGetHeight()

    // 3. Recurse for children
    for _, child := range node.Children {
        layoutNode(ctx, child)
    }
}
```

**Reference**: [MDN - Flexbox](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Flexible_Box_Layout) | [Firefox Flexbox](https://searchfox.org/mozilla-central/source/layout/generic/nsFlexContainerFrame.cpp) | [kjk/flex](https://github.com/kjk/flex)

---

## 5.11 Painting: From Layout to Pixels

After layout, `paintTree` draws each element.

### Paint Dispatch

```go
func paintNode(ctx *gg.Context, node *hotdog.NodeDOM) {
    switch node.Style.Display {
    case "block":
        paintBlockElement(ctx, node)
    case "inline":
        paintInlineElement(ctx, node)
    case "list-item":
        paintListItemElement(ctx, node)
    }
}

func paintTree(ctx *gg.Context, node *hotdog.NodeDOM) {
    if node == nil {
        return
    }
    paintNode(ctx, node)           // Paint this node
    for _, child := range node.Children {
        paintTree(ctx, child)      // Paint children (on top)
    }
    paintFocusRing(ctx, node)      // Paint focus indicator last
}
```

### Block Painting

```go
func paintBlockElement(ctx *gg.Context, node *hotdog.NodeDOM) {
    box := node.RenderBox
    if box.Width <= 0 || box.Height <= 0 {
        return
    }

    ctx.Push()
    defer ctx.Pop()

    // 1. Background
    if node.Style.BackgroundColor != nil {
        ctx.SetColor(node.Style.BackgroundColor)
        ctx.DrawRectangle(box.Left, box.Top, box.Width, box.Height)
        ctx.Fill()
    }

    // 2. Border
    if node.Style.BorderTopWidth != "" {
        drawBorders(ctx, box, node.Style)
    }

    // 3. Content (text, images) - handled by children or inline paint
}
```

**Painting Order** (bottom to top):
1. Background
2. Border
3. Children (recursive)
4. Focus ring (if focused)

**Reference**: [MDN - Paint](https://developer.mozilla.org/en-US/docs/Web/Performance/How_browsers_work#paint) | [Firefox Painting](https://searchfox.org/mozilla-central/source/layout/painting)

---

## 5.12 Focus Ring: Visual Feedback

```go
func paintFocusRing(ctx *gg.Context, node *hotdog.NodeDOM) {
    if node == nil || node.RenderBox == nil || node.RenderBox.Width <= 0 || node.RenderBox.Height <= 0 {
        return
    }

    // Find the actual focusable element
    source := node
    if node.RenderBox.Node != nil {
        source = node.RenderBox.Node
    }
    if source.WindowCtx == nil || source.WindowCtx.GetFocusedElement() != source {
        return
    }

    box := node.RenderBox
    ctx.Push()
    ctx.SetHexColor("#1a73e8")  // Blue focus ring
    ctx.SetLineWidth(2)
    ctx.DrawRectangle(box.Left-2, box.Top-2, box.Width+4, box.Height+4)
    ctx.Stroke()
    ctx.Pop()
}
```

---

## 5.13 Page Title Extraction

```go
// GetPageTitle finds the <title> element content in the DOM tree.
func GetPageTitle(TreeDOM *hotdog.NodeDOM) string {
    nodeChildren := getNodeChildren(TreeDOM)
    pageTitle := "Sem Titulo"  // Portuguese: "No Title"

    if getElementName(TreeDOM) == "title" {
        return getNodeContent(TreeDOM)
    }

    for i := 0; i < len(nodeChildren); i++ {
        nPageTitle := GetPageTitle(nodeChildren[i])
        if nPageTitle != "Sem Titulo" {
            pageTitle = nPageTitle
        }
    }

    return pageTitle
}
```

---

## 5.14 Canvas Rendering: Offscreen Drawing

thdwb can render to a **canvas** (for `<canvas>` element, printing, etc.):

```go
// CanvasContext interface for mustard canvas compatibility.
type CanvasContext interface {
    GetContext() *gg.Context
}

// RenderDocumentToCanvas draws a document into a mustard-compatible canvas.
func RenderDocumentToCanvas(canvas CanvasContext, document *hotdog.Document, experimentalLayout bool) error {
    if canvas == nil {
        return fmt.Errorf("canvas has no drawing context")
    }
    context := canvas.GetContext()
    if context == nil {
        return fmt.Errorf("canvas has no drawing context")
    }
    return RenderDocument(context, document, experimentalLayout)
}
```

---

## 5.15 Cross-References

| Concept | Chapter |
|---------|---------|
| How styles are computed | [Chapter 4: CSS Styling](../04-css-styling.md) |
| How graphics are drawn | [Chapter 6: Rendering & Graphics](../06-rendering-graphics.md) |
| How JS triggers relayout | [Chapter 7: JavaScript Runtime](../07-javascript-runtime.md) |
| Box model details | [Glossary: Box Model](../glossary.md#box-model) |
| Flexbox details | [Glossary: Flexbox](../glossary.md#flexbox) |

---

## 5.16 Example Test: Legacy Layout

```go
// Example_RenderDocument demonstrates legacy layout and rendering.
func Example_RenderDocument() {
    html := `<!DOCTYPE html>
<html>
  <head><title>Layout Test</title></head>
  <body style="margin: 0; font-family: sans-serif;">
    <header style="background: #333; color: white; padding: 20px;">
      <h1 style="margin: 0;">My Site</h1>
    </header>
    <main style="padding: 20px;">
      <p>Hello, layout engine!</p>
    </main>
  </body>
</html>`

    ctx := hotdog.NewWindowContext()
    defer ctx.Close()

    doc := ketchup.ParseHTML(html, ctx)

    // Create drawing context (800x600)
    dc := gg.NewContext(800, 600)

    // Render with legacy layout
    err := bun.RenderDocument(dc, doc, false)
    if err != nil {
        log.Fatal(err)
    }

    // Save to file for inspection
    dc.SavePNG("layout_output.png")
    fmt.Println("Saved layout_output.png")

    // Output: (file created)
}
```

Run with: `go test -run Example_RenderDocument ./bun`

---

## 5.17 Example Test: Flexbox Layout

```go
// Example_RenderDocumentFlex demonstrates experimental flexbox layout.
func Example_RenderDocumentFlex() {
    html := `<!DOCTYPE html>
<html>
  <head><title>Flex Test</title></head>
  <body style="display: flex; min-height: 100vh; margin: 0;">
    <nav style="flex: 0 0 200px; background: #f0f0f0; padding: 20px;">Navigation</nav>
    <main style="flex: 1; padding: 20px;">Main Content</main>
    <aside style="flex: 0 0 200px; background: #f0f0f0; padding: 20px;">Sidebar</aside>
  </body>
</html>`

    ctx := hotdog.NewWindowContext()
    defer ctx.Close()

    doc := ketchup.ParseHTML(html, ctx)
    dc := gg.NewContext(800, 600)

    // Render with experimental flexbox layout
    err := bun.RenderDocument(dc, doc, true)
    if err != nil {
        log.Fatal(err)
    }

    dc.SavePNG("flex_layout_output.png")
    fmt.Println("Saved flex_layout_output.png")
}
```

Run with: `go test -run Example_RenderDocumentFlex ./bun`

---

## 5.18 Summary

- **Layout** = calculating x, y, width, height for every element
- **Legacy mode**: Custom block/inline layout (vertical stacking + horizontal flow)
- **Experimental mode**: Flexbox via `kjk/flex` (Yoga port)
- **RenderBox** = output of layout, input to painting
- **Block layout**: Stack vertically, width from parent, height from children
- **Inline layout**: Flow horizontally, wrap at container edge
- **Flexbox**: Powerful row/column layout with grow/shrink/align
- **Painting**: Background → Border → Children → Focus ring

---

## 5.19 Next Chapter

👉 **[Chapter 6: Rendering & Graphics — gg + mustard](../06-rendering-graphics.md)**

Learn how the 2D graphics library (gg) draws shapes, text, and images, and how mustard manages windows and OpenGL.

---

*Key terms from this chapter: [Block Element](../glossary.md#block-element), [Box Model](../glossary.md#box-model), [Flexbox](../glossary.md#flexbox), [Inline Element](../glossary.md#inline-element), [Layout](../glossary.md#layout), [Paint](../glossary.md#paint), [Reflow](../glossary.md#reflow), [Render Box](../glossary.md#render-box), [Viewport](../glossary.md#viewport)*