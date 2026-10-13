# Chapter 3: HTML Parsing & DOM — ketchup + hotdog

> **In this chapter**: How thdwb turns raw HTML text into a structured DOM tree. This is the "blueprint reader" of the browser.

---

## 3.1 What Is HTML Parsing?

When the network layer delivers HTML text (like `<html><body><h1>Hello</h1></body></html>`), the browser can't use it directly. It needs to **parse** it — analyze the structure and build a tree of objects representing the document.

**Analogy**: HTML text is like a recipe written in shorthand. The parser reads it and builds a detailed, organized cookbook (the DOM tree) that the rest of the browser can follow.

```
HTML TEXT (bytes)          TOKENIZATION          TREE CONSTRUCTION          DOM TREE
─────────────────          ─────────────         ───────────────────        ────────
<html>                     Tokens:               Build tree:                Document
  <body>                     <html>                 html                      │
    <h1>Hello</h1>           <body>                  ├─ head                  ├─ html
  </body>                    <h1>                   │  └─ title               │  ├─ head
</html>                      "Hello"                 │     └─ "My Page"        │  │  └─ title
                            </h1>                   └─ body                     │  └─ body
                            </body>                      ├─ h1                     │     ├─ h1
                            </html>                      │  └─ "Hello"             │     │  └─ "Hello"
                                                         └─ (text nodes)           │     └─ (text)
```

---

## 3.2 Key Terms (Defined in [Glossary](../glossary.md))

- **Parser** — Program that reads text and builds structured data
- **Tokenization** — Breaking text into meaningful chunks (tags, text, comments)
- **Tree Construction** — Building the DOM tree from tokens
- **DOM (Document Object Model)** — Tree of nodes representing the document
- **Node** — Single item in the DOM tree (element, text, comment, etc.)
- **Element** — HTML tag pair with content (e.g., `<p>text</p>`)
- **Void Element** — Self-closing tag (e.g., `<img>`, `<br>`)
- **NodeDOM** — thdwb's internal DOM node structure

---

## 3.3 The Two Parsers in thdwb

thdwb has **two HTML parsers** for different use cases:

| Parser | Package | Use Case | Standards Compliance |
|--------|---------|----------|---------------------|
| **ketchup** | `ketchup` | Main parser, custom implementation | Simplified, learning-focused |
| **golang.org/x/net/html** | (stdlib) | Fallback, testing | Full HTML5 spec |

The **ketchup** package is thdwb's own parser — built from scratch for learning. It's simpler than the HTML5 spec but handles common cases.

---

## 3.4 ketchup Package: Custom HTML Parser

### File Overview

```
ketchup/
├── html.go        # Main parser: ParseHTML, ParsePlainText
├── parser.go      # Core parsing logic (tokenization + tree building)
├── helpers.go     # Utility functions (void elements, attributes)
└── ketchup_test.go
```

### Entry Points

```go
// ParseHTML parses a full HTML document string into a Document.
// Uses the custom ketchup parser (not the stdlib).
// Returns a *hotdog.Document with a populated DOM tree.
func ParseHTML(document string, windowCtx *hotdog.WindowContext) *hotdog.Document

// ParsePlainText creates a minimal document for plain text (no HTML tags).
// Wraps each line in <p> tags inside a basic HTML structure.
func ParsePlainText(document string, windowCtx *hotdog.WindowContext) *hotdog.Document
```

---

## 3.5 Tokenization: Breaking Text into Tokens

### The Tokenizer (in `parser.go`)

The tokenizer uses **regular expressions** to identify tokens:

