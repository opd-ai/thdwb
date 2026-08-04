package bun

import (
	"io/ioutil"
	"runtime/debug"
	"testing"

	gg "github.com/danfragoso/thdwb/gg"
	"github.com/danfragoso/thdwb/hotdog"
	"github.com/danfragoso/thdwb/ketchup"
	profiler "github.com/danfragoso/thdwb/profiler"
	"github.com/kjk/flex"
)

func TestRenderDocument_noBody(t *testing.T) {
	html, err := ioutil.ReadFile("test-data/no-body.html")
	if err != nil {
		t.Fatalf("got unexpected error: %s", err)
	}

	settings := &hotdog.Settings{}
	buildInfo := &hotdog.BuildInfo{}
	prof := profiler.CreateProfiler()
	windowCtx := hotdog.NewWindowContext(settings, buildInfo, prof)

	doc := ketchup.ParseHTML(string(html), windowCtx)
	if doc == nil {
		t.Fatal("got nil document")
	}

	dctx := gg.NewContext(1024, 1024)

	defer func() {
		if err := recover(); err != nil {
			stack := debug.Stack()
			t.Fatalf("got unexpected panic: %s. Stack: %s", err, stack)
		}
	}()

	RenderDocument(dctx, doc, false)
}

func TestRenderDocument_flexbox(t *testing.T) {
	html, err := ioutil.ReadFile("test-data/flexbox-test.html")
	if err != nil {
		t.Fatalf("got unexpected error: %s", err)
	}

	settings := &hotdog.Settings{}
	buildInfo := &hotdog.BuildInfo{}
	prof := profiler.CreateProfiler()
	windowCtx := hotdog.NewWindowContext(settings, buildInfo, prof)

	doc := ketchup.ParseHTMLDocument(string(html), windowCtx)
	if doc == nil {
		t.Fatal("got nil document")
	}

	dctx := gg.NewContext(1024, 1024)

	defer func() {
		if err := recover(); err != nil {
			stack := debug.Stack()
			t.Fatalf("got unexpected panic: %s. Stack: %s", err, stack)
		}
	}()

	// Test with experimental layout (flexbox)
	RenderDocument(dctx, doc, true)

	// Verify that flex nodes were created
	body, err := doc.DOM.FindChildByName("body")
	if err != nil || body == nil {
		t.Fatal("body element not found")
	}

	// Find the flex container
	flexContainer := findElementByClass(body, "flex-container")
	if flexContainer == nil {
		t.Fatal("flex-container element not found")
	}

	// Verify flex node exists
	if flexContainer.FlexNode == nil {
		t.Fatal("flex node not created for flex-container")
	}

	// Verify children have flex nodes
	flexItemCount := 0
	for _, child := range flexContainer.Children {
		if child.Type == hotdog.NodeTypeElement {
			flexItemCount++
			if child.FlexNode == nil {
				t.Fatalf("flex node not created for flex-item %d", flexItemCount-1)
			}
		}
	}
	if flexItemCount != 3 {
		t.Fatalf("expected 3 flex items, got %d", flexItemCount)
	}
}

// findElementByClass finds an element with the given class attribute
func findElementByClass(root *hotdog.NodeDOM, className string) *hotdog.NodeDOM {
	for _, attr := range root.Attributes {
		if attr.Name == "class" && attr.Value == className {
			return root
		}
	}
	for _, child := range root.Children {
		if found := findElementByClass(child, className); found != nil {
			return found
		}
	}
	return nil
}

