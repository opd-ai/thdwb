package mayo

import (
	"regexp"
	"strconv"
	"strings"

	hotdog "github.com/danfragoso/thdwb/hotdog"
)

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
		BackgroundColor: &hotdog.ColorRGBA{1, 1, 1, 0},
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
			elementStylesheet.Color = &hotdog.ColorRGBA{0, 0, 0, 1}
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
