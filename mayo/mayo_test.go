package mayo

import (
	"io/ioutil"
	"testing"

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
