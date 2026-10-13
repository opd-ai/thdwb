package bun

import (
	"math"
	"net/http"
	"net/url"
	"testing"

	gg "github.com/danfragoso/thdwb/gg"
	"github.com/danfragoso/thdwb/hotdog"
	"github.com/kjk/flex"
)

func layoutTestStyle(display string) *hotdog.Stylesheet {
	return &hotdog.Stylesheet{
		Display: display, FontSize: 14, FontWeight: 400,
		Color:           &hotdog.ColorRGBA{R: 0, G: 0, B: 0, A: 1},
		BackgroundColor: &hotdog.ColorRGBA{R: 1, G: 1, B: 1, A: 1},
	}
}

func TestGetPageTitle(t *testing.T) {
	root := &hotdog.NodeDOM{Element: "html", Children: []*hotdog.NodeDOM{
		{Element: "head", Children: []*hotdog.NodeDOM{{Element: "title", Content: "Example"}}},
	}}
	if got := GetPageTitle(root); got != "Example" {
		t.Fatalf("GetPageTitle() = %q, want Example", got)
	}
	if got := GetPageTitle(&hotdog.NodeDOM{Element: "body"}); got != "Sem Titulo" {
		t.Fatalf("missing title = %q, want fallback", got)
	}
}

func TestRenderDocumentAndCanvasErrors(t *testing.T) {
	ctx := gg.NewContext(20, 20)
	if err := RenderDocumentToCanvas(nil, nil, false); err == nil {
		t.Fatal("nil canvas should fail")
	}
	if err := RenderDocumentToCanvas(testCanvasContext{}, nil, false); err == nil {
		t.Fatal("nil context should fail")
	}
	missingHTML := &hotdog.Document{DOM: &hotdog.NodeDOM{Element: "root"}}
	if err := RenderDocument(ctx, missingHTML, true); err == nil {
		t.Fatal("experimental layout without html should fail")
	}
	html := &hotdog.NodeDOM{Element: "html", Style: layoutTestStyle("block"), RenderBox: &hotdog.RenderBox{}}
	body := &hotdog.NodeDOM{Element: "body", Parent: html, Style: layoutTestStyle("block")}
	html.Children = []*hotdog.NodeDOM{body}
	doc := &hotdog.Document{DOM: html}
	if err := RenderDocumentToCanvas(testCanvasContext{context: ctx}, doc, false); err != nil {
		t.Fatalf("fallback root rendering failed: %v", err)
	}
}

func TestLayoutHelpers(t *testing.T) {
	root := &hotdog.NodeDOM{Element: "root", Content: "root"}
	child := &hotdog.NodeDOM{Element: "child", Content: "child"}
	root.Children = []*hotdog.NodeDOM{child}
	if getNodeContent(root) != "root" || getElementName(child) != "child" {
		t.Fatal("node accessors returned incorrect content")
	}
	walkDOM(root, "")
	paintText(gg.NewContext(10, 10), root)
	paintTree(gg.NewContext(10, 10), nil)
}

func TestBlockLayoutBehavior(t *testing.T) {
	ctx := gg.NewContext(100, 80)
	parent := &hotdog.NodeDOM{RenderBox: &hotdog.RenderBox{Top: 7, Width: 90}}
	first := &hotdog.NodeDOM{Type: hotdog.NodeTypeText, Content: "wrapped text", Parent: parent, Style: layoutTestStyle("block"), RenderBox: &hotdog.RenderBox{}}
	calculateBlockLayout(ctx, first, 0)
	if first.RenderBox.Width != 90 || first.RenderBox.Top != 7 || first.RenderBox.Height <= 0 {
		t.Fatalf("unexpected first text layout: %+v", first.RenderBox)
	}
	previous := &hotdog.NodeDOM{RenderBox: &hotdog.RenderBox{Top: 20, Height: 9}}
	second := &hotdog.NodeDOM{Type: hotdog.NodeTypeText, Content: "next", Parent: parent, Style: layoutTestStyle("block"), RenderBox: &hotdog.RenderBox{}}
	parent.Children = []*hotdog.NodeDOM{previous, second}
	calculateBlockLayout(ctx, second, 1)
	if second.RenderBox.Top != 29 {
		t.Fatalf("second text top = %v, want 29", second.RenderBox.Top)
	}
	element := &hotdog.NodeDOM{Content: "block", Parent: parent, Style: layoutTestStyle("block"), RenderBox: &hotdog.RenderBox{Width: 90}}
	calculateBlockLayout(ctx, element, 0)
	if element.RenderBox.Width != 90 || element.RenderBox.Height <= 0 || element.RenderBox.Top != 7 {
		t.Fatalf("unexpected element layout: %+v", element.RenderBox)
	}
}

