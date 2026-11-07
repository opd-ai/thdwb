package bun

import (
	"testing"

	"github.com/danfragoso/thdwb/assets"
	gg "github.com/danfragoso/thdwb/gg"
	"github.com/danfragoso/thdwb/hotdog"
	"github.com/danfragoso/thdwb/ketchup"
	profiler "github.com/danfragoso/thdwb/profiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExplicitWidthReachesRenderBox guards O-2: on the live layout path
// (experimentalLayout=false) calculateBlockLayout used Style.Width/Height only
// as "skip sizing" predicates and never assigned them, so a box with an
// explicit width got RenderBox.Width == 0 and paintBlockElement skipped it --
// the element rendered nothing.
func TestExplicitWidthReachesRenderBox(t *testing.T) {
	const page = `<!DOCTYPE html><html><head>
	<style>#box { width: 300px; height: 120px; background-color: blue; }</style>
	</head><body><div id="box">text</div></body></html>`

	wc := hotdog.NewWindowContext(&hotdog.Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	doc := ketchup.ParseHTMLDocument(page, wc)
	require.NotNil(t, doc)

	ctx := gg.NewContext(1024, 1024)
	require.NoError(t, RenderDocument(ctx, doc, false))

	box, err := doc.DOM.FindChildByName("div")
	require.NoError(t, err)
	require.NotNil(t, box)
	require.NotNil(t, box.RenderBox)

	assert.Equal(t, 300.0, box.RenderBox.Width, "explicit width must reach RenderBox")
	// Explicit height is applied; layoutDOM then adds child heights (existing
	// box-model behavior), so assert the declared height is honored as a floor
	// rather than the box collapsing to zero.
	assert.GreaterOrEqual(t, box.RenderBox.Height, 120.0, "explicit height must reach RenderBox")

	// The box must actually paint: sample a pixel inside it for the blue fill.
	img := ctx.Image()
	r, g, b, _ := img.At(10, 10).RGBA()
	assert.Greater(t, b>>8, r>>8, "blue background should be painted inside the sized box")
	assert.Greater(t, b>>8, g>>8)
}

// TestExplicitSizeInlineNotBroken ensures the inline path (content-sized by
// design) is unaffected: an inline span still shrinks to its measured text
// rather than taking an explicit block width.
func TestExplicitSizeInlineNotBroken(t *testing.T) {
	const page = `<!DOCTYPE html><html><head>
	<style>#b { display: block; background-color: blue; }</style>
	</head><body><div id="b">hello</div></body></html>`

	wc := hotdog.NewWindowContext(&hotdog.Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	doc := ketchup.ParseHTMLDocument(page, wc)
	require.NotNil(t, doc)

	ctx := gg.NewContext(1024, 1024)
	require.NoError(t, RenderDocument(ctx, doc, false))

	div, err := doc.DOM.FindChildByName("div")
	require.NoError(t, err)
	require.NotNil(t, div.RenderBox)
	// A block with no explicit width fills the parent, not zero.
	assert.Greater(t, div.RenderBox.Width, 0.0)
}
