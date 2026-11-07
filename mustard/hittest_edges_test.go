package mustard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsPointInsideEdgeInclusive (audit O-12) pins the hit-test boundary: the
// top/left edge pixel must count as inside (strict > used to exclude it, so a
// click exactly on a control's first column/row missed), while the
// bottom/right edge is exclusive so two adjacent widgets never both claim the
// seam pixel.
func TestIsPointInsideEdgeInclusive(t *testing.T) {
	widget := &baseWidget{
		computedBox: box{top: 10, left: 20, width: 100, height: 50},
		window:      &WindowBase{width: 400, height: 400},
	}

	assert.True(t, widget.IsPointInside(20, 10), "top-left corner pixel must be inside")
	assert.True(t, widget.IsPointInside(119.9, 59.9), "just below bottom-right must be inside")

	// Exclusive upper edges: the seam belongs to the neighbor.
	assert.False(t, widget.IsPointInside(120, 10), "x == left+width must be outside")
	assert.False(t, widget.IsPointInside(20, 60), "y == top+height must be outside")

	// Outside on the low side stays outside.
	assert.False(t, widget.IsPointInside(19.9, 55))
	assert.False(t, widget.IsPointInside(60, 9.9))
}
