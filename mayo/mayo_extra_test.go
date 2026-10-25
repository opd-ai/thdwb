package mayo

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	hotdog "github.com/danfragoso/thdwb/hotdog"
	"golang.org/x/net/html"
)

func almostEqual(t *testing.T, got, want float64) bool {
	t.Helper()
	return math.Abs(got-want) < 1e-9
}

// TestColorFloatRegression covers the integer-division regression in
// hexToFloatInRange / RGBAToColor: channels must be float values in 0-1.
func TestColorFloatRegression(t *testing.T) {
	c := HexStringToColor("#cc3300")
	require.NotNil(t, c)
	assert.True(t, almostEqual(t, c.R, 204/255.0), "R: %v", c.R)
	assert.True(t, almostEqual(t, c.G, 51/255.0), "G: %v", c.G)
	assert.Equal(t, 0.0, c.B)
	assert.Equal(t, 1.0, c.A)

	grey := HexStringToColor("#333333")
	for name, ch := range map[string]float64{"R": grey.R, "G": grey.G, "B": grey.B} {
		assert.True(t, almostEqual(t, ch, 51/255.0), "%s: %v", name, ch)
	}

	// 4-digit short form: #f00 -> R=1.0.
	short := HexStringToColor("#f00")
	assert.Equal(t, 1.0, short.R)
	assert.Equal(t, 0.0, short.G)
	assert.Equal(t, 0.0, short.B)
	assert.Equal(t, 1.0, short.A)

	// 5-digit short form doubles the alpha digit too.
	shortA := HexStringToColor("#f00a")
	assert.Equal(t, 1.0, shortA.R)
	assert.True(t, almostEqual(t, shortA.A, 0xaa/255.0), "A: %v", shortA.A)

	// Bad hex (non-hex chars) returns 0, not a panic.
	bad := HexStringToColor("#zz0000")
	assert.Equal(t, 0.0, bad.R)
	assert.Equal(t, 0.0, bad.G)
	assert.Equal(t, 0.0, bad.B)
}

func TestRGBAToColorForms(t *testing.T) {
	// rgb() with int channels on 0-255 scale.
	c := RGBAToColor("rgb(204, 51, 0)")
	require.NotNil(t, c)
	assert.True(t, almostEqual(t, c.R, 204/255.0))
	assert.True(t, almostEqual(t, c.G, 51/255.0))
	assert.Equal(t, 0.0, c.B)

	// Percent form: 50% -> 0.5.
	p := RGBAToColor("rgba(50%, 50%, 50%)")
	require.NotNil(t, p)
	assert.Equal(t, 0.5, p.R)
	assert.Equal(t, 0.5, p.G)
	assert.Equal(t, 0.5, p.B)

	// Float channels pass through unchanged.
	f := RGBAToColor("rgb(0.5, 0.25, 1.0)")
	require.NotNil(t, f)
	assert.Equal(t, 0.5, f.R)
	assert.Equal(t, 0.25, f.G)
	assert.Equal(t, 1.0, f.B)

	// Mixed forms parse independently per channel.
	mixed := RGBAToColor("rgb(127.5, 50%, 255)")
	require.NotNil(t, mixed)
	assert.Equal(t, 127.5, mixed.R)
	assert.Equal(t, 0.5, mixed.G)
	assert.True(t, almostEqual(t, mixed.B, 1.0))

	// Fewer than 3 params -> nil (zero color).
	assert.Nil(t, RGBAToColor("rgb(0, 0)"))
	assert.Nil(t, RGBAToColor("not a color"))
}

func TestHexToFloatInRange(t *testing.T) {
	assert.True(t, almostEqual(t, hexToFloatInRange("ff"), 1.0))
	assert.True(t, almostEqual(t, hexToFloatInRange("80"), 128/255.0))
	assert.Equal(t, 0.0, hexToFloatInRange("zz"))
	assert.Equal(t, 0.0, hexToFloatInRange(""))
}

