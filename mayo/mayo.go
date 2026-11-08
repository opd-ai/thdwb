package mayo

import (
	"errors"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	cascadia "github.com/andybalholm/cascadia"
	hotdog "github.com/danfragoso/thdwb/hotdog"
	"golang.org/x/net/html"
)

// Inheritable properties - CSS properties that inherit by default from parent to child.
// When a child element doesn't have its own value for these properties, it uses the parent's value.
// Matches CSS spec inheritance behavior.
// See: https://developer.mozilla.org/en-US/docs/Web/CSS/inheritance
// See: https://searchfox.org/mozilla-central/source/layout/style/nsCSSProps.cpp
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

// _ = inheritableProperties keeps the symbol referenced while it awaits
// integration with the CSS inheritance cascade.
var _ = inheritableProperties

// getDefaultElementDisplay returns the default 'display' value for an HTML element.
// Implements the User Agent (UA) stylesheet defaults per HTML spec.
// Unknown elements default to 'block'.
// See: https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Box_Model/Introduction_to_the_CSS_box_model#user_agent_styles
// See: https://searchfox.org/mozilla-central/source/layout/style/res
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

// getDefaultElementFontWeight returns the default font-weight for an element.
// Headings (h1-h6) default to 600 (semi-bold), others default to 400 (normal).
// Matches UA stylesheet defaults.
func getDefaultElementFontWeight(element string) int {
	switch element {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return 600
	}

	return 400
}

// mapSizeValue converts a CSS size string to a float64 pixel value.
// Extracts the numeric portion from strings like "16px", "1.5em", "100%", "-5px".
// Returns the numeric value and the unit suffix ("px", "%", "em", ...).
// Returns 0 for keyword values ("auto", "none", "inherit", ...) so the layout
// engine computes the intrinsic size instead of pinning the box to 14px
// (audit O-22).
func mapSizeValue(sizeValue string) (float64, string) {
	switch strings.ToLower(strings.TrimSpace(sizeValue)) {
	case "auto", "none", "inherit", "initial", "unset", "fit-content",
		"min-content", "max-content", "content":
		return 0, ""
	}
	re := regexp.MustCompile(`[+-]?[0-9]+(\.[0-9]+)?`)
	valueString := re.FindString(sizeValue)
	if valueString == "" {
		return float64(14), "px"
	}
	value, err := strconv.ParseFloat(valueString, 64)
	if err != nil {
		return float64(14), "px"
	}

	// Extract the unit suffix after the number.
	unit := strings.TrimPrefix(sizeValue, valueString)
	unit = strings.TrimSpace(unit)
	return value, unit
}

// mapIntValue extracts an integer from a CSS value string.
// Used for properties like z-index, font-weight (numeric).
// Returns 0 if no number found. A number that overflows int32 saturates to
// +/-MaxInt32 instead of collapsing to 0: z-index: 99999999999999 must keep its
// "topmost" meaning, and dropping it to 0 put the element BELOW everything
// (audit O-15).
func mapIntValue(intValue string) int {
	re := regexp.MustCompile(`[+-]?[0-9]+`)
	valueString := re.FindString(intValue)
	if valueString == "" {
		return 0
	}
	value, err := strconv.ParseInt(valueString, 10, 32)
	if err != nil {
		// Range error on a non-empty numeric token: saturate, don't drop.
		if errors.Is(err, strconv.ErrRange) {
			if strings.HasPrefix(valueString, "-") {
				return math.MinInt32
			}
			return math.MaxInt32
		}
		return 0
	}
	return int(value)
}

