package ketchup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	hotdog "github.com/danfragoso/thdwb/hotdog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixturesDir points at the repository-level test fixtures used by the whole
// browser test-suite.
const fixturesDir = "../test-fixtures"

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixturesDir, name))
	require.NoError(t, err, "failed to load fixture %s", name)
	return string(data)
}

// findByElement walks the DOM tree and returns the first node with the given
// element name.
func findByElement(root *hotdog.NodeDOM, element string) *hotdog.NodeDOM {
	if root == nil {
		return nil
	}
	if root.Element == element {
		return root
	}
	for _, child := range root.Children {
		if found := findByElement(child, element); found != nil {
			return found
		}
	}
	return nil
}

// firstTextChild returns the content of the first text-node child of node,
// or "" if there is none.
func firstTextChild(node *hotdog.NodeDOM) string {
	if node == nil {
		return ""
	}
	for _, child := range node.Children {
		if child.Type == hotdog.NodeTypeText {
			return child.Content
		}
	}
	return ""
}

// countByElement counts all nodes in the tree with the given element name.
func countByElement(root *hotdog.NodeDOM, element string) int {
	count := 0
	for _, node := range flattenTree(root) {
		if node.Element == element {
			count++
		}
	}
	return count
}

// flattenTree returns every node in the subtree rooted at root, pre-order.
func flattenTree(root *hotdog.NodeDOM) []*hotdog.NodeDOM {
	var out []*hotdog.NodeDOM
	var walk func(n *hotdog.NodeDOM)
	walk = func(n *hotdog.NodeDOM) {
		if n == nil {
			return
		}
		out = append(out, n)
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(root)
	return out
}

func TestParseHTMLDocument_SimpleFixture(t *testing.T) {
	doc := ParseHTMLDocument(loadFixture(t, "simple.html"), nil)
	require.NotNil(t, doc)
	require.NotNil(t, doc.DOM)

	// html.Parse returns a document node at the root with the html element
	// as its first element child.
	htmlEl := findByElement(doc.DOM, "html")
	require.NotNil(t, htmlEl)

	// html has head and body children.
	head, err := htmlEl.FindChildByName("head")
	require.NoError(t, err)
	body, err := htmlEl.FindChildByName("body")
	require.NoError(t, err)

	// h1 and p exist inside body.
	require.NotNil(t, findByElement(body, "h1"), "expected an h1 element")
	assert.Equal(t, 1, countByElement(body, "p"))

	// Title text is preserved on the child text node.
	title := findByElement(head, "title")
	require.NotNil(t, title)
	assert.Equal(t, "Simple Page", strings.TrimSpace(firstTextChild(title)))
}

func TestParseHTMLDocument_StructureFlags(t *testing.T) {
	doc := ParseHTMLDocument("<html><body><div id=\"x\">hi</div></body></html>", nil)
	require.NotNil(t, doc.DOM)

	div := findByElement(doc.DOM, "div")
	require.NotNil(t, div)
	assert.Equal(t, "x", div.Attr("id"))
	assert.NotNil(t, div.Style, "element should have a default stylesheet")
	assert.NotNil(t, div.RenderBox)
	assert.True(t, div.NeedsReflow)
	assert.True(t, div.NeedsRepaint)
	assert.Equal(t, doc, div.Document)
	// Parent linkage is set.
	assert.Equal(t, "body", div.Parent.Element)
}

func TestParseHTMLDocument_TextNodesAndEntities(t *testing.T) {
	src := `<html><body><p>A &amp; B &lt;tag&gt; &#x2603; snow</p></body></html>`
	doc := ParseHTMLDocument(src, nil)
	require.NotNil(t, doc.DOM)

	p := findByElement(doc.DOM, "p")
	require.NotNil(t, p)
	// The decoded text lives on the child text node; golang.org/x/net/html
	// decodes entities (&amp; -> &, &lt; -> <, &#x2603; -> snowman).
	var text string
	for _, child := range p.Children {
		if child.Type == hotdog.NodeTypeText {
			text += child.Content
		}
	}
	assert.Contains(t, text, "A & B")
	assert.Contains(t, text, "<tag>")
	assert.Contains(t, text, "snow")
}

func TestParseHTMLDocument_StyleElementExtracted(t *testing.T) {
	src := `<html><head><style>h1 { color: #cc3300; }</style></head><body><h1>x</h1></body></html>`
	doc := ParseHTMLDocument(src, nil)
	require.NotNil(t, doc.DOM)

	// The <style> content becomes a parsed StyleElement on the document.
	require.NotEmpty(t, doc.StyleSheets, "style element should be parsed into the document")
	assert.Equal(t, "h1", doc.StyleSheets[0].Selector)
	require.NotNil(t, doc.StyleSheets[0].Style)
	require.NotNil(t, doc.StyleSheets[0].Style.Color)
	// Color channels are float 0-1 (regression: no integer truncation).
	assert.InDelta(t, 204/255.0, doc.StyleSheets[0].Style.Color.R, 1e-6)
	assert.InDelta(t, 51/255.0, doc.StyleSheets[0].Style.Color.G, 1e-6)
	assert.InDelta(t, 0.0, doc.StyleSheets[0].Style.Color.B, 1e-6)

	// The h1 element should have the stylesheet color merged in.
	h1 := findByElement(doc.DOM, "h1")
	require.NotNil(t, h1)
	require.NotNil(t, h1.Style)
	require.NotNil(t, h1.Style.Color)
	assert.InDelta(t, 204/255.0, h1.Style.Color.R, 1e-6)
}

func TestParseHTMLDocument_IFrameAttributes(t *testing.T) {
	src := `<html><body><iframe src="http://example.com" sandbox="allow-scripts"></iframe></body></html>`
	doc := ParseHTMLDocument(src, nil)
	require.NotNil(t, doc.DOM)

	iframe := findByElement(doc.DOM, "iframe")
	require.NotNil(t, iframe)
	assert.Equal(t, "http://example.com", iframe.IframeSrc)
	assert.True(t, iframe.SandboxFlags.HasSandboxFlag(hotdog.SandboxAllowScripts),
		"allow-scripts sandbox flag should be set")
}

func TestParseHTMLDocument_CommentAndDoctype(t *testing.T) {
	src := `<!DOCTYPE html><html><body><!-- a comment -->text</body></html>`
	doc := ParseHTMLDocument(src, nil)
	require.NotNil(t, doc.DOM)

	nodes := flattenTree(doc.DOM)
	var sawComment, sawDoctype, sawText bool
	for _, n := range nodes {
		switch n.Type {
		case hotdog.NodeTypeComment:
			sawComment = true
			assert.Equal(t, " a comment ", n.Content)
		case hotdog.NodeTypeDoctype:
			sawDoctype = true
		case hotdog.NodeTypeText:
			sawText = true
		}
	}
	assert.True(t, sawDoctype, "expected a doctype node")
	assert.True(t, sawComment, "expected a comment node")
	assert.True(t, sawText, "expected a text node")
}

func TestParseHTMLDocument_CSSFixture(t *testing.T) {
	doc := ParseHTMLDocument(loadFixture(t, "css.html"), nil)
	require.NotNil(t, doc.DOM)
	require.NotEmpty(t, doc.StyleSheets, "css.html should produce stylesheets")

	// At least one rule should target the .highlight class.
	var sawHighlight bool
	for _, se := range doc.StyleSheets {
		if se.Selector == ".highlight" {
			sawHighlight = true
			require.NotNil(t, se.Style.BackgroundColor)
			assert.InDelta(t, 255/255.0, se.Style.BackgroundColor.R, 1e-6)
		}
	}
	assert.True(t, sawHighlight, "expected a .highlight rule in the parsed stylesheets")
}

func TestParseHTMLDocument_MalformedRecovery(t *testing.T) {
	// The edge-cases fixture is deliberately broken HTML (unclosed tags,
	// stray attributes, unterminated scripts). The parser must not panic and
	// must still produce a usable DOM tree with a body.
	doc := ParseHTMLDocument(loadFixture(t, "edge-cases.html"), nil)
	require.NotNil(t, doc)
	require.NotNil(t, doc.DOM)
	_, err := doc.DOM.FindChildByName("body")
	assert.NoError(t, err, "malformed document should still expose a body element")
}

func TestParsePlainText(t *testing.T) {
	src := "line one\nline two\n\nline four"
	doc := ParsePlainText(src, nil)
	require.NotNil(t, doc)
	assert.Equal(t, "Text Document", doc.Title)
	assert.Equal(t, src, doc.RawDocument)

	body, err := doc.DOM.FindChildByName("body")
	require.NoError(t, err)
	// Four input lines -> four <p> elements.
	assert.Equal(t, 4, len(body.Children))
	assert.Equal(t, "p", body.Children[0].Element)
	assert.Equal(t, "line one", body.Children[0].Content)
	assert.Equal(t, "line four", body.Children[3].Content)

	// head + body are the two direct children of the html root.
	require.Len(t, doc.DOM.Children, 2)
	assert.Equal(t, "head", doc.DOM.Children[0].Element)
	assert.Equal(t, "body", doc.DOM.Children[1].Element)
}

func TestParsePlainText_Empty(t *testing.T) {
	doc := ParsePlainText("", nil)
	require.NotNil(t, doc)
	body, err := doc.DOM.FindChildByName("body")
	require.NoError(t, err)
	// An empty string still splits into a single empty line.
	require.Len(t, body.Children, 1)
	assert.Equal(t, "", body.Children[0].Content)
}

func TestCreateHTMLBase(t *testing.T) {
	root := CreateHTMLBase(nil)
	require.NotNil(t, root)
	assert.Equal(t, "html", root.Element)
	assert.Equal(t, hotdog.NodeTypeElement, root.Type)
	require.Len(t, root.Children, 2)

	head := root.Children[0]
	body := root.Children[1]
	assert.Equal(t, "head", head.Element)
	assert.Equal(t, "body", body.Element)
	assert.Equal(t, root, head.Parent)
	assert.Equal(t, root, body.Parent)
	assert.NotNil(t, head.Style)
	assert.NotNil(t, body.Style)
	assert.NotNil(t, root.RenderBox)
	assert.True(t, root.NeedsReflow)
	assert.True(t, root.NeedsRepaint)
}

func TestParseHTML_CreatesBaseFromEmpty(t *testing.T) {
	// No tags at all: DOM stays nil until a tag is found, so an empty string
	// yields an empty document.
	doc := ParseHTML("", nil)
	require.NotNil(t, doc)
	assert.Equal(t, "", doc.RawDocument)
}

func TestParseHTML_BuildsTree(t *testing.T) {
	src := `<html><body><div class="a"><p>hello</p></div></body></html>`
	doc := ParseHTML(src, nil)
	require.NotNil(t, doc)
	require.NotNil(t, doc.DOM)

	// Walk to the nested p.
	body := findByElement(doc.DOM, "body")
	require.NotNil(t, body)
	div := findByElement(body, "div")
	require.NotNil(t, div)
	assert.Equal(t, "a", div.Attr("class"))
	p := findByElement(div, "p")
	require.NotNil(t, p)
	assert.Equal(t, "hello", p.Content)
}

func TestParseHTML_VoidElements(t *testing.T) {
	// Void elements (br, img) must not swallow subsequent siblings.
	src := `<html><body><p>a<br>b</p><img src="x.png"><p>c</p></body></html>`
	doc := ParseHTML(src, nil)
	require.NotNil(t, doc)
	require.NotNil(t, doc.DOM)

	body := findByElement(doc.DOM, "body")
	require.NotNil(t, body)
	assert.Equal(t, 2, countByElement(body, "p"))
	require.NotNil(t, findByElement(body, "img"))
}

func TestIsVoidElement(t *testing.T) {
	for _, tag := range []string{"br", "img", "hr", "input", "meta", "link"} {
		assert.True(t, isVoidElement(tag), "expected %s to be a void element", tag)
	}
	for _, tag := range []string{"div", "p", "span", "a"} {
		assert.False(t, isVoidElement(tag), "expected %s not to be a void element", tag)
	}
}

func TestExtractAttributes(t *testing.T) {
	attrs := extractAttributes(`class="foo" id="bar" data-x="1"`)
	require.Len(t, attrs, 3)
	assert.Equal(t, "class", attrs[0].Name)
	assert.Equal(t, "foo", attrs[0].Value)
	assert.Equal(t, "id", attrs[1].Name)
	assert.Equal(t, "bar", attrs[1].Value)

	// No attributes.
	assert.Empty(t, extractAttributes(`plain`))
}

func TestBuildNodeDOMFromHTML_RawText(t *testing.T) {
	// buildNodeDOMFromHTML is exercised via the exported parsers above, but
	// verify the text-node shape directly with a synthetic html.Node tree by
	// parsing a small document and inspecting a text leaf.
	doc := ParseHTMLDocument(`<html><body><span>hi</span></body></html>`, nil)
	span := findByElement(doc.DOM, "span")
	require.NotNil(t, span)
	textNode := span.Children[0]
	assert.Equal(t, hotdog.NodeTypeText, textNode.Type)
	assert.Equal(t, "html:text", textNode.Element)
	assert.Equal(t, "hi", textNode.Content)
	assert.Equal(t, span, textNode.Parent)
}
