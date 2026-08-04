package mayo

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/andybalholm/cascadia"
	hotdog "github.com/danfragoso/thdwb/hotdog"
	"golang.org/x/net/html"
)

// Inheritable properties - CSS properties that inherit by default
var inheritableProperties = map[string]bool{
	"color":          true,
	"font-family":    true,
	"font-size":      true,
	"font-weight":    true,
	"font-style":     true,
	"line-height":    true,
	"letter-spacing": true,
	"text-align":     true,
	"text-indent":    true,
	"text-transform": true,
	"visibility":     true,
	"white-space":    true,
	"word-spacing":   true,
	"list-style":     true,
	"cursor":         true,
	"direction":      true,
	"unicode-bidi":   true,
	"opacity":        true,
}

func getDefaultElementDisplay(element string) string {
	displayType := "block"

	switch element {
	case "script", "style", "meta", "link", "head", "title":
		displayType = "none"
	case "li":
		displayType = "list-item"
	case "html:text", "a", "abbr", "acronym", "b", "bdo", "big", "br",
		"button", "cite", "code", "dfn", "em", "i", "img", "input", "kbd",
		"label", "map", "object", "output", "q", "samp", "select", "small",
		"span", "strong", "sub", "sup", "textarea", "time", "tt", "var", "font":
		displayType = "inline"
	}

	return displayType
}

func getDefaultElementFontWeight(element string) int {
	switch element {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return 600
	}

	return 400
}

func mapSizeValue(sizeValue string) float64 {
	re := regexp.MustCompile(`[0-9]+(\.[0-9]+)?`)
	valueString := re.FindString(sizeValue)
	if valueString == "" {
		return float64(14)
	}
	value, err := strconv.ParseFloat(valueString, 64)
	if err != nil {
		return float64(14)
	}

	return value
}

func mapIntValue(intValue string) int {
	re := regexp.MustCompile(`[0-9]+`)
	valueString := re.FindString(intValue)
	if valueString == "" {
		return 0
	}
	value, err := strconv.Atoi(valueString)
	if err != nil {
		return 0
	}
	return value
}