func TestInlineLayoutBehavior(t *testing.T) {
	ctx := gg.NewContext(100, 80)
	parent := &hotdog.NodeDOM{RenderBox: &hotdog.RenderBox{Top: 4, Left: 6, Width: 80}}
	text := &hotdog.NodeDOM{Type: hotdog.NodeTypeText, Content: "text", Parent: parent, Style: layoutTestStyle("inline"), RenderBox: &hotdog.RenderBox{}}
	calculateInlineLayout(ctx, text, 0)
	if text.RenderBox.Left != 6 || text.RenderBox.Top != 4 || text.RenderBox.Width <= 0 {
		t.Fatalf("unexpected first inline text layout: %+v", text.RenderBox)
	}
	prevInline := &hotdog.NodeDOM{Style: layoutTestStyle("inline"), RenderBox: &hotdog.RenderBox{Left: 9, Top: 10, Width: 20, Height: 6}}
	next := &hotdog.NodeDOM{Content: "next", Parent: parent, Style: layoutTestStyle("inline"), RenderBox: &hotdog.RenderBox{}}
	parent.Children = []*hotdog.NodeDOM{prevInline, next}
	calculateInlineLayout(ctx, next, 1)
	if next.RenderBox.Left != 29 || next.RenderBox.Top != 10 {
		t.Fatalf("inline sibling position = (%v,%v), want (29,10)", next.RenderBox.Left, next.RenderBox.Top)
	}
	prevBlock := &hotdog.NodeDOM{Style: layoutTestStyle("block"), RenderBox: &hotdog.RenderBox{Top: 30, Height: 8}}
	next = &hotdog.NodeDOM{Content: "block sibling", Parent: parent, Style: layoutTestStyle("inline"), RenderBox: &hotdog.RenderBox{}}
	parent.Children = []*hotdog.NodeDOM{prevBlock, next}
	calculateInlineLayout(ctx, next, 1)
	if next.RenderBox.Left != 6 || next.RenderBox.Top != 38 {
		t.Fatalf("block sibling position = (%v,%v), want (6,38)", next.RenderBox.Left, next.RenderBox.Top)
	}
}

func TestListItemLayoutAndPainting(t *testing.T) {
	ctx := gg.NewContext(100, 80)
	parent := &hotdog.NodeDOM{RenderBox: &hotdog.RenderBox{Top: 5, Width: 90}}
	first := &hotdog.NodeDOM{Content: "item", Parent: parent, Style: layoutTestStyle("list-item"), RenderBox: &hotdog.RenderBox{}}
	calculateListItemLayout(ctx, first, 0)
	if first.RenderBox.Width != 60 || first.RenderBox.Left != 30 || first.RenderBox.Top != 5 || first.RenderBox.Height <= 0 {
		t.Fatalf("unexpected first list item layout: %+v", first.RenderBox)
	}
	secondStyle := layoutTestStyle("list-item")
	secondStyle.Width = 40
	second := &hotdog.NodeDOM{Content: "second", Parent: parent, Style: secondStyle, RenderBox: &hotdog.RenderBox{Width: 40, Height: 12}}
	parent.Children = []*hotdog.NodeDOM{first, second}
	calculateListItemLayout(ctx, second, 1)
	if second.RenderBox.Width != 40 || second.RenderBox.Top != first.RenderBox.Top+first.RenderBox.Height {
		t.Fatalf("unexpected following list item layout: %+v", second.RenderBox)
	}
	paintListItemElement(ctx, &hotdog.NodeDOM{Content: "paint", Style: layoutTestStyle("list-item"), RenderBox: &hotdog.RenderBox{Width: 60, Height: 20}})
}