```go
var (
    // Matches any tag: <tag>, </tag>, <tag/>, <tag attr="val">
    xmlTag = regexp.MustCompile(`(\<.+?\>)|(\<//?\w+\>\\?)`)

    // Matches closing tags: </tag>
    clTag = regexp.MustCompile(`\<\/\w+\>`)

    // Matches self-closing tags: <tag/>
    selfClosingTag = regexp.MustCompile(`\<.+\/\>`)

    // Matches content between tags
    tagContent = regexp.MustCompile(`(.+?)\<\/`)

    // Matches tag name: <tagname
    tagName = regexp.MustCompile(`(\<\w+)`)

    // Matches attributes: name="value"
    attr = regexp.MustCompile(`\w+=".+?"`)
)
```

**How it works:**
1. Find next tag with `xmlTag.FindString()`
2. Determine tag type (opening, closing, self-closing, doctype)
3. Extract tag name and attributes
4. Extract text content between tags
5. Repeat until no more tags

**Analogy**: Like highlighting different parts of speech in a sentence — nouns, verbs, punctuation — each gets a different color.

---

## 3.6 Tree Construction: Building the DOM

### The Algorithm (Simplified)

```go
func ParseHTML(document string, windowCtx *hotdog.WindowContext) *hotdog.Document {
    HTMLDocument := &hotdog.Document{}
    HTMLDocument.RawDocument = document
    lastNode := HTMLDocument.DOM
    parseDocument := xmlTag.MatchString(document)
    document = strings.ReplaceAll(document, "\n", "")

    for parseDocument == true {
        currentTag := xmlTag.FindString(document)
        currentTagIndex := xmlTag.FindStringIndex(document)

        if string(currentTag[1]) == "!" {
            // Doctype or comment — skip
            document = strings.Replace(document, currentTag, "", 1)
        } else if clTag.MatchString(currentTag) {
            // CLOSING TAG: </tag>
            // Extract text content, assign to lastNode, move up to parent
            contentString := extractContent(document)
            lastNode.Content = strings.TrimSpace(contentString)
            lastNode = lastNode.Parent
        } else {
            // OPENING TAG: <tag> or <tag/>
            currentTagName := extractTagName(currentTag)
            extractedAttributes := extractAttributes(currentTag)
            elementStylesheet := mayo.GetElementStylesheet(currentTagName, extractedAttributes)

            // Create new DOM node
            currentNode = &hotdog.NodeDOM{
                Element:    currentTagName,
                Type:       hotdog.NodeTypeElement,
                Content:    "",
                Children:   []*hotdog.NodeDOM{},
                Attributes: extractedAttributes,
                Style:      elementStylesheet,
                Parent:     lastNode,
                NeedsReflow:  true,
                NeedsRepaint: true,
                RenderBox:    &hotdog.RenderBox{},
                Document:  HTMLDocument,
                WindowCtx: windowCtx,
            }

            if currentTagName == "html" {
                HTMLDocument.DOM = currentNode
                lastNode = HTMLDocument.DOM
            } else {
                lastNode.Children = append(lastNode.Children, currentNode)
                if !isVoidElement(currentTagName) {
                    lastNode = currentNode  // Descend into this element
                }
            }
        }

        document = document[currentTagIndex[1]:]
        parseDocument = xmlTag.MatchString(document)
    }

    // Apply CSS inheritance after all stylesheets are processed
    mayo.ApplyInheritance(HTMLDocument.DOM)

    return HTMLDocument
}
```

### Visual: Tree Construction Step by Step

```
Input: <html><body><h1>Hi</h1></body></html>

Step 1: <html>          Step 2: <body>          Step 3: <h1>
┌─────────┐             ┌─────────┐             ┌─────────┐
│ html    │             │ html    │             │ html    │
└─────────┘             │  └─ body│             │  └─ body│
                        └─────────┘             │     └─ h1│
                                                └─────────┘

Step 4: "Hi" (text)     Step 5: </h>            Step 6: </body>
┌─────────┐             ┌─────────┐             ┌─────────┐
│ html    │             │ html    │             │ html    │
│  └─ body│             │  └─ body│             │  └─ body│
│     └─ h1│             │     └─ h1│            │     └─ h1│
│        └─ "Hi"        │        └─ "Hi"        │        └─ "Hi"│
└─────────┘             └─────────┘             └─────────┘