func mapPropToStylesheet(parsedStyleSheet *hotdog.Stylesheet, propSlice []string) *hotdog.Stylesheet {
	propName := strings.TrimSpace(strings.ToLower(propSlice[0]))
	propValue := strings.TrimSpace(propSlice[1])

	switch propName {
	case "color":
		parsedStyleSheet.Color = MapCSSColor(propValue)
	case "background-color":
		parsedStyleSheet.BackgroundColor = MapCSSColor(propValue)
	case "font-size":
		parsedStyleSheet.FontSize = mapSizeValue(propValue)
	case "font-weight":
		parsedStyleSheet.FontWeight = mapIntValue(propValue)
	case "font-family":
		parsedStyleSheet.FontFamily = strings.Trim(propValue, "\"'")
	case "font-style":
		parsedStyleSheet.FontStyle = propValue
	case "line-height":
		parsedStyleSheet.LineHeight = mapSizeValue(propValue)
	case "letter-spacing":
		parsedStyleSheet.LetterSpacing = mapSizeValue(propValue)
	case "text-align":
		parsedStyleSheet.TextAlign = propValue
	case "text-decoration":
		parsedStyleSheet.TextDecoration = propValue
	case "white-space":
		parsedStyleSheet.WhiteSpace = propValue
	case "display":
		parsedStyleSheet.Display = propValue
	case "position":
		parsedStyleSheet.Position = propValue
	case "flex-direction":
		parsedStyleSheet.FlexDirection = propValue
	case "flex-wrap":
		parsedStyleSheet.FlexWrap = propValue
	case "justify-content":
		parsedStyleSheet.JustifyContent = propValue
	case "align-items":
		parsedStyleSheet.AlignItems = propValue
	case "align-content":
		parsedStyleSheet.AlignContent = propValue
	case "flex-grow":
		parsedStyleSheet.FlexGrow = mapSizeValue(propValue)
	case "flex-shrink":
		parsedStyleSheet.FlexShrink = mapSizeValue(propValue)
	case "flex-basis":
		parsedStyleSheet.FlexBasis = propValue
	case "order":
		parsedStyleSheet.Order = mapIntValue(propValue)
	case "align-self":
		parsedStyleSheet.AlignSelf = propValue
	case "width":
		parsedStyleSheet.Width = mapSizeValue(propValue)
	case "height":
		parsedStyleSheet.Height = mapSizeValue(propValue)
	case "min-width":
		parsedStyleSheet.MinWidth = mapSizeValue(propValue)
	case "min-height":
		parsedStyleSheet.MinHeight = mapSizeValue(propValue)
	case "max-width":
		parsedStyleSheet.MaxWidth = mapSizeValue(propValue)
	case "max-height":
		parsedStyleSheet.MaxHeight = mapSizeValue(propValue)
	case "top":
		parsedStyleSheet.Top = mapSizeValue(propValue)
	case "right":
		parsedStyleSheet.Right = mapSizeValue(propValue)
	case "bottom":
		parsedStyleSheet.Bottom = mapSizeValue(propValue)
	case "left":
		parsedStyleSheet.Left = mapSizeValue(propValue)
	case "z-index":
		parsedStyleSheet.ZIndex = mapIntValue(propValue)
	case "margin":
		parseMargin(propValue, parsedStyleSheet)
	case "margin-top":
		parsedStyleSheet.MarginTop = mapSizeValue(propValue)
	case "margin-right":
		parsedStyleSheet.MarginRight = mapSizeValue(propValue)
	case "margin-bottom":
		parsedStyleSheet.MarginBottom = mapSizeValue(propValue)
	case "margin-left":
		parsedStyleSheet.MarginLeft = mapSizeValue(propValue)
	case "padding":
		parsePadding(propValue, parsedStyleSheet)
	case "padding-top":
		parsedStyleSheet.PaddingTop = mapSizeValue(propValue)
	case "padding-right":
		parsedStyleSheet.PaddingRight = mapSizeValue(propValue)
	case "padding-bottom":
		parsedStyleSheet.PaddingBottom = mapSizeValue(propValue)
	case "padding-left":
		parsedStyleSheet.PaddingLeft = mapSizeValue(propValue)
	case "border-width", "border-top-width":
		parsedStyleSheet.BorderTopWidth = mapSizeValue(propValue)
	case "border-right-width":
		parsedStyleSheet.BorderRightWidth = mapSizeValue(propValue)
	case "border-bottom-width":
		parsedStyleSheet.BorderBottomWidth = mapSizeValue(propValue)
	case "border-left-width":
		parsedStyleSheet.BorderLeftWidth = mapSizeValue(propValue)
	case "border-style", "border-top-style":
		parsedStyleSheet.BorderTopStyle = propValue
	case "border-right-style":
		parsedStyleSheet.BorderRightStyle = propValue
	case "border-bottom-style":
		parsedStyleSheet.BorderBottomStyle = propValue
	case "border-left-style":
		parsedStyleSheet.BorderLeftStyle = propValue
	case "border-color", "border-top-color":
		parsedStyleSheet.BorderTopColor = MapCSSColor(propValue)
	case "border-right-color":
		parsedStyleSheet.BorderRightColor = MapCSSColor(propValue)
	case "border-bottom-color":
		parsedStyleSheet.BorderBottomColor = MapCSSColor(propValue)
	case "border-left-color":
		parsedStyleSheet.BorderLeftColor = MapCSSColor(propValue)
	case "overflow":
		parsedStyleSheet.OverflowX = propValue
		parsedStyleSheet.OverflowY = propValue
	case "overflow-x":
		parsedStyleSheet.OverflowX = propValue
	case "overflow-y":
		parsedStyleSheet.OverflowY = propValue
	case "visibility":
		parsedStyleSheet.Visibility = propValue
	case "opacity":
		if val, err := strconv.ParseFloat(propValue, 64); err == nil {
			parsedStyleSheet.Opacity = val
		}
	}

	return parsedStyleSheet
}

func parseInlineStylesheet(attributes []*hotdog.Attribute, elementStylesheet *hotdog.Stylesheet) *hotdog.Stylesheet {
	for i := 0; i < len(attributes); i++ {
		attributeName := attributes[i].Name
		if attributeName == "style" {

			styleString := attributes[i].Value
			styleProps := strings.Split(strings.Replace(styleString, " ", "", -1), ";")

			for i := 0; i < len(styleProps); i++ {
				styledProperty := strings.Split(styleProps[i], ":")
				if len(styledProperty) >= 2 {
					elementStylesheet = mapPropToStylesheet(elementStylesheet, styledProperty)
				}
			}
		}
	}

	return elementStylesheet
}

func hasInlineStyle(attributes []*hotdog.Attribute) bool {
	inlineStyle := false

	for i := 0; i < len(attributes); i++ {
		attributeName := attributes[i].Name
		if attributeName == "style" {
			inlineStyle = true
		}
	}

	return inlineStyle
}

