# Chapter 1: What Is a Web Browser?

> **Welcome to the thdwb Browser Textbook!**
>
> This chapter introduces the big picture: what a browser is, how the web works, and the journey from typing a URL to seeing a webpage. By the end, you'll understand the main parts of a browser and how they fit together.

---

## 1.1 The Browser as a Translator

Imagine you're visiting a foreign country where everyone speaks a language you don't know. You need a **translator** — someone who:
1. Listens to what people say (receives data)
2. Understands the meaning (parses and interprets)
3. Explains it to you in your language (renders visually)
4. Helps you respond (handles your clicks, typing, etc.)

**A web browser is exactly that translator.** The "foreign language" is the web's languages: **HTML**, **CSS**, and **JavaScript**. The browser translates them into pixels on your screen that you can see and interact with.

```
┌─────────────────────────────────────────────────────────────┐
│                    THE BROWSER TRANSLATOR                    │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   🌐 INTERNET          📥 FETCH          🧠 PARSE           │
│   ───────────         ─────────         ─────────           │
│   Server sends        Browser           Browser reads       │
│   HTML, CSS, JS       downloads         HTML, builds        │
│   files               files             DOM tree            │
│        │                   │                   │             │
│        ▼                   ▼                   ▼             │
│   ┌─────────────────────────────────────────────────┐       │
│   │              BROWSER ENGINE                      │       │
│   │  ┌──────────┐ ┌──────────┐ ┌──────────┐        │       │
│   │  │  HTML    │ │   CSS    │ │   JS     │        │       │
│   │  │ Parser   │ │ Parser   │ │ Engine   │        │       │
│   │  └──────────┘ └──────────┘ └──────────┘        │       │
│   │        │           │           │                │       │
│   │        ▼           ▼           ▼                │       │
│   │  ┌──────────────────────────────────────────┐   │       │
│   │  │         LAYOUT ENGINE                     │   │       │
│   │  │    (calculates position & size)          │   │       │
│   │  └──────────────────────────────────────────┘   │       │
│   │        │                                       │       │
│   │        ▼                                       │       │
│   │  ┌──────────────────────────────────────────┐   │       │
│   │  │         RENDERER / PAINTER                │   │       │
│   │  │    (draws pixels on screen)              │   │       │
│   │  └──────────────────────────────────────────┘   │       │
│   └─────────────────────────────────────────────────┘       │
│        │                                                       │
│        ▼                                                       │
│   🖥️  YOUR SCREEN                                              │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 1.2 The Web's Three Languages

Before we dive into the browser, let's meet the three languages it translates:

### HTML (HyperText Markup Language) — The Structure
**HTML is the skeleton.** It defines *what* things are: headings, paragraphs, links, images, forms.

```html
<!DOCTYPE html>
<html>
  <head>
    <title>My Page</title>
  </head>
  <body>
    <h1>Hello, World!</h1>
    <p>This is a paragraph.</p>
    <a href="https://example.com">A link</a>
  </body>
</html>
```

**Key terms:**
- **Tag**: Markup like `<p>` or `</p>` that labels content
- **Element**: A tag pair + its content (e.g., `<p>Hello</p>`)
- **Attribute**: Extra info inside a tag (e.g., `href="..."`)

> 📖 **Reference**: [MDN - HTML](https://developer.mozilla.org/en-US/docs/Web/HTML) | [Firefox HTML Parser](https://searchfox.org/mozilla-central/source/parser/html)

### CSS (Cascading Style Sheets) — The Appearance
**CSS is the skin and clothes.** It defines *how* things look: colors, fonts, spacing, layout.

```css
body {
  font-family: Arial, sans-serif;
  background-color: #f0f0f0;
}

h1 {
  color: #333;
  text-align: center;
}

p {
  line-height: 1.6;
  margin: 1rem 0;
}
```

**Key terms:**
- **Selector**: Which elements to style (e.g., `h1`, `.class`, `#id`)
- **Property**: What to change (e.g., `color`, `font-size`)
- **Value**: The setting (e.g., `red`, `16px`)
- **Cascade**: How conflicting rules are resolved

