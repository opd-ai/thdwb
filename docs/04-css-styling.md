# Chapter 4: CSS Styling — The mayo Package

> **In this chapter**: How thdwb reads CSS, computes styles for each element, handles the cascade, inheritance, and default styles. This is the "decorating guide reader" of the browser.

---

## 4.1 What Is CSS Parsing & Styling?

After HTML parsing builds the DOM tree, the browser needs to know **how each element should look**. CSS (Cascading Style Sheets) provides the rules. The CSS parser reads CSS text and produces **computed styles** for every DOM node.

**Analogy**: HTML builds the house frame. CSS is the interior design plan — paint colors, furniture layout, window treatments. The styling engine reads the plan and creates a detailed spec for each room.

```
CSS TEXT                    PARSE & CASCADE              COMPUTED STYLES
─────────                   ───────────────              ───────────────
body {                      1. Parse all                body: {
  color: black;               stylesheets                 color: black,
  font: 16px sans;            2. Match selectors          font-family: sans-serif,
}                             3. Cascade (specificity)    font-size: 16px,
                              4. Inherit                  line-height: normal,
h1 {                          5. Apply defaults           ... }
  color: blue;                ────────────────────────────────────────
  font-size: 2em;             For EACH element:
}                             - Final value for EVERY property
                              - Ready for layout
```

---

## 4.2 Key Terms (Defined in [Glossary](../glossary.md))

- **CSS Selector** — Pattern matching elements to style
- **Cascade** — Algorithm resolving conflicting rules
- **Specificity** — Score determining which rule wins
- **Inheritance** — Child elements getting parent's styles
- **Computed Style** — Final resolved value for every property
- **User Agent Stylesheet** — Browser's default styles
- **Stylesheet** — Collection of CSS rules
- **Property/Value** — What to change and what to change it to

---

## 4.3 The mayo Package: File Overview

```
mayo/
├── mayo.go           # Main: parsing, cascade, inheritance, defaults
├── css.go            # CSS property parsing helpers
├── color.go          # Color parsing (named, hex, rgb, hsl)
├── reflow.go         # Style invalidation triggers
├── tables.go         # Table-specific layout styles
├── mayo_test.go      # Tests
└── fonts.go          # (in assets/) Font handling
```

---

## 4.4 CSS Parsing: From Text to Rules

### The Parser Entry Point

```go
// ParseCSS parses a CSS string into a Stylesheet (map of selectors to rules).
// Returns a map: selector string -> map of property:value pairs.
func ParseCSS(css string) map[string]map[string]string
```

**Simplified flow:**
1. Split CSS into rules (by `}`)
2. For each rule, split selector and declarations (by `{`)
3. Parse declarations into property:value pairs
4. Store in map keyed by selector

### Selector Matching (via Cascadia)

thdwb uses the **cascadia** library for selector matching:

```go
import cascadia "github.com/andybalholm/cascadia"

// Compile selector once
sel, _ := cascadia.Compile("div.content > p:first-child")

// Match against html.Node (stdlib)
matches := sel.MatchAll(htmlNode)
// Returns []*html.Node
```