func TestRenderDocument_complexNestedFlexbox(t *testing.T) {
	html, err := ioutil.ReadFile("test-data/complex-nested-flexbox.html")
	if err != nil {
		t.Fatalf("got unexpected error: %s", err)
	}

	settings := &hotdog.Settings{}
	buildInfo := &hotdog.BuildInfo{}
	prof := profiler.CreateProfiler()
	windowCtx := hotdog.NewWindowContext(settings, buildInfo, prof)

	doc := ketchup.ParseHTMLDocument(string(html), windowCtx)
	if doc == nil {
		t.Fatal("got nil document")
	}

	dctx := gg.NewContext(1024, 1024)

	defer func() {
		if err := recover(); err != nil {
			stack := debug.Stack()
			t.Fatalf("got unexpected panic: %s. Stack: %s", err, stack)
		}
	}()

	// Test with experimental layout (flexbox)
	RenderDocument(dctx, doc, true)

	// Verify that flex nodes were created for the root container
	body, err := doc.DOM.FindChildByName("body")
	if err != nil || body == nil {
		t.Fatal("body element not found")
	}

	// Find the root container
	rootContainer := findElementByClass(body, "root-container")
	if rootContainer == nil {
		t.Fatal("root-container element not found")
	}

	// Verify flex node exists for root container
	if rootContainer.FlexNode == nil {
		t.Fatal("flex node not created for root-container")
	}

	// Verify root container has correct flex direction (column)
	flexNode := rootContainer.FlexNode.(*flex.Node)
	if flexNode.Style.FlexDirection != flex.FlexDirectionColumn {
		t.Errorf("expected flex-direction column for root-container, got %v", flexNode.Style.FlexDirection)
	}

	// Verify children of root container have flex nodes
	expectedRootChildren := 3 // horizontal-container, vertical-container, relative-container
	// Filter for element nodes only (skip text nodes and comments)
	var elementChildren []*hotdog.NodeDOM
	for _, child := range rootContainer.Children {
		if child.Type == hotdog.NodeTypeElement {
			elementChildren = append(elementChildren, child)
		}
	}
	if len(elementChildren) != expectedRootChildren {
		t.Fatalf("expected %d root element children, got %d", expectedRootChildren, len(elementChildren))
	}

	for i, child := range elementChildren {
		if child.FlexNode == nil {
			t.Fatalf("flex node not created for root child %d", i)
		}
	}

	// Test horizontal container (first child)
	horizontalContainer := findElementByClass(rootContainer, "horizontal-container")
	if horizontalContainer == nil {
		t.Fatal("horizontal-container not found in root")
	}
	if horizontalContainer.FlexNode == nil {
		t.Fatal("flex node not created for horizontal-container")
	}
	hFlexNode := horizontalContainer.FlexNode.(*flex.Node)
	if hFlexNode.Style.FlexDirection != flex.FlexDirectionRow {
		t.Errorf("expected flex-direction row for horizontal-container, got %v", hFlexNode.Style.FlexDirection)
	}

	// Verify horizontal container has 3 children (element nodes only)
	var hElementChildren []*hotdog.NodeDOM
	for _, child := range horizontalContainer.Children {
		if child.Type == hotdog.NodeTypeElement {
			hElementChildren = append(hElementChildren, child)
		}
	}
	if len(hElementChildren) != 3 {
		t.Fatalf("expected 3 horizontal element children, got %d", len(hElementChildren))
	}

	// Test vertical container (second child)
	verticalContainer := findElementByClass(rootContainer, "vertical-container")
	if verticalContainer == nil {
		t.Fatal("vertical-container not found in root")
	}
	if verticalContainer.FlexNode == nil {
		t.Fatal("flex node not created for vertical-container")
	}
	vFlexNode := verticalContainer.FlexNode.(*flex.Node)
	if vFlexNode.Style.FlexDirection != flex.FlexDirectionColumn {
		t.Errorf("expected flex-direction column for vertical-container, got %v", vFlexNode.Style.FlexDirection)
	}

	// Verify vertical container has 3 children (element nodes only)
	var vElementChildren []*hotdog.NodeDOM
	for _, child := range verticalContainer.Children {
		if child.Type == hotdog.NodeTypeElement {
			vElementChildren = append(vElementChildren, child)
		}
	}
	if len(vElementChildren) != 3 {
		t.Fatalf("expected 3 vertical element children, got %d", len(vElementChildren))
	}

	// Test nested horizontal container inside vertical
	nestedHorizontal := findElementByClass(verticalContainer, "horizontal-container")
	if nestedHorizontal == nil {
		t.Fatal("nested horizontal-container not found in vertical-container")
	}
	if nestedHorizontal.FlexNode == nil {
		t.Fatal("flex node not created for nested horizontal-container")
	}

	// Test deep nested container (wrap)
	deepNested := findElementByClass(nestedHorizontal, "deep-nested")
	if deepNested == nil {
		t.Fatal("deep-nested not found")
	}
	if deepNested.FlexNode == nil {
		t.Fatal("flex node not created for deep-nested")
	}
	dnFlexNode := deepNested.FlexNode.(*flex.Node)
	if dnFlexNode.Style.FlexDirection != flex.FlexDirectionRow {
		t.Errorf("expected flex-direction row for deep-nested, got %v", dnFlexNode.Style.FlexDirection)
	}
	if dnFlexNode.Style.FlexWrap != flex.WrapWrap {
		t.Errorf("expected flex-wrap wrap for deep-nested, got %v", dnFlexNode.Style.FlexWrap)
	}

	// Verify deep nested has 4 children (element nodes only)
	var dnElementChildren []*hotdog.NodeDOM
	for _, child := range deepNested.Children {
		if child.Type == hotdog.NodeTypeElement {
			dnElementChildren = append(dnElementChildren, child)
		}
	}
	if len(dnElementChildren) != 4 {
		t.Fatalf("expected 4 deep nested element children, got %d", len(dnElementChildren))
	}

	// Test relative container with absolute positioning
	relativeContainer := findElementByClass(rootContainer, "relative-container")
	if relativeContainer == nil {
		t.Fatal("relative-container not found")
	}
	if relativeContainer.FlexNode == nil {
		t.Fatal("flex node not created for relative-container")
	}

	// Verify absolute item exists as child
	absoluteItem := findElementByClass(relativeContainer, "absolute-item")
	if absoluteItem == nil {
		t.Fatal("absolute-item not found in relative-container")
	}
	if absoluteItem.FlexNode == nil {
		t.Fatal("flex node not created for absolute-item")
	}
	absFlexNode := absoluteItem.FlexNode.(*flex.Node)
	if absFlexNode.Style.PositionType != flex.PositionTypeAbsolute {
		t.Errorf("expected position absolute for absolute-item, got %v", absFlexNode.Style.PositionType)
	}
}