func TestMapPropToStylesheetFullPropertySet(t *testing.T) {
	sheet := &hotdog.Stylesheet{}
	props := [][]string{
		{"font-style", "italic"},
		{"line-height", "1.5"},
		{"letter-spacing", "2px"},
		{"text-align", "center"},
		{"text-decoration", "underline"},
		{"white-space", "nowrap"},
		{"flex-direction", "row"},
		{"flex-wrap", "wrap"},
		{"justify-content", "space-between"},
		{"align-items", "center"},
		{"align-content", "stretch"},
		{"flex-grow", "1"},
		{"flex-shrink", "2"},
		{"flex-basis", "auto"},
		{"order", "3"},
		{"align-self", "baseline"},
		{"min-width", "10px"},
		{"min-height", "20px"},
		{"max-width", "30px"},
		{"max-height", "40px"},
		{"top", "1px"},
		{"right", "2px"},
		{"bottom", "3px"},
		{"left", "4px"},
		{"z-index", "5"},
		{"margin-top", "6px"},
		{"margin-right", "7px"},
		{"margin-bottom", "8px"},
		{"margin-left", "9px"},
		{"padding-top", "10px"},
		{"padding-right", "11px"},
		{"padding-bottom", "12px"},
		{"padding-left", "13px"},
		{"border-top-width", "1px"},
		{"border-right-width", "2px"},
		{"border-bottom-width", "3px"},
		{"border-left-width", "4px"},
		{"border-top-style", "solid"},
		{"border-right-style", "dashed"},
		{"border-bottom-style", "dotted"},
		{"border-left-style", "double"},
		{"border-top-color", "red"},
		{"border-right-color", "green"},
		{"border-bottom-color", "blue"},
		{"border-left-color", "black"},
		{"overflow", "hidden"},
		{"overflow-x", "auto"},
		{"overflow-y", "scroll"},
		{"visibility", "hidden"},
		{"opacity", "0.5"},
	}
	for _, p := range props {
		mapPropToStylesheet(sheet, p)
	}

	assert.Equal(t, "italic", sheet.FontStyle)
	assert.Equal(t, 1.5, sheet.LineHeight)
	assert.Equal(t, 2.0, sheet.LetterSpacing)
	assert.Equal(t, "center", sheet.TextAlign)
	assert.Equal(t, "underline", sheet.TextDecoration)
	assert.Equal(t, "nowrap", sheet.WhiteSpace)
	assert.Equal(t, "row", sheet.FlexDirection)
	assert.Equal(t, "wrap", sheet.FlexWrap)
	assert.Equal(t, "space-between", sheet.JustifyContent)
	assert.Equal(t, "center", sheet.AlignItems)
	assert.Equal(t, "stretch", sheet.AlignContent)
	assert.Equal(t, 1.0, sheet.FlexGrow)
	assert.Equal(t, 2.0, sheet.FlexShrink)
	assert.Equal(t, "auto", sheet.FlexBasis)
	assert.Equal(t, 3, sheet.Order)
	assert.Equal(t, "baseline", sheet.AlignSelf)
	assert.Equal(t, 10.0, sheet.MinWidth)
	assert.Equal(t, 20.0, sheet.MinHeight)
	assert.Equal(t, 30.0, sheet.MaxWidth)
	assert.Equal(t, 40.0, sheet.MaxHeight)
	assert.Equal(t, 1.0, sheet.Top)
	assert.Equal(t, 2.0, sheet.Right)
	assert.Equal(t, 3.0, sheet.Bottom)
	assert.Equal(t, 4.0, sheet.Left)
	assert.Equal(t, 5, sheet.ZIndex)
	assert.Equal(t, 6.0, sheet.MarginTop)
	assert.Equal(t, 7.0, sheet.MarginRight)
	assert.Equal(t, 8.0, sheet.MarginBottom)
	assert.Equal(t, 9.0, sheet.MarginLeft)
	assert.Equal(t, 10.0, sheet.PaddingTop)
	assert.Equal(t, 11.0, sheet.PaddingRight)
	assert.Equal(t, 12.0, sheet.PaddingBottom)
	assert.Equal(t, 13.0, sheet.PaddingLeft)
	assert.Equal(t, 1.0, sheet.BorderTopWidth)
	assert.Equal(t, 2.0, sheet.BorderRightWidth)
	assert.Equal(t, 3.0, sheet.BorderBottomWidth)
	assert.Equal(t, 4.0, sheet.BorderLeftWidth)
	assert.Equal(t, "solid", sheet.BorderTopStyle)
	assert.Equal(t, "dashed", sheet.BorderRightStyle)
	assert.Equal(t, "dotted", sheet.BorderBottomStyle)
	assert.Equal(t, "double", sheet.BorderLeftStyle)
	require.NotNil(t, sheet.BorderTopColor)
	assert.Equal(t, 1.0, sheet.BorderTopColor.R)
	require.NotNil(t, sheet.BorderRightColor)
	assert.Equal(t, 0.5, sheet.BorderRightColor.G) // colorTable defines "green" as {0, 0.5, 0, 1}
	require.NotNil(t, sheet.BorderBottomColor)
	assert.Equal(t, 1.0, sheet.BorderBottomColor.B)
	// overflow set both axes; the per-axis rules then overwrote each.
	assert.Equal(t, "auto", sheet.OverflowX)
	assert.Equal(t, "scroll", sheet.OverflowY)
	assert.Equal(t, "hidden", sheet.Visibility)
	assert.Equal(t, 0.5, sheet.Opacity)

	// Invalid opacity leaves the previous value.
	bad := &hotdog.Stylesheet{Opacity: 0.5}
	mapPropToStylesheet(bad, []string{"opacity", "not-a-number"})
	assert.Equal(t, 0.5, bad.Opacity)

	// Unknown properties are ignored without error.
	unknown := &hotdog.Stylesheet{}
	mapPropToStylesheet(unknown, []string{"tab-size", "4"})
	assert.Equal(t, 0.0, unknown.Opacity)
}

