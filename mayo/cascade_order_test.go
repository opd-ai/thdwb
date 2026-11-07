package mayo

import (
	"testing"

	"github.com/danfragoso/thdwb/hotdog"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

// buildCascadeDoc assembles a Document whose body contains a single
// <div class="c">, with the supplied parsed stylesheets, wired so
// ApplyStylesheets can match selectors against the html node tree.
func buildCascadeDoc(sheets []*hotdog.StyleElement) (*hotdog.Document, *hotdog.NodeDOM) {
	return buildCascadeDocWithAttrs(sheets, [][2]string{{"class", "c"}})
}

// buildCascadeDocWithAttrs is buildCascadeDoc with explicit div attributes so
// tests can target specific selectors (id/class).
func buildCascadeDocWithAttrs(sheets []*hotdog.StyleElement, attrs [][2]string) (*hotdog.Document, *hotdog.NodeDOM) {
	divAttrs := make([]*hotdog.Attribute, 0, len(attrs))
	htmlAttrs := make([]html.Attribute, 0, len(attrs))
	for _, a := range attrs {
		divAttrs = append(divAttrs, &hotdog.Attribute{Name: a[0], Value: a[1]})
		htmlAttrs = append(htmlAttrs, html.Attribute{Key: a[0], Val: a[1]})
	}

	div := &hotdog.NodeDOM{Element: "div", Style: &hotdog.Stylesheet{}, Attributes: divAttrs}
	body := &hotdog.NodeDOM{Element: "body", Style: &hotdog.Stylesheet{}, Children: []*hotdog.NodeDOM{div}}
	root := &hotdog.NodeDOM{Element: "html", Style: &hotdog.Stylesheet{}, Children: []*hotdog.NodeDOM{body}}

	htmlDiv := &html.Node{Type: html.ElementNode, Data: "div", Attr: htmlAttrs}
	htmlBody := &html.Node{Type: html.ElementNode, Data: "body", FirstChild: htmlDiv}
	htmlRoot := &html.Node{Type: html.ElementNode, Data: "html", FirstChild: htmlBody}

	root.HTMLNode = htmlRoot
	body.HTMLNode = htmlBody
	div.HTMLNode = htmlDiv

	return &hotdog.Document{DOM: root, HTMLRoot: htmlRoot, StyleSheets: sheets}, div
}

// TestParseStylesheetFromIsMonotonicAcrossBlocks guards O-3: ApplyStylesheets
// uses StyleElement.Index as the final source-order tiebreak, so Index must be
// globally increasing across <style> blocks. Restarting at 0 per block let an
// earlier block's rule win a later block's equal-specificity rule.
func TestParseStylesheetFromIsMonotonicAcrossBlocks(t *testing.T) {
	first := ParseStylesheetFrom(`.c { color: red; }`, 0)
	second := ParseStylesheetFrom(`.c { color: blue; }`, len(first))

	assert.Equal(t, 0, first[0].Index)
	assert.Equal(t, 1, second[0].Index, "second block must continue the index sequence")

	doc, div := buildCascadeDoc(append(append([]*hotdog.StyleElement{}, first...), second...))
	ApplyStylesheets(doc)

	// Equal specificity: the later block (blue) must win.
	assert.NotNil(t, div.Style.Color)
	assert.Equal(t, 1.0, div.Style.Color.B, "later <style> block should win the source-order tiebreak")
	assert.Equal(t, 0.0, div.Style.Color.R)
}

// TestParseStylesheetFromSkipsGaps checks that malformed rules do not leave
// Index gaps that could collide with the next block's base.
func TestParseStylesheetFromSkipsGaps(t *testing.T) {
	// The middle "rule" has no declaration block and must not consume an index.
	blocked := ParseStylesheetFrom(".a { color: red; } broken .b { color: blue; }", 0)
	for i, el := range blocked {
		assert.Equal(t, i, el.Index, "Index must be contiguous from base")
	}
}

// TestMapPropStripsImportant guards O-5: the !important flag must not leak into
// the stored value. "display: none !important" used to store "none !important",
// which matched no dispatch case, so display was silently dropped.
func TestMapPropStripsImportant(t *testing.T) {
	sheet := &hotdog.Stylesheet{}
	mapPropToStylesheet(sheet, []string{"display", "none !important"})
	assert.Equal(t, "none", sheet.Display, "!important must be stripped from the value")

	mapPropToStylesheet(sheet, []string{"color", "blue !IMPORTANT"})
	assert.NotNil(t, sheet.Color, "case-insensitive !important must still parse the color")
	assert.Equal(t, 1.0, sheet.Color.B)
}

// TestGroupArmUsesOwnSpecificity guards O-6: a comma group like "#foo, .b" must
// record each arm's own specificity, not the max across arms. Previously the
// group stored (1,0,0) from the #foo arm, so an element that only matched via
// the low-specificity ".b" arm still beat a later, equally-specific .b rule.
func TestGroupArmUsesOwnSpecificity(t *testing.T) {
	// Rule 1: "#foo, .b" -> red. Rule 2: ".b" -> blue (later source order).
	first := ParseStylesheetFrom(`#foo, .b { color: red; }`, 0)
	second := ParseStylesheetFrom(`.b { color: blue; }`, len(first))

	// The group must expand to two arms with distinct specificities.
	assert.Len(t, first, 2, "group selector must emit one element per arm")
	var idArm, classArm *hotdog.StyleElement
	for _, el := range first {
		switch el.Selector {
		case "#foo":
			idArm = el
		case ".b":
			classArm = el
		}
	}
	assert.NotNil(t, idArm)
	assert.NotNil(t, classArm)
	assert.EqualValues(t, [3]int{1, 0, 0}, [3]int(idArm.Specificity), "#foo arm specificity")
	assert.EqualValues(t, [3]int{0, 1, 0}, [3]int(classArm.Specificity), ".b arm specificity")

	// Element has class b but NOT id foo: it matches the .b arm (0,1,0), which
	// ties the later standalone .b rule, so source order makes blue win. Under
	// the old max-specificity behavior the group stored (1,0,0) and red won.
	doc, div := buildCascadeDocWithAttrs(append(append([]*hotdog.StyleElement{}, first...), second...),
		[][2]string{{"class", "b"}})
	ApplyStylesheets(doc)
	assert.NotNil(t, div.Style.Color)
	assert.Equal(t, 1.0, div.Style.Color.B, "class-only element must not be beaten by the #foo arm")
	assert.Equal(t, 0.0, div.Style.Color.R)
}

// TestSplitTopLevelQuoteAware guards O-7: splitTopLevel must not treat
// parentheses/brackets inside quoted strings as nesting. div[title="("], .b
// used to fail to split (the ( raised depth), so the whole group was stored as
// a single arm with zero specificity.
func TestSplitTopLevelQuoteAware(t *testing.T) {
	got := splitTopLevel(`div[title="("], .b`, ',')
	assert.Equal(t, []string{`div[title="("]`, ` .b`}, got)

	// Single quotes and a paren that closes only inside the string.
	got = splitTopLevel(`a[content="x)y"], b`, ',')
	assert.Equal(t, []string{`a[content="x)y"]`, ` b`}, got)

	// Real functional pseudo-classes still keep their commas.
	got = splitTopLevel(`:not(a, b), c`, ',')
	assert.Equal(t, []string{`:not(a, b)`, ` c`}, got)

	// A quoted group parses to two properly-specific arms, not one zero-spec arm.
	arms := ParseStylesheet(`div[title="("], .b { color: red; }`)
	assert.Len(t, arms, 2, "quote-containing attribute selector must split into arms")
}

// TestApplyStylesheetsStableOnEqualKeys guards O-11: the cascade sort must be
// stable so equal origin/specificity/index rules resolve deterministically.
func TestApplyStylesheetsStableOnEqualKeys(t *testing.T) {
	sheets := ParseStylesheet(`.c { color: red; }`)
	// Force identical Index on two rules to create a tie the comparator cannot
	// break; stable sort must preserve input order (blue applied last wins).
	red := sheets[0]
	blue := ParseStylesheet(`.c { color: blue; }`)[0]
	blue.Index = red.Index

	doc, div := buildCascadeDoc([]*hotdog.StyleElement{red, blue})
	ApplyStylesheets(doc)

	assert.NotNil(t, div.Style.Color)
	// With a stable sort and a tie, the later-in-input rule is applied last.
	assert.Equal(t, 1.0, div.Style.Color.B, "stable sort must keep input order for equal keys")
}
