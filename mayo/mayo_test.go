package mayo

import (
	"os"
	"testing"

	"golang.org/x/net/html"

	cascadia "github.com/andybalholm/cascadia"
	hotdog "github.com/danfragoso/thdwb/hotdog"

	"github.com/stretchr/testify/assert"
)

func TestParseStylesheet(t *testing.T) {
	tests := [...]string{}
	for _, testName := range tests {
		testData, err := os.ReadFile("test_assets/" + testName + ".css")
		if err != nil {
			t.Fatalf("got unexpected error: %s", err)
		}

		t.Log(string(testData))
	}
}

func TestHexStringToColor(t *testing.T) {
	validColors := []string{"#0000FF", "#00f", "#00ff", "#0000ffff"}
	invalidColors := []string{"#00000", "#d0"}

	for _, validColor := range validColors {
		result := HexStringToColor(validColor)
		expected := &hotdog.ColorRGBA{R: 0, G: 0, B: 1, A: 1}
		assert.Equal(t, expected, result, "Expecting: &hotdog.ColorRGBA{0, 0, 1, 1}")
	}

	for _, invalidColor := range invalidColors {
		result := HexStringToColor(invalidColor)
		expected := &hotdog.ColorRGBA{}
		assert.Equal(t, expected, result, "Expecting: &hotdog.ColorRGBA{}")
	}
}

func TestRGBAToColor(t *testing.T) {
	blues := []string{
		"rgba(0, 0, 255)", "rgba(0%, 0%, 100%)",
		"rgb(0, 0, 255, 1)", "rgb(0%, 0%, 100%, 100%)",
	}

	for _, blue := range blues {
		result := RGBAToColor(blue)
		expected := &hotdog.ColorRGBA{R: 0, G: 0, B: 1, A: 1}
		assert.Equal(t, expected, result, "Expecting: &hotdog.ColorRGBA{0, 0, 1, 1}")
	}
}

func TestMapCSSColor(t *testing.T) {
	blues := []string{
		"#0000FF", "blue", "#00f", "rgba(0, 0, 255)", "rgba(0%, 0%, 100%)",
		"rgb(0, 0, 255, 1)", "rgb(0%, 0%, 100%)",
	}

	for _, blue := range blues {
		result := MapCSSColor(blue)
		expected := &hotdog.ColorRGBA{R: 0, G: 0, B: 1, A: 1}
		assert.Equal(t, expected, result, "Expecting: &hotdog.ColorRGBA{0, 0, 1, 1}")
	}
}

func TestCSSInheritance(t *testing.T) {
	// Build a simple DOM tree manually
	root := &hotdog.NodeDOM{
		Element: "html",
		Style: &hotdog.Stylesheet{
			Color:      &hotdog.ColorRGBA{R: 0, G: 0, B: 0, A: 1}, // black
			FontFamily: "Times",
			FontSize:   16,
			LineHeight: 1.2,
		},
		Children: []*hotdog.NodeDOM{},
	}

	body := &hotdog.NodeDOM{
		Element: "body",
		Style: &hotdog.Stylesheet{
			Color:      &hotdog.ColorRGBA{R: 1, G: 0, B: 0, A: 1}, // red
			FontFamily: "Arial",
			FontSize:   18,
			LineHeight: 1.5,
		},
		Parent:   root,
		Children: []*hotdog.NodeDOM{},
	}
	root.Children = append(root.Children, body)

	div := &hotdog.NodeDOM{
		Element:  "div",
		Style:    &hotdog.Stylesheet{}, // No explicit styles - should inherit
		Parent:   body,
		Children: []*hotdog.NodeDOM{},
	}
	body.Children = append(body.Children, div)

	p := &hotdog.NodeDOM{
		Element:  "p",
		Style:    &hotdog.Stylesheet{}, // No explicit styles - should inherit
		Parent:   div,
		Children: []*hotdog.NodeDOM{},
	}
	div.Children = append(div.Children, p)

	span := &hotdog.NodeDOM{
		Element: "span",
		Style: &hotdog.Stylesheet{
			FontSize: 24, // Explicit font-size override
		},
		Parent:   div,
		Children: []*hotdog.NodeDOM{},
	}
	div.Children = append(div.Children, span)

	// Apply inheritance
	ApplyInheritance(root)

	// Body should have explicit styles
	assert.NotNil(t, body.Style.Color)
	assert.Equal(t, 1.0, body.Style.Color.R)
	assert.Equal(t, 0.0, body.Style.Color.G)
	assert.Equal(t, 0.0, body.Style.Color.B)
	assert.Equal(t, "Arial", body.Style.FontFamily)
	assert.Equal(t, 18.0, body.Style.FontSize)
	assert.Equal(t, 1.5, body.Style.LineHeight)

	// Div should inherit from body
	assert.NotNil(t, div.Style.Color)
	assert.Equal(t, 1.0, div.Style.Color.R)
	assert.Equal(t, 0.0, div.Style.Color.G)
	assert.Equal(t, 0.0, div.Style.Color.B)
	assert.Equal(t, "Arial", div.Style.FontFamily)
	assert.Equal(t, 18.0, div.Style.FontSize)
	assert.Equal(t, 1.5, div.Style.LineHeight)

	// p should inherit from div (which inherited from body)
	assert.NotNil(t, p.Style.Color)
	assert.Equal(t, 1.0, p.Style.Color.R)
	assert.Equal(t, "Arial", p.Style.FontFamily)
	assert.Equal(t, 18.0, p.Style.FontSize)
	assert.Equal(t, 1.5, p.Style.LineHeight)

	// span should inherit color, font-family, line-height from div but have overridden font-size
	assert.NotNil(t, span.Style.Color)
	assert.Equal(t, 1.0, span.Style.Color.R)
	assert.Equal(t, "Arial", span.Style.FontFamily)
	assert.Equal(t, 24.0, span.Style.FontSize) // Overridden
	assert.Equal(t, 1.5, span.Style.LineHeight)
}