Step 7: </html> (done)
┌─────────┐
│ html    │
│  └─ body│
│     └─ h1│
│        └─ "Hi"        ← Complete DOM Tree!
└─────────┘
```

---

## 3.7 Void Elements: Self-Closing Tags

Some HTML elements **never have children** — they're "void":

```go
func isVoidElement(tagName string) bool {
    switch tagName {
    case "area", "base", "br", "col", "command", "embed", "hr",
         "img", "input", "keygen", "link", "meta", "param",
         "source", "track", "wbr":
        return true
    default:
        return false
    }
}
```

When the parser sees `<img src="x.png">`, it:
1. Creates the `img` node
2. Does NOT descend into it (no `lastNode = currentNode`)
3. Continues with the next sibling

**Reference**: [MDN - Void Elements](https://developer.mozilla.org/en-US/docs/Glossary/Void_element) | [Firefox Void Elements List](https://searchfox.org/mozilla-central/source/parser/html/HTMLTreeBuilder.cpp#1200)

---

## 3.8 hotdog Package: The DOM Tree

### NodeDOM: The Core Node Structure

```go
// NodeDOM represents a single node in the DOM tree.
// This is thdwb's internal representation — similar to browser's internal Node.
type NodeDOM struct {
    // Node identity
    Type      NodeType       // Element, Text, Comment, Doctype, Raw
    Element   string         // Tag name: "div", "span", "html:text", etc.
    Content   string         // Text content (for text nodes, or inner text)

    // Tree structure
    Children    []*NodeDOM   // Child nodes
    Parent      *NodeDOM     // Parent node (nil for root)
    FirstChild  *NodeDOM     // First child (linked list)
    NextSibling *NodeDOM     // Next sibling
    PrevSibling *NodeDOM     // Previous sibling

    // Styling & Layout
    Attributes  []*Attribute // HTML attributes
    Style       *Stylesheet  // Computed CSS styles
    RenderBox   *RenderBox   // Layout result: x, y, width, height

    // JavaScript & Runtime
    WindowCtx   *WindowContext `json:"-"` // Owning window (for JS bindings)
    Document    *Document      `json:"-"` // Owner document
    HTMLNode    *html.Node     `json:"-"` // Reference to stdlib html.Node (for selectors)

    // Flexbox layout (kjk/flex)
    FlexNode interface{} `json:"-"`

    // Dirty flags for incremental updates
    Dirty        bool
    NeedsReflow  bool
    NeedsRepaint bool

    // Iframe support
    IframeDocument *Document
    SandboxFlags   SandboxFlags
    IframeSrc      string
}
```

### Node Types

```go
type NodeType int

