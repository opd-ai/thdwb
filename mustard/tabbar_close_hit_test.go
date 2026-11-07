package mustard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCloseButtonHitMatchesPaint (audit O-13) verifies the close "X" hit box is
// centered vertically like the glyph draw() paints it. The hit box used to sit
// at the bottom of the tab (tabHeight-closeSize-4), so clicking the visible X
// missed and clicking the empty bottom strip closed the tab.
func TestCloseButtonHitMatchesPaint(t *testing.T) {
	tb := CreateTabBar(40) // tabHeight=40, closeSize=12
	tab := tb.AddTab("First", "https://a.example")
	tb.computedBox.SetCoords(0, 0, 600, 40)

	const tabWidth = 140.0
	const closeSize = 12.0
	// draw() paints the glyph at closeY = top + (height-closeSize)/2 and it
	// spans closeSize px. In click-relative coords (OnTabClick subtracts top)
	// that is [(40-12)/2, (40-12)/2+12] = [14, 26].
	glyphMidY := (40 - closeSize) / 2 + closeSize/2 // == 20
	closeX := tabWidth - closeSize - 4 + closeSize/2

	assert.True(t, tb.closeButtonAt(closeX, glyphMidY, tab),
		"a click on the painted X center must hit the close button")

	// The old bottom-aligned band (y in [24,36]) must no longer hit at its old
	// center y=30, which is below the painted glyph's bottom (26).
	assert.False(t, tb.closeButtonAt(closeX, 30, tab),
		"a click below the painted glyph must not hit the close button")
}

// TestCloseButtonHitEndToEnd runs through the public OnTabClick path so the
// coordinate normalization (subtracting computedBox.left/top) is exercised.
func TestCloseButtonHitEndToEnd(t *testing.T) {
	tb := CreateTabBar(40)
	tb.AddTab("First", "https://a.example")
	tb.computedBox.SetCoords(100, 50, 600, 40) // absolute origin

	// Absolute click on the first tab's painted X center.
	absX := float64(50 + (140 - 12 - 4) + 6) // left + closeX + half glyph
	absY := float64(100 + 20)                 // top + glyphMidY
	closed, isClose := tb.OnTabClick(absX, absY)
	assert.True(t, isClose, "OnTabClick on the painted X must report a close")
	assert.NotNil(t, closed)
}