func TestCSSInheritanceOverride(t *testing.T) {
	// Build a DOM tree with explicit overrides
	root := &hotdog.NodeDOM{
		Element: "html",
		Style: &hotdog.Stylesheet{
			Color:      &hotdog.ColorRGBA{R: 0, G: 0, B: 0, A: 1},
			FontFamily: "Times",
			FontSize:   16,
			LineHeight: 1.2,
		},
		Children: []*hotdog.NodeDOM{},
	}

	body := &hotdog.NodeDOM{
		Element: "body",
		Style: &hotdog.Stylesheet{
			Color:      &hotdog.ColorRGBA{R: 1, G: 0, B: 0, A: 1}, // red
			FontFamily: "Arial",
			FontSize:   16,
			LineHeight: 1.5,
		},
		Parent:   root,
		Children: []*hotdog.NodeDOM{},
	}
	root.Children = append(root.Children, body)

	div := &hotdog.NodeDOM{
		Element: "div",
		Style: &hotdog.Stylesheet{
			Color: &hotdog.ColorRGBA{R: 0, G: 0, B: 1, A: 1}, // blue (override)
		},
		Parent:   body,
		Children: []*hotdog.NodeDOM{},
	}
	body.Children = append(body.Children, div)

	p := &hotdog.NodeDOM{
		Element: "p",
		Style: &hotdog.Stylesheet{
			FontSize: 24, // Override font-size
		},
		Parent:   div,
		Children: []*hotdog.NodeDOM{},
	}
	div.Children = append(div.Children, p)

	span := &hotdog.NodeDOM{
		Element:  "span",
		Style:    &hotdog.Stylesheet{}, // No explicit styles
		Parent:   div,
		Children: []*hotdog.NodeDOM{},
	}
	div.Children = append(div.Children, span)

	// Apply inheritance
	ApplyInheritance(root)

	// Body has explicit styles
	assert.Equal(t, 1.0, body.Style.Color.R)
	assert.Equal(t, "Arial", body.Style.FontFamily)
	assert.Equal(t, 16.0, body.Style.FontSize)

	// Div overrides color but inherits font-family and font-size
	assert.Equal(t, 1.0, div.Style.Color.B) // blue
	assert.Equal(t, "Arial", div.Style.FontFamily)
	assert.Equal(t, 16.0, div.Style.FontSize)

	// p inherits color from div (blue) but overrides font-size
	assert.Equal(t, 1.0, p.Style.Color.B)
	assert.Equal(t, "Arial", p.Style.FontFamily)
	assert.Equal(t, 24.0, p.Style.FontSize) // overridden

	// span inherits color from div (blue) and font-size from body
	assert.Equal(t, 1.0, span.Style.Color.B)
	assert.Equal(t, "Arial", span.Style.FontFamily)
	assert.Equal(t, 16.0, span.Style.FontSize)
}

func TestCSSSpecificity(t *testing.T) {
	// Test that specificity is calculated correctly for different selector types
	// Note: cascadia's specificity calculation may differ from standard CSS specificity
	// for pseudo-classes and pseudo-elements
	tests := []struct {
		selector     string
		expectedSpec cascadia.Specificity
		description  string
		usePseudo    bool // whether to use ParseWithPseudoElement
	}{
		{"div", cascadia.Specificity{0, 0, 1}, "element selector", false},
		{".class", cascadia.Specificity{0, 1, 0}, "class selector", false},
		{"#id", cascadia.Specificity{1, 0, 0}, "ID selector", false},
		{"div.class", cascadia.Specificity{0, 1, 1}, "element + class", false},
		{"#id.class", cascadia.Specificity{1, 1, 0}, "ID + class", false},
		{"div#id.class", cascadia.Specificity{1, 1, 1}, "element + ID + class", false},
		{"div .class", cascadia.Specificity{0, 1, 1}, "descendant element + class", false},
		{"div > .class", cascadia.Specificity{0, 1, 1}, "child element + class", false},
		{"div + .class", cascadia.Specificity{0, 1, 1}, "adjacent sibling element + class", false},
		{"div ~ .class", cascadia.Specificity{0, 1, 1}, "general sibling element + class", false},
		{".class1.class2", cascadia.Specificity{0, 2, 0}, "two classes", false},
		{"#id1 #id2", cascadia.Specificity{2, 0, 0}, "two IDs", false},
		{"div:hover", cascadia.Specificity{0, 0, 1}, "element + pseudo-class (cascadia treats as element)", false},
		{"::before", cascadia.Specificity{0, 0, 1}, "pseudo-element", true},
		{"div::before", cascadia.Specificity{0, 0, 2}, "element + pseudo-element", true},
	}

	for _, tc := range tests {
		var sel cascadia.Sel
		var err error
		if tc.usePseudo {
			sel, err = cascadia.ParseWithPseudoElement(tc.selector)
		} else {
			sel, err = cascadia.Parse(tc.selector)
		}
		assert.NoError(t, err, "Failed to parse selector: %s", tc.selector)
		assert.Equal(t, tc.expectedSpec, sel.Specificity(), tc.description)
	}
}