const (
    NodeTypeElement NodeType = iota  // <div>, <span>, etc.
    NodeTypeText                     // Text content
    NodeTypeComment                  // <!-- comment -->
    NodeTypeDoctype                  // <!DOCTYPE html>
    NodeTypeRaw                      // Raw HTML (unparsed)
)
```

**Analogy**: Each `NodeDOM` is a **card in a card catalog** — it has a label (element name), content (text), links to related cards (parent, children, siblings), and styling info.

---

## 3.9 Document: The Root Container

```go
// Document represents a parsed HTML document.
// Contains the DOM tree root and metadata.
type Document struct {
    Title        string      // Page title (from <title>)
    URL          string      // Document URL
    RawDocument  string      // Original HTML source
    DOM          *NodeDOM    // Root of DOM tree (usually <html>)
}
```

### Creating the Base Structure

```go
// CreateHTMLBase builds the minimal HTML structure:
// <html><head></head><body></body></html>
func CreateHTMLBase(windowCtx *hotdog.WindowContext) *hotdog.NodeDOM {
    html := &hotdog.NodeDOM{
        Element:      "html",
        Type:         hotdog.NodeTypeElement,
        NeedsReflow:  true,
        NeedsRepaint: true,
        Style:        mayo.GetElementStylesheet("html", []*hotdog.Attribute{}),
        RenderBox:    &hotdog.RenderBox{},
        WindowCtx:    windowCtx,
    }

    head := &hotdog.NodeDOM{
        Element:      "head",
        Type:         hotdog.NodeTypeElement,
        NeedsReflow:  true,
        NeedsRepaint: true,
        Style:        mayo.GetElementStylesheet("head", []*hotdog.Attribute{}),
        RenderBox:    &hotdog.RenderBox{},
        Parent:       html,
        WindowCtx:    windowCtx,
    }

    body := &hotdog.NodeDOM{
        Element:      "body",
        Type:         hotdog.NodeTypeElement,
        NeedsReflow:  true,
        NeedsRepaint: true,
        Style:        mayo.GetElementStylesheet("body", []*hotdog.Attribute{}),
        RenderBox:    &hotdog.RenderBox{},
        Parent:       html,
        WindowCtx:    windowCtx,
    }

    html.Children = append(html.Children, head, body)
    return html
}
```

This ensures every document has a valid structure even if the HTML is malformed.

---

## 3.10 DOM Tree Operations

### Finding Elements

```go
// FindChildByName finds first direct child with given element name.
func (node *NodeDOM) FindChildByName(name string) (*NodeDOM, error)

// FindByXPath finds node by XPath expression.
func (node *NodeDOM) FindByXPath(xPath string) (*NodeDOM, error)

// GetXPath returns the XPath of this node.
func (node *NodeDOM) GetXPath() string
```

### Tree Traversal

```go
// Print prints the tree structure (for debugging).
func (node *NodeDOM) Print(d int) {
    spacing := strings.Repeat("-", d)
    fmt.Printf("|%s> %s [%s]\n", spacing, node.Element, node.Content)
    for _, child := range node.Children {
        child.Print(d + 1)
    }
}
```

**Output example:**
```
|> html []
|-> head []
|--> title [My Page]
|-> body []
|--> h1 [Hello]
|--> p [This is a paragraph]
```

### JSON Serialization

```go
// JSON returns the node and its subtree as indented JSON.
func (node *NodeDOM) JSON() string
```

---

## 3.11 CSS Selector Integration (Cascadia)

thdwb uses **cascadia** (Go CSS selector library) for `querySelector`:

```go
// QuerySelector finds first descendant matching CSS selector.
// Uses cascadia library for selector matching.
func (node *NodeDOM) QuerySelector(selector string) (*NodeDOM, error)