func TestMergeStylesheetCoverage(t *testing.T) {
	// nil src leaves dest untouched.
	dest := &hotdog.Stylesheet{Display: "block"}
	mergeStylesheet(dest, nil)
	assert.Equal(t, "block", dest.Display)

	src := &hotdog.Stylesheet{
		Color:         &hotdog.ColorRGBA{R: 0.1, G: 0.2, B: 0.3, A: 1},
		BackgroundColor: &hotdog.ColorRGBA{R: 1, G: 1, B: 1, A: 0},
		FontSize:      18,
		FontWeight:    600,
		FontFamily:    "Arial",
		FontStyle:     "italic",
		LineHeight:    1.4,
		LetterSpacing: 1,
		TextAlign:     "right",
		TextDecoration: "line-through",
		WhiteSpace:    "pre",
		Display:       "flex",
		Position:      "relative",
		FlexDirection: "column",
		FlexWrap:      "nowrap",
		JustifyContent: "flex-start",
		AlignItems:    "flex-end",
		AlignContent:  "center",
		FlexGrow:      1,
		FlexShrink:    2,
		FlexBasis:     "0",
		Order:         4,
		AlignSelf:     "auto",
		Width:         100,
		Height:        50,
		MinWidth:      10,
		MinHeight:     20,
		MaxWidth:      30,
		MaxHeight:     40,
		Top:           1,
		Right:         2,
		Bottom:        3,
		Left:          4,
		ZIndex:        5,
		MarginTop:     6,
		MarginRight:   7,
		MarginBottom:  8,
		MarginLeft:    9,
		PaddingTop:    10,
		PaddingRight:  11,
		PaddingBottom: 12,
		PaddingLeft:   13,
	}
	dest = &hotdog.Stylesheet{}
	mergeStylesheet(dest, src)
	assert.Equal(t, 18.0, dest.FontSize)
	assert.Equal(t, 600, dest.FontWeight)
	assert.Equal(t, "Arial", dest.FontFamily)
	assert.Equal(t, "flex", dest.Display)
	assert.Equal(t, "relative", dest.Position)
	assert.Equal(t, 4, dest.Order)
	assert.Equal(t, 100.0, dest.Width)
	assert.Equal(t, 5, dest.ZIndex)
	require.NotNil(t, dest.Color)
	assert.Equal(t, 0.1, dest.Color.R)
	assert.Equal(t, 13.0, dest.PaddingLeft)

	// Zero-value src fields do not overwrite dest.
	dest2 := &hotdog.Stylesheet{FontSize: 16, Display: "grid"}
	mergeStylesheet(dest2, &hotdog.Stylesheet{})
	assert.Equal(t, 16.0, dest2.FontSize)
	assert.Equal(t, "grid", dest2.Display)
}

func TestApplyStylesheetMatchingAndGaps(t *testing.T) {
	css := `
		/* a comment to strip */
		div { color: red; }
		.unclosed { color: blue
		#target { background-color: blue; width: 42px; }
	`
	elements := ParseStylesheet(css)
	// The unclosed rule swallows the "#target {" text, so the parser yields the
	// div rule plus one malformed rule whose stray "width: 42px" declaration is
	// still mapped. Exercises both the valid and degraded paths.
	assert.Len(t, elements, 2)
	require.NotNil(t, elements[1].Style)
	assert.Equal(t, 42.0, elements[1].Style.Width)

	// Invalid selector compiles to error and is skipped by ApplyStylesheets.
	doc := &hotdog.Document{
		StyleSheets: []*hotdog.StyleElement{
			{Selector: "#", Style: &hotdog.Stylesheet{Width: 1}},
			{Selector: "div", Style: &hotdog.Stylesheet{Width: 2}},
		},
	}
	divDOM := &hotdog.NodeDOM{Element: "div", Style: &hotdog.Stylesheet{}}
	divHTML := &html.Node{Type: html.ElementNode, Data: "div"}
	divDOM.HTMLNode = divHTML
	rootDOM := &hotdog.NodeDOM{Element: "html", Style: &hotdog.Stylesheet{}, Children: []*hotdog.NodeDOM{divDOM}}
	rootHTML := &html.Node{Type: html.ElementNode, Data: "html", FirstChild: divHTML}
	rootDOM.HTMLNode = rootHTML
	doc.DOM = rootDOM
	doc.HTMLRoot = rootHTML

	ApplyStylesheets(doc)
	assert.Equal(t, 2.0, divDOM.Style.Width)

	// Nil / incomplete documents are no-ops.
	ApplyStylesheets(nil)
	ApplyStylesheets(&hotdog.Document{})
	ApplyStylesheets(&hotdog.Document{DOM: rootDOM})
	// Entries with nil style are skipped.
	doc.StyleSheets = append(doc.StyleSheets, &hotdog.StyleElement{Selector: "div", Style: nil})
	doc.StyleSheets = append(doc.StyleSheets, &hotdog.StyleElement{Selector: "", Style: &hotdog.Stylesheet{}})
	ApplyStylesheets(doc)

	// findNodeDOMByHTMLNode: match at root, in subtree, and no match.
	assert.Same(t, rootDOM, findNodeDOMByHTMLNode(rootDOM, rootHTML))
	assert.Nil(t, findNodeDOMByHTMLNode(nil, divHTML))
	assert.Nil(t, findNodeDOMByHTMLNode(rootDOM, &html.Node{Type: html.ElementNode, Data: "nope"}))
}

