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

// TestCommentNotPaintedAsText (audit R-12) verifies HTML comments are not laid
// out or painted. A comment node gets the default block stylesheet (its
// synthetic name "html:comment" is unknown to the UA display table), so before
// the fix it received a real box and paintBlockElement drew node.Content --
// rendering the comment text as visible page content.
func TestCommentNotPaintedAsText(t *testing.T) {
	const page = `<!DOCTYPE html><html><body><div id="d" style="width:100px;height:40px;background-color:white"><!--SECRETTEXT--></div></body></html>`

	wc := hotdog.NewWindowContext(&hotdog.Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	doc := ketchup.ParseHTMLDocument(page, wc)
	require.NotNil(t, doc)

	ctx := gg.NewContext(400, 400)
	require.NoError(t, RenderDocument(ctx, doc, false))

	comment := findNodeByType(doc, hotdog.NodeTypeComment)
	require.NotNil(t, comment, "the comment node should exist in the tree")
	require.NotNil(t, comment.RenderBox)
	assert.Zero(t, comment.RenderBox.Height, "comment must have no layout height")
	assert.Zero(t, comment.RenderBox.Width)
}

// TestDoctypeNotPainted covers the sibling non-renderable types.
func TestDoctypeNotPainted(t *testing.T) {
	const page = `<!DOCTYPE html><html><body><p>hi</p></body></html>`

	wc := hotdog.NewWindowContext(&hotdog.Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	doc := ketchup.ParseHTMLDocument(page, wc)
	require.NotNil(t, doc)

	ctx := gg.NewContext(400, 400)
	require.NoError(t, RenderDocument(ctx, doc, false))

	dt := findNodeByType(doc, hotdog.NodeTypeDoctype)
	if dt != nil {
		require.NotNil(t, dt.RenderBox)
		assert.Zero(t, dt.RenderBox.Height, "doctype must have no layout height")
	}
}

func findNodeByType(doc *hotdog.Document, t hotdog.NodeType) *hotdog.NodeDOM {
	var found *hotdog.NodeDOM
	var walk func(n *hotdog.NodeDOM)
	walk = func(n *hotdog.NodeDOM) {
		if found != nil {
			return
		}
		if n.Type == t {
			found = n
			return
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	if doc.DOM != nil {
		walk(doc.DOM)
	}
	return found
}
