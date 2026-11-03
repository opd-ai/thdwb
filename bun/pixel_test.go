package bun

import (
	"os"
	"runtime/debug"
	"testing"

	"github.com/danfragoso/thdwb/assets"
	gg "github.com/danfragoso/thdwb/gg"
	"github.com/danfragoso/thdwb/hotdog"
	"github.com/danfragoso/thdwb/ketchup"
	profiler "github.com/danfragoso/thdwb/profiler"
)

// stdPathBluePanel is a page whose block gets a blue background purely from a
// <style> rule (no inline style). The standard layout path gives the block its
// width from the parent and its height from the text content, so the blue fill
// is painted regardless of font metrics — a deterministic, CSS-driven signal.
const stdPathBluePanel = `<!DOCTYPE html><html><head>
<style>#panel { background-color: blue; }</style>
</head><body><div id="panel">Content that gives the block a height for the standard layout to measure.</div></body></html>`

// TestRenderDocument_paintsNonWhitePixels is the pixel-output invariant the
// plan calls out: the bun suite was 88% covered yet rendered nothing, so a
// render that paints zero pixels would pass. This renders a page through the
// standard layout path and asserts the canvas is no longer entirely white, and
// specifically that the CSS-declared blue background actually reached the
// canvas (not just stray text pixels).
func TestRenderDocument_paintsNonWhitePixels(t *testing.T) {
	wc := hotdog.NewWindowContext(&hotdog.Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	doc := ketchup.ParseHTMLDocument(stdPathBluePanel, wc)
	if doc == nil {
		t.Fatal("ketchup.ParseHTMLDocument returned nil")
	}

	ctx := gg.NewContext(1024, 1024)
	defer func() {
		if err := recover(); err != nil {
			t.Fatalf("RenderDocument panicked: %v\n%s", err, debug.Stack())
		}
	}()
	if err := RenderDocument(ctx, doc, false); err != nil {
		t.Fatalf("RenderDocument failed: %v", err)
	}
	img := ctx.Image()
	nonWhite, bluish := 0, 0
	for y := 0; y < ctx.Height(); y++ {
		for x := 0; x < ctx.Width(); x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r < 0xffff || g < 0xffff || b < 0xffff {
				nonWhite++
			}
			if b > r && b > g {
				bluish++
			}
		}
	}
	if nonWhite == 0 {
		t.Fatal("standard-path RenderDocument painted no non-white pixels; canvas is blank")
	}
	if bluish == 0 {
		t.Errorf("standard-path RenderDocument painted no blue pixels for the CSS background (nonWhite=%d)", nonWhite)
	}
}

// TestRenderDocument_experimentalPaintsNonWhitePixels is the same invariant for
// the experimental (flex) layout path, which is the one driven by
// TestRenderDocument_flexbox.
func TestRenderDocument_experimentalPaintsNonWhitePixels(t *testing.T) {
	doc := parseFlexboxFixture(t)
	ctx := gg.NewContext(1024, 1024)
	defer func() {
		if err := recover(); err != nil {
			t.Fatalf("RenderDocument panicked: %v\n%s", err, debug.Stack())
		}
	}()
	if err := RenderDocumentToCanvas(testCanvasContext{context: ctx}, doc, true); err != nil {
		t.Fatalf("RenderDocumentToCanvas failed: %v", err)
	}
	img := ctx.Image()
	nonWhite := 0
	for y := 0; y < ctx.Height(); y++ {
		for x := 0; x < ctx.Width(); x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r < 0xffff || g < 0xffff || b < 0xffff {
				nonWhite++
			}
		}
	}
	if nonWhite == 0 {
		t.Fatal("experimental RenderDocument painted no non-white pixels; canvas is blank")
	}
}

// parseFlexboxFixture loads and parses the shared flexbox fixture (blue 50x50
// items) the way the other bun render tests do.
func parseFlexboxFixture(t *testing.T) *hotdog.Document {
	t.Helper()
	html, err := os.ReadFile("test-data/flexbox-test.html")
	if err != nil {
		t.Fatalf("read flexbox fixture: %v", err)
	}
	windowCtx := hotdog.NewWindowContext(&hotdog.Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	doc := ketchup.ParseHTMLDocument(string(html), windowCtx)
	if doc == nil {
		t.Fatal("ketchup.ParseHTMLDocument returned nil")
	}
	return doc
}