**Reference**: [MDN - CSS Selectors](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Selectors) | [Cascadia](https://github.com/andybalholm/cascadia)

---

## 4.5 The Stylesheet Struct: Per-Element Computed Styles

Each DOM node gets a `Stylesheet` with **all computed properties**:

```go
// Stylesheet holds computed CSS styles for a single element.
// Every CSS property has a field. Zero values = "not set" (will inherit or default).
type Stylesheet struct {
    // Display & Box Model
    Display        string  // block, inline, flex, none, list-item, etc.
    Position       string  // static, relative, absolute, fixed, sticky
    Width          string  // auto, length, percentage
    Height         string  // auto, length, percentage
    MinWidth       string
    MaxWidth       string
    MinHeight      string
    MaxHeight      string

    // Box Model: Margin, Border, Padding
    MarginTop      string
    MarginRight    string
    MarginBottom   string
    MarginLeft     string
    PaddingTop     string
    PaddingRight   string
    PaddingBottom  string
    PaddingLeft    string
    BorderTopWidth string
    BorderRightWidth string
    BorderBottomWidth string
    BorderLeftWidth string
    BorderTopStyle string
    BorderRightStyle string
    BorderBottomStyle string
    BorderLeftStyle string
    BorderTopColor color.Color
    BorderRightColor color.Color
    BorderBottomColor color.Color
    BorderLeftColor color.Color

    // Flexbox
    FlexDirection  string  // row, column, row-reverse, column-reverse
    FlexWrap       string  // nowrap, wrap, wrap-reverse
    JustifyContent string  // flex-start, center, space-between, etc.
    AlignItems     string  // stretch, flex-start, center, etc.
    AlignContent   string
    FlexGrow       float64
    FlexShrink     float64
    FlexBasis      string

    // Typography
    Color          color.Color
    FontFamily     string
    FontSize       float64  // in pixels
    FontWeight     int      // 100-900
    FontStyle      string   // normal, italic, oblique
    LineHeight     float64
    TextAlign      string   // left, center, right, justify
    TextDecoration string
    TextTransform  string
    LetterSpacing  float64
    WordSpacing    float64
    WhiteSpace     string   // normal, nowrap, pre, pre-wrap, pre-line

    // Background
    BackgroundColor color.Color
    BackgroundImage string
    BackgroundRepeat string
    BackgroundPosition string
    BackgroundSize string

    // Visual
    Opacity        float64
    Visibility     string  // visible, hidden, collapse
    Overflow       string  // visible, hidden, scroll, auto
    ZIndex         int

    // List
    ListStyleType  string
    ListStylePosition string

    // Cursor
    Cursor         string

    // Internal: raw property map for cascade
    Properties map[string]string
}
```

**Key insight**: Every possible CSS property has a field. This makes lookup fast during layout — no map access needed.

---

## 4.6 Default Styles: The User Agent Stylesheet

Before any author CSS, elements get **default styles** from the browser:

```go
// getDefaultElementDisplay returns the default 'display' value for an element.
// Matches HTML spec and Firefox's UA stylesheet.
func getDefaultElementDisplay(element string) string {
    displayType := "block"  // Default for unknown elements

    switch element {
    case "script", "style", "meta", "link", "head", "title":
        displayType = "none"           // Never rendered
    case "li":
        displayType = "list-item"      // List items
    case "html:text", "a", "abbr", "acronym", "b", "bdo", "big", "br",
         "button", "cite", "code", "dfn", "em", "i", "img", "input", "kbd",
         "label", "map", "object", "output", "q", "samp", "select", "small",
         "span", "strong", "sub", "sup", "textarea", "time", "tt", "var", "font":
        displayType = "inline"         // Text-level elements
    }

    return displayType
}

// getDefaultElementFontWeight returns default font-weight for headings.
func getDefaultElementFontWeight(element string) int {
    switch element {
    case "h1", "h2", "h3", "h4", "h5", "h6":
        return 600  // Bold
    }
    return 400  // Normal
}
```

**Reference**: [MDN - User Agent Stylesheet](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Box_Model/Introduction_to_the_CSS_box_model#user_agent_styles) | [Firefox UA Stylesheet](https://searchfox.org/mozilla-central/source/layout/style/res)

### Creating an Element's Stylesheet

```go
// GetElementStylesheet creates a Stylesheet for an element with its attributes.
// 1. Start with defaults (display, font-weight)
// 2. Apply inline style="" attribute
// 3. Apply matched CSS rules (cascade)
// 4. Return computed Stylesheet
func GetElementStylesheet(element string, attributes []*hotdog.Attribute) *Stylesheet {
    sheet := &Stylesheet{
        Display:     getDefaultElementDisplay(element),
        FontWeight:  getDefaultElementFontWeight(element),
        FontSize:    14,  // Default 14px
        LineHeight:  1.2,
        Color:       color.Black,
        Properties:  make(map[string]string),
    }

    // Parse inline style="..." attribute
    for _, attr := range attributes {
        if attr.Name == "style" {
            parseInlineStyle(sheet, attr.Value)
        }
    }

    return sheet
}
```

---

## 4.7 The Cascade: Resolving Conflicts

When multiple rules target the same element, the **cascade** decides which wins.

### Cascade Order (Lowest to Highest Priority)

```
1. User Agent Stylesheet     (browser defaults)
2. User Stylesheet           (user custom styles - not implemented in thdwb)
3. Author Stylesheet         (site's CSS)
   3a. External stylesheets  (<link rel="stylesheet">)
   3b. Internal stylesheets  (<style> tags)
   3c. Inline styles         (style="..." attribute)
4. !important                (overrides everything above)
   4a. User Agent !important
   4b. User !important
   4c. Author !important
```

### Specificity: The Tiebreaker

When two author rules have same origin and importance, **specificity** wins:

```
Specificity = (A, B, C) where:
  A = number of ID selectors      (#myId)
  B = number of class/attr/pseudo (:hover, .class, [attr])
  C = number of type/pseudo-elem  (div, ::before)

Examples:
  *               → (0,0,0)
  div             → (0,0,1)
  .class          → (0,1,0)
  #id             → (1,0,0)
  div.class       → (0,1,1)
  #id .class      → (1,1,0)
  style=""        → (1,0,0,0)  // Inline = highest (treated as ID)
```

**Analogy**: Specificity is like a tiebreaker score. More specific selector = higher score = wins.

### thdwb's Cascade Implementation

```go
// ApplyStylesheet applies all matched CSS rules to an element's Stylesheet.
// Rules are pre-sorted by specificity (cascadia does this).
func ApplyStylesheet(sheet *Stylesheet, matchedRules []MatchedRule) {
    for _, rule := range matchedRules {
        for prop, value := range rule.Declarations {
            // !important handling
            if strings.HasSuffix(value, "!important") {
                value = strings.TrimSuffix(value, "!important")
                sheet.SetImportant(prop, value)
            } else if !sheet.IsImportant(prop) {
                sheet.Set(prop, value)
            }
        }
    }
}
```

**Reference**: [MDN - Cascade](https://developer.mozilla.org/en-US/docs/Web/CSS/Cascade) | [Firefox Cascade](https://searchfox.org/mozilla-central/source/layout/style/Cascade.cpp)

---

## 4.8 Inheritance: Parent → Child

Some CSS properties **inherit** by default — children get parent's value unless overridden.

### Inheritable Properties (in thdwb)

```go
var inheritableProperties = map[string]bool{
    "color":           true,
    "font-family":     true,
    "font-size":       true,
    "font-weight":     true,
    "font-style":      true,
    "line-height":     true,
    "letter-spacing":  true,
    "text-align":      true,
    "text-indent":     true,
    "text-transform":  true,
    "visibility":      true,
    "white-space":     true,
    "word-spacing":    true,
    "list-style":      true,
    "cursor":          true,
    "direction":       true,
    "unicode-bidi":    true,
    "opacity":         true,
}
```

**Non-inheritable** (by default): `display`, `width`, `height`, `margin`, `border`, `padding`, `position`, `background`, `flex-*`, etc.

### Applying Inheritance

```go
// ApplyInheritance walks the DOM tree top-down, inheriting properties.
// Called after all stylesheets are computed.
func ApplyInheritance(node *hotdog.NodeDOM) {
    for _, child := range node.Children {
        if child.Style != nil && node.Style != nil {
            for prop, inherits := range inheritableProperties {
                if inherits {
                    // Only inherit if child doesn't have its own value
                    if child.Style.Get(prop) == "" {
                        child.Style.Set(prop, node.Style.Get(prop))
                    }
                }
            }
        }
        ApplyInheritance(child)  // Recurse to grandchildren
    }
}
```

**Visual Example:**

```html
<div style="color: blue; font-size: 20px;">
  <p>This paragraph inherits blue color and 20px font</p>
  <span>This span also inherits</span>
  <div style="color: red;">This div OVERRIDES color to red</div>
</div>
```

```
Computed Styles:
div (parent):     color=blue, font-size=20px
  p:              color=blue (inherited), font-size=20px (inherited)
  span:           color=blue (inherited), font-size=20px (inherited)
  div (child):    color=red (own), font-size=20px (inherited)
```

**Reference**: [MDN - Inheritance](https://developer.mozilla.org/en-US/docs/Web/CSS/inheritance) | [Firefox ComputedStyle](https://searchfox.org/mozilla-central/source/layout/style/ComputedStyle.cpp)

---

## 4.9 Property Parsing: Strings → Typed Values

CSS values are strings. The parser converts them to typed Go values.

### Size Values (px, em, rem, %, etc.)

```go
// mapSizeValue converts "16px", "1.5em", "100%" to float64 pixels.
// For relative units (em, rem, %), returns the numeric part.
// Layout engine resolves relative units later.
func mapSizeValue(sizeValue string) float64 {
    re := regexp.MustCompile(`[0-9]+(\.[0-9]+)?`)
    valueString := re.FindString(sizeValue)
    if valueString == "" {
        return 14  // Default font size
    }
    value, _ := strconv.ParseFloat(valueString, 64)
    return value
}
```

### Color Parsing

```go
// MapCSSColor parses: named colors, #rgb, #rrggbb, rgb(), rgba(), hsl(), hsla()
func MapCSSColor(colorString string) color.Color {
    colorString = strings.TrimSpace(strings.ToLower(colorString))

    // Named colors
    if c, ok := namedColors[colorString]; ok {
        return c
    }

    // Hex: #rgb or #rrggbb
    if strings.HasPrefix(colorString, "#") {
        return parseHexColor(colorString)
    }

    // rgb(), rgba()
    if strings.HasPrefix(colorString, "rgb") {
        return parseRGBColor(colorString)
    }

    // hsl(), hsla()
    if strings.HasPrefix(colorString, "hsl") {
        return parseHSLColor(colorString)
    }

    return color.Black  // Fallback
}
```

**Reference**: [MDN - Color Values](https://developer.mozilla.org/en-US/docs/Web/CSS/color_value) | [Firefox Color Parsing](https://searchfox.org/mozilla-central/source/layout/style/nsCSSValue.cpp)

---

## 4.10 Style Resolution: The Complete Flow

```
┌─────────────────────────────────────────────────────────────────────┐
                    STYLE RESOLUTION PIPELINE
└─────────────────────────────────────────────────────────────────────┘

  HTML Element + Attributes
         │
         ▼
  ┌──────────────────┐
  │ 1. DEFAULT STYLES │  getDefaultElementDisplay(), getDefaultElementFontWeight()
  │    (UA Sheet)     │
  └────────┬─────────┘
           │
           ▼
  ┌──────────────────┐
  │ 2. INLINE STYLE  │  Parse style="color: red; font-size: 20px"
  │    ATTRIBUTE     │
  └────────┬─────────┘
           │
           ▼
  ┌──────────────────┐
  │ 3. MATCHED RULES │  cascadia matches selectors against element
  │    (CASCADE)     │  Sorted by specificity, applied in order
  └────────┬─────────┘
           │
           ▼
  ┌──────────────────┐
  │ 4. INHERITANCE   │  ApplyInheritance() walks tree top-down
  │    (TOP-DOWN)    │  Children get parent's inheritable properties
  └────────┬─────────┘
           │
           ▼
  ┌──────────────────┐
  │ 5. COMPUTED      │  Stylesheet struct with ALL properties resolved
  │    STYLESHEET    │  Ready for layout engine
  └──────────────────┘
```

---

## 4.11 Style Invalidation: When Styles Change

When DOM or CSS changes, affected elements need **reflow** (relayout) and **repaint**.

```go
// MarkStyleDirty marks an element and its descendants as needing style recalculation.
// Called when:
// - Element's class/id/style attribute changes
// - Stylesheet is added/removed/modified
// - Parent's style changes (might affect inheritance)
func MarkStyleDirty(node *hotdog.NodeDOM) {
    node.Dirty = true
    node.NeedsReflow = true
    node.NeedsRepaint = true

    // Propagate to children (inheritance might change)
    for _, child := range node.Children {
        MarkStyleDirty(child)
    }
}
```

**Reference**: [MDN - Reflow](https://developer.mozilla.org/en-US/docs/Web/Performance/How_browsers_work#reflow)

---

## 4.12 Cross-References

| Concept | Chapter |
|---------|---------|
| How HTML parser creates elements | [Chapter 3: HTML Parsing](../03-html-parsing-dom.md) |
| How layout uses computed styles | [Chapter 5: Layout Engine](../05-layout-engine.md) |
| How JS reads/changes styles | [Chapter 7: JavaScript Runtime](../07-javascript-runtime.md) |
| Color parsing details | [Glossary: CSS](../glossary.md#css) |
| Specificity & cascade | [Glossary: Specificity](../glossary.md#specificity), [Glossary: Cascading](../glossary.md#cascading) |

---

## 4.13 Example Test: Creating Stylesheets

```go
// Example_GetElementStylesheet demonstrates creating a stylesheet for an element.
func Example_GetElementStylesheet() {
    // Element with inline style
    attrs := []*hotdog.Attribute{
        {Name: "style", Value: "color: red; font-size: 20px; margin: 10px"},
    }

    sheet := mayo.GetElementStylesheet("div", attrs)

    fmt.Printf("Display: %s\n", sheet.Display)
    fmt.Printf("Color: %v\n", sheet.Color)
    fmt.Printf("FontSize: %.0fpx\n", sheet.FontSize)
    fmt.Printf("MarginTop: %s\n", sheet.MarginTop)

    // Output:
    // Display: block
    // Color: {0 0 0 255}  (red)
    // FontSize: 20px
    // MarginTop: 10px
}
```

Run with: `go test -run Example_GetElementStylesheet ./mayo`

---

## 4.14 Example Test: Inheritance

```go
// Example_ApplyInheritance demonstrates CSS inheritance.
func Example_ApplyInheritance() {
    ctx := hotdog.NewWindowContext()
    defer ctx.Close()

    // Parent with color and font-size
    parent := &hotdog.NodeDOM{
        Element: "div",
        Style:   mayo.GetElementStylesheet("div", nil),
    }
    parent.Style.Color = color.RGBA{0, 0, 255, 255}  // Blue
    parent.Style.FontSize = 20

    // Child (no explicit color)
    child := &hotdog.NodeDOM{
        Element: "p",
        Style:   mayo.GetElementStylesheet("p", nil),
        Parent:  parent,
    }
    parent.Children = []*hotdog.NodeDOM{child}

    // Apply inheritance
    mayo.ApplyInheritance(parent)

    fmt.Printf("Parent color: %v\n", parent.Style.Color)
    fmt.Printf("Child color (inherited): %v\n", child.Style.Color)
    fmt.Printf("Child font-size (inherited): %.0fpx\n", child.Style.FontSize)

    // Output:
    // Parent color: {0 0 255 255}
    // Child color (inherited): {0 0 255 255}
    // Child font-size (inherited): 20px
}
```

Run with: `go test -run Example_ApplyInheritance ./mayo`

---

## 4.15 Summary

- **mayo** parses CSS and computes styles for every DOM element
- **Stylesheet** struct holds all computed properties as typed fields
- **Default styles** (UA stylesheet) provide base display types, font weights
- **Cascade** resolves conflicts: origin → specificity → source order
- **Inheritance** passes certain properties (color, font, text) parent→child
- **Property parsing** converts CSS strings to Go types (float64, color.Color, int)
- **Style invalidation** marks dirty nodes when DOM/CSS changes

---

## 4.16 Next Chapter

👉 **[Chapter 5: Layout Engine — The bun Package](../05-layout-engine.md)**

Learn how computed styles + DOM tree become positioned boxes (x, y, width, height) through block, inline, and flexbox layout.

---

*Key terms from this chapter: [Cascade](../glossary.md#cascading), [Computed Style](../glossary.md#computed-style), [CSS Selector](../glossary.md#css-selector), [Inheritance](../glossary.md#inheritance), [Property](../glossary.md#property), [Specificity](../glossary.md#specificity), [Stylesheet](../glossary.md#stylesheet), [User Agent Stylesheet](../glossary.md#user-agent-stylesheet), [Value](../glossary.md#value)*