func TestApplyInheritanceGaps(t *testing.T) {
	ApplyInheritance(nil)

	// Node with nil style is skipped; children still processed.
	child := &hotdog.NodeDOM{Element: "span", Style: &hotdog.Stylesheet{}}
	odd := &hotdog.NodeDOM{Element: "div", Style: nil, Children: []*hotdog.NodeDOM{child}}
	ApplyInheritance(odd)

	// Child of a node whose Style is nil never receives a parent style.
	assert.Nil(t, child.Style.Color)

	// inheritFromParent nil guards.
	inheritFromParent(nil, &hotdog.Stylesheet{})
	inheritFromParent(&hotdog.Stylesheet{}, nil)
}

func TestReflowNodeOffsets(t *testing.T) {
	grandchild := &hotdog.NodeDOM{Element: "span", Style: &hotdog.Stylesheet{Height: 2}}
	child := &hotdog.NodeDOM{Element: "div", Style: &hotdog.Stylesheet{Height: 10}, Children: []*hotdog.NodeDOM{grandchild}}
	root := &hotdog.NodeDOM{Element: "body", Style: &hotdog.Stylesheet{Height: 100}, Children: []*hotdog.NodeDOM{child}}

	// node != prev returns Height; offset accumulates through siblings:
	// grandchild Top = 2 + 0; child Top = 10 + 2; root Top = 100 + 10.
	total := ReflowNode(root, nil, 0)
	assert.Equal(t, 100.0, total)
	assert.Equal(t, 12.0, child.Style.Top)
	assert.Equal(t, 2.0, grandchild.Style.Top)

	// node == prev (leaf case) still returns its Height.
	leaf := &hotdog.NodeDOM{Element: "p", Style: &hotdog.Stylesheet{Height: 7}}
	assert.Equal(t, 7.0, ReflowNode(leaf, leaf, 0))
}

func TestGetElementStylesheetDefaults(t *testing.T) {
	// Unknown element gets defaults: font 14, black color, weight 400, display block.
	sheet := GetElementStylesheet("unknown-thing", nil)
	assert.Equal(t, 14.0, sheet.FontSize)
	require.NotNil(t, sheet.Color)
	assert.Equal(t, 0.0, sheet.Color.R)
	assert.Equal(t, 400, sheet.FontWeight)
	assert.Equal(t, "block", sheet.Display)
	assert.Equal(t, "Normal", sheet.Position)
	assert.Equal(t, 1.0, sheet.Opacity)
	assert.Equal(t, "visible", sheet.Visibility)

	// Heading element: weight 600 from table.
	h1 := GetElementStylesheet("h1", nil)
	assert.Equal(t, 600, h1.FontWeight)

	// Inline element: display inline.
	a := GetElementStylesheet("a", nil)
	assert.Equal(t, "inline", a.Display)

	// Inline style beats element table defaults.
	styled := GetElementStylesheet("h1", []*hotdog.Attribute{
		{Name: "style", Value: "color:red;font-size:30px"},
	})
	assert.Equal(t, 30.0, styled.FontSize)
	require.NotNil(t, styled.Color)
	assert.Equal(t, 1.0, styled.Color.R)

	// Non-style attributes are ignored.
	plain := GetElementStylesheet("div", []*hotdog.Attribute{{Name: "class", Value: "x"}})
	assert.Equal(t, 14.0, plain.FontSize)
}