func TestCascadeSpecificityOrder(t *testing.T) {
	// Test that higher specificity selectors override lower specificity ones
	css := `
		div { color: red; }
		.class { color: blue; }
		#id { color: lime; }
	`

	doc := &hotdog.Document{
		StyleSheets: ParseStylesheet(css),
	}

	// Build DOM: <div id="id" class="class">Test</div>
	root := &hotdog.NodeDOM{
		Element: "html",
		Style:   &hotdog.Stylesheet{},
		Children: []*hotdog.NodeDOM{
			{
				Element: "body",
				Style:   &hotdog.Stylesheet{},
				Children: []*hotdog.NodeDOM{
					{
						Element: "div",
						Style:   &hotdog.Stylesheet{},
						Attributes: []*hotdog.Attribute{
							{Name: "id", Value: "id"},
							{Name: "class", Value: "class"},
						},
					},
				},
			},
		},
	}
	doc.DOM = root

	// Create a minimal HTML tree for cascadia matching
	htmlRoot := &html.Node{
		Type: html.ElementNode,
		Data: "html",
		FirstChild: &html.Node{
			Type: html.ElementNode,
			Data: "body",
			FirstChild: &html.Node{
				Type: html.ElementNode,
				Data: "div",
				Attr: []html.Attribute{
					{Key: "id", Val: "id"},
					{Key: "class", Val: "class"},
				},
			},
		},
	}
	doc.HTMLRoot = htmlRoot

	// Link DOM nodes to HTML nodes
	root.HTMLNode = htmlRoot
	root.Children[0].HTMLNode = htmlRoot.FirstChild
	root.Children[0].Children[0].HTMLNode = htmlRoot.FirstChild.FirstChild

	ApplyStylesheets(doc)

	// The div has id="id" and class="class", so #id selector (specificity 1,0,0) should win
	// over .class (0,1,0) and div (0,0,1)
	// lime color is {R: 0.0, G: 1.0, B: 0.0, A: 1.0}
	divNode := root.Children[0].Children[0]
	assert.NotNil(t, divNode.Style.Color)
	assert.Equal(t, 0.0, divNode.Style.Color.R) // lime
	assert.Equal(t, 1.0, divNode.Style.Color.G)
	assert.Equal(t, 0.0, divNode.Style.Color.B)
}

func TestCascadeSourceOrder(t *testing.T) {
	// Test that later rules with same specificity override earlier ones
	css := `
		.class1 { color: red; }
		.class2 { color: blue; }
	`

	doc := &hotdog.Document{
		StyleSheets: ParseStylesheet(css),
	}

	// Build DOM: <div class="class1 class2">Test</div>
	root := &hotdog.NodeDOM{
		Element: "html",
		Style:   &hotdog.Stylesheet{},
		Children: []*hotdog.NodeDOM{
			{
				Element: "body",
				Style:   &hotdog.Stylesheet{},
				Children: []*hotdog.NodeDOM{
					{
						Element: "div",
						Style:   &hotdog.Stylesheet{},
						Attributes: []*hotdog.Attribute{
							{Name: "class", Value: "class1 class2"},
						},
					},
				},
			},
		},
	}
	doc.DOM = root

	htmlRoot := &html.Node{
		Type: html.ElementNode,
		Data: "html",
		FirstChild: &html.Node{
			Type: html.ElementNode,
			Data: "body",
			FirstChild: &html.Node{
				Type: html.ElementNode,
				Data: "div",
				Attr: []html.Attribute{
					{Key: "class", Val: "class1 class2"},
				},
			},
		},
	}
	doc.HTMLRoot = htmlRoot

	root.HTMLNode = htmlRoot
	root.Children[0].HTMLNode = htmlRoot.FirstChild
	root.Children[0].Children[0].HTMLNode = htmlRoot.FirstChild.FirstChild

	ApplyStylesheets(doc)

	// Both selectors have same specificity (0,1,0), so later rule (.class2) should win
	divNode := root.Children[0].Children[0]
	assert.NotNil(t, divNode.Style.Color)
	assert.Equal(t, 0.0, divNode.Style.Color.R) // blue
	assert.Equal(t, 0.0, divNode.Style.Color.G)
	assert.Equal(t, 1.0, divNode.Style.Color.B)
}

func TestGetDefaultElementDisplay(t *testing.T) {
	tests := []struct {
		element  string
		expected string
	}{
		{"script", "none"},
		{"style", "none"},
		{"meta", "none"},
		{"link", "none"},
		{"head", "none"},
		{"title", "none"},
		{"li", "list-item"},
		{"a", "inline"},
		{"span", "inline"},
		{"div", "block"},
		{"p", "block"},
		{"unknown", "block"},
	}

	for _, tc := range tests {
		result := getDefaultElementDisplay(tc.element)
		assert.Equal(t, tc.expected, result, "Element: %s", tc.element)
	}
}

func TestGetDefaultElementFontWeight(t *testing.T) {
	tests := []struct {
		element  string
		expected int
	}{
		{"h1", 600},
		{"h2", 600},
		{"h3", 600},
		{"h4", 600},
		{"h5", 600},
		{"h6", 600},
		{"div", 400},
		{"p", 400},
		{"span", 400},
	}

	for _, tc := range tests {
		result := getDefaultElementFontWeight(tc.element)
		assert.Equal(t, tc.expected, result, "Element: %s", tc.element)
	}
}

