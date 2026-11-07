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
	div := &hotdog.NodeDOM{
		Element:    "div",
		Style:      &hotdog.Stylesheet{},
		Attributes: []*hotdog.Attribute{{Name: "class", Value: "c"}},
	}
	body := &hotdog.NodeDOM{Element: "body", Style: &hotdog.Stylesheet{}, Children: []*hotdog.NodeDOM{div}}
	root := &hotdog.NodeDOM{Element: "html", Style: &hotdog.Stylesheet{}, Children: []*hotdog.NodeDOM{body}}

	htmlDiv := &html.Node{Type: html.ElementNode, Data: "div",
		Attr: []html.Attribute{{Key: "class", Val: "c"}}}
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
