package bun

import (
	"runtime/debug"
	"testing"

	"github.com/danfragoso/thdwb/assets"
	gg "github.com/danfragoso/thdwb/gg"
	"github.com/danfragoso/thdwb/hotdog"
	"github.com/danfragoso/thdwb/ketchup"
	profiler "github.com/danfragoso/thdwb/profiler"
)

// renderBodyless runs the live render path (x/net-backed ParseHTMLDocument,
// the same one browser/shell uses) over src and lays it out with the default
// experimentalLayout=false. It returns without error if no panic occurred.
func renderBodyless(t *testing.T, src string) {
	t.Helper()

	windowCtx := hotdog.NewWindowContext(&hotdog.Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	doc := ketchup.ParseHTMLDocument(src, windowCtx)
	if doc == nil {
		t.Fatal("got nil document")
	}

	dctx := gg.NewContext(1024, 1024)
	if err := RenderDocument(dctx, doc, false); err != nil {
		t.Fatalf("RenderDocument failed: %v", err)
	}
}

// TestRenderDocumentFramesetNoPanic guards the nil-parent crash on the live
// layout path: when a document has no <body> (frameset documents, which
// x/net/html does not synthesize one for), RenderDocument falls back to the
// root node as the layout root, and block/list-item layout then dereferenced
// the root's nil Parent for childIdx == 0.
func TestRenderDocumentFramesetNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v\n%s", r, debug.Stack())
		}
	}()

	renderBodyless(t, `<html><frameset cols="50%,50%"><frame src="a.html"><frame src="b.html"></frameset></html>`)
	renderBodyless(t, `<html><head><title>no body here</title></head></html>`)
}

// TestResolveBlockTopNilParent exercises the helper directly for the root case
// and the sibling-stacking case.
func TestResolveBlockTopNilParent(t *testing.T) {
	root := &hotdog.NodeDOM{
		Type:      hotdog.NodeTypeElement,
		RenderBox: &hotdog.RenderBox{Top: 7, Height: 20},
	}
	// A node with no parent must resolve to 0 rather than dereferencing nil.
	orphan := &hotdog.NodeDOM{Type: hotdog.NodeTypeElement, RenderBox: &hotdog.RenderBox{}}
	if got := resolveBlockTop(orphan, 0); got != 0 {
		t.Fatalf("nil-parent top = %v, want 0", got)
	}

	child := &hotdog.NodeDOM{Type: hotdog.NodeTypeElement, Parent: root, RenderBox: &hotdog.RenderBox{Top: 7, Height: 20}}
	root.Children = []*hotdog.NodeDOM{child}

	// childIdx == 0 aligns with the parent top.
	if got := resolveBlockTop(child, 0); got != 7 {
		t.Fatalf("first-child top = %v, want 7", got)
	}

	// childIdx == 1 stacks below the previous sibling (7 + 20).
	second := &hotdog.NodeDOM{Type: hotdog.NodeTypeElement, Parent: root, RenderBox: &hotdog.RenderBox{}}
	root.Children = append(root.Children, second)
	if got := resolveBlockTop(second, 1); got != 27 {
		t.Fatalf("second-child top = %v, want 27", got)
	}

	// Out-of-range childIdx must not panic.
	if got := resolveBlockTop(second, 9); got != 7 {
		t.Fatalf("out-of-range top = %v, want 7 (parent top fallback)", got)
	}

	// A nil previous sibling box must not panic.
	root.Children = []*hotdog.NodeDOM{{Type: hotdog.NodeTypeElement}, second}
	if got := resolveBlockTop(second, 1); got != 7 {
		t.Fatalf("nil-prev-box top = %v, want 7", got)
	}
}