func TestMapSizeValue(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"16px", 16},
		{"1.5em", 1.5},
		{"100%", 100},
		{"12.5pt", 12.5},
		{"2rem", 2},
		{"invalid", 14}, // default
		{"", 14},        // default
	}

	for _, tc := range tests {
		result := mapSizeValue(tc.input)
		assert.Equal(t, tc.expected, result, "Input: %s", tc.input)
	}
}

func TestMapIntValue(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"10", 10},
		{"400", 400},
		{"600", 600},
		{"z-index: 5", 5},
		{"invalid", 0},
		{"", 0},
	}

	for _, tc := range tests {
		result := mapIntValue(tc.input)
		assert.Equal(t, tc.expected, result, "Input: %s", tc.input)
	}
}

func TestMapPropToStylesheet(t *testing.T) {
	sheet := &hotdog.Stylesheet{}

	// Test color
	mapPropToStylesheet(sheet, []string{"color", "red"})
	assert.NotNil(t, sheet.Color)
	assert.Equal(t, 1.0, sheet.Color.R)

	// Test background-color
	mapPropToStylesheet(sheet, []string{"background-color", "blue"})
	assert.NotNil(t, sheet.BackgroundColor)
	assert.Equal(t, 1.0, sheet.BackgroundColor.B)

	// Test font-size
	mapPropToStylesheet(sheet, []string{"font-size", "20px"})
	assert.Equal(t, 20.0, sheet.FontSize)

	// Test font-weight
	mapPropToStylesheet(sheet, []string{"font-weight", "600"})
	assert.Equal(t, 600, sheet.FontWeight)

	// Test font-family
	mapPropToStylesheet(sheet, []string{"font-family", "\"Arial\""})
	assert.Equal(t, "Arial", sheet.FontFamily)

	// Test display
	mapPropToStylesheet(sheet, []string{"display", "flex"})
	assert.Equal(t, "flex", sheet.Display)

	// Test position
	mapPropToStylesheet(sheet, []string{"position", "absolute"})
	assert.Equal(t, "absolute", sheet.Position)

	// Test width/height
	mapPropToStylesheet(sheet, []string{"width", "100px"})
	assert.Equal(t, 100.0, sheet.Width)
	mapPropToStylesheet(sheet, []string{"height", "50px"})
	assert.Equal(t, 50.0, sheet.Height)

	// Test margin
	mapPropToStylesheet(sheet, []string{"margin", "10px"})
	assert.Equal(t, 10.0, sheet.MarginTop)
	assert.Equal(t, 10.0, sheet.MarginRight)
	assert.Equal(t, 10.0, sheet.MarginBottom)
	assert.Equal(t, 10.0, sheet.MarginLeft)

	// Test padding
	mapPropToStylesheet(sheet, []string{"padding", "5px"})
	assert.Equal(t, 5.0, sheet.PaddingTop)
	assert.Equal(t, 5.0, sheet.PaddingRight)
	assert.Equal(t, 5.0, sheet.PaddingBottom)
	assert.Equal(t, 5.0, sheet.PaddingLeft)

	// Test border
	mapPropToStylesheet(sheet, []string{"border-width", "2px"})
	assert.Equal(t, 2.0, sheet.BorderTopWidth)
}

func TestParseInlineStylesheet(t *testing.T) {
	attributes := []*hotdog.Attribute{
		{Name: "style", Value: "color: red; font-size: 20px; background-color: blue"},
	}
	sheet := &hotdog.Stylesheet{}
	result := parseInlineStylesheet(attributes, sheet)

	assert.NotNil(t, result.Color)
	assert.Equal(t, 1.0, result.Color.R)
	assert.Equal(t, 20.0, result.FontSize)
	assert.NotNil(t, result.BackgroundColor)
	assert.Equal(t, 1.0, result.BackgroundColor.B)
}

func TestHasInlineStyle(t *testing.T) {
	// With style attribute
	attributesWithStyle := []*hotdog.Attribute{
		{Name: "style", Value: "color: red"},
		{Name: "class", Value: "test"},
	}
	assert.True(t, hasInlineStyle(attributesWithStyle))

	// Without style attribute
	attributesWithoutStyle := []*hotdog.Attribute{
		{Name: "class", Value: "test"},
		{Name: "id", Value: "test"},
	}
	assert.False(t, hasInlineStyle(attributesWithoutStyle))

	// Empty attributes
	assert.False(t, hasInlineStyle([]*hotdog.Attribute{}))
}

func TestGetElementStylesheet(t *testing.T) {
	// Test with no attributes
	sheet := GetElementStylesheet("div", []*hotdog.Attribute{})
	assert.NotNil(t, sheet)
	assert.Equal(t, "block", sheet.Display)
	assert.Equal(t, 400, sheet.FontWeight)
	assert.NotNil(t, sheet.Color)
	assert.Equal(t, 14.0, sheet.FontSize)

	// Test with style attribute
	attributes := []*hotdog.Attribute{
		{Name: "style", Value: "color: red; font-size: 24px"},
	}
	sheet = GetElementStylesheet("span", attributes)
	assert.NotNil(t, sheet.Color)
	assert.Equal(t, 1.0, sheet.Color.R)
	assert.Equal(t, 24.0, sheet.FontSize)
	assert.Equal(t, "inline", sheet.Display)

	// Test heading element
	sheet = GetElementStylesheet("h1", []*hotdog.Attribute{})
	assert.Equal(t, 600, sheet.FontWeight)
}

