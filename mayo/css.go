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
	var quote rune // non-zero while inside a "..." or '...' string
	for i, c := range s {
		switch {
		case quote != 0:
			// Inside a quoted string, brackets/commas are literal.
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '(' || c == '[' || c == '{':
			depth++
		case c == ')' || c == ']' || c == '}':
			if depth > 0 {
				depth--
			}
		case c == delim:
			// delim is always ASCII (','), so i+1 is a valid byte offset.
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// parseDeclarations parses a "prop: val; prop: val" block into dst.
func parseDeclarations(declarations string, dst *hotdog.Stylesheet) {
	for _, decl := range strings.Split(declarations, ";") {
		decl = strings.TrimSpace(decl)
		if decl == "" {
			continue
		}
		propVal := strings.SplitN(decl, ":", 2)
		if len(propVal) == 2 {
			mapPropToStylesheet(dst, []string{strings.TrimSpace(propVal[0]), strings.TrimSpace(propVal[1])})
		}
	}
}

// ParseStylesheet parses a CSS stylesheet string and returns a slice of StyleElement.
// Index values start at 0; use ParseStylesheetFrom when appending several sheets to a
// document so source order is preserved across <style> blocks.
func ParseStylesheet(cssString string) []*hotdog.StyleElement {
	return ParseStylesheetFrom(cssString, 0)
}

// ParseStylesheetFrom parses CSS and assigns cascade Index values starting at
// indexBase. ApplyStylesheets uses Index as the final source-order tiebreak, so
// every rule in a document must get a globally increasing index; restarting at
// 0 per <style> block let an earlier block's rule win a later block's rule.
func ParseStylesheetFrom(cssString string, indexBase int) []*hotdog.StyleElement {
	var styleSheet []*hotdog.StyleElement

	// Remove comments
	cssString = regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(cssString, "")

	// Split by rules (selector { declarations })
	ruleRegex := regexp.MustCompile(`([^{]+)\{([^}]+)\}`)
	matches := ruleRegex.FindAllStringSubmatch(cssString, -1)

	// emitted counts rules actually produced, so Index stays contiguous from
	// indexBase. Using the raw match index would leave gaps when a malformed
	// rule is skipped, and those gaps could collide with the next <style>
	// block's base index.
	emitted := 0
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		selector := strings.TrimSpace(match[1])
		declarations := strings.TrimSpace(match[2])

		// cascadia.Parse rejects comma group selectors (e.g. "a, b"), so split
		// on top-level commas and emit one StyleElement per arm. Each arm
		// carries its own specificity: recording the max across the group (the
		// old behavior) let "#foo, .b" beat a legitimate rule that actually
		// matched via the low-specificity ".b" arm. Splitting also lets
		// ApplyStylesheets compile and match each arm independently.
		for _, arm := range splitTopLevel(selector, ',') {
			arm = strings.TrimSpace(arm)
			if arm == "" {
				continue
			}
			sel, err := cascadia.Parse(arm)
			if err != nil {
				continue
			}

			styleElement := &hotdog.StyleElement{
				Selector:    arm,
				Style:       &hotdog.Stylesheet{},
				Specificity: sel.Specificity(),
				Origin:      "author",
				Index:       indexBase + emitted,
			}
			parseDeclarations(declarations, styleElement.Style)

			styleSheet = append(styleSheet, styleElement)
			emitted++
		}
	}

	return styleSheet
}
