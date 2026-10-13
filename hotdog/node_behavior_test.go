package hotdog

import (
	"net/url"
	"strings"
	"testing"
)

func TestNodeDebugAndXPathHelpers(t *testing.T) {
	root := &NodeDOM{Element: "html"}
	body := &NodeDOM{Element: "body", Parent: root}
	first := &NodeDOM{Element: "p", Parent: body}
	second := &NodeDOM{Element: "p", Parent: body}
	root.Children = []*NodeDOM{body}
	body.Children = []*NodeDOM{first, second}
	if got := second.GetXPath(); got != "/html/body[1]/p[2]" {
		t.Fatalf("GetXPath() = %q", got)
	}
	if found, err := root.FindByXPath("/html/body[1]/p[2]"); err != nil || found != second {
		t.Fatalf("FindByXPath() = %p, %v", found, err)
	}
	if _, err := root.FindByXPath("/missing"); err == nil {
		t.Fatal("FindByXPath accepted a missing path")
	}
	if found, err := root.FindChildByName("body"); err != nil || found != body {
		t.Fatalf("FindChildByName() = %p, %v", found, err)
	}
	root.Print(0)
	if got := root.JSON(); got == "{}" || !strings.Contains(got, `"element": "html"`) {
		t.Fatalf("JSON() = %s", got)
	}
}

func TestNodeHTMLSerialization(t *testing.T) {
	cases := []struct {
		node *NodeDOM
		want string
	}{
		{&NodeDOM{Type: NodeTypeText, Content: "<&>"}, "<&>"},
		{&NodeDOM{Type: NodeTypeComment, Content: "note"}, "<!--note-->"},
		{&NodeDOM{Type: NodeTypeDoctype, Content: "html"}, "<!DOCTYPE html>"},
		{&NodeDOM{Type: NodeTypeRaw, Content: "<raw>"}, "<raw>"},
		{&NodeDOM{Type: NodeType(99)}, ""},
	}
	for _, tc := range cases {
		if got := tc.node.toHTML(); got != tc.want {
			t.Errorf("toHTML() = %q, want %q", got, tc.want)
		}
	}
	element := &NodeDOM{Type: NodeTypeElement, Element: "p", Attributes: []*Attribute{{Name: "id", Value: "x"}}}
	element.Children = []*NodeDOM{{Type: NodeTypeText, Content: "hello"}}
	if got := element.InnerHTML(); got != "hello" {
		t.Fatalf("InnerHTML() = %q", got)
	}
}

func TestNodeSelectorConvenienceMethods(t *testing.T) {
	wc := NewWindowContext(&Settings{}, nil, nil)
	defer wc.Destroy()
	if err := wc.SetOrigin("https://example.com"); err != nil {
		t.Fatal(err)
	}
	doc, err := parseHTMLForTest(`<html><body><p id="item:1" class="entry"></p><p class="entry"></p></body></html>`, wc)
	if err != nil {
		t.Fatal(err)
	}
	first, err := doc.DOM.GetElementById("item:1")
	if err != nil || first == nil || first.Attr("id") != "item:1" {
		t.Fatalf("GetElementById() = %p, %v", first, err)
	}
	classNodes, err := doc.DOM.GetElementsByClassName("entry")
	if err != nil || len(classNodes) != 2 {
		t.Fatalf("GetElementsByClassName() returned %d nodes, %v", len(classNodes), err)
	}
	tagNodes, err := doc.DOM.GetElementsByTagName("p")
	if err != nil || len(tagNodes) != 2 {
		t.Fatalf("GetElementsByTagName() returned %d nodes, %v", len(tagNodes), err)
	}
}

func TestSandboxFlagsAndIframePolicies(t *testing.T) {
	flags := ParseSandboxAttribute("allow-scripts allow-same-origin allow-forms")
	for _, flag := range []SandboxFlags{SandboxAllowScripts, SandboxAllowSameOrigin, SandboxAllowForms} {
		if !flags.HasSandboxFlag(flag) {
			t.Errorf("sandbox flag %d was not parsed", flag)
		}
	}
	if ParseSandboxAttribute("").HasSandboxFlag(SandboxAllowScripts) {
		t.Fatal("empty sandbox unexpectedly allowed scripts")
	}
	if !(&NodeDOM{Element: "div"}).CanExecuteScripts() {
		t.Fatal("non-iframe should allow scripts")
	}
	if (&NodeDOM{Element: "iframe"}).CanSubmitForms() || (&NodeDOM{Element: "iframe"}).CanNavigateTop() {
		t.Fatal("unsandboxed iframe flags should remain restricted")
	}
	if !(&NodeDOM{Element: "iframe", SandboxFlags: SandboxAllowForms | SandboxAllowTopNavigation}).CanSubmitForms() {
		t.Fatal("allow-forms flag was not honored")
	}
}