func TestParseMargin(t *testing.T) {
	sheet := &hotdog.Stylesheet{}

	// 1 value
	parseMargin("10px", sheet)
	assert.Equal(t, 10.0, sheet.MarginTop)
	assert.Equal(t, 10.0, sheet.MarginRight)
	assert.Equal(t, 10.0, sheet.MarginBottom)
	assert.Equal(t, 10.0, sheet.MarginLeft)

	// 2 values
	sheet2 := &hotdog.Stylesheet{}
	parseMargin("10px 20px", sheet2)
	assert.Equal(t, 10.0, sheet2.MarginTop)
	assert.Equal(t, 10.0, sheet2.MarginBottom)
	assert.Equal(t, 20.0, sheet2.MarginRight)
	assert.Equal(t, 20.0, sheet2.MarginLeft)

	// 3 values
	sheet3 := &hotdog.Stylesheet{}
	parseMargin("10px 20px 30px", sheet3)
	assert.Equal(t, 10.0, sheet3.MarginTop)
	assert.Equal(t, 20.0, sheet3.MarginRight)
	assert.Equal(t, 30.0, sheet3.MarginBottom)
	assert.Equal(t, 20.0, sheet3.MarginLeft)

	// 4 values
	sheet4 := &hotdog.Stylesheet{}
	parseMargin("10px 20px 30px 40px", sheet4)
	assert.Equal(t, 10.0, sheet4.MarginTop)
	assert.Equal(t, 20.0, sheet4.MarginRight)
	assert.Equal(t, 30.0, sheet4.MarginBottom)
	assert.Equal(t, 40.0, sheet4.MarginLeft)
}

func TestParsePadding(t *testing.T) {
	sheet := &hotdog.Stylesheet{}

	// 1 value
	parsePadding("10px", sheet)
	assert.Equal(t, 10.0, sheet.PaddingTop)
	assert.Equal(t, 10.0, sheet.PaddingRight)
	assert.Equal(t, 10.0, sheet.PaddingBottom)
	assert.Equal(t, 10.0, sheet.PaddingLeft)

	// 2 values
	sheet2 := &hotdog.Stylesheet{}
	parsePadding("10px 20px", sheet2)
	assert.Equal(t, 10.0, sheet2.PaddingTop)
	assert.Equal(t, 10.0, sheet2.PaddingBottom)
	assert.Equal(t, 20.0, sheet2.PaddingRight)
	assert.Equal(t, 20.0, sheet2.PaddingLeft)

	// 3 values
	sheet3 := &hotdog.Stylesheet{}
	parsePadding("10px 20px 30px", sheet3)
	assert.Equal(t, 10.0, sheet3.PaddingTop)
	assert.Equal(t, 20.0, sheet3.PaddingRight)
	assert.Equal(t, 30.0, sheet3.PaddingBottom)
	assert.Equal(t, 20.0, sheet3.PaddingLeft)

	// 4 values
	sheet4 := &hotdog.Stylesheet{}
	parsePadding("10px 20px 30px 40px", sheet4)
	assert.Equal(t, 10.0, sheet4.PaddingTop)
	assert.Equal(t, 20.0, sheet4.PaddingRight)
	assert.Equal(t, 30.0, sheet4.PaddingBottom)
	assert.Equal(t, 40.0, sheet4.PaddingLeft)
}

func TestReflowNode(t *testing.T) {
	// Build a simple DOM tree
	root := &hotdog.NodeDOM{
		Element: "html",
		Style: &hotdog.Stylesheet{
			Height: 100,
		},
		Children: []*hotdog.NodeDOM{},
	}

	body := &hotdog.NodeDOM{
		Element: "body",
		Style: &hotdog.Stylesheet{
			Height: 80,
		},
		Parent:   root,
		Children: []*hotdog.NodeDOM{},
	}
	root.Children = append(root.Children, body)

	div := &hotdog.NodeDOM{
		Element: "div",
		Style: &hotdog.Stylesheet{
			Height: 50,
		},
		Parent:   body,
		Children: []*hotdog.NodeDOM{},
	}
	body.Children = append(body.Children, div)

	// Call ReflowNode
	offset := ReflowNode(root, root, 0)

	// The function returns the height of the last processed node
	// and sets Top on each node
	// The function processes children first, then sets Top = Height + siblingsOffset
	// For leaf nodes (div), siblingsOffset = 0, so Top = 50, returns 50
	// For body, siblingsOffset = 50 (from div), so Top = 80 + 50 = 130, returns 80
	// For root, siblingsOffset = 80 (from body), so Top = 100 + 80 = 180, returns 100
	assert.Equal(t, 100.0, offset)
	assert.Equal(t, 180.0, root.Style.Top)
	assert.Equal(t, 130.0, body.Style.Top)
	assert.Equal(t, 50.0, div.Style.Top)
}

