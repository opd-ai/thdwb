package hotdog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetachedNodesAreNotPinned (audit C-22) verifies that nodes created by
// page JS but never attached to the document tree do not accumulate in the
// wrapper map. wrapNode used to cache every node that crossed into JS keyed by
// *NodeDOM, and unbindSubtree only ran on removal-from-parent, so scratch
// nodes (createElement without appendChild, or dropped by `x.y = null`) were
// pinned for the life of the tab -- the probe retained 99.3 MB after GC for
// 5000 detached divs.
func TestDetachedNodesAreNotPinned(t *testing.T) {
	wc := newBindingTestContext(t, `<html><head></head><body></body></html>`)

	_, err := wc.RunJS(`
		for (var i = 0; i < 500; i++) {
			var d = document.createElement("div");
			d.setAttribute("id", "x" + i);
			var t = document.createTextNode("t");
			d.appendChild(t); // parent is detached: still not in the tree
			void d.textContent;
		}
	`)
	require.NoError(t, err)

	wc.jsRuntimeMu.lock()
	pinned := len(wc.nodeWrappers)
	wc.jsRuntimeMu.unlock()

	// Only the document root / body wrappers created during init may remain;
	// the 500 detached divs and their text children must not be cached.
	assert.LessOrEqual(t, pinned, 4,
		"detached nodes must not accumulate in nodeWrappers (got %d)", pinned)
}

// TestAttachedNodeKeepsWrapperIdentity verifies the attach paths cache the
// object JS handed us, so identity survives `el === parent.appendChild(el)`
// and tree traversals return the same wrapper for the same node.
func TestAttachedNodeKeepsWrapperIdentity(t *testing.T) {
	wc := newBindingTestContext(t, `<html><head></head><body></body></html>`)

	value, err := wc.RunJS(`
		var el = document.createElement("section");
		var returned = document.body.appendChild(el);
		var same = (returned === el);
		var inTree = (document.body.lastChild() === el);
		same + "," + inTree
	`)
	require.NoError(t, err)
	assert.Equal(t, "true,true", value.String())
}
