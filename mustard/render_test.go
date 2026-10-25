package mustard

import (
	"image"
	"image/color"
	"testing"
)

// newRGBAFill returns an image.RGBA of size w x h filled with c.
func newRGBAFill(w, h int, c color.NRGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

// TestRender_PaintsFrameBuffer verifies that WindowBase.render composites the
// widget tree into a non-empty frame buffer instead of returning a blank image.
func TestRender_PaintsFrameBuffer(t *testing.T) {
	window := &WindowBase{title: "test", width: 200, height: 120, visible: true}

	root := CreateFrame(VerticalFrame)
	title := CreateLabelWidget("hello")
	root.AttachWidget(title)
	window.SetRootFrame(root)

	// Mirror the real usage contract: App.AddWindow associates the root
	// frame (and its widget tree) with the window.
	app := CreateNewApp(nil, "test")
	app.AddWindow(window)
	window.Show()

	frame := window.render()
	if frame == nil {
		t.Fatal("render returned nil frame buffer")
	}

	// The root frame fills its box with its background color (#fff), so a
	// corner pixel must be fully opaque white rather than transparent.
	pix := frame.At(195, 115)
	r, g, b, a := pix.RGBA()
	if a>>8 == 0 {
		t.Fatalf("frame pixel at (10,10) is transparent; render did not paint: %+v", pix)
	}
	if r>>8 != 255 || g>>8 != 255 || b>>8 != 255 {
		t.Errorf("expected white background, got rgba(%d,%d,%d,%d)", r>>8, g>>8, b>>8, a>>8)
	}
}

// TestRender_NoRootFrame verifies render is safe before a root frame exists.
func TestRender_NoRootFrame(t *testing.T) {
	window := &WindowBase{title: "test", width: 64, height: 48, visible: true}
	frame := window.render()
	if frame == nil {
		t.Fatal("render returned nil frame buffer")
	}
	if got, want := frame.Bounds().Dx(), 64; got != want {
		t.Errorf("frame width = %d, want %d", got, want)
	}
	if got, want := frame.Bounds().Dy(), 48; got != want {
		t.Errorf("frame height = %d, want %d", got, want)
	}
}

// TestRender_RefreshAfterResize verifies a resize invalidates the frame buffer
// and the next render recreates it at the new size.
func TestRender_RefreshAfterResize(t *testing.T) {
	window := &WindowBase{title: "test", width: 100, height: 100, visible: true}
	window.SetRootFrame(CreateFrame(HorizontalFrame))
	app := CreateNewApp(nil, "test")
	app.AddWindow(window)
	window.Show()

	window.render()
	window.resize(300, 200, 1, 1)

	frame := window.render()
	if got, want := frame.Bounds().Dx(), 300; got != want {
		t.Errorf("frame width after resize = %d, want %d", got, want)
	}
	if got, want := frame.Bounds().Dy(), 200; got != want {
		t.Errorf("frame height after resize = %d, want %d", got, want)
	}
}

// TestRender_Overlays verifies static and active overlays are composited onto
// the frame buffer at their position.
func TestRender_Overlays(t *testing.T) {
	window := &WindowBase{title: "test", width: 100, height: 100, visible: true}
	window.SetRootFrame(CreateFrame(VerticalFrame))
	app := CreateNewApp(nil, "test")
	app.AddWindow(window)
	window.Show()

	red := color.NRGBA{R: 255, A: 255}
	overlayBuffer := newRGBAFill(20, 20, red)
	window.AddStaticOverlay(CreateOverlay("red", overlayBuffer, image.Point{X: 10, Y: 10}))

	frame := window.render()

	r, g, b, a := frame.At(15, 15).RGBA()
	if r>>8 != 255 || g>>8 != 0 || b>>8 != 0 || a>>8 != 255 {
		t.Errorf("expected red overlay pixel at (15,15), got rgba(%d,%d,%d,%d)", r>>8, g>>8, b>>8, a>>8)
	}
}