> 📖 **Reference**: [MDN - CSS](https://developer.mozilla.org/en-US/docs/Web/CSS) | [Firefox CSS Parser](https://searchfox.org/mozilla-central/source/layout/style)

### JavaScript — The Behavior
**JavaScript is the muscles and brain.** It makes pages interactive: responding to clicks, fetching data, animating things.

```javascript
// When the button is clicked, show an alert
document.getElementById('myButton').addEventListener('click', () => {
  alert('You clicked me!');
});

// Fetch data from a server
fetch('/api/data')
  .then(response => response.json())
  .then(data => console.log(data));
```

**Key terms:**
- **DOM API**: JavaScript's interface to read/change HTML
- **Event**: Something that happened (click, load, keypress)
- **Event Listener**: Code that runs when an event occurs
- **Fetch/XHR**: Getting data from a server without reloading

> 📖 **Reference**: [MDN - JavaScript](https://developer.mozilla.org/en-US/docs/Web/JavaScript) | [Firefox SpiderMonkey](https://searchfox.org/mozilla-central/source/js/src)

---

## 1.3 The Browser's Main Loop: From URL to Pixels

When you type `https://example.com` and press Enter, here's what happens:

### Step 1: DNS Lookup 📞
**What**: Convert the domain name (`example.com`) to an IP address (`93.184.216.34`).
**Analogy**: Looking up a phone number in a phone book.
**thdwb component**: `sauce` package (network layer)

### Step 2: TCP Connection 📡
**What**: Open a connection to the server at that IP address.
**Analogy**: Dialing the phone and waiting for the other end to pick up.

### Step 3: TLS Handshake 🔐 (for HTTPS)
**What**: Negotiate encryption so nobody can eavesdrop.
**Analogy**: Agreeing on a secret code before talking.
**thdwb component**: `sauce` package (certificate validation, TOFU)

### Step 4: HTTP Request 📤
**What**: Send a request like `GET / HTTP/1.1`.
**Analogy**: Sending a letter asking for a specific document.

### Step 5: HTTP Response 📥
**What**: Server sends back HTML, CSS, JS files.
**Analogy**: Receiving the requested document in the mail.

### Step 6: HTML Parsing 🌳
**What**: Browser reads HTML text and builds the **DOM Tree**.
**Analogy**: Reading a blueprint and building a model.
**thdwb component**: `ketchup` package (parser) → `hotdog` package (DOM)

```
HTML Text                    DOM Tree
─────────────                ─────────
<html>                       Document
  <head>       ──────►         ├─ html
    <title>                     │  ├─ head
    </title>                    │  │  └─ title "My Page"
  </head>                      │  └─ body
  <body>                          ├─ h1 "Hello"
    <h1>Hello</h1>               └─ p "Paragraph"
    <p>Paragraph</p>
  </body>
</html>
```

### Step 7: CSS Parsing 🎨
**What**: Browser reads CSS and builds **Stylesheets** with computed styles for each element.
**Analogy**: Reading a decorating guide and making a shopping list for each room.
**thdwb component**: `mayo` package (CSS parser)

### Step 8: Style Computation 🧮
**What**: For each DOM node, figure out its final styles (combining cascade, inheritance, defaults).
**Analogy**: For each room, decide the final paint color, furniture, etc.
**thdwb component**: `mayo` package (cascade, inheritance)

### Step 9: Layout / Reflow 📐
**What**: Calculate exact position (x, y) and size (width, height) of every element.
**Analogy**: The architect measuring and marking where every wall goes.
**thdwb component**: `bun` package (layout engine, flexbox)

```
Layout Output (Render Boxes):
┌─────────────────────────────────────┐
│ html: x=0, y=0, w=800, h=600       │
│   ├─ head: x=0, y=0, w=0, h=0      │ (display: none)
│   │  └─ title: x=0, y=0, w=0, h=0  │
│   └─ body: x=0, y=0, w=800, h=600  │
│      ├─ h1: x=100, y=50, w=600, h=40│
│      └─ p: x=100, y=120, w=600, h=80│
└─────────────────────────────────────┘
```

### Step 10: Paint 🎨
**What**: Draw pixels on screen — backgrounds, borders, text, images.
**Analogy**: The painter filling in the blueprint with colors.
**thdwb component**: `gg` package (2D graphics) + `mustard` (UI toolkit)

### Step 11: Compositing 🖼️
**What**: Combine painted layers into the final image on screen.
**Analogy**: Stacking transparent sheets to make the final picture.

### Step 12: JavaScript Execution ⚡
**What**: Run JS code, which can modify the DOM, triggering steps 6-11 again!
**Analogy**: The brain sending signals to move muscles.
**thdwb component**: `hotdog/js_bindings.go` (Goja JS engine)

---

## 1.4 thdwb's Architecture: Packages as Building Blocks

thdwb is organized into packages, each handling one stage of the pipeline:

| Package | Role | Analogy |
|---------|------|---------|
| **sauce** | Network: HTTP, cookies, cache, certificates | Postal service + security guard |
| **ketchup** | HTML Parser: text → DOM tree | Blueprint reader |
| **hotdog** | DOM Tree + JS Runtime: nodes, window context, Goja VM | Model builder + brain |
| **mayo** | CSS Parser: text → stylesheets, cascade, inheritance | Decorating guide reader |
| **bun** | Layout Engine: DOM + styles → positioned boxes | Architect/measurer |
| **gg** | 2D Graphics: drawing primitives, text shaping | Painter's tools |
| **mustard** | UI Toolkit: windows, widgets, input, OpenGL | House frame + furniture |

```
thdwb PACKAGE DEPENDENCIES
────────────────────────────────────────────────────────────
                              ┌─────────┐
                              │  main   │ (mustard/app.go)
                              └────┬────┘
                                   │
              ┌────────────────────┼────────────────────┐
              ▼                    ▼                    ▼
         ┌─────────┐         ┌─────────┐         ┌─────────┐
         │ mustard │         │  hotdog │         │  sauce  │
         │ (UI)    │◄────────│ (DOM+JS)│────────►│ (Net)   │
         └────┬────┘         └────┬────┘         └────┬────┘
              │                   │                   │
              ▼                   ▼                   │
         ┌─────────┐         ┌─────────┐              │
         │   gg    │         │  bun    │              │
         │(Graphics)         │(Layout) │              │
         └────┬────┘         └────┬────┘              │
              │                   │                   │
              ▼                   ▼                   │
         ┌─────────────────────────────────┐          │
         │           mayo                  │          │
         │         (CSS)                   │          │
         └─────────────────────────────────┘          │
              │                                       │
              ▼                                       │
         ┌─────────────────────────────────┐          │
         │           ketchup               │          │
         │         (HTML Parser)           │          │
         └─────────────────────────────────┘          │
                                                      │
              ◄───────────────────────────────────────┘
                    (sauce provides HTTP client)
```

---

## 1.5 How to Read This Textbook

Each chapter follows this structure:

1. **Concept Overview** — What this component does and why it matters
2. **Key Terms** — Defined in plain language (see [Glossary](../glossary.md))
3. **Code Walkthrough** — Tour of the actual Go code with explanations
4. **Analogy** — Concrete comparison to everyday things
5. **Mozilla/Firefox Reference** — Verified links to specs and source
6. **Cross-References** — Links to related chapters
7. **Example Tests** — Runnable Go code demonstrating the concept

**Reading order**: Chapters 1-8 build on each other. Start here and go sequentially.

---

## 1.6 Quick Reference: The Rendering Pipeline

```
┌─────────────┐   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│   NETWORK   │──►│    HTML     │──►│    CSS      │──►│   LAYOUT    │
│   (sauce)   │   │  (ketchup)  │   │   (mayo)    │   │   (bun)     │
└─────────────┘   └─────────────┘   └─────────────┘   └──────┬──────┘
                                                              │
                                                              ▼
┌─────────────┐   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│  JAVASCRIPT │◄──│   PAINT     │◄──│  COMPOSITE  │◄──│   RENDER    │
│ (hotdog/JS) │   │   (gg)      │   │ (mustard)   │   │  (gg+mustard)│
└─────────────┘   └─────────────┘   └─────────────┘   └─────────────┘
```

---

## 1.7 Exercises

1. **Explore**: Open your browser's DevTools (F12), go to the Network tab, reload a page. Watch the requests — which are HTML, CSS, JS, images?

2. **Inspect**: In DevTools, go to Elements tab. Hover over DOM nodes — see the box model (margin, border, padding, content) highlighted.

3. **Experiment**: In Console tab, type `document.body.style.background = 'red'` — the page changes instantly! That's JavaScript modifying the DOM.

---

## 1.8 Summary

- A **browser** translates HTML, CSS, and JavaScript into interactive pixels
- The **rendering pipeline** has 12+ steps from URL to screen
- **thdwb** splits this pipeline into 7 Go packages, each with a clear job
- This textbook walks through each package, building your mental model step by step

---

## 1.9 Next Chapter

👉 **[Chapter 2: Network Layer — The sauce Package](../02-network-layer.md)**

Learn how thdwb fetches resources, handles cookies, validates certificates, and caches responses.

---

*Key terms defined in this chapter: [API](../glossary.md#api), [ASCII](../glossary.md#ascii), [Block Element](../glossary.md#block-element), [Box Model](../glossary.md#box-model), [Browser Engine](../glossary.md#browser-engine), [Cache](../glossary.md#cache), [Cascading](../glossary.md#cascading), [CORS](../glossary.md#cors), [CSS](../glossary.md#css), [CSS Selector](../glossary.md#css-selector), [DOM](../glossary.md#dom), [DOCTYPE](../glossary.md#doctype), [Element](../glossary.md#element), [Event](../glossary.md#event), [Event Listener](../glossary.md#event-listener), [Flexbox](../glossary.md#flexbox), [Frame](../glossary.md#frame), [GPU](../glossary.md#gpu), [HTML](../glossary.md#html), [HTTP](../glossary.md#http), [HTTPS](../glossary.md#https), [Inline Element](../glossary.md#inline-element), [Inheritance](../glossary.md#inheritance), [JavaScript](../glossary.md#javascript), [JS Engine](../glossary.md#js-engine), [Layout](../glossary.md#layout), [Margin](../glossary.md#margin), [Markup](../glossary.md#markup), [Node](../glossary.md#node), [Origin](../glossary.md#origin), [Padding](../glossary.md#padding), [Paint](../glossary.md#paint), [Parser](../glossary.md#parser), [Reflow](../glossary.md#reflow), [Render Tree](../glossary.md#render-tree), [Rendering Engine](../glossary.md#rendering-engine), [Same-Origin Policy](../glossary.md#same-origin-policy), [Specificity](../glossary.md#specificity), [Stylesheet](../glossary.md#stylesheet), [Tag](../glossary.md#tag), [Tokenization](../glossary.md#tokenization), [Tree](../glossary.md#tree), [URL](../glossary.md#url), [User Agent](../glossary.md#user-agent), [Viewport](../glossary.md#viewport), [Web Standards](../glossary.md#web-standards), [XPath](../glossary.md#xpath)*