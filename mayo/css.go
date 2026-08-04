package mayo

import (
	"regexp"
	"strings"

	hotdog "github.com/danfragoso/thdwb/hotdog"
)

// ParseStylesheet parses a CSS stylesheet string and returns a slice of StyleElement
func ParseStylesheet(cssString string) []*hotdog.StyleElement {
	var styleSheet []*hotdog.StyleElement

	// Remove comments
	cssString = regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(cssString, "")

	// Split by rules (selector { declarations })
	ruleRegex := regexp.MustCompile(`([^{]+)\{([^}]+)\}`)
	matches := ruleRegex.FindAllStringSubmatch(cssString, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		selector := strings.TrimSpace(match[1])
		declarations := strings.TrimSpace(match[2])

		styleElement := &hotdog.StyleElement{
			Selector: selector,
			Style:    &hotdog.Stylesheet{},
		}

		// Parse declarations
		declPairs := strings.Split(declarations, ";")
		for _, decl := range declPairs {
			decl = strings.TrimSpace(decl)
			if decl == "" {
				continue
			}
			propVal := strings.SplitN(decl, ":", 2)
			if len(propVal) == 2 {
				mapPropToStylesheet(styleElement.Style, []string{strings.TrimSpace(propVal[0]), strings.TrimSpace(propVal[1])})
			}
		}

		styleSheet = append(styleSheet, styleElement)
	}

	return styleSheet
}
