package bun

import (
	"testing"

	"github.com/danfragoso/thdwb/assets"
	gg "github.com/danfragoso/thdwb/gg"
	"github.com/danfragoso/thdwb/hotdog"
	"github.com/danfragoso/thdwb/ketchup"
	"github.com/danfragoso/thdwb/mayo"
	profiler "github.com/danfragoso/thdwb/profiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBlockStackingSkipsPrunedSiblings (audit C-20) verifies that a block
// following a display:none sibling, an HTML comment, or a <script> stacks
// below the last sibling that actually generated a box — not at the parent's
// top. The R-10/R-12 prunes zero out the hidden node's RenderBox; the flow
// anchor must skip those zeroed boxes.
func TestBlockStackingSkipsPrunedSiblings(t *testing.T) {
	const page = `<!DOCTYPE html><html><head>
	<style>
		#a { display: block; width: 100px; height: 70px; background-color: rgb(255,0,0); }
		#b { display: none; width: 100px; height: 90px; }
		#c { display: block; width: 100px; height: 70px; background-color: rgb(0,0,255); }
	</style></head><body><div id="a"></div><div id="b"></div><!-- mid comment --><div id="c"></div></body></html>`

	wc := hotdog.NewWindowContext(&hotdog.Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	doc := ketchup.ParseHTMLDocument(page, wc)
	require.NotNil(t, doc)

	ctx := gg.NewContext(400, 400)
	require.NoError(t, RenderDocument(ctx, doc, false))

	a := findDivByID(doc, "a")
	c := findDivByID(doc, "c")
	require.NotNil(t, a)
	require.NotNil(t, c)
	require.NotNil(t, a.RenderBox)
	require.NotNil(t, c.RenderBox)

	// #c must flow below #a (Top == a.Top + a.Height), not collapse onto the
	// parent top because the hidden/comment siblings have zeroed boxes.
	assert.InDelta(t, a.RenderBox.Top+a.RenderBox.Height, c.RenderBox.Top, 1.0,
		"block after display:none + comment must stack below the previous visible block")

	// And #c must not overprint #a: the pixel inside #a must stay red.
	r, g, b, _ := ctx.Image().At(50, int(a.RenderBox.Top)+5).RGBA()
	assert.Greater(t, r>>8, g>>8, "pixel inside #a must remain red, not overprinted by #c")
	assert.Greater(t, r>>8, b>>8, "pixel inside #a must remain red, not overprinted by #c")
}

// TestCommentBetweenInlineElementsNoPanic (audit C-21) verifies a comment node
// between inline elements does not nil-panic the layout walk. R-12's
// comment/doctype/raw early return ran before the nil-Style default, and
// fragment-parsed comments (innerHTML) are created with Style == nil, so the
// next sibling's inline layout dereferenced prev.Style and panicked.
func TestCommentBetweenInlineElementsNoPanic(t *testing.T) {
	const page = `<!DOCTYPE html><html><body></body></html>`

	wc := hotdog.NewWindowContext(&hotdog.Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	doc := ketchup.ParseHTMLDocument(page, wc)
	require.NotNil(t, doc)

	body, err := doc.DOM.FindChildByName("body")
	require.NoError(t, err)
	require.NotNil(t, body)

	// The innerHTML fragment path builds comment nodes with a nil Style --
	// exactly the shape that panicked before the fix.
	require.NoError(t, body.SetInnerHTML(`<span id="s1">A</span><!--c--><span id="s2">B</span>`))

	ctx := gg.NewContext(400, 200)
	require.NoError(t, RenderDocument(ctx, doc, false))

	s2 := findDivByID(doc, "s2")
	require.NotNil(t, s2)
	require.NotNil(t, s2.RenderBox)
	// Pre-fix this panicked on the nil-Style comment sibling. Post-fix the
	// span lays out and its style is defaulted.
	assert.GreaterOrEqual(t, s2.RenderBox.Height, 1.0, "span after comment must lay out")
	require.NotNil(t, s2.Style, "span style must be defaulted, not nil")
}

// TestCommentNilStyleBetweenInlineSpans reproduces the exact tree shape the
// audit probe built via JS (createElement/createComment/appendChild): a
// comment node with Style == nil between two inline spans. Before the fix,
// R-12's comment early-return ran before the nil-Style default, so the
// following inline span's layout dereferenced prev.Style.Display -> panic.
func TestCommentNilStyleBetweenInlineSpans(t *testing.T) {
	wc := hotdog.NewWindowContext(&hotdog.Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()

	doc := &hotdog.Document{
		DOM: &hotdog.NodeDOM{
			Type:     hotdog.NodeTypeElement,
			Element:  "html",
			Children: make([]*hotdog.NodeDOM, 0),
			Style:    mayo.GetElementStylesheet("html", nil),
		},
	}
	doc.DOM.Document = doc
	body := &hotdog.NodeDOM{
		Type:       hotdog.NodeTypeElement,
		Element:    "body",
		Children:   make([]*hotdog.NodeDOM, 0),
		Attributes: make([]*hotdog.Attribute, 0),
		Document:   doc,
		WindowCtx:  wc,
		Style:      mayo.GetElementStylesheet("body", nil),
	}
	doc.DOM.Children = append(doc.DOM.Children, body)
	body.Parent = doc.DOM

	mkSpan := func(id string) *hotdog.NodeDOM {
		span := doc.CreateElement("span")
		span.Attributes = append(span.Attributes, &hotdog.Attribute{Name: "id", Value: id})
		span.WindowCtx = wc
		span.Style = mayo.GetElementStylesheet("span", span.Attributes)
		span.RenderBox = &hotdog.RenderBox{}
		return span
	}

	s1 := mkSpan("s1")
	s2 := mkSpan("s2")
	// JS document.createComment returns a node with NO Style -- the shape that
	// panicked. InsertBefore is the public linking path page JS uses.
	comment := &hotdog.NodeDOM{
		Type:      hotdog.NodeTypeComment,
		Content:   "c",
		Document:  doc,
		WindowCtx: wc,
	}
	body.InsertBefore(s1, nil)
	body.InsertBefore(comment, nil)
	body.InsertBefore(s2, nil)
	require.Nil(t, comment.Style, "the reproducing shape requires a nil-Style comment")

	ctx := gg.NewContext(400, 200)
	require.NoError(t, RenderDocument(ctx, doc, false))

	require.NotNil(t, s2.RenderBox)
	require.NotNil(t, comment.Style, "layout must default a nil-Style comment, not leave it nil")
}
