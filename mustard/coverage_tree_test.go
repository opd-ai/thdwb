package mustard

import (
	"testing"

	assets "github.com/danfragoso/thdwb/assets"
	gg "github.com/danfragoso/thdwb/gg"
)

// newTreeWidgetWithFont returns a TreeWidget with a real font face and icons so
// its draw path (which casts font.(font.Face) and draws the chevron icons) is
// exercised without any platform backend.
func newTreeWidgetWithFont(t *testing.T) *TreeWidget {
	t.Helper()
	tree := CreateTreeWidget()
	face, _ := loadFontFace(assets.OpenSans(400), 20)
	tree.font = face
	openIcon, _ := gg.LoadAsset(assets.DownChevron())
	closeIcon, _ := gg.LoadAsset(assets.RightChevron())
	tree.openIcon = openIcon
	tree.closeIcon = closeIcon
	tree.fontSize = 20
	tree.fontColor = "#000"
	return tree
}

func TestGetIntersectedNode(t *testing.T) {
	// The open root's box is tall enough to include its child region, as
	// flowNode would produce (base row plus the child rows).
	root := CreateTreeWidgetNode("root", "r")
	root.box.SetCoords(0, 0, 100, 60)
	child := CreateTreeWidgetNode("child", "c")
	child.box.SetCoords(20, 0, 100, 40)
	root.AddNode(child)

	// Point inside the closed root returns the root directly.
	if got := getIntersectedNode([]*TreeWidgetNode{root}, 10, 10); got != root {
		t.Fatalf("closed root hit = %v, want root", got)
	}

	// Open root with a point inside the child descends to the child.
	root.isOpen = true
	if got := getIntersectedNode([]*TreeWidgetNode{root}, 10, 30); got != child {
		t.Fatalf("open root, child hit = %v, want child", got)
	}

	// Point inside the open root (not the child) returns the root.
	if got := getIntersectedNode([]*TreeWidgetNode{root}, 10, 5); got != root {
		t.Fatalf("open root, self hit = %v, want root", got)
	}

	// A miss returns nil.
	if got := getIntersectedNode([]*TreeWidgetNode{root}, 200, 200); got != nil {
		t.Fatalf("miss = %v, want nil", got)
	}
}

func TestFlowNodePositionsRootsSiblingsAndOpenChildren(t *testing.T) {
	window := newTestWindow(t, 200, 200)
	tree := newTreeWidgetWithFont(t)
	tree.window = window
	tree.computedBox.SetCoords(0, 0, 200, 200)
	ctx := window.context.(*gg.Context)

	rootA := CreateTreeWidgetNode("A", "a")
	tree.AddNode(rootA)
	childA := CreateTreeWidgetNode("A1", "a1")
	childA2 := CreateTreeWidgetNode("A2", "a2")
	rootA.AddNode(childA)
	rootA.AddNode(childA2)
	rootA.isOpen = true

	flowNode(ctx, rootA, tree, 0)

	// First root has no parent/sibling so it anchors at the top.
	if rootA.box.top != 0 {
		t.Fatalf("rootA.top = %v, want 0", rootA.box.top)
	}
	// Open root grows to include both children (24 base + 24 + 24).
	if rootA.box.height != 72 {
		t.Fatalf("open rootA height = %v, want 72", rootA.box.height)
	}
	// First child indented by fontSize * level and stacked under its parent.
	if childA.box.left != 20 || childA.box.top != 24 {
		t.Fatalf("childA = left %v top %v, want 20/24", childA.box.left, childA.box.top)
	}
	// Second child stacks below the first child.
	if childA2.box.top != 48 {
		t.Fatalf("childA2.top = %v, want 48", childA2.box.top)
	}
}

func TestFlowNodeChildWithNoParentOffset(t *testing.T) {
	window := newTestWindow(t, 200, 200)
	tree := newTreeWidgetWithFont(t)
	tree.window = window
	tree.computedBox.SetCoords(0, 0, 200, 200)
	ctx := window.context.(*gg.Context)

	// Top-level nodes have a nil Parent, so PreviousSibling() is also nil and
	// each anchors at top 0 (sibling stacking only happens within a parent).
	first := CreateTreeWidgetNode("f", "f")
	second := CreateTreeWidgetNode("s", "s")
	tree.AddNode(first)
	tree.AddNode(second)
	flowNode(ctx, first, tree, 0)
	flowNode(ctx, second, tree, 0)
	if second.box.top != 0 {
		t.Fatalf("second.root top = %v, want 0", second.box.top)
	}
}

func TestDrawTreeWithOpenAndClosedChildren(t *testing.T) {
	window := newTestWindow(t, 200, 200)
	tree := newTreeWidgetWithFont(t)
	tree.window = window
	tree.computedBox.SetCoords(0, 0, 200, 200)
	ctx := window.context.(*gg.Context)

	openRoot := CreateTreeWidgetNode("open", "o")
	openRoot.isOpen = true
	child := CreateTreeWidgetNode("child", "c")
	openRoot.AddNode(child)
	closedRoot := CreateTreeWidgetNode("closed", "cl")
	tree.AddNode(openRoot)
	tree.AddNode(closedRoot)

	flowNode(ctx, openRoot, tree, 0)
	flowNode(ctx, closedRoot, tree, 0)
	drawNode(ctx, openRoot, tree, 0)
	drawNode(ctx, closedRoot, tree, 0)

	if window.context == nil {
		t.Fatal("drawing the tree lost the window context")
	}
}

func TestTreeSelectNodeByValueAndDeselect(t *testing.T) {
	tree := newTreeWidgetWithFont(t)
	root := CreateTreeWidgetNode("root", "r")
	child := CreateTreeWidgetNode("child", "target")
	root.AddNode(child)
	tree.AddNode(root)

	tree.SelectNodeByValue("target")
	if !child.isSelected || root.isSelected {
		t.Fatalf("select by value: root.sel=%v child.sel=%v, want root=false child=true", root.isSelected, child.isSelected)
	}

	// Selecting a different node deselects the previous one.
	tree.SelectNode(root)
	if !root.isSelected || child.isSelected {
		t.Fatalf("select root: root.sel=%v child.sel=%v, want root=true child=false", root.isSelected, child.isSelected)
	}
}

func TestFireNodeEventsIgnoresClickOutsideNodeHeight(t *testing.T) {
	tree := newTreeWidgetWithFont(t)
	tree.window = newTestWindow(t, 100, 100)
	node := CreateTreeWidgetNode("n", "n")
	node.box.SetCoords(50, 0, 100, 20) // top=50
	tree.AddNode(node)
	fired := false
	tree.selectCallback = func(*TreeWidgetNode) { fired = true }

	// y below the node row must not select/toggle.
	fireNodeEvents(tree, node, 40, 80, 20)
	if fired || node.isSelected {
		t.Fatal("expected no selection when clicking below the node row")
	}
}

func TestCreateTreeWidgetConstructor(t *testing.T) {
	tree := CreateTreeWidget()
	if tree == nil || tree.widgetType != treeWidget {
		t.Fatalf("CreateTreeWidget = %v, want a non-nil tree widget", tree)
	}
}