func GetElementStylesheet(elementName string, attributes []*hotdog.Attribute) *hotdog.Stylesheet {
	elementStylesheet := &hotdog.Stylesheet{
		BackgroundColor: &hotdog.ColorRGBA{R: 1, G: 1, B: 1, A: 0},
		FontSize:        0,
		Display:         "",
		Position:        "Normal",
		Opacity:         1,
		Visibility:      "visible",
	}

	if hasInlineStyle(attributes) {
		elementStylesheet = parseInlineStylesheet(attributes, elementStylesheet)
	}

	if elementStylesheet.FontSize == float64(0) {
		fontSize := elementFontTable[elementName]

		if fontSize != float64(0) {
			elementStylesheet.FontSize = fontSize
		} else {
			elementStylesheet.FontSize = float64(14)
		}
	}

	if elementStylesheet.Color == nil {
		color := elementColorTable[elementName]
		if color != nil {
			elementStylesheet.Color = color
		} else {
			elementStylesheet.Color = &hotdog.ColorRGBA{R: 0, G: 0, B: 0, A: 1}
		}
	}

	if elementStylesheet.FontWeight == 0 {
		elementStylesheet.FontWeight = getDefaultElementFontWeight(elementName)
	}

	if elementStylesheet.Display == "" {
		elementStylesheet.Display = getDefaultElementDisplay(elementName)
	}

	return elementStylesheet
}

func parseMargin(value string, sheet *hotdog.Stylesheet) {
	values := strings.Fields(value)
	switch len(values) {
	case 1:
		v := mapSizeValue(values[0])
		sheet.MarginTop = v
		sheet.MarginRight = v
		sheet.MarginBottom = v
		sheet.MarginLeft = v
	case 2:
		v1 := mapSizeValue(values[0])
		v2 := mapSizeValue(values[1])
		sheet.MarginTop = v1
		sheet.MarginBottom = v1
		sheet.MarginRight = v2
		sheet.MarginLeft = v2
	case 3:
		sheet.MarginTop = mapSizeValue(values[0])
		sheet.MarginRight = mapSizeValue(values[1])
		sheet.MarginBottom = mapSizeValue(values[2])
		sheet.MarginLeft = sheet.MarginRight
	case 4:
		sheet.MarginTop = mapSizeValue(values[0])
		sheet.MarginRight = mapSizeValue(values[1])
		sheet.MarginBottom = mapSizeValue(values[2])
		sheet.MarginLeft = mapSizeValue(values[3])
	}
}

func parsePadding(value string, sheet *hotdog.Stylesheet) {
	values := strings.Fields(value)
	switch len(values) {
	case 1:
		v := mapSizeValue(values[0])
		sheet.PaddingTop = v
		sheet.PaddingRight = v
		sheet.PaddingBottom = v
		sheet.PaddingLeft = v
	case 2:
		v1 := mapSizeValue(values[0])
		v2 := mapSizeValue(values[1])
		sheet.PaddingTop = v1
		sheet.PaddingBottom = v1
		sheet.PaddingRight = v2
		sheet.PaddingLeft = v2
	case 3:
		sheet.PaddingTop = mapSizeValue(values[0])
		sheet.PaddingRight = mapSizeValue(values[1])
		sheet.PaddingBottom = mapSizeValue(values[2])
		sheet.PaddingLeft = sheet.PaddingRight
	case 4:
		sheet.PaddingTop = mapSizeValue(values[0])
		sheet.PaddingRight = mapSizeValue(values[1])
		sheet.PaddingBottom = mapSizeValue(values[2])
		sheet.PaddingLeft = mapSizeValue(values[3])
	}
}

// ApplyStylesheets applies all parsed stylesheets in a document to matching DOM elements.
// It uses cascadia for CSS selector matching against the HTML parse tree.
func ApplyStylesheets(document *hotdog.Document) {
	if document == nil || document.DOM == nil || document.HTMLRoot == nil {
		return
	}

	for _, styleElement := range document.StyleSheets {
		if styleElement == nil || styleElement.Style == nil || styleElement.Selector == "" {
			continue
		}

		sel, err := cascadia.Compile(styleElement.Selector)
		if err != nil {
			continue
		}

		matchedNodes := sel.MatchAll(document.HTMLRoot)
		for _, htmlNode := range matchedNodes {
			nodeDOM := findNodeDOMByHTMLNode(document.DOM, htmlNode)
			if nodeDOM != nil {
				mergeStylesheet(nodeDOM.Style, styleElement.Style)
			}
		}
	}
}

