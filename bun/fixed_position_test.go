package bun

import (
	"testing"

	"github.com/danfragoso/thdwb/hotdog"
	"github.com/kjk/flex"
	"github.com/stretchr/testify/assert"
)

// TestFixedPositionMapsToAbsolute (audit O-16) verifies position:fixed is taken
// out of flow. The vendored kjk/flex (2017) has no Fixed position type, and the
// switch in applyStylesToFlexNode had no "fixed" case at all, so a fixed node
// silently kept the default Relative: it stayed in flow and its offsets
// resolved against the flow parent instead of behaving like absolute.

func TestFixedPositionMapsToAbsolute(t *testing.T) {
	node := flex.NewNode()
	applyStylesToFlexNode(node, &hotdog.Stylesheet{
		Display: "block", Position: "fixed", Top: 10, Left: 20,
		Width: 100, Height: 50,
	})

	assert.Equal(t, flex.PositionTypeAbsolute, node.Style.PositionType,
		"position:fixed must degrade to Absolute (out of flow)")

	// Offsets must be applied too (the outer gate includes fixed).
	assert.EqualValues(t, 10, node.Style.Position[flex.EdgeTop].Value)
	assert.EqualValues(t, 20, node.Style.Position[flex.EdgeLeft].Value)
}

// TestFixedActuallyLeavesFlow runs a real layout: a fixed child must not push a
// following sibling down the way an in-flow (relative) child would.
func TestFixedActuallyLeavesFlow(t *testing.T) {
	root := flex.NewNode()
	applyStylesToFlexNode(root, &hotdog.Stylesheet{Display: "flex", Width: 200, Height: 200})

	fixed := flex.NewNode()
	applyStylesToFlexNode(fixed, &hotdog.Stylesheet{
		Display: "block", Position: "fixed", Top: 0, Left: 0, Width: 200, Height: 100,
	})
	root.InsertChild(fixed, 0)

	next := flex.NewNode()
	applyStylesToFlexNode(next, &hotdog.Stylesheet{Display: "block", Width: 200, Height: 30})
	root.InsertChild(next, 1)

	flex.CalculateLayout(root, 200, 200, flex.DirectionLTR)

	// With fixed degraded to Absolute the sibling starts at the top of the
	// container; the buggy Relative case left it in flow below the 100px box.
	assert.LessOrEqual(t, next.Layout.Position[flex.EdgeTop], float32(1),
		"a position:fixed sibling must not consume flow space")
}
