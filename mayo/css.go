package mayo

import (
	"regexp"
	"strings"

	cascadia "github.com/andybalholm/cascadia"
	hotdog "github.com/danfragoso/thdwb/hotdog"
)

// splitTopLevel splits s on the given delimiter, but only at the top level
// (i.e., not inside parentheses, brackets, or braces). This is used to split
// comma group selectors without breaking functional pseudo-classes like
// :not(a, b).
func splitTopLevel(s string, delim rune) []string {
	var parts []string
	depth := 0
	start := 0
	for i, c := range s {
		switch c {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case delim:
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// ParseStylesheet parses a CSS stylesheet string and returns a slice of StyleElement
func ParseStylesheet(cssString string) []*hotdog.StyleElement {
	var styleSheet []*hotdog.StyleElement

	// Remove comments
	cssString = regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(cssString, "")

	// Split by rules (selector { declarations })
	ruleRegex := regexp.MustCompile(`([^{]+)\{([^}]+)\}`)
	matches := ruleRegex.FindAllStringSubmatch(cssString, -1)

	for idx, match := range matches {
		if len(match) < 3 {
			continue
		}

		selector := strings.TrimSpace(match[1])
		declarations := strings.TrimSpace(match[2])

		// Parse selector to get specificity. cascadia.Parse rejects comma
		// group selectors (e.g. "a, b"), so split on top-level commas and
		// take the maximum specificity across the group's arms.
		var specificity cascadia.Specificity
		for _, arm := range splitTopLevel(selector, ',') {
			arm = strings.TrimSpace(arm)
			if arm == "" {
				continue
			}
			if sel, err := cascadia.Parse(arm); err == nil {
				s := sel.Specificity()
				if specificity.Less(s) {
					specificity = s
				}
			}
		}

		styleElement := &hotdog.StyleElement{
			Selector:    selector,
			Style:       &hotdog.Stylesheet{},
			Specificity: specificity,
			Origin:      "author",
			Index:       idx,
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
