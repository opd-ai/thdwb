package hotdog

import (
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestToHTMLAndInnerHTML(t *testing.T) {
	wc := newBareWindowContext(t)
	doc := &Document{URL: mustParseURL(t, "https://doc.example/")}
	root := &NodeDOM{Type: NodeTypeElement, Element: "div", Document: doc, WindowCtx: wc}
	child := &NodeDOM{Type: NodeTypeElement, Element: "span", Document: doc, WindowCtx: wc}
	text := &NodeDOM{Type: NodeTypeText, Content: "a<b & >c", Document: doc, WindowCtx: wc}
	attachNode(t, root, child)
	attachNode(t, child, text)

	got := root.toHTML()
	want := `<div><span>a&lt;b &amp; &gt;c</span></div>`
	if got != want {
		t.Fatalf("toHTML() = %q, want %q", got, want)
	}
	if root.InnerHTML() != `<span>a&lt;b &amp; &gt;c</span>` {
		t.Fatalf("InnerHTML() = %q", root.InnerHTML())
	}
	if child.toHTML() != `<span>a&lt;b &amp; &gt;c</span>` {
		t.Fatalf("child toHTML() = %q", child.toHTML())
	}

	// escapeHTML directly for all entities.
	if got := escapeHTML(`<>"'&`); got != `&lt;&gt;&quot;&#39;&amp;` {
		t.Fatalf("escapeHTML = %q", got)
	}
	// Text-only and empty nodes.
	leaf := &NodeDOM{Type: NodeTypeText, Content: "x"}
	if leaf.toHTML() != "x" {
		t.Fatalf("text toHTML = %q", leaf.toHTML())
	}
	empty := &NodeDOM{Type: NodeTypeElement, Element: "p"}
	if empty.toHTML() != `<p></p>` {
		t.Fatalf("empty element toHTML = %q", empty.toHTML())
	}
	// Element with attributes.
	attrEl := &NodeDOM{Type: NodeTypeElement, Element: "a", Attributes: []*Attribute{{Name: "href", Value: "/x"}}}
	if attrEl.toHTML() != `<a href="/x"></a>` {
		t.Fatalf("attributed toHTML = %q", attrEl.toHTML())
	}
}

func attachNode(t *testing.T, parent, child *NodeDOM) {
	t.Helper()
	parent.appendChildNode(child)
}

func TestGetTextContentWhitespaceModes(t *testing.T) {
	// Non-text nodes return raw content.
	el := &NodeDOM{Type: NodeTypeElement, Element: "div", Content: "  raw  "}
	if el.GetTextContent() != "  raw  " {
		t.Fatalf("element GetTextContent = %q", el.GetTextContent())
	}

	text := &NodeDOM{Type: NodeTypeText, Content: "  a   b \n\t c  "}
	if got := text.GetTextContent(); got != "a b c" {
		t.Fatalf("normal GetTextContent = %q", got)
	}

	// pre-wrap / pre-line branches are driven via the white-space value
	// selected inside GetTextContent; the normal path is default. Exercise
	// the helpers directly for the other modes.
	if got := CollapseWhitespacePreserveNewlines("  a  b \n c\t d  "); got != "a b\nc d" {
		t.Fatalf("CollapseWhitespacePreserveNewlines = %q", got)
	}
	if got := CollapseWhitespace("  a  b \n\t c  "); got != "a b c" {
		t.Fatalf("CollapseWhitespace = %q", got)
	}
}

func TestEscapeCSSIdentifier(t *testing.T) {
	cases := map[string]string{
		"normal":  "normal",
		"_dash":   "_dash",
		"9digit":  `\9digit`,
		"-9digit": `\-9digit`,
		"sp ace":  `sp\ ace`,
		"unicode": "unicode",
		"é":       "é",
		"-":       "-",
	}
	for in, want := range cases {
		if got := escapeCSSIdentifier(in); got != want {
			t.Fatalf("escapeCSSIdentifier(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseSandboxAttributeAndCaps(t *testing.T) {
	if got := ParseSandboxAttribute(""); got != SandboxNone {
		t.Fatalf("ParseSandboxAttribute(\"\") = %v, want SandboxNone", got)
	}
	all := ParseSandboxAttribute("allow-scripts allow-forms allow-same-origin allow-top-navigation allow-popups allow-presentation allow-modals allow-orientation-lock allow-pointer-lock")
	for _, flag := range []SandboxFlags{SandboxAllowScripts, SandboxAllowForms, SandboxAllowSameOrigin, SandboxAllowTopNavigation, SandboxAllowPopups, SandboxAllowPresentation, SandboxAllowModals, SandboxAllowOrientationLock, SandboxAllowPointerLock} {
		if !all.HasSandboxFlag(flag) {
			t.Fatalf("flag %v missing from %v", flag, all)
		}
	}
	if all.HasSandboxFlag(SandboxNone) {
		t.Fatal("SandboxNone should not be a member of parsed flags")
	}
	// Case-insensitive token parsing.
	mixed := ParseSandboxAttribute("ALLOW-SCRIPTS")
	if !mixed.HasSandboxFlag(SandboxAllowScripts) {
		t.Fatal("case-insensitive parsing failed")
	}

	nonIframe := &NodeDOM{Type: NodeTypeElement, Element: "div"}
	if !nonIframe.CanExecuteScripts() || !nonIframe.CanSubmitForms() || !nonIframe.CanNavigateTop() {
		t.Fatal("non-iframe must allow scripts/forms/navigation")
	}
	iframe := &NodeDOM{Type: NodeTypeElement, Element: "iframe"}
	if iframe.CanExecuteScripts() || iframe.CanSubmitForms() || iframe.CanNavigateTop() {
		t.Fatal("sandboxed iframe must block scripts/forms/navigation by default")
	}
	iframe.SandboxFlags = SandboxAllowScripts | SandboxAllowForms | SandboxAllowTopNavigation
	if !iframe.CanExecuteScripts() || !iframe.CanSubmitForms() || !iframe.CanNavigateTop() {
		t.Fatal("allowed iframe flags not honored")
	}
}

func TestLoadIframeContent(t *testing.T) {
	notIframe := &NodeDOM{Type: NodeTypeElement, Element: "div"}
	if _, err := notIframe.LoadIframeContent(); err == nil {
		t.Fatal("LoadIframeContent accepted non-iframe")
	}
	noSrc := &NodeDOM{Type: NodeTypeElement, Element: "iframe"}
	if _, err := noSrc.LoadIframeContent(); err == nil {
		t.Fatal("LoadIframeContent accepted iframe without src")
	}
	// Cross-origin without allow-same-origin: empty isolated document.
	plain := &NodeDOM{Type: NodeTypeElement, Element: "iframe", IframeSrc: "https://other.example/"}
	doc, err := plain.LoadIframeContent()
	if err != nil || doc != nil && doc.DOM != nil {
		t.Fatalf("expected empty document, got %+v, %v", doc, err)
	}
	// With allow-same-origin and same document URL.
	same := &NodeDOM{
		Type:         NodeTypeElement,
		Element:      "iframe",
		IframeSrc:    "https://doc.example/frame.html",
		SandboxFlags: SandboxAllowSameOrigin,
	}
	same.Document = &Document{URL: mustParseURL(t, "https://doc.example/")}
	same.WindowCtx = newBareWindowContext(t)
	doc2, err := same.LoadIframeContent()
	if err != nil || doc2 == nil {
		t.Fatalf("same-origin LoadIframeContent = %+v, %v", doc2, err)
	}
	// Bad relative URL fails.
	badURL := &NodeDOM{
		Type:         NodeTypeElement,
		Element:      "iframe",
		IframeSrc:    "https://doc.example/%zz",
		SandboxFlags: SandboxAllowSameOrigin,
	}
	badURL.Document = &Document{URL: mustParseURL(t, "https://doc.example/")}
	badURL.WindowCtx = newBareWindowContext(t)
	if _, err := badURL.LoadIframeContent(); err == nil {
		t.Fatal("LoadIframeContent accepted invalid src")
	}
	// Cross-origin (no allow-same-origin) returns isolated document.
	cross := &NodeDOM{
		Type:      NodeTypeElement,
		Element:   "iframe",
		IframeSrc: "https://other.example/x.html",
	}
	cross.Document = &Document{URL: mustParseURL(t, "https://doc.example/")}
	cross.WindowCtx = newBareWindowContext(t)
	doc3, err := cross.LoadIframeContent()
	if err != nil || doc3 == nil || doc3.DOM != nil {
		t.Fatalf("cross-origin LoadIframeContent = %+v, %v", doc3, err)
	}
}

func TestConvertHTMLNodeToNodeDOMAndClone(t *testing.T) {
	wc := newBareWindowContext(t)
	doc := &Document{URL: mustParseURL(t, "https://conv.example/")}

	nodes, err := parseHTMLFragment(`<div id="root" class="box"><!-- note --><span>hi</span>tail</div>`, doc)
	if err != nil || len(nodes) != 1 {
		t.Fatalf("parseHTMLFragment = %v nodes, %v", nodes, err)
	}
	div := nodes[0]
	if div.Element != "div" || div.Attr("id") != "root" {
		t.Fatalf("div = %+v", div)
	}
	if len(div.Children) != 3 {
		t.Fatalf("div children = %d, want 3 (comment + span + text)", len(div.Children))
	}
	// Comment node type mapping.
	if div.Children[0].Type != NodeTypeComment || div.Children[0].Content != " note " {
		t.Fatalf("comment child = %+v", div.Children[0])
	}
	// Element children with attributes and nested text.
	span := div.Children[1]
	if span.Element != "span" || len(span.Children) != 1 || span.Children[0].Content != "hi" {
		t.Fatalf("span child = %+v", span)
	}
	if div.Children[2].Type != NodeTypeText || div.Children[2].Content != "tail" {
		t.Fatalf("text child = %+v", div.Children[2])
	}

	// CloneNode shallow vs deep.
	shallow := div.CloneNode(false)
	if shallow.Element != "div" || len(shallow.Children) != 0 {
		t.Fatalf("shallow clone = %+v children %d", shallow.Element, len(shallow.Children))
	}
	deep := div.CloneNode(true)
	if len(deep.Children) != 3 || deep.Children[1].Element != "span" {
		t.Fatalf("deep clone children = %d", len(deep.Children))
	}

	// Document creation helpers.
	p := doc.CreateElement("p")
	if p.Element != "p" || p.Document != doc {
		t.Fatalf("CreateElement = %+v", p)
	}
	txn := doc.CreateTextNode("data")
	if txn.Type != NodeTypeText || txn.Content != "data" {
		t.Fatalf("CreateTextNode = %+v", txn)
	}
	_ = wc
}

func TestSetInnerHTMLTextContentAndInsert(t *testing.T) {
	wc := newBareWindowContext(t)
	doc := &Document{URL: mustParseURL(t, "https://mut.example/")}
	doc.DOM = &NodeDOM{Type: NodeTypeElement, Element: "html", Document: doc}
	div := &NodeDOM{Type: NodeTypeElement, Element: "div", Document: doc, WindowCtx: wc}
	oldChild := &NodeDOM{Type: NodeTypeElement, Element: "b", Document: doc, WindowCtx: wc}
	div.appendChildNode(oldChild)
	// Pre-wrap old child in JS so SetInnerHTML cleans the wrapper map.
	_ = wc.nodeWrappers[oldChild]

	if err := div.SetInnerHTML(`<i>inner</i>text`); err != nil {
		t.Fatal(err)
	}
	if len(div.Children) != 2 || div.Children[0].Element != "i" || div.Children[1].Content != "text" {
		t.Fatalf("SetInnerHTML children = %+v", div.Children)
	}
	if _, ok := wc.nodeWrappers[oldChild]; ok {
		t.Fatal("old child wrapper not cleaned up by SetInnerHTML")
	}
	// SetInnerHTML on a text node fails.
	if err := (&NodeDOM{Type: NodeTypeText}).SetInnerHTML("<i></i>"); err == nil {
		t.Fatal("SetInnerHTML accepted text node")
	}

	// SetTextContent on element replaces children with one text node.
	div.SetTextContent("flat")
	if len(div.Children) != 1 || div.Children[0].Type != NodeTypeText || div.Children[0].Content != "flat" {
		t.Fatalf("SetTextContent children = %+v", div.Children)
	}
	// SetTextContent on a text node mutates content.
	txn := &NodeDOM{Type: NodeTypeText, Content: "old", Document: doc}
	txn.SetTextContent("new")
	if txn.Content != "new" || len(txn.Children) != 0 {
		t.Fatalf("text SetTextContent = %+v", txn)
	}

	// InsertBefore: nil new child, invalid reference, append-at-end.
	if err := div.InsertBefore(nil, nil); err == nil {
		t.Fatal("InsertBefore(nil) accepted")
	}
	a := &NodeDOM{Type: NodeTypeElement, Element: "a", Document: doc, WindowCtx: wc}
	b := &NodeDOM{Type: NodeTypeElement, Element: "b", Document: doc, WindowCtx: wc}
	c := &NodeDOM{Type: NodeTypeElement, Element: "c", Document: doc, WindowCtx: wc}
	div.Children = nil
	div.FirstChild = nil
	div.appendChildNode(a)
	div.appendChildNode(b)
	if err := div.InsertBefore(c, nil); err != nil {
		t.Fatal(err)
	}
	if len(div.Children) != 3 || div.Children[2] != c {
		t.Fatalf("append insert = %+v", div.Children)
	}
	d := &NodeDOM{Type: NodeTypeElement, Element: "d", Document: doc, WindowCtx: wc}
	if err := div.InsertBefore(d, a); err != nil {
		t.Fatal(err)
	}
	if div.Children[0] != d || div.Children[1] != a {
		t.Fatalf("insert before a = %+v", div.Children)
	}
	if d.NextSibling != a || a.PrevSibling != d {
		t.Fatal("sibling pointers broken by InsertBefore")
	}
	// InsertBefore with a reference that is not a child is an error.
	notChild := &NodeDOM{Type: NodeTypeElement, Element: "z", Document: doc}
	if err := div.InsertBefore(notChild, notChild); err == nil {
		t.Fatal("InsertBefore accepted non-child reference")
	}
	// InsertBefore moves a node that already has a parent.
	moved := &NodeDOM{Type: NodeTypeElement, Element: "m", Document: doc, WindowCtx: wc}
	div.appendChildNode(moved)
	if err := div.InsertBefore(moved, c); err != nil {
		t.Fatal(err)
	}
	foundMoved := false
	for _, ch := range div.Children {
		if ch == moved {
			foundMoved = true
		}
	}
	if !foundMoved || len(div.Children) != 5 {
		t.Fatalf("moved node not repositioned: %d children", len(div.Children))
	}

	// ReplaceChild: error paths and success.
	if _, err := div.ReplaceChild(nil, a); err == nil {
		t.Fatal("ReplaceChild(nil) accepted")
	}
	e := &NodeDOM{Type: NodeTypeElement, Element: "e", Document: doc, WindowCtx: wc}
	orph := &NodeDOM{Type: NodeTypeElement, Element: "orph", Document: doc, WindowCtx: wc}
	if _, err := div.ReplaceChild(e, orph); err == nil {
		t.Fatal("ReplaceChild accepted foreign oldChild")
	}
	repl := &NodeDOM{Type: NodeTypeElement, Element: "f", Document: doc, WindowCtx: wc}
	old, err := div.ReplaceChild(repl, a)
	if err != nil || old != a {
		t.Fatalf("ReplaceChild = %v, %v", old, err)
	}
	if div.Children[1] != repl || repl.PrevSibling != d {
		t.Fatalf("replace result = %+v", div.Children)
	}
}

func TestQueryHelpersErrors(t *testing.T) {
	doc := &Document{URL: mustParseURL(t, "https://q.example/")}
	rootHTML, err := html.Parse(strings.NewReader(`<html><body><p id="one" class="x">t</p></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	doc.HTMLRoot = rootHTML
	node := &NodeDOM{Type: NodeTypeElement, Element: "div", Document: doc}
	// No HTMLNode set: queries return nil results without error.
	if got, err := node.QuerySelector("#missing"); err != nil || got != nil {
		t.Fatalf("QuerySelector without HTMLNode = %v, %v", got, err)
	}
	if got, err := node.QuerySelectorAll("#missing"); err != nil || len(got) != 0 {
		t.Fatalf("QuerySelectorAll without HTMLNode = %v, %v", got, err)
	}
	// Invalid CSS selectors compile to errors (requires a set HTMLNode so
	// the query reaches the cascadia compile step).
	qNode := &NodeDOM{Type: NodeTypeElement, Element: "div", Document: doc, HTMLNode: rootHTML}
	if _, err := qNode.QuerySelector("p["); err == nil {
		t.Fatal("QuerySelector accepted invalid selector")
	}
	if _, err := qNode.QuerySelectorAll("[bad"); err == nil {
		t.Fatal("QuerySelectorAll accepted invalid selector")
	}
	if _, err := node.GetElementById("x"); err != nil {
		t.Fatal(err)
	}
	if _, err := node.GetElementsByClassName("x"); err != nil {
		t.Fatal(err)
	}
	if _, err := node.GetElementsByTagName("p"); err != nil {
		t.Fatal(err)
	}

	// Cross-origin document is blocked by checkOrigin.
	otherDoc := &Document{URL: mustParseURL(t, "https://other.example/")}
	otherDoc.HTMLRoot = rootHTML
	other := &NodeDOM{Type: NodeTypeElement, Element: "div", Document: otherDoc, WindowCtx: newBareWindowContext(t)}
	_ = other.WindowCtx.SetOrigin("https://q.example")
	if _, err := other.QuerySelector("p"); err == nil {
		t.Fatal("QuerySelector accepted cross-origin document")
	}
}

func TestDocumentHelpers(t *testing.T) {
	doc := &Document{URL: mustParseURL(t, "https://d.example/")}
	if got, err := doc.GetBody(); err != nil || got != nil {
		t.Fatalf("GetBody without DOM = %v, %v", got, err)
	}
	if got, err := doc.GetDocumentElement(); err != nil || got != nil {
		t.Fatalf("GetDocumentElement without DOM = %v, %v", got, err)
	}
	if doc.GetTitle() != "" {
		t.Fatal("GetTitle should be empty without DOM")
	}
	// With DOM and HTMLRoot for selector matching.
	rootHTML, err := html.Parse(strings.NewReader(`<html><body></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	doc.HTMLRoot = rootHTML
	// Link the NodeDOM tree to the parsed html.Node tree so that
	// findByHTMLNode can resolve selector matches back to NodeDOM values.
	// html.Parse inserts a <head> before <body>, so body is html's second child.
	htmlElNode := rootHTML.FirstChild             // <html> element
	bodyNode := htmlElNode.FirstChild.NextSibling // <body>, sibling of <head>
	root := &NodeDOM{Type: NodeTypeElement, Element: "html", Document: doc, HTMLNode: htmlElNode}
	b := &NodeDOM{Type: NodeTypeElement, Element: "body", Document: doc, HTMLNode: bodyNode}
	root.appendChildNode(b)
	doc.DOM = root
	got, err := doc.GetBody()
	if err != nil || got == nil {
		t.Fatalf("GetBody = %v, %v", got, err)
	}
	el, err := doc.GetDocumentElement()
	if err != nil || el != root {
		t.Fatalf("GetDocumentElement = %v, %v", el, err)
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
