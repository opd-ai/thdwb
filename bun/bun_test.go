package bun

import (
	"io/ioutil"
	"runtime/debug"
	"testing"

	gg "github.com/danfragoso/thdwb/gg"
	"github.com/danfragoso/thdwb/hotdog"
	"github.com/danfragoso/thdwb/ketchup"
	profiler "github.com/danfragoso/thdwb/profiler"
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
	if len(flexContainer.Children) != 3 {
		t.Fatalf("expected 3 flex items, got %d", len(flexContainer.Children))
	}

	for i, child := range flexContainer.Children {
		if child.FlexNode == nil {
			t.Fatalf("flex node not created for flex-item %d", i)
		}
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