// findNodeDOMByHTMLNode finds the NodeDOM corresponding to an html.Node by traversing the DOM tree.
func findNodeDOMByHTMLNode(root *hotdog.NodeDOM, target *html.Node) *hotdog.NodeDOM {
	if root == nil || root.HTMLNode == target {
		return root
	}

	for _, child := range root.Children {
		if result := findNodeDOMByHTMLNode(child, target); result != nil {
			return result
		}
	}

	return nil
}

// ApplyInheritance applies CSS inheritance rules to the DOM tree.
// Inheritable properties that are not explicitly set on an element
// will inherit their computed value from the parent element.
func ApplyInheritance(root *hotdog.NodeDOM) {
	if root == nil {
		return
	}

	// Start inheritance from root with no parent styles
	applyInheritanceRecursive(root, nil)
}

// applyInheritanceRecursive walks the DOM tree and applies inherited styles.
func applyInheritanceRecursive(node *hotdog.NodeDOM, parentStyle *hotdog.Stylesheet) {
	if node == nil || node.Style == nil {
		return
	}

	// For each inheritable property, if not explicitly set on this node,
	// inherit from parent
	if parentStyle != nil {
		inheritFromParent(node.Style, parentStyle)
	}

	// Recurse to children with this node's style as parent
	for _, child := range node.Children {
		applyInheritanceRecursive(child, node.Style)
	}
}

// inheritFromParent copies inheritable properties from parent to child
// if they are not explicitly set on the child.
func inheritFromParent(childStyle, parentStyle *hotdog.Stylesheet) {
	if childStyle == nil || parentStyle == nil {
		return
	}

	// Color properties
	if childStyle.Color == nil && parentStyle.Color != nil {
		childStyle.Color = parentStyle.Color
	}

	// Font properties
	if childStyle.FontFamily == "" && parentStyle.FontFamily != "" {
		childStyle.FontFamily = parentStyle.FontFamily
	}
	if childStyle.FontSize == 0 && parentStyle.FontSize != 0 {
		childStyle.FontSize = parentStyle.FontSize
	}
	if childStyle.FontWeight == 0 && parentStyle.FontWeight != 0 {
		childStyle.FontWeight = parentStyle.FontWeight
	}
	if childStyle.FontStyle == "" && parentStyle.FontStyle != "" {
		childStyle.FontStyle = parentStyle.FontStyle
	}
	if childStyle.LineHeight == 0 && parentStyle.LineHeight != 0 {
		childStyle.LineHeight = parentStyle.LineHeight
	}
	if childStyle.LetterSpacing == 0 && parentStyle.LetterSpacing != 0 {
		childStyle.LetterSpacing = parentStyle.LetterSpacing
	}

	// Text properties
	if childStyle.TextAlign == "" && parentStyle.TextAlign != "" {
		childStyle.TextAlign = parentStyle.TextAlign
	}
	if childStyle.TextDecoration == "" && parentStyle.TextDecoration != "" {
		childStyle.TextDecoration = parentStyle.TextDecoration
	}
	if childStyle.WhiteSpace == "" && parentStyle.WhiteSpace != "" {
		childStyle.WhiteSpace = parentStyle.WhiteSpace
	}

	// Visibility
	if childStyle.Visibility == "" && parentStyle.Visibility != "" {
		childStyle.Visibility = parentStyle.Visibility
	}

	// Opacity
	if childStyle.Opacity == 0 && parentStyle.Opacity != 0 {
		childStyle.Opacity = parentStyle.Opacity
	}
}