func TestMergeStylesheet(t *testing.T) {
	dest := &hotdog.Stylesheet{
		Color:      &hotdog.ColorRGBA{R: 1, G: 0, B: 0, A: 1}, // red
		FontSize:   16,
		FontWeight: 400,
		Display:    "block",
	}

	src := &hotdog.Stylesheet{
		Color:           &hotdog.ColorRGBA{R: 0, G: 1, B: 0, A: 1}, // green (should override)
		BackgroundColor: &hotdog.ColorRGBA{R: 0, G: 0, B: 1, A: 1}, // blue (new)
		FontSize:        24,                                        // should override
		FontFamily:      "Arial",                                   // new
		FontWeight:      600,                                       // should override
		LineHeight:      1.5,                                       // new
		TextAlign:       "center",                                  // new
		Display:         "flex",                                    // should override
		Position:        "absolute",                                // new
		FlexDirection:   "column",                                  // new
		Width:           100,                                       // new
		Height:          200,                                       // new
		MarginTop:       10,                                        // new
		PaddingTop:      5,                                         // new
		BorderTopWidth:  2,                                         // new
		BorderTopColor:  &hotdog.ColorRGBA{R: 0, G: 0, B: 0, A: 1}, // new
		OverflowX:       "auto",                                    // new
		Opacity:         0.5,                                       // new
	}

	mergeStylesheet(dest, src)

	// Check overridden values
	assert.Equal(t, 0.0, dest.Color.R) // green
	assert.Equal(t, 1.0, dest.Color.G)
	assert.Equal(t, 24.0, dest.FontSize)
	assert.Equal(t, 600, dest.FontWeight)
	assert.Equal(t, "flex", dest.Display)

	// Check new values
	assert.NotNil(t, dest.BackgroundColor)
	assert.Equal(t, 1.0, dest.BackgroundColor.B)
	assert.Equal(t, "Arial", dest.FontFamily)
	assert.Equal(t, 1.5, dest.LineHeight)
	assert.Equal(t, "center", dest.TextAlign)
	assert.Equal(t, "absolute", dest.Position)
	assert.Equal(t, "column", dest.FlexDirection)
	assert.Equal(t, 100.0, dest.Width)
	assert.Equal(t, 200.0, dest.Height)
	assert.Equal(t, 10.0, dest.MarginTop)
	assert.Equal(t, 5.0, dest.PaddingTop)
	assert.Equal(t, 2.0, dest.BorderTopWidth)
	assert.NotNil(t, dest.BorderTopColor)
	assert.Equal(t, "auto", dest.OverflowX)
	assert.Equal(t, 0.5, dest.Opacity)
}

func TestMergeStylesheetNilSrc(t *testing.T) {
	dest := &hotdog.Stylesheet{
		Color:    &hotdog.ColorRGBA{R: 1, G: 0, B: 0, A: 1},
		FontSize: 16,
	}

	// Should not panic
	mergeStylesheet(dest, nil)

	// Values should remain unchanged
	assert.NotNil(t, dest.Color)
	assert.Equal(t, 16.0, dest.FontSize)
}

func TestInheritFromParent(t *testing.T) {
	child := &hotdog.Stylesheet{
		FontSize: 24, // explicitly set - should NOT inherit
	}

	parent := &hotdog.Stylesheet{
		Color:          &hotdog.ColorRGBA{R: 1, G: 0, B: 0, A: 1}, // red
		FontFamily:     "Arial",
		FontSize:       16, // should NOT override child's 24
		FontWeight:     600,
		LineHeight:     1.5,
		TextAlign:      "center",
		TextDecoration: "underline",
		WhiteSpace:     "nowrap",
		Visibility:     "hidden",
		Opacity:        0.5,
	}

	inheritFromParent(child, parent)

	// Child's explicit values should remain
	assert.Equal(t, 24.0, child.FontSize)

	// Child should inherit parent's values for unset properties
	assert.NotNil(t, child.Color)
	assert.Equal(t, 1.0, child.Color.R)
	assert.Equal(t, "Arial", child.FontFamily)
	assert.Equal(t, 600, child.FontWeight)
	assert.Equal(t, 1.5, child.LineHeight)
	assert.Equal(t, "center", child.TextAlign)
	assert.Equal(t, "underline", child.TextDecoration)
	assert.Equal(t, "nowrap", child.WhiteSpace)
	assert.Equal(t, "hidden", child.Visibility)
	assert.Equal(t, 0.5, child.Opacity)
}

func TestInheritFromParentNil(t *testing.T) {
	child := &hotdog.Stylesheet{
		FontSize: 16,
	}

	// Should not panic
	inheritFromParent(child, nil)
	inheritFromParent(nil, &hotdog.Stylesheet{Color: &hotdog.ColorRGBA{R: 1, G: 0, B: 0, A: 1}})
	inheritFromParent(nil, nil)

	assert.Equal(t, 16.0, child.FontSize)
}