func TestLoadIframeContentSandboxAndOriginBranches(t *testing.T) {
	iframe := &NodeDOM{Element: "div"}
	if _, err := iframe.LoadIframeContent(); err == nil {
		t.Fatal("non-iframe node was accepted")
	}
	iframe.Element = "iframe"
	if _, err := iframe.LoadIframeContent(); err == nil {
		t.Fatal("iframe without src was accepted")
	}
	iframe.IframeSrc = "https://example.com/frame"
	iframe.SandboxFlags = SandboxNone
	iframeDoc, err := iframe.LoadIframeContent()
	if err != nil || iframeDoc == nil {
		t.Fatalf("sandboxed iframe load = %v, %v", iframeDoc, err)
	}
	origin, _ := url.Parse("https://example.com")
	wc := NewWindowContext(&Settings{}, nil, nil)
	defer wc.Destroy()
	wc.Origin = origin
	iframe.Document = &Document{URL: origin}
	iframe.WindowCtx = wc
	iframe.SandboxFlags = SandboxAllowSameOrigin
	iframeDoc, err = iframe.LoadIframeContent()
	if err != nil || iframeDoc == nil {
		t.Fatalf("same-origin iframe load = %v, %v", iframeDoc, err)
	}
}

func TestWhitespaceNormalizationAndDocumentHelpers(t *testing.T) {
	if got := CollapseWhitespace("  one\t two\n three  "); got != "one two three" {
		t.Fatalf("CollapseWhitespace() = %q", got)
	}
	if got := CollapseWhitespacePreserveNewlines(" one\t two \n three "); got != "one two\nthree" {
		t.Fatalf("CollapseWhitespacePreserveNewlines() = %q", got)
	}
	text := &NodeDOM{Type: NodeTypeText, Content: "  one\t two\n three  "}
	if got := text.GetTextContent(); got != "one two three" {
		t.Fatalf("GetTextContent() = %q", got)
	}
	doc := &Document{Title: "Example"}
	if doc.GetTitle() != "Example" {
		t.Fatal("GetTitle() did not return the document title")
	}
	if element := doc.CreateElement("main"); element.Element != "main" {
		t.Fatalf("CreateElement() = %#v", element)
	}
	if node := doc.CreateTextNode("text"); node.Content != "text" || node.Type != NodeTypeText {
		t.Fatalf("CreateTextNode() = %#v", node)
	}
}

func TestDocumentElementAndBodyHelpers(t *testing.T) {
	doc := &Document{}
	if body, err := doc.GetBody(); err != nil || body != nil {
		t.Fatalf("GetBody() on empty document = %v, %v", body, err)
	}
	if element, err := doc.GetDocumentElement(); err != nil || element != nil {
		t.Fatalf("GetDocumentElement() on empty document = %v, %v", element, err)
	}
	wc := NewWindowContext(&Settings{}, nil, nil)
	defer wc.Destroy()
	doc, err := parseHTMLForTest(`<html><body><p>ok</p></body></html>`, wc)
	if err != nil {
		t.Fatal(err)
	}
	if body, err := doc.GetBody(); err != nil || body == nil || body.Element != "body" {
		t.Fatalf("GetBody() = %v, %v", body, err)
	}
	if element, err := doc.GetDocumentElement(); err != nil || element == nil || element.Element != "html" {
		t.Fatalf("GetDocumentElement() = %v, %v", element, err)
	}
}

func TestClearDirtyAndNoOpNodeMutations(t *testing.T) {
	child := &NodeDOM{Dirty: true}
	root := &NodeDOM{Dirty: true, Children: []*NodeDOM{child}}
	root.ClearDirty()
	if root.Dirty || child.Dirty {
		t.Fatal("ClearDirty() left a node dirty")
	}
	if err := (&NodeDOM{Type: NodeTypeText}).SetInnerHTML("<b>bad</b>"); err == nil {
		t.Fatal("SetInnerHTML accepted a text node")
	}
	if _, err := root.ReplaceChild(child, &NodeDOM{}); err == nil {
		t.Fatal("ReplaceChild accepted a missing old child")
	}
}