// mapPropToStylesheet applies a single CSS property:value pair to a Stylesheet.
// Parses the property name and value, converts to typed Go fields.
// Handles: color, background-color, font-size, and many others.
// Properties not explicitly handled are stored in the Properties map for cascade.
// Parameters:
//
//	parsedStyleSheet - target Stylesheet to modify
//	propSlice - [property, value] pair (e.g., ["color", "red"])
//
// Returns the modified Stylesheet.
func mapPropToStylesheet(parsedStyleSheet *hotdog.Stylesheet, propSlice []string) *hotdog.Stylesheet {
	propName := strings.TrimSpace(strings.ToLower(propSlice[0]))
	propValue := strings.TrimSpace(propSlice[1])

	// Strip the !important flag before value parsing. It must not leak into the
	// stored value: "display: none !important" used to store "none !important",
	// which matches no dispatch case, so the display was silently dropped and
	// the element stayed visible. !important priority ordering across rules is
	// not yet implemented; stripping at least makes the value itself correct.
	if i := strings.Index(strings.ToLower(propValue), "!important"); i >= 0 {
		propValue = strings.TrimSpace(propValue[:i])
	}

	switch propName {
	case "color":
		parsedStyleSheet.Color = MapCSSColor(propValue)
	case "background-color":
		parsedStyleSheet.BackgroundColor = MapCSSColor(propValue)
	case "font-size":
		parsedStyleSheet.FontSize, _ = mapSizeValue(propValue)
	case "font-weight":
		parsedStyleSheet.FontWeight = mapIntValue(propValue)
	case "font-family":
		parsedStyleSheet.FontFamily = strings.Trim(propValue, "\"'")
	case "font-style":
		parsedStyleSheet.FontStyle = propValue
	case "line-height":
		parsedStyleSheet.LineHeight, _ = mapSizeValue(propValue)
	case "letter-spacing":
		parsedStyleSheet.LetterSpacing, _ = mapSizeValue(propValue)
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
		parsedStyleSheet.FlexGrow, _ = mapSizeValue(propValue)
	case "flex-shrink":
		parsedStyleSheet.FlexShrink, _ = mapSizeValue(propValue)
	case "flex-basis":
		parsedStyleSheet.FlexBasis = propValue
	case "order":
		parsedStyleSheet.Order = mapIntValue(propValue)
	case "align-self":
		parsedStyleSheet.AlignSelf = propValue
	case "width":
		parsedStyleSheet.Width, _ = mapSizeValue(propValue)
	case "height":
		parsedStyleSheet.Height, _ = mapSizeValue(propValue)
	case "min-width":
		parsedStyleSheet.MinWidth, _ = mapSizeValue(propValue)
	case "min-height":
		parsedStyleSheet.MinHeight, _ = mapSizeValue(propValue)
	case "max-width":
		parsedStyleSheet.MaxWidth, _ = mapSizeValue(propValue)
	case "max-height":
		parsedStyleSheet.MaxHeight, _ = mapSizeValue(propValue)
	case "top":
		parsedStyleSheet.Top, _ = mapSizeValue(propValue)
	case "right":
		parsedStyleSheet.Right, _ = mapSizeValue(propValue)
	case "bottom":
		parsedStyleSheet.Bottom, _ = mapSizeValue(propValue)
	case "left":
		parsedStyleSheet.Left, _ = mapSizeValue(propValue)
	case "z-index":
		parsedStyleSheet.ZIndex = mapIntValue(propValue)
	case "margin":
		parseMargin(propValue, parsedStyleSheet)
	case "margin-top":
		parsedStyleSheet.MarginTop, _ = mapSizeValue(propValue)
	case "margin-right":
		parsedStyleSheet.MarginRight, _ = mapSizeValue(propValue)
	case "margin-bottom":
		parsedStyleSheet.MarginBottom, _ = mapSizeValue(propValue)
	case "margin-left":
		parsedStyleSheet.MarginLeft, _ = mapSizeValue(propValue)
	case "padding":
		parsePadding(propValue, parsedStyleSheet)
	case "padding-top":
		parsedStyleSheet.PaddingTop, _ = mapSizeValue(propValue)
	case "padding-right":
		parsedStyleSheet.PaddingRight, _ = mapSizeValue(propValue)
	case "padding-bottom":
		parsedStyleSheet.PaddingBottom, _ = mapSizeValue(propValue)
	case "padding-left":
		parsedStyleSheet.PaddingLeft, _ = mapSizeValue(propValue)
	case "border-width", "border-top-width":
		parsedStyleSheet.BorderTopWidth, _ = mapSizeValue(propValue)
	case "border-right-width":
		parsedStyleSheet.BorderRightWidth, _ = mapSizeValue(propValue)
	case "border-bottom-width":
		parsedStyleSheet.BorderBottomWidth, _ = mapSizeValue(propValue)
	case "border-left-width":
		parsedStyleSheet.BorderLeftWidth, _ = mapSizeValue(propValue)
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
			// Split on ';' first, then on ':' per declaration, trimming only
			// outer whitespace. Deleting all whitespace (the old behavior)
			// collapses multi-value shorthand like "margin: 10px 20px" into
			// "margin:10px20px".
			for _, decl := range strings.Split(styleString, ";") {
				prop, val, ok := strings.Cut(decl, ":")
				if !ok {
					continue
				}
				elementStylesheet = mapPropToStylesheet(elementStylesheet, []string{strings.TrimSpace(prop), strings.TrimSpace(val)})
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
		v, _ := mapSizeValue(values[0])
		sheet.MarginTop = v
		sheet.MarginRight = v
		sheet.MarginBottom = v
		sheet.MarginLeft = v
	case 2:
		v1, _ := mapSizeValue(values[0])
		v2, _ := mapSizeValue(values[1])
		sheet.MarginTop = v1
		sheet.MarginBottom = v1
		sheet.MarginRight = v2
		sheet.MarginLeft = v2
	case 3:
		sheet.MarginTop, _ = mapSizeValue(values[0])
		sheet.MarginRight, _ = mapSizeValue(values[1])
		sheet.MarginBottom, _ = mapSizeValue(values[2])
		sheet.MarginLeft = sheet.MarginRight
	case 4:
		sheet.MarginTop, _ = mapSizeValue(values[0])
		sheet.MarginRight, _ = mapSizeValue(values[1])
		sheet.MarginBottom, _ = mapSizeValue(values[2])
		sheet.MarginLeft, _ = mapSizeValue(values[3])
	}
}

func parsePadding(value string, sheet *hotdog.Stylesheet) {
	values := strings.Fields(value)
	switch len(values) {
	case 1:
		v, _ := mapSizeValue(values[0])
		sheet.PaddingTop = v
		sheet.PaddingRight = v
		sheet.PaddingBottom = v
		sheet.PaddingLeft = v
	case 2:
		v1, _ := mapSizeValue(values[0])
		v2, _ := mapSizeValue(values[1])
		sheet.PaddingTop = v1
		sheet.PaddingBottom = v1
		sheet.PaddingRight = v2
		sheet.PaddingLeft = v2
	case 3:
		sheet.PaddingTop, _ = mapSizeValue(values[0])
		sheet.PaddingRight, _ = mapSizeValue(values[1])
		sheet.PaddingBottom, _ = mapSizeValue(values[2])
		sheet.PaddingLeft = sheet.PaddingRight
	case 4:
		sheet.PaddingTop, _ = mapSizeValue(values[0])
		sheet.PaddingRight, _ = mapSizeValue(values[1])
		sheet.PaddingBottom, _ = mapSizeValue(values[2])
		sheet.PaddingLeft, _ = mapSizeValue(values[3])
	}
}

// ApplyStylesheets applies all parsed stylesheets in a document to matching DOM elements.
// It uses cascadia for CSS selector matching against the HTML parse tree.
// Stylesheets are applied in cascade order: origin, specificity, then source order.
func ApplyStylesheets(document *hotdog.Document) {
	if document == nil || document.DOM == nil || document.HTMLRoot == nil {
		return
	}

	// Create a copy of stylesheets for sorting by cascade order
	type styleEntry struct {
		element  *hotdog.StyleElement
		selector cascadia.Selector
	}

	var entries []styleEntry
	for _, styleElement := range document.StyleSheets {
		if styleElement == nil || styleElement.Style == nil || styleElement.Selector == "" {
			continue
		}
		sel, err := cascadia.Compile(styleElement.Selector)
		if err != nil {
			continue
		}
		entries = append(entries, styleEntry{element: styleElement, selector: sel})
	}

	// Sort by cascade order: origin priority, then specificity, then source
	// order. SliceStable keeps equal-keyed rules in their original relative
	// order so the result is deterministic even if Index values collide.
	sort.SliceStable(entries, func(i, j int) bool {
		a := entries[i].element
		b := entries[j].element

		// Origin priority: user-agent < user < author
		originPriority := map[string]int{"user-agent": 0, "user": 1, "author": 2}
		aPriority := originPriority[a.Origin]
		bPriority := originPriority[b.Origin]
		if aPriority != bPriority {
			return aPriority < bPriority
		}

		// Higher specificity wins
		if a.Specificity != b.Specificity {
			return a.Specificity.Less(b.Specificity)
		}

		// Later rules win (higher index)
		return a.Index < b.Index
	})

	// Apply stylesheets in cascade order
	for _, entry := range entries {
		styleElement := entry.element
		sel := entry.selector

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
