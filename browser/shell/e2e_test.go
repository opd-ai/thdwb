package shell

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// e2eHTML is a fixture page for the end-to-end test. It carries a <style> rule
// (blue background) so the render assertion can check a CSS-driven fill that
// the standard layout path paints deterministically.
const e2eHTML = `<!DOCTYPE html><html><head><title>e2e</title>
<style>#panel { background-color: blue; }</style>
</head><body>
<h1>End to End</h1>
<div id="panel">Panel content that gives the block a measurable height.</div>
</body></html>`

// TestEndToEnd_LoadURL_Renders is the true end-to-end invariant the plan calls
// out: launch the engine headless, LoadURL a fixture server on 127.0.0.1, and
// assert both the DOM (querySelector) and the rendered pixels. This is the test
// that would have caught the render-pipeline disconnect (findings 1-6): a
// document that fetches and parses but never paints would fail the pixel check.
func TestEndToEnd_LoadURL_Renders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(e2eHTML))
	}))
	defer srv.Close()

	bs := NewBrowserShell(nil, nil)
	defer bs.Close()

	tab := bs.TabManager().Add("e2e", srv.URL+"/", "", "", "")
	if err := bs.loadTabContent(tab); err != nil {
		t.Fatalf("loadTabContent: %v", err)
	}

	doc := activeDocument(t, bs, tab)

	// DOM: the ketchup DOM resolves selectors.
	h1, err := doc.DOM.QuerySelector("h1")
	if err != nil || h1 == nil {
		t.Fatalf(`QuerySelector("h1") = (%v, %v), want a match`, h1, err)
	}
	if got := subtreeText(h1); !strings.Contains(got, "End to End") {
		t.Errorf(`h1 subtree text = %q, want to contain "End to End"`, got)
	}

	// Render: the CSS blue background of #panel must reach the canvas.
	if got := countPixels(t, doc, isBlue); got == 0 {
		t.Error("expected the engine to paint the blue background of #panel end-to-end")
	}
}

// TestEndToEnd_RealHost is the opt-in real-host variant: it fetches a live page
// and asserts the DOM + pixels. It is gated on THDWB_NETWORK=1 so the default
// CI run stays network-free and fast, matching the item-2 opt-in tests.
func TestEndToEnd_RealHost(t *testing.T) {
	if os.Getenv("THDWB_NETWORK") != "1" {
		t.Skip("set THDWB_NETWORK=1 to run the real-host end-to-end test")
	}

	bs := NewBrowserShell(nil, nil)
	defer bs.Close()

	tab := bs.TabManager().Add("e2e real", "https://example.com/", "", "", "")
	if err := bs.loadTabContent(tab); err != nil {
		t.Fatalf("loadTabContent: %v", err)
	}

	doc := activeDocument(t, bs, tab)
	if doc == nil {
		t.Fatal("expected an active document")
	}

	// DOM: the ketchup DOM must resolve the body element of a live page.
	body, err := doc.DOM.QuerySelector("body")
	if err != nil || body == nil {
		t.Fatalf(`QuerySelector("body") = (%v, %v), want a match`, body, err)
	}

	// Render: a real page paints something non-white (its background and text).
	if got := countPixels(t, doc, isNonWhite); got == 0 {
		t.Error("expected the real-host page to render non-white pixels")
	}
}

// isBlue matches a pixel from a blue background (blue channel dominant).
func isBlue(r, g, b uint32) bool {
	return b > r && b > g && b > 0x8000
}

// isNonWhite matches any pixel that differs from the white background.
func isNonWhite(r, g, b uint32) bool {
	return r < 0xff00 || g < 0xff00 || b < 0xff00
}
