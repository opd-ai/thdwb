package shell

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danfragoso/thdwb/bun"
	gg "github.com/danfragoso/thdwb/gg"
	hotdog "github.com/danfragoso/thdwb/hotdog"
)

// loadTabHTML is a minimal page that exercises both style sources bun needs:
// an inline background-color on a block element and a <style> rule.
const loadTabHTML = `<!DOCTYPE html><html><head><title>load tab</title></head><body>
<h1>Load Tab</h1>
<div id="box" style="background-color: rgb(255, 0, 0);">Panel content that gives the block a measurable height.</div>
<style>#box { margin: 0; }</style>
</body></html>`

// TestLoadTabContentKetchupPipeline verifies that a URL load wires the ketchup
// DOM into the active document so that querySelector and bun rendering work.
//
// This is the regression test for the render pipeline: loadTabContent must
// produce an active document whose DOM is bun-renderable (RenderBox + Style
// populated) rather than an x/net fragment DOM, and bun must paint the styled
// content to the canvas.
func TestLoadTabContentKetchupPipeline(t *testing.T) {
	srv := newHTMLTestServer(t, loadTabHTML)

	bs := NewBrowserShell(nil, nil)
	defer bs.Close()

	tab := bs.TabManager().Add("load tab", srv.URL+"/", "", "", "")
	if tab == nil {
		t.Fatal("expected a new tab")
	}
	bs.loadTabContent(tab)

	doc := activeDocument(t, bs, tab)
	if doc.HTMLRoot == nil {
		t.Error("expected ketchup parsing to set HTMLRoot")
	}
	if len(doc.StyleSheets) == 0 {
		t.Error("expected <style> to be parsed into StyleSheets")
	}

	// querySelector must resolve against the ketchup DOM.
	h1, err := doc.DOM.QuerySelector("h1")
	if err != nil || h1 == nil {
		t.Fatalf(`QuerySelector("h1") = (%v, %v), want a match`, h1, err)
	}
	if got := subtreeText(h1); !strings.Contains(got, "Load Tab") {
		t.Errorf(`QuerySelector("h1") subtree text = %q, want to contain "Load Tab"`, got)
	}
	box, err := doc.DOM.QuerySelector("#box")
	if err != nil || box == nil {
		t.Fatalf(`QuerySelector("#box") = (%v, %v), want a match`, box, err)
	}

	// bun must lay out and paint the document, and the CSS-declared red
	// background of #box must reach the canvas (a deterministic, style-driven
	// signal rather than font-dependent text pixels).
	if got := countPixels(t, doc, isRed); got == 0 {
		t.Error("expected bun to paint the red background of #box")
	}
}

// TestLoadTabContentQuerySelector verifies the CSS selector path against the
// ketchup DOM built during a URL load.
func TestLoadTabContentQuerySelector(t *testing.T) {
	srv := newHTMLTestServer(t, loadTabHTML)

	bs := NewBrowserShell(nil, nil)
	defer bs.Close()

	tab := bs.TabManager().Add("load tab", srv.URL+"/", "", "", "")
	bs.loadTabContent(tab)

	doc := activeDocument(t, bs, tab)

	for _, sel := range []string{"#box", "div", "body"} {
		node, err := doc.DOM.QuerySelector(sel)
		if err != nil || node == nil {
			t.Errorf(`QuerySelector(%q) = (%v, %v), want a match`, sel, node, err)
		}
	}

	all, err := doc.DOM.QuerySelectorAll("div")
	if err != nil {
		t.Fatalf(`QuerySelectorAll("div") error: %v`, err)
	}
	if len(all) != 1 {
		t.Errorf(`QuerySelectorAll("div") returned %d nodes, want 1`, len(all))
	}
}

// activeDocument fetches the tab's active document, failing the test if absent.
func activeDocument(t *testing.T, bs *BrowserShell, tab *Tab) *hotdog.Document {
	t.Helper()
	wc := bs.tabWindowContext(tab)
	if wc == nil {
		t.Fatal("expected a window context for the tab")
	}
	if wc.ActiveDocument == nil {
		t.Fatal("expected an active document after load")
	}
	return wc.ActiveDocument
}

// countPixels renders doc with bun into a canvas and returns the number of
// pixels satisfying match, sampled on a 2px grid to keep the loop fast.
func countPixels(t *testing.T, doc *hotdog.Document, match func(r, g, b uint32) bool) int {
	t.Helper()
	ctx := gg.NewContext(1024, 768)
	if err := bun.RenderDocument(ctx, doc, false); err != nil {
		t.Fatalf("bun.RenderDocument: %v", err)
	}
	img := ctx.Image()
	bounds := img.Bounds()
	n := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 2 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 2 {
			r, g, b, _ := img.At(x, y).RGBA()
			if match(r, g, b) {
				n++
			}
		}
	}
	return n
}

// isRed matches a pixel whose red channel dominates, i.e. a pixel from the
// #box element's rgb(255, 0, 0) background.
func isRed(r, g, b uint32) bool {
	return r > g && r > b && r > 0x8000
}

// subtreeText returns the concatenated text of node and its descendants. The
// ketchup DOM stores text in child text nodes, so an element's own Content is
// empty; gathering the subtree is how the test reads the rendered text.
func subtreeText(n *hotdog.NodeDOM) string {
	var sb strings.Builder
	var walk func(node *hotdog.NodeDOM)
	walk = func(node *hotdog.NodeDOM) {
		if node == nil {
			return
		}
		if node.Type == hotdog.NodeTypeText && node.Content != "" {
			sb.WriteString(node.Content)
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(n)
	return sb.String()
}

// newHTMLTestServer serves html with a text/html content type and registers a
// cleanup with the test.
func newHTMLTestServer(t *testing.T, html string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	}))
	t.Cleanup(srv.Close)
	return srv
}
