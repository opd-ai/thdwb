package mustard

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	gg "github.com/danfragoso/thdwb/gg"
)

func TestProcessScrollUpdatesScrollableCanvasWithinBounds(t *testing.T) {
	window := &Window{cursorX: 10, cursorY: 10}
	root := &Frame{baseWidget: baseWidget{visible: true}}
	canvas := &CanvasWidget{
		baseWidget:     baseWidget{visible: true},
		context:        gg.NewContext(50, 50),
		drawingContext: gg.NewContext(50, 150),
		scrollable:     true,
	}
	root.AttachWidget(canvas)
	root.window = window
	canvas.window = window
	root.computedBox.SetCoords(0, 0, 50, 50)
	canvas.computedBox.SetCoords(0, 0, 50, 50)
	window.rootFrame = root

	window.ProcessScroll(0, -1)
	window.ProcessScroll(0, -1)
	window.ProcessScroll(0, -1)
	if canvas.GetOffset() != 100 {
		t.Fatalf("scroll offset = %d, want max offset 100", canvas.GetOffset())
	}
	window.ProcessScroll(0, 1)
	if canvas.GetOffset() != 60 {
		t.Fatalf("scroll offset after moving up = %d, want 60", canvas.GetOffset())
	}
}

func TestCanvasOffsetSetBeforeLayoutSurvivesContextInitialization(t *testing.T) {
	canvas := &CanvasWidget{
		context:        gg.NewContext(0, 0),
		drawingContext: gg.NewContext(0, 0),
	}
	canvas.SetOffset(37)
	canvas.computedBox.SetCoords(0, 0, 10, 20)
	canvas.ensureContexts(10, 20)
	if canvas.GetOffset() != 37 {
		t.Fatalf("offset after context initialization = %d, want 37", canvas.GetOffset())
	}
	canvas.EnableScrolling()
	canvas.DisableScrolling()
	if canvas.GetOffset() != 0 || !canvas.NeedsRepaint() {
		t.Fatal("disabling scrolling did not reset the offset and request repaint")
	}
}

func TestCanvasScrollClearsOldViewportPixels(t *testing.T) {
	window := &Window{width: 4, height: 2, context: gg.NewContext(4, 2)}
	root := CreateFrame(VerticalFrame)
	root.SetVisible(true)
	canvas := &CanvasWidget{
		baseWidget:     baseWidget{visible: true, needsRepaint: true},
		context:        gg.NewContext(0, 0),
		drawingContext: gg.NewContext(0, 0),
		drawingRepaint: true,
		renderer: func(canvas *CanvasWidget) {
			content := canvas.GetContext().Image().(*image.RGBA)
			content.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
			content.SetRGBA(0, 1, color.RGBA{G: 255, A: 255})
			content.SetRGBA(0, 3, color.RGBA{B: 255, A: 255})
		},
	}
	root.AttachWidget(canvas)
	canvas.SetVisible(true)
	setWidgetWindow(root, window)
	window.rootFrame = root

	drawRootFrame(window)
	canvas.SetOffset(2)
	drawRootFrame(window)

	pixels := window.context.Image().(*image.RGBA)
	if got := pixels.RGBAAt(0, 0); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("pixel at scrolled transparent row = %v, want cleared frame background", got)
	}
	if got := pixels.RGBAAt(0, 1); got != (color.RGBA{B: 255, A: 255}) {
		t.Fatalf("pixel at scrolled content row = %v, want blue content", got)
	}
}

func TestCanvasDrawUsesLocalViewportCoordinates(t *testing.T) {
	window := &Window{context: gg.NewContext(8, 5)}
	canvas := &CanvasWidget{
		baseWidget:     baseWidget{needsRepaint: true, window: window},
		context:        gg.NewContext(3, 2),
		drawingContext: gg.NewContext(3, 2),
		drawingRepaint: true,
		renderer: func(canvas *CanvasWidget) {
			draw.Draw(canvas.GetContext().Image().(draw.Image), image.Rect(0, 0, 3, 2), image.NewUniform(color.RGBA{R: 255, A: 255}), image.Point{}, draw.Src)
		},
	}
	canvas.computedBox.SetCoords(1, 2, 3, 2)

	canvas.draw()

	pixels := window.context.Image().(*image.RGBA)
	if got := pixels.RGBAAt(2, 1); got != (color.RGBA{R: 255, A: 255}) {
		t.Fatalf("canvas pixel at its origin = %v, want red", got)
	}
	if got := pixels.RGBAAt(0, 0); got != (color.RGBA{}) {
		t.Fatalf("canvas changed a pixel outside its bounds: %v", got)
	}
}
