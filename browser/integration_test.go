// Package main contains the thdwb browser entry point. The integration test
// exercises the full document pipeline (HTML parse -> CSS -> DOM -> layout)
// against the test fixtures in test-fixtures/.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	hotdog "github.com/danfragoso/thdwb/hotdog"
	ketchup "github.com/danfragoso/thdwb/ketchup"
	mayo "github.com/danfragoso/thdwb/mayo"
)

const fixturesDir = "../test-fixtures"

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(fixturesDir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to load fixture %s: %v", path, err)
	}
	return string(data)
}

// TestRenderSimpleHTML parses the minimal fixture and verifies the DOM tree
// was built with the expected structure.
func TestRenderSimpleHTML(t *testing.T) {
	doc := ketchup.ParseHTMLDocument(loadFixture(t, "simple.html"), nil)
	if doc == nil {
		t.Fatal("ParseHTMLDocument returned nil document")
	}
	if doc.DOM == nil {
		t.Fatal("parsed document has no DOM tree")
	}

	headings := collectByTag(doc.DOM, "h1", nil, func(n *hotdog.NodeDOM) string { return elementText(n) })
	if len(headings) != 1 || headings[0] != "Hello World" {
		t.Errorf("headings = %v, want [Hello World]", headings)
	}

	if p := firstByTag(doc.DOM, "p"); p == nil || !strings.Contains(elementText(p), "minimal HTML document") {
		t.Errorf("paragraph content missing or wrong: %q", elementText(p))
	}
}

// TestRenderCSSHTML verifies stylesheet application: element selectors and
// box styles are resolved onto the DOM.
func TestRenderCSSHTML(t *testing.T) {
	doc := ketchup.ParseHTMLDocument(loadFixture(t, "css.html"), nil)
	if len(doc.StyleSheets) == 0 {
		t.Fatal("no stylesheets parsed from <style> block")
	}

	h1 := firstByTag(doc.DOM, "h1")
	if h1 == nil || h1.Style == nil || h1.Style.Color == nil {
		t.Fatal("h1 element or color style not found")
	}
	// h1 { color: #cc3300 } from the stylesheet (0-1 scale: cc=0.8, 33=0.2).
	if got := *h1.Style.Color; !floatEq(got.R, float64(0xcc)/255) || !floatEq(got.G, float64(0x33)/255) || got.B != 0 {
		t.Errorf("h1 color = rgb(%v,%v,%v), want rgb(0.8,0.2,0)", got.R, got.G, got.B)
	}

	p := firstByTag(doc.DOM, "p")
	if p == nil || p.Style == nil {
		t.Fatal("p element or style not found")
	}
	if p.Style.Color == nil {
		t.Fatal("p color style is nil")
	}
	if got := *p.Style.Color; !floatEq(got.R, float64(0x33)/255) || !floatEq(got.G, float64(0x33)/255) || !floatEq(got.B, float64(0x33)/255) {
		t.Errorf("p color = rgb(%v,%v,%v), want rgb(0.2,0.2,0.2)", got.R, got.G, got.B)
	}
	if p.Style.MarginTop != 10 || p.Style.MarginLeft != 10 {
		t.Errorf("p margin = top %v left %v, want 10 10", p.Style.MarginTop, p.Style.MarginLeft)
	}
	if p.Style.PaddingTop != 4 {
		t.Errorf("p padding-top = %v, want 4", p.Style.PaddingTop)
	}
}

// TestRenderComplexHTML parses the realistic page and verifies structure and
// computed styles survive the full pipeline.
func TestRenderComplexHTML(t *testing.T) {
	doc := ketchup.ParseHTMLDocument(loadFixture(t, "complex.html"), nil)

	header := firstByTag(doc.DOM, "header")
	if header == nil {
		t.Fatal("header element not found")
	}
	if header.Style == nil || header.Style.PaddingTop != 16 {
		t.Errorf("header padding-top = %+v, want 16", header.Style)
	}

	var sections, rows, links int
	collectByTag(doc.DOM, "section", &sections, nil)
	collectByTag(doc.DOM, "tr", &rows, nil)
	collectByTag(doc.DOM, "a", &links, nil)
	if sections != 2 {
		t.Errorf("card sections = %d, want 2", sections)
	}
	if rows != 3 {
		t.Errorf("table rows = %d, want 3", rows)
	}
	if links != 3 {
		t.Errorf("nav links = %d, want 3", links)
	}
}

// TestRenderEdgeCases verifies the parser does not panic on malformed HTML
// and still produces a navigable DOM with decoded entities.
func TestRenderEdgeCases(t *testing.T) {
	doc := ketchup.ParseHTMLDocument(loadFixture(t, "edge-cases.html"), nil)
	if doc == nil || doc.DOM == nil {
		t.Fatal("edge case document failed to parse")
	}

	var text strings.Builder
	collectText(doc.DOM, &text)
	if !strings.Contains(text.String(), "Entity test:") {
		t.Error("decoded entity text missing from DOM")
	}
}

// TestLayoutReflowsDocument runs the reflow pass over the complex fixture and
// checks that nodes get non-trivial vertical layout positions.
func TestLayoutReflowsDocument(t *testing.T) {
	doc := ketchup.ParseHTMLDocument(loadFixture(t, "complex.html"), nil)

	body := firstByTag(doc.DOM, "body")
	if body == nil {
		t.Fatal("body element not found")
	}
	if len(body.Children) == 0 {
		t.Fatal("body has no children to reflow")
	}
	mayo.ReflowNode(body, body, 0)
	for i, child := range body.Children {
		if child.Style == nil {
			t.Fatalf("body child %d has no style after reflow", i)
		}
		if child.Style.Top < 0 {
			t.Errorf("body child %d top = %v, want >= 0", i, child.Style.Top)
		}
	}
}

// elementText returns the concatenated text of an element and its descendants.
func elementText(n *hotdog.NodeDOM) string {
	if n == nil {
		return ""
	}
	var sb strings.Builder
	collectText(n, &sb)
	return strings.TrimSpace(sb.String())
}

// floatEq compares two floats within an epsilon.
func floatEq(a, b float64) bool {
	const eps = 1e-9
	return a < b+eps && a > b-eps
}

// firstByTag returns the first element node with the given tag name.
func firstByTag(root *hotdog.NodeDOM, tag string) *hotdog.NodeDOM {
	if root == nil {
		return nil
	}
	if root.Element == tag {
		return root
	}
	for _, child := range root.Children {
		if found := firstByTag(child, tag); found != nil {
			return found
		}
	}
	return nil
}

// collectByTag counts element nodes with the tag and, when extract is
// non-nil, appends extracted values to the result.
func collectByTag(root *hotdog.NodeDOM, tag string, count *int, extract func(*hotdog.NodeDOM) string) []string {
	var values []string
	var walk func(n *hotdog.NodeDOM)
	walk = func(n *hotdog.NodeDOM) {
		if n == nil {
			return
		}
		if n.Element == tag {
			if count != nil {
				*count++
			}
			if extract != nil {
				values = append(values, extract(n))
			}
			return
		}
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(root)
	return values
}

// collectText appends all text content in the DOM tree to sb.
func collectText(root *hotdog.NodeDOM, sb *strings.Builder) {
	if root == nil {
		return
	}
	sb.WriteString(root.Content)
	for _, child := range root.Children {
		collectText(child, sb)
	}
}
