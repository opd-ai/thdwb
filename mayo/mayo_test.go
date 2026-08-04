package mayo

import (
	"io/ioutil"
	"testing"

	"golang.org/x/net/html"

	cascadia "github.com/andybalholm/cascadia"
	hotdog "github.com/danfragoso/thdwb/hotdog"

	"github.com/stretchr/testify/assert"
)

func TestParseStylesheet(t *testing.T) {
	tests := [...]string{}
	for _, testName := range tests {
		testData, err := ioutil.ReadFile("test_assets/" + testName + ".css")
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

func TestCascadeOriginPriority(t *testing.T) {
	// Test that author styles override user-agent styles
	// (We simulate this by manually setting origins)
	css := `
		div { color: red; }
	`

	styleSheets := ParseStylesheet(css)
	// Simulate user-agent stylesheet by changing origin
	if len(styleSheets) > 0 {
		styleSheets[0].Origin = "user-agent"
	}

	// Add author stylesheet
	authorCSS := `
		div { color: blue; }
	`
	authorSheets := ParseStylesheet(authorCSS)
	styleSheets = append(styleSheets, authorSheets...)

	doc := &hotdog.Document{
		StyleSheets: styleSheets,
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
			},
		},
	}
	doc.HTMLRoot = htmlRoot

	root.HTMLNode = htmlRoot
	root.Children[0].HTMLNode = htmlRoot.FirstChild
	root.Children[0].Children[0].HTMLNode = htmlRoot.FirstChild.FirstChild

	ApplyStylesheets(doc)

	// Author stylesheet should win over user-agent
	divNode := root.Children[0].Children[0]
	assert.NotNil(t, divNode.Style.Color)
	assert.Equal(t, 0.0, divNode.Style.Color.R) // blue
	assert.Equal(t, 0.0, divNode.Style.Color.G)
	assert.Equal(t, 1.0, divNode.Style.Color.B)
}

func TestParseStylesheetSpecificity(t *testing.T) {
	// Test that ParseStylesheet correctly populates specificity
	css := `
		div { color: red; }
		.class { color: blue; }
		#id { color: green; }
	`

	sheets := ParseStylesheet(css)
	assert.Len(t, sheets, 3)

	// Check specificity values
	assert.Equal(t, cascadia.Specificity{0, 0, 1}, sheets[0].Specificity) // div
	assert.Equal(t, cascadia.Specificity{0, 1, 0}, sheets[1].Specificity) // .class
	assert.Equal(t, cascadia.Specificity{1, 0, 0}, sheets[2].Specificity) // #id

	// Check origin is set to "author"
	for _, s := range sheets {
		assert.Equal(t, "author", s.Origin)
	}

	// Check index is set correctly
	assert.Equal(t, 0, sheets[0].Index)
	assert.Equal(t, 1, sheets[1].Index)
	assert.Equal(t, 2, sheets[2].Index)
}
