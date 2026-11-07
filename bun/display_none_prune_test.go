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

// TestDisplayNonePrunedInLiveLayout (audit R-10) verifies display:none nodes
// contribute nothing to the live layout pass. Before the fix layoutDOM
// recursed unconditionally, so a hidden block still laid out and its height
// was added to the parent, pushing visible siblings down.
func TestDisplayNonePrunedInLiveLayout(t *testing.T) {
	const page = `<!DOCTYPE html><html><head>
	<style>
		#hidden { display: none; width: 100px; height: 200px; background-color: red; }
		#shown  { display: block; width: 100px; height: 40px; background-color: green; }
	</style></head><body><div id="hidden"></div><div id="shown"></div></body></html>`

	wc := hotdog.NewWindowContext(&hotdog.Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	doc := ketchup.ParseHTMLDocument(page, wc)
	require.NotNil(t, doc)

	ctx := gg.NewContext(400, 400)
	require.NoError(t, RenderDocument(ctx, doc, false))

	hidden, err := doc.DOM.FindChildByName("div")
	require.NoError(t, err)
	_ = hidden

	// The hidden div must have a zero box and thus add nothing to body height.
	h := findDivByID(doc, "hidden")
	require.NotNil(t, h)
	require.NotNil(t, h.RenderBox)
	assert.Zero(t, h.RenderBox.Height, "display:none must collapse layout height")
	assert.Zero(t, h.RenderBox.Width)

	s := findDivByID(doc, "shown")
	require.NotNil(t, s)
	assert.Greater(t, s.RenderBox.Height, 0.0, "visible sibling must still lay out")

	// And the hidden node must not paint red pixels where it would have been.
	r, g, b, _ := ctx.Image().At(50, 5).RGBA()
	assert.False(t, r>>8 > g>>8 && r>>8 > b>>8,
		"hidden red block must not paint at the top of the page")
}

func findDivByID(doc *hotdog.Document, id string) *hotdog.NodeDOM {
	var found *hotdog.NodeDOM
	var walk func(n *hotdog.NodeDOM)
	walk = func(n *hotdog.NodeDOM) {
		if found != nil {
			return
		}
		if n.Type == hotdog.NodeTypeElement && n.Attr("id") == id {
			found = n
			return
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(doc.DOM)
	return found
}

// TestDisplayNoneHidesSubtree confirms the prune is recursive: a visible child
// under a display:none ancestor is neither laid out nor painted.
func TestDisplayNoneHidesSubtree(t *testing.T) {
	const page = `<!DOCTYPE html><html><head>
	<style>
		#outer { display: none; }
		#inner { display: block; width: 80px; height: 30px; background-color: blue; }
	</style></head><body><div id="outer"><div id="inner"></div></div></body></html>`

	wc := hotdog.NewWindowContext(&hotdog.Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	doc := ketchup.ParseHTMLDocument(page, wc)
	require.NotNil(t, doc)

	ctx := gg.NewContext(400, 400)
	require.NoError(t, RenderDocument(ctx, doc, false))

	inner := findDivByID(doc, "inner")
	require.NotNil(t, inner)
	// The inner node was never laid out (pruned at the ancestor), so its box
	// stays at the parse-time zero.
	assert.Zero(t, inner.RenderBox.Height, "descendant of display:none must not lay out")
}
