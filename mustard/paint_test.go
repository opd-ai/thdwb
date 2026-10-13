package mustard

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func TestChangedWidgetBufferResizesForEitherDimension(t *testing.T) {
	widget := &LabelWidget{baseWidget: baseWidget{buffer: image.NewRGBA(image.Rect(0, 0, 4, 4))}}
	source := image.NewRGBA(image.Rect(0, 0, 12, 12))
	widget.computedBox.SetCoords(0, 0, 6, 4)
	copyWidgetToBuffer(widget, source)
	if got := widget.Buffer().Bounds().Size(); got.X != 6 || got.Y != 4 {
		t.Fatalf("width-only repaint buffer size = %v, want (6,4)", got)
	}
	widget.computedBox.SetCoords(0, 0, 6, 7)
	copyWidgetToBuffer(widget, source)
	if got := widget.Buffer().Bounds().Size(); got.X != 6 || got.Y != 7 {
		t.Fatalf("height-only repaint buffer size = %v, want (6,7)", got)
	}
}

func TestCompositeWidgetUpdatesOnlyDirtyBounds(t *testing.T) {
	background := color.RGBA{B: 255, A: 255}
	red := color.RGBA{R: 255, A: 255}
	frame := image.NewRGBA(image.Rect(0, 0, 6, 6))
	draw.Draw(frame, frame.Bounds(), image.NewUniform(background), image.Point{}, draw.Src)
	widget := &LabelWidget{baseWidget: baseWidget{buffer: image.NewRGBA(image.Rect(0, 0, 2, 2))}}
	draw.Draw(widget.Buffer(), widget.Buffer().Bounds(), image.NewUniform(red), image.Point{}, draw.Src)
	widget.computedBox.SetCoords(1, 1, 2, 2)
	compositeWidget(frame, widget)
	if got := frame.RGBAAt(1, 1); got != background {
		t.Fatalf("clean widget changed frame pixel: got %v, want %v", got, background)
	}
	widget.SetNeedsRepaint(true)
	compositeWidget(frame, widget)
	if got := frame.RGBAAt(1, 1); got != red || frame.RGBAAt(0, 0) != background || widget.NeedsRepaint() {
		t.Fatal("dirty widget did not update only its bounds and clear its repaint flag")
	}
}
