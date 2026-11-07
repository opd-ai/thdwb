package hotdog

import (
	"testing"
	"time"

	"github.com/danfragoso/thdwb/assets"
	profiler "github.com/danfragoso/thdwb/profiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRemoveChildUnbindsSubtree (audit R-4) verifies that removing an element
// detaches wrapper and event-registry entries for the WHOLE subtree. Before the
// fix only the direct child was deleted from nodeWrappers, so every wrapped
// descendant stayed reachable through the map key forever -- a permanent leak
// of the detached subtree and its goja objects.
func TestRemoveChildUnbindsSubtree(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	require.NoError(t, wc.LoadHTML(`<html><body></body></html>`, "https://dom.example"))

	w := &JSDOMWrapper{runtime: wc.GetJSRuntime(), windowCtx: wc}

	elem := func(name string) *NodeDOM {
		n := &NodeDOM{Type: NodeTypeElement, Element: name, WindowCtx: wc}
		w.wrapNode(n)
		return n
	}

	// parent -> [c1 -> g1, c2 -> g2]
	parent, c1, g1, c2, g2 := elem("div"), elem("span"), elem("b"), elem("span"), elem("b")
	c1.appendChildNode(g1)
	c2.appendChildNode(g2)
	parent.appendChildNode(c1)
	parent.appendChildNode(c2)

	host := &NodeDOM{Type: NodeTypeElement, Element: "body", WindowCtx: wc, Children: []*NodeDOM{}}
	host.appendChildNode(parent)

	// Register a listener deep inside the subtree.
	wc.registerNodeEvent(g1, "click", "k", func(interface{}) {})

	wc.eventMu.Lock()
	before := len(wc.nodeWrappers)
	_, g1Wrapped := wc.nodeWrappers[g1]
	_, g1HasEvents := wc.nodeEventRegistry[g1]
	wc.eventMu.Unlock()
	require.True(t, g1Wrapped, "grandchild must be wrapped before removal")
	require.True(t, g1HasEvents, "grandchild listener must be registered before removal")
	require.Equal(t, 5, before, "parent + 2 children + 2 grandchildren wrapped")

	host.removeChildNode(parent)

	wc.eventMu.Lock()
	after := len(wc.nodeWrappers)
	for _, n := range []*NodeDOM{parent, c1, g1, c2, g2} {
		_, stillWrapped := wc.nodeWrappers[n]
		assert.False(t, stillWrapped, "wrapper for %q must be removed with the subtree", n.Element)
		_, stillEvents := wc.nodeEventRegistry[n]
		assert.False(t, stillEvents, "event registry for %q must be removed with the subtree", n.Element)
	}
	wc.eventMu.Unlock()
	assert.Equal(t, 0, after, "wrapper map must shrink to zero on subtree removal")
}

// TestUnbindSubtreeIsCycleSafe guards the traversal itself: a corrupted tree
// with a Parent/Children loop must not send unbindSubtree into infinite
// recursion (a Go stack overflow is unrecoverable and kills the process).
func TestUnbindSubtreeTerminates(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	require.NoError(t, wc.LoadHTML(`<html><body></body></html>`, "https://dom.example"))

	a := &NodeDOM{Type: NodeTypeElement, Element: "a", WindowCtx: wc}
	b := &NodeDOM{Type: NodeTypeElement, Element: "b", WindowCtx: wc, Parent: a}
	a.Children = append(a.Children, b)
	// Corrupt the tree so b lists its own ancestor a as a child.
	b.Children = append(b.Children, a)

	// Must terminate (visited-set) rather than recurse forever.
	done := make(chan struct{})
	go func() {
		wc.unbindSubtree(a)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("unbindSubtree did not terminate on a cyclic tree")
	}

	// Both nodes of the cycle must be unbound exactly once without hanging.
	wc.eventMu.Lock()
	_, aBound := wc.nodeWrappers[a]
	_, bBound := wc.nodeWrappers[b]
	wc.eventMu.Unlock()
	assert.False(t, aBound, "cyclic node a must be unbound")
	assert.False(t, bBound, "cyclic node b must be unbound")
}