func TestApplyInheritanceRecursive(t *testing.T) {
	root := &hotdog.NodeDOM{
		Element: "html",
		Style: &hotdog.Stylesheet{
			Color:      &hotdog.ColorRGBA{R: 0, G: 0, B: 0, A: 1}, // black
			FontFamily: "Times",
			FontSize:   16,
			LineHeight: 1.2,
		},
		Children: []*hotdog.NodeDOM{},
	}

	body := &hotdog.NodeDOM{
		Element: "body",
		Style: &hotdog.Stylesheet{
			Color:      &hotdog.ColorRGBA{R: 1, G: 0, B: 0, A: 1}, // red
			FontFamily: "Arial",
			FontSize:   18,
			LineHeight: 1.5,
		},
		Parent:   root,
		Children: []*hotdog.NodeDOM{},
	}
	root.Children = append(root.Children, body)

	div := &hotdog.NodeDOM{
		Element:  "div",
		Style:    &hotdog.Stylesheet{}, // No explicit styles - should inherit
		Parent:   body,
		Children: []*hotdog.NodeDOM{},
	}
	body.Children = append(body.Children, div)

	p := &hotdog.NodeDOM{
		Element:  "p",
		Style:    &hotdog.Stylesheet{}, // No explicit styles - should inherit
		Parent:   div,
		Children: []*hotdog.NodeDOM{},
	}
	div.Children = append(div.Children, p)

	span := &hotdog.NodeDOM{
		Element: "span",
		Style: &hotdog.Stylesheet{
			FontSize: 24, // Explicit font-size override
		},
		Parent:   div,
		Children: []*hotdog.NodeDOM{},
	}
	div.Children = append(div.Children, span)

	// Apply inheritance recursively
	applyInheritanceRecursive(root, nil)

	// Root should have its own styles
	assert.NotNil(t, root.Style.Color)
	assert.Equal(t, 0.0, root.Style.Color.R)
	assert.Equal(t, "Times", root.Style.FontFamily)
	assert.Equal(t, 16.0, root.Style.FontSize)
	assert.Equal(t, 1.2, root.Style.LineHeight)

	// Body should have explicit styles
	assert.NotNil(t, body.Style.Color)
	assert.Equal(t, 1.0, body.Style.Color.R)
	assert.Equal(t, "Arial", body.Style.FontFamily)
	assert.Equal(t, 18.0, body.Style.FontSize)
	assert.Equal(t, 1.5, body.Style.LineHeight)

	// Div should inherit from body
	assert.NotNil(t, div.Style.Color)
	assert.Equal(t, 1.0, div.Style.Color.R)
	assert.Equal(t, "Arial", div.Style.FontFamily)
	assert.Equal(t, 18.0, div.Style.FontSize)
	assert.Equal(t, 1.5, div.Style.LineHeight)

	// p should inherit from div (which inherited from body)
	assert.NotNil(t, p.Style.Color)
	assert.Equal(t, 1.0, p.Style.Color.R)
	assert.Equal(t, "Arial", p.Style.FontFamily)
	assert.Equal(t, 18.0, p.Style.FontSize)
	assert.Equal(t, 1.5, p.Style.LineHeight)

	// span should inherit color, font-family, line-height from div but have overridden font-size
	assert.NotNil(t, span.Style.Color)
	assert.Equal(t, 1.0, span.Style.Color.R)
	assert.Equal(t, "Arial", span.Style.FontFamily)
	assert.Equal(t, 24.0, span.Style.FontSize) // Overridden
	assert.Equal(t, 1.5, span.Style.LineHeight)
}

func TestApplyInheritance(t *testing.T) {
	root := &hotdog.NodeDOM{
		Element: "html",
		Style: &hotdog.Stylesheet{
			Color:      &hotdog.ColorRGBA{R: 0, G: 0, B: 0, A: 1},
			FontFamily: "Times",
			FontSize:   16,
			LineHeight: 1.2,
		},
		Children: []*hotdog.NodeDOM{},
	}

	body := &hotdog.NodeDOM{
		Element: "body",
		Style: &hotdog.Stylesheet{
			Color:      &hotdog.ColorRGBA{R: 1, G: 0, B: 0, A: 1},
			FontFamily: "Arial",
			FontSize:   18,
			LineHeight: 1.5,
		},
		Parent:   root,
		Children: []*hotdog.NodeDOM{},
	}
	root.Children = append(root.Children, body)

	div := &hotdog.NodeDOM{
		Element:  "div",
		Style:    &hotdog.Stylesheet{},
		Parent:   body,
		Children: []*hotdog.NodeDOM{},
	}
	body.Children = append(body.Children, div)

	// Apply inheritance
	ApplyInheritance(root)

	// Root should have its own styles
	assert.NotNil(t, root.Style.Color)
	assert.Equal(t, 0.0, root.Style.Color.R)

	// Body should have explicit styles
	assert.NotNil(t, body.Style.Color)
	assert.Equal(t, 1.0, body.Style.Color.R)

	// Div should inherit from body
	assert.NotNil(t, div.Style.Color)
	assert.Equal(t, 1.0, div.Style.Color.R)
	assert.Equal(t, "Arial", div.Style.FontFamily)
	assert.Equal(t, 18.0, div.Style.FontSize)
	assert.Equal(t, 1.5, div.Style.LineHeight)
}

