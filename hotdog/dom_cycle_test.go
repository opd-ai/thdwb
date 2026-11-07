package hotdog

import (
	"testing"

	"github.com/danfragoso/thdwb/assets"
	profiler "github.com/danfragoso/thdwb/profiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildCycleTree returns root -> mid -> leaf, wired as parent/child.
func buildCycleTree() (root, mid, leaf *NodeDOM) {
	root = &NodeDOM{Type: NodeTypeElement, Element: "div", Children: []*NodeDOM{}}
	mid = &NodeDOM{Type: NodeTypeElement, Element: "p", Parent: root, Children: []*NodeDOM{}}
	leaf = &NodeDOM{Type: NodeTypeElement, Element: "span", Parent: mid, Children: []*NodeDOM{}}
	root.Children = append(root.Children, mid)
	mid.Children = append(mid.Children, leaf)
	return root, mid, leaf
}

func childElems(n *NodeDOM) []string {
	var out []string
	for _, c := range n.Children {
		out = append(out, c.Element)
	}
	return out
}

// TestInsertBeforeRejectsAncestor guards the fatal stack overflow that occurred
// when a node was inserted into one of its own ancestors: MarkDirty recurses up
// the Parent chain, so a Parent/Children loop overflows the stack, and a Go
// stack overflow is not recoverable — it kills the process from page JS.
func TestInsertBeforeRejectsAncestor(t *testing.T) {
	root, mid, leaf := buildCycleTree()

	// Inserting an ancestor (root) under its own descendant (mid) must error
	// and must not link any pointer.
	err := mid.InsertBefore(root, leaf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ancestor")
	assert.Same(t, root, mid.Parent, "failed insert must leave Parent untouched")

	// Self-insertion is also a cycle.
	require.Error(t, leaf.InsertBefore(leaf, nil))

	// A legitimate sibling insert into the same parent still succeeds.
	sib := &NodeDOM{Type: NodeTypeElement, Element: "em"}
	require.NoError(t, mid.InsertBefore(sib, leaf))
	assert.Equal(t, 2, len(mid.Children))
}

// TestReplaceChildRejectsAncestor guards the same class via ReplaceChild.
func TestReplaceChildRejectsAncestor(t *testing.T) {
	root, mid, leaf := buildCycleTree()

	got, err := mid.ReplaceChild(root, leaf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ancestor")
	assert.Nil(t, got)
	assert.Same(t, root, mid.Parent, "failed replace must leave Parent untouched")

	// A legitimate replacement still works.
	repl := &NodeDOM{Type: NodeTypeElement, Element: "b"}
	old, err := mid.ReplaceChild(repl, leaf)
	require.NoError(t, err)
	assert.Same(t, leaf, old)
	assert.Equal(t, 1, len(mid.Children))
}

// TestAppendChildNodeRejectsAncestor confirms the existing appendChild guard
// still holds after the refactor to the shared wouldCreateCycle helper.
func TestAppendChildNodeRejectsAncestor(t *testing.T) {
	root, mid, leaf := buildCycleTree()

	mid.appendChildNode(root) // ancestor -> must be rejected
	assert.NotContains(t, childElems(mid), "div", "appendChild must not add an ancestor")
	assert.Same(t, root, mid.Parent, "rejected append must not reparent")

	mid.appendChildNode(&NodeDOM{Type: NodeTypeElement, Element: "i"}) // valid
	assert.Contains(t, childElems(mid), "i")
	_ = leaf
}

// TestJSInsertBeforeAncestorThrows exercises the page-JS surface (the real
// attack path) and asserts it throws instead of building a cycle.
func TestJSInsertBeforeAncestorThrows(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	if err := wc.LoadHTML("<html><body><div id='x'></div></body></html>", "https://dom.example"); err != nil {
		t.Fatalf("LoadHTML: %v", err)
	}

	v, err := wc.RunJS(`
		var d = document.createElement("div");
		var p = document.createElement("p");
		var s = document.createElement("span");
		d.appendChild(p);
		p.appendChild(s);
		var outcome = "accepted";
		try { p.insertBefore(d, s); } catch (e) { outcome = "rejected"; }
		outcome;
	`)
	require.NoError(t, err)
	assert.Equal(t, "rejected", v.String(), "insertBefore(ancestor) must throw, not build a cycle")

	v, err = wc.RunJS(`
		var d2 = document.createElement("div");
		var p2 = document.createElement("p");
		var s2 = document.createElement("span");
		d2.appendChild(p2);
		p2.appendChild(s2);
		var outcome2 = "accepted";
		try { p2.replaceChild(d2, s2); } catch (e) { outcome2 = "rejected"; }
		outcome2;
	`)
	require.NoError(t, err)
	assert.Equal(t, "rejected", v.String(), "replaceChild(ancestor) must throw, not build a cycle")
}