// mergeStylesheet merges source stylesheet into destination, overwriting non-zero values.
func mergeStylesheet(dest, src *hotdog.Stylesheet) {
	if src == nil {
		return
	}

	// Color properties
	if src.Color != nil {
		dest.Color = src.Color
	}
	if src.BackgroundColor != nil {
		dest.BackgroundColor = src.BackgroundColor
	}

	// Font properties
	if src.FontSize != 0 {
		dest.FontSize = src.FontSize
	}
	if src.FontWeight != 0 {
		dest.FontWeight = src.FontWeight
	}
	if src.FontFamily != "" {
		dest.FontFamily = src.FontFamily
	}
	if src.FontStyle != "" {
		dest.FontStyle = src.FontStyle
	}
	if src.LineHeight != 0 {
		dest.LineHeight = src.LineHeight
	}
	if src.LetterSpacing != 0 {
		dest.LetterSpacing = src.LetterSpacing
	}

	// Text properties
	if src.TextAlign != "" {
		dest.TextAlign = src.TextAlign
	}
	if src.TextDecoration != "" {
		dest.TextDecoration = src.TextDecoration
	}
	if src.WhiteSpace != "" {
		dest.WhiteSpace = src.WhiteSpace
	}

	// Display and position
	if src.Display != "" {
		dest.Display = src.Display
	}
	if src.Position != "" {
		dest.Position = src.Position
	}

	// Flexbox properties
	if src.FlexDirection != "" {
		dest.FlexDirection = src.FlexDirection
	}
	if src.FlexWrap != "" {
		dest.FlexWrap = src.FlexWrap
	}
	if src.JustifyContent != "" {
		dest.JustifyContent = src.JustifyContent
	}
	if src.AlignItems != "" {
		dest.AlignItems = src.AlignItems
	}
	if src.AlignContent != "" {
		dest.AlignContent = src.AlignContent
	}
	if src.FlexGrow != 0 {
		dest.FlexGrow = src.FlexGrow
	}
	if src.FlexShrink != 0 {
		dest.FlexShrink = src.FlexShrink
	}
	if src.FlexBasis != "" {
		dest.FlexBasis = src.FlexBasis
	}
	if src.Order != 0 {
		dest.Order = src.Order
	}
	if src.AlignSelf != "" {
		dest.AlignSelf = src.AlignSelf
	}

	// Size properties
	if src.Width != 0 {
		dest.Width = src.Width
	}
	if src.Height != 0 {
		dest.Height = src.Height
	}
	if src.MinWidth != 0 {
		dest.MinWidth = src.MinWidth
	}
	if src.MinHeight != 0 {
		dest.MinHeight = src.MinHeight
	}
	if src.MaxWidth != 0 {
		dest.MaxWidth = src.MaxWidth
	}
	if src.MaxHeight != 0 {
		dest.MaxHeight = src.MaxHeight
	}

	// Position properties
	if src.Top != 0 {
		dest.Top = src.Top
	}
	if src.Right != 0 {
		dest.Right = src.Right
	}
	if src.Bottom != 0 {
		dest.Bottom = src.Bottom
	}
	if src.Left != 0 {
		dest.Left = src.Left
	}
	if src.ZIndex != 0 {
		dest.ZIndex = src.ZIndex
	}

	// Margin properties
	if src.MarginTop != 0 {
		dest.MarginTop = src.MarginTop
	}
	if src.MarginRight != 0 {
		dest.MarginRight = src.MarginRight
	}
	if src.MarginBottom != 0 {
		dest.MarginBottom = src.MarginBottom
	}
	if src.MarginLeft != 0 {
		dest.MarginLeft = src.MarginLeft
	}

	// Padding properties
	if src.PaddingTop != 0 {
		dest.PaddingTop = src.PaddingTop
	}
	if src.PaddingRight != 0 {
		dest.PaddingRight = src.PaddingRight
	}
	if src.PaddingBottom != 0 {
		dest.PaddingBottom = src.PaddingBottom
	}
	if src.PaddingLeft != 0 {
		dest.PaddingLeft = src.PaddingLeft
	}

	// Border properties
	if src.BorderTopWidth != 0 {
		dest.BorderTopWidth = src.BorderTopWidth
	}
	if src.BorderRightWidth != 0 {
		dest.BorderRightWidth = src.BorderRightWidth
	}
	if src.BorderBottomWidth != 0 {
		dest.BorderBottomWidth = src.BorderBottomWidth
	}
	if src.BorderLeftWidth != 0 {
		dest.BorderLeftWidth = src.BorderLeftWidth
	}
	if src.BorderTopStyle != "" {
		dest.BorderTopStyle = src.BorderTopStyle
	}
	if src.BorderRightStyle != "" {
		dest.BorderRightStyle = src.BorderRightStyle
	}
	if src.BorderBottomStyle != "" {
		dest.BorderBottomStyle = src.BorderBottomStyle
	}
	if src.BorderLeftStyle != "" {
		dest.BorderLeftStyle = src.BorderLeftStyle
	}
	if src.BorderTopColor != nil {
		dest.BorderTopColor = src.BorderTopColor
	}
	if src.BorderRightColor != nil {
		dest.BorderRightColor = src.BorderRightColor
	}
	if src.BorderBottomColor != nil {
		dest.BorderBottomColor = src.BorderBottomColor
	}
	if src.BorderLeftColor != nil {
		dest.BorderLeftColor = src.BorderLeftColor
	}

	// Overflow and visibility
	if src.OverflowX != "" {
		dest.OverflowX = src.OverflowX
	}
	if src.OverflowY != "" {
		dest.OverflowY = src.OverflowY
	}
	if src.Visibility != "" {
		dest.Visibility = src.Visibility
	}
	if src.Opacity != 0 {
		dest.Opacity = src.Opacity
	}
}