func TestFindNodeDOMByHTMLNode(t *testing.T) {
	root := &hotdog.NodeDOM{
		Element:  "html",
		Style:    &hotdog.Stylesheet{},
		Children: []*hotdog.NodeDOM{},
	}

	body := &hotdog.NodeDOM{
		Element:  "body",
		Style:    &hotdog.Stylesheet{},
		Parent:   root,
		Children: []*hotdog.NodeDOM{},
	}
	root.Children = append(root.Children, body)

	div := &hotdog.NodeDOM{
		Element:  "div",
		Style:    &hotdog.Stylesheet{},
		Parent:   body,
		Children: []*hotdog.NodeDOM{},
	}
	body.Children = append(body.Children, div)

	// Create matching HTML tree
	htmlRoot := &html.Node{
		Type: html.ElementNode,
		Data: "html",
		FirstChild: &html.Node{
			Type: html.ElementNode,
			Data: "body",
			FirstChild: &html.Node{
				Type: html.ElementNode,
				Data: "div",
			},
		},
	}

	// Link DOM nodes to HTML nodes
	root.HTMLNode = htmlRoot
	body.HTMLNode = htmlRoot.FirstChild
	div.HTMLNode = htmlRoot.FirstChild.FirstChild

	// Find root
	result := findNodeDOMByHTMLNode(root, htmlRoot)
	assert.Equal(t, root, result)

	// Find body
	result = findNodeDOMByHTMLNode(root, htmlRoot.FirstChild)
	assert.Equal(t, body, result)

	// Find div
	result = findNodeDOMByHTMLNode(root, htmlRoot.FirstChild.FirstChild)
	assert.Equal(t, div, result)

	// Find non-existent
	result = findNodeDOMByHTMLNode(root, &html.Node{Type: html.ElementNode, Data: "span"})
	assert.Nil(t, result)

	// Find with nil root
	result = findNodeDOMByHTMLNode(nil, htmlRoot)
	assert.Nil(t, result)
}

func TestApplyStylesheetsWithMultipleSelectors(t *testing.T) {
	css := `
		div { color: red; }
		.class1 { color: blue; }
		.class2 { color: green; }
		#id { color: yellow; }
	`

	doc := &hotdog.Document{
		StyleSheets: ParseStylesheet(css),
	}

	// Build DOM: <div id="id" class="class1 class2">Test</div>
	root := &hotdog.NodeDOM{
		Element: "html",
		Style:   &hotdog.Stylesheet{},
		Children: []*hotdog.NodeDOM{
			{
				Element: "body",
				Style:   &hotdog.Stylesheet{},
				Children: []*hotdog.NodeDOM{
					{
						Element: "div",
						Style:   &hotdog.Stylesheet{},
						Attributes: []*hotdog.Attribute{
							{Name: "id", Value: "id"},
							{Name: "class", Value: "class1 class2"},
						},
					},
				},
			},
		},
	}
	doc.DOM = root

	htmlRoot := &html.Node{
		Type: html.ElementNode,
		Data: "html",
		FirstChild: &html.Node{
			Type: html.ElementNode,
			Data: "body",
			FirstChild: &html.Node{
				Type: html.ElementNode,
				Data: "div",
				Attr: []html.Attribute{
					{Key: "id", Val: "id"},
					{Key: "class", Val: "class1 class2"},
				},
			},
		},
	}
	doc.HTMLRoot = htmlRoot

	root.HTMLNode = htmlRoot
	root.Children[0].HTMLNode = htmlRoot.FirstChild
	root.Children[0].Children[0].HTMLNode = htmlRoot.FirstChild.FirstChild

	ApplyStylesheets(doc)

	// #id has highest specificity (1,0,0), should win
	divNode := root.Children[0].Children[0]
	assert.NotNil(t, divNode.Style.Color)
	assert.Equal(t, 1.0, divNode.Style.Color.R) // yellow
	assert.Equal(t, 1.0, divNode.Style.Color.G)
	assert.Equal(t, 0.0, divNode.Style.Color.B)
}

func TestApplyStylesheetsWithNestedElements(t *testing.T) {
	css := `
		div { color: blue; }
		p { font-size: 20px; }
	`

	doc := &hotdog.Document{
		StyleSheets: ParseStylesheet(css),
	}

	root := &hotdog.NodeDOM{
		Element: "html",
		Style:   &hotdog.Stylesheet{},
		Children: []*hotdog.NodeDOM{
			{
				Element: "body",
				Style:   &hotdog.Stylesheet{},
				Children: []*hotdog.NodeDOM{
					{
						Element: "div",
						Style:   &hotdog.Stylesheet{},
						Children: []*hotdog.NodeDOM{
							{
								Element: "p",
								Style:   &hotdog.Stylesheet{},
							},
						},
					},
				},
			},
		},
	}
	doc.DOM = root

	htmlRoot := &html.Node{
		Type: html.ElementNode,
		Data: "html",
		FirstChild: &html.Node{
			Type: html.ElementNode,
			Data: "body",
			FirstChild: &html.Node{
				Type: html.ElementNode,
				Data: "div",
				FirstChild: &html.Node{
					Type: html.ElementNode,
					Data: "p",
				},
			},
		},
	}
	doc.HTMLRoot = htmlRoot

	root.HTMLNode = htmlRoot
	root.Children[0].HTMLNode = htmlRoot.FirstChild
	root.Children[0].Children[0].HTMLNode = htmlRoot.FirstChild.FirstChild
	root.Children[0].Children[0].Children[0].HTMLNode = htmlRoot.FirstChild.FirstChild.FirstChild

	ApplyStylesheets(doc)

	// Div gets blue color
	divNode := root.Children[0].Children[0]
	assert.NotNil(t, divNode.Style.Color)
	assert.Equal(t, 0.0, divNode.Style.Color.R)
	assert.Equal(t, 1.0, divNode.Style.Color.B)

	// P gets 20px font-size, inherits blue color from div
	pNode := root.Children[0].Children[0].Children[0]
	assert.Equal(t, 20.0, pNode.Style.FontSize)
	if pNode.Style.Color != nil {
		assert.Equal(t, 0.0, pNode.Style.Color.R)
		assert.Equal(t, 1.0, pNode.Style.Color.B)
	}
}
