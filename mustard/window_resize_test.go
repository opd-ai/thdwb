package mustard

import (
	"image"
	"testing"

	gg "github.com/danfragoso/thdwb/gg"
)

func TestWindowResizeRecreatesPerWindowViewport(t *testing.T) {
	first := &Window{width: 400, height: 300, context: gg.NewContext(400, 300)}
	second := &Window{width: 250, height: 200, context: gg.NewContext(250, 200)}
	oldContext := first.context
	first.frameBuffer = image.NewRGBA(image.Rect(0, 0, 400, 300))
	first.resize(1600, 1200, 2, 2)
	if first.width != 800 || first.height != 600 || first.context == oldContext {
		t.Fatalf("resize did not update logical viewport and context: %dx%d", first.width, first.height)
	}
	if first.context.Image().Bounds().Dx() != 800 || first.context.Image().Bounds().Dy() != 600 || first.frameBuffer != nil || !first.needsReflow {
		t.Fatal("resize did not recreate the drawing surface, discard stale pixels, and request reflow")
	}
	if second.width != 250 || second.height != 200 || second.needsReflow {
		t.Fatal("resizing one window changed another window")
	}
	first.needsReflow = false
	context := first.context
	first.resize(0, 0, 2, 2)
	if first.context != context || first.needsReflow {
		t.Fatal("invalid minimized size replaced the viewport or requested reflow")
	}
}