// QuerySelectorAll finds all descendants matching CSS selector.
func (node *NodeDOM) QuerySelectorAll(selector string) ([]*NodeDOM, error)
```

**How it works:**
1. Convert `NodeDOM` tree to `html.Node` tree (stdlib)
2. Use cascadia to match selectors against `html.Node`
3. Map results back to `NodeDOM`

**Reference**: [MDN - querySelector](https://developer.mozilla.org/en-US/docs/Web/API/Document/querySelector) | [Cascadia Library](https://github.com/andybalholm/cascadia)

---

## 3.12 CSS Inheritance Application

After parsing, computed styles are applied with inheritance:

```go
// ApplyInheritance walks the DOM tree and applies inheritable CSS properties
// from parent to child (color, font-family, font-size, etc.).
func ApplyInheritance(node *NodeDOM) {
    inheritableProps := map[string]bool{
        "color": true, "font-family": true, "font-size": true,
        "font-weight": true, "font-style": true, "line-height": true,
        "letter-spacing": true, "text-align": true, "text-indent": true,
        "text-transform": true, "visibility": true, "white-space": true,
        "word-spacing": true, "list-style": true, "cursor": true,
        "direction": true, "unicode-bidi": true, "opacity": true,
    }

    // For each child, inherit applicable properties from parent
    for _, child := range node.Children {
        if child.Style != nil && node.Style != nil {
            for prop, inherits := range inheritableProps {
                if inherits && child.Style.Get(prop) == "" {
                    child.Style.Set(prop, node.Style.Get(prop))
                }
            }
        }
        ApplyInheritance(child)  // Recurse
    }
}
```

**Reference**: [MDN - Inheritance](https://developer.mozilla.org/en-US/docs/Web/CSS/inheritance) | [Firefox ComputedStyle](https://searchfox.org/mozilla-central/source/layout/style/ComputedStyle.cpp)

---

## 3.13 Cross-References

| Concept | Chapter |
|---------|---------|
| How network fetches HTML | [Chapter 2: Network Layer](../02-network-layer.md) |
| How CSS styles the DOM | [Chapter 4: CSS Styling](../04-css-styling.md) |
| How layout uses DOM | [Chapter 5: Layout Engine](../05-layout-engine.md) |
| How JS accesses DOM | [Chapter 7: JavaScript Runtime](../07-javascript-runtime.md) |
| Node types & tree terms | [Glossary: Node](../glossary.md#node), [Glossary: Tree](../glossary.md#tree) |

---

## 3.14 Example Test: Parsing HTML

```go
// Example_ParseHTML demonstrates parsing HTML with the ketchup package.
func Example_ParseHTML() {
    html := `<!DOCTYPE html>
<html>
  <head><title>Test Page</title></head>
  <body>
    <h1>Hello, World!</h1>
    <p>This is a <strong>test</strong>.</p>
  </body>
</html>`

    ctx := hotdog.NewWindowContext()
    defer ctx.Close()

    doc := ketchup.ParseHTML(html, ctx)

    fmt.Println("Title:", doc.Title)
    fmt.Println("DOM Tree:")
    doc.DOM.Print(0)

    // Output:
    // Title: Test Page
    // DOM Tree:
    // |> html []
    // |-> head []
    // |--> title [Test Page]
    // |-> body []
    // |--> h1 [Hello, World!]
    // |--> p [This is a test.]
    // |---> strong [test]
}
```

Run with: `go test -run Example_ParseHTML ./ketchup`

---

## 3.15 Example Test: DOM Traversal

```go
// Example_DOMTraversal demonstrates walking the DOM tree.
func Example_DOMTraversal() {
    html := `<div><span>A</span><span>B</span></div>`
    ctx := hotdog.NewWindowContext()
    defer ctx.Close()

    doc := ketchup.ParseHTML(html, ctx)
    div := doc.DOM.Children[1].Children[0] // body > div

    // Find all spans
    spans, _ := div.QuerySelectorAll("span")
    for i, span := range spans {
        fmt.Printf("Span %d: %s\n", i, span.Content)
    }

    // Output:
    // Span 0: A
    // Span 1: B
}
```

Run with: `go test -run Example_DOMTraversal ./hotdog`

---

## 3.16 Summary

- **HTML Parsing** = Tokenization + Tree Construction
- **ketchup** = thdwb's custom parser (regex-based, simplified)
- **hotdog.NodeDOM** = Internal DOM node with tree links, styles, layout boxes
- **Document** = Root container with title, URL, DOM tree
- **Void elements** = Self-closing tags that never have children
- **CSS Inheritance** = Applied after parsing via `mayo.ApplyInheritance`
- **Selectors** = Powered by cascadia library on stdlib html.Node

---

## 3.17 Next Chapter

👉 **[Chapter 4: CSS Styling — The mayo Package](../04-css-styling.md)**

Learn how CSS text becomes computed styles, how cascade and inheritance work, and how the default stylesheet styles HTML elements.

---

*Key terms from this chapter: [DOM](../glossary.md#dom), [Element](../glossary.md#element), [Node](../glossary.md#node), [Parser](../glossary.md#parser), [Tokenization](../glossary.md#tokenization), [Tree](../glossary.md#tree), [Void Element](../glossary.md#void-element)*