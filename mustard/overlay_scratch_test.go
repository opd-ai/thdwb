package mustard

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

// newTestOverlay builds an active overlay with a solid-color buffer at pos.
func newTestOverlay(ref string, size image.Point, pos image.Point, c color.RGBA) *Overlay {
	buf := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	draw.Draw(buf, buf.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	return &Overlay{ref: ref, active: true, buffer: buf, position: pos}
}

func TestCompositeOverlaysReusesScratchBuffer(t *testing.T) {
	window := &WindowBase{}
	frame := image.NewRGBA(image.Rect(0, 0, 8, 6))
	draw.Draw(frame, frame.Bounds(), &image.Uniform{C: color.RGBA{R: 9, G: 9, B: 9, A: 255}}, image.Point{}, draw.Src)

	ov := newTestOverlay("menu", image.Pt(2, 2), image.Pt(1, 1), color.RGBA{R: 0, G: 255, B: 0, A: 255})

	compositeOverlays(window, frame, []*Overlay{ov})
	first := window.overlayScratch
	if first == nil {
		t.Fatal("no scratch buffer allocated")
	}
	if window.frameBuffer != first {
		t.Fatal("frameBuffer should point at the scratch after compositing")
	}
	// The live context image must not be bled into by the overlay blit.
	if got := frame.RGBAAt(1, 1); got.R != 9 {
		t.Fatalf("source frame was mutated by overlay: %v", got)
	}

	// Second pass (fresh context image, same size) must reuse the buffer.
	frame2 := image.NewRGBA(image.Rect(0, 0, 8, 6))
	draw.Draw(frame2, frame2.Bounds(), &image.Uniform{C: color.RGBA{R: 1, G: 2, B: 3, A: 255}}, image.Point{}, draw.Src)
	compositeOverlays(window, frame2, []*Overlay{ov})
	if window.overlayScratch != first {
		t.Fatal("scratch reallocated for a same-size frame; expected reuse")
	}

	// A resize must reallocate.
	frame3 := image.NewRGBA(image.Rect(0, 0, 10, 6))
	compositeOverlays(window, frame3, []*Overlay{ov})
	if window.overlayScratch == first {
		t.Fatal("scratch not reallocated after resize")
	}
}

// TestCompositeOverlaysPixelParity checks the reused-scratch path produces
// exactly the same output as the old allocate-a-fresh-copy-every-pass
// behavior, including the two-pass static+dynamic overlay case.
func TestCompositeOverlaysPixelParity(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 6, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 6; x++ {
			base.SetRGBA(x, y, color.RGBA{R: uint8(x * 10), G: uint8(y * 20), B: 5, A: 255})
		}
	}
	staticOv := newTestOverlay("s", image.Pt(2, 2), image.Pt(0, 0), color.RGBA{R: 200, G: 0, B: 0, A: 255})
	dynOv := newTestOverlay("d", image.Pt(1, 3), image.Pt(3, 1), color.RGBA{B: 200, A: 255})

	// Reference: old behavior (fresh copy per pass).
	refBuf := image.NewRGBA(base.Bounds())
	draw.Draw(refBuf, refBuf.Bounds(), base, image.Point{}, draw.Src)
	for _, ov := range []*Overlay{staticOv, dynOv} {
		next := image.NewRGBA(refBuf.Bounds())
		draw.Draw(next, next.Bounds(), refBuf, image.Point{}, draw.Over)
		draw.Draw(next, ov.buffer.Bounds().Add(ov.position), ov.buffer, image.Point{}, draw.Over)
		refBuf = next
	}

	// New behavior: same two-pass call sequence render() uses.
	window := &WindowBase{}
	window.frameBuffer = base
	compositeOverlays(window, window.frameBuffer, []*Overlay{staticOv})
	compositeOverlays(window, window.frameBuffer, []*Overlay{dynOv})
	fb := window.frameBuffer
	if fb == nil {
		t.Fatal("expected composited frameBuffer")
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 6; x++ {
			if got, want := fb.RGBAAt(x, y), refBuf.RGBAAt(x, y); got != want {
				t.Fatalf("pixel (%d,%d): got %v want %v", x, y, got, want)
			}
		}
	}
}