func TestParseFlexValue(t *testing.T) {
	tests := []struct {
		input string
		unit  flex.Unit
		value float32
	}{
		{" auto ", flex.UnitAuto, flex.Undefined},
		{" 25% ", flex.UnitPercent, 25},
		{"12.5", flex.UnitPoint, 12.5},
		{"nonsense", flex.UnitAuto, flex.Undefined},
		{"bad%", flex.UnitAuto, flex.Undefined},
	}
	for _, test := range tests {
		got := parseFlexValue(test.input)
		valueMatches := got.Value == test.value || (math.IsNaN(float64(got.Value)) && math.IsNaN(float64(test.value)))
		if got.Unit != test.unit || !valueMatches {
			t.Errorf("parseFlexValue(%q) = (%v,%v), want (%v,%v)", test.input, got.Unit, got.Value, test.unit, test.value)
		}
	}
}

func TestApplyStylesToFlexNode(t *testing.T) {
	style := &hotdog.Stylesheet{
		Display: "inline-flex", FlexDirection: "row-reverse", FlexWrap: "wrap-reverse",
		JustifyContent: "space-evenly", AlignItems: "center", AlignContent: "space-around",
		AlignSelf: "flex-end", FlexGrow: 2, FlexShrink: 3, FlexBasis: "50%",
		Width: 40, Height: 30, MinWidth: 10, MinHeight: 11, MaxWidth: 80, MaxHeight: 70,
		Position: "absolute", Top: 1, Right: 2, Bottom: 3, Left: 4,
		MarginTop: 5, MarginRight: 6, MarginBottom: 7, MarginLeft: 8,
		PaddingTop: 9, PaddingRight: 10, PaddingBottom: 11, PaddingLeft: 12,
		BorderTopWidth: 1, BorderRightWidth: 2, BorderBottomWidth: 3, BorderLeftWidth: 4,
		OverflowX: "hidden",
	}
	node := flex.NewNode()
	applyStylesToFlexNode(node, style)
	if node.Style.Display != flex.DisplayFlex || node.Style.FlexDirection != flex.FlexDirectionRowReverse || node.Style.FlexWrap != flex.WrapWrapReverse {
		t.Fatalf("flex display/direction/wrap were not applied: %+v", node.Style)
	}
	if node.Style.JustifyContent != flex.JustifySpaceAround || node.Style.AlignItems != flex.AlignCenter || node.Style.AlignContent != flex.AlignSpaceAround || node.Style.AlignSelf != flex.AlignFlexEnd {
		t.Fatalf("flex alignment styles were not applied: %+v", node.Style)
	}
	if node.Style.FlexGrow != 2 || node.Style.FlexShrink != 3 || node.Style.FlexBasis.Unit != flex.UnitPercent ||
		node.Style.Dimensions[flex.DimensionWidth].Value != 40 || node.Style.Dimensions[flex.DimensionHeight].Value != 30 {
		t.Fatalf("flex sizing styles were not applied: %+v", node.Style)
	}
	if node.Style.PositionType != flex.PositionTypeAbsolute || node.Style.Overflow != flex.OverflowHidden {
		t.Fatalf("flex positioning/overflow styles were not applied: %+v", node.Style)
	}
	applyStylesToFlexNode(flex.NewNode(), nil)
}

func TestInlineImageUsesFallbackWhenCSPBlocksSource(t *testing.T) {
	pageURL, _ := url.Parse("https://example.com/index.html")
	document := &hotdog.Document{URL: pageURL}
	headers := make(http.Header)
	headers.Set("Content-Security-Policy", "img-src 'self'")
	document.ApplyContentSecurityPolicyHeaders(headers)
	node := &hotdog.NodeDOM{
		Element: "img", Document: document,
		Attributes: []*hotdog.Attribute{{Name: "src", Value: "https://blocked.example/image.png"}},
		Style:      layoutTestStyle("inline"), RenderBox: &hotdog.RenderBox{Width: 16, Height: 16},
	}
	paintInlineElement(gg.NewContext(30, 30), node)
}

func TestInlineTextPaintIgnoresInvalidBounds(t *testing.T) {
	ctx := gg.NewContext(10, 10)
	node := &hotdog.NodeDOM{Type: hotdog.NodeTypeText, Style: layoutTestStyle("inline"), RenderBox: &hotdog.RenderBox{}}
	paintTextWithinBounds(ctx, node, "hidden", 0, 1)
	node.RenderBox = nil
	paintTextWithinBounds(ctx, node, "hidden", 0, 1)
}
