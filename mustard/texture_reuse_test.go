package mustard

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestDrawReusesGPUTexture locks in audit R-16: ebitenGame.Draw must upload
// into one persistent texture across frames rather than allocating a new
// ebiten.Image (a fresh GPU texture) every frame.
func TestDrawReusesGPUTexture(t *testing.T) {
	g := &ebitenGame{frame: image.NewRGBA(image.Rect(0, 0, 8, 6))}
	screen := ebiten.NewImage(8, 6)

	g.Draw(screen)
	first := g.gpuFrame
	if first == nil {
		t.Fatal("Draw did not create a GPU texture")
	}

	// A second frame of the same size must reuse the very same texture object.
	g.frame = image.NewRGBA(image.Rect(0, 0, 8, 6))
	g.Draw(screen)
	if g.gpuFrame != first {
		t.Fatal("gpuFrame was re-allocated for a same-size frame; expected reuse")
	}

	// A resize must allocate a new texture sized to the new frame.
	g.frame = image.NewRGBA(image.Rect(0, 0, 12, 6))
	g.Draw(screen)
	if g.gpuFrame == first {
		t.Fatal("gpuFrame not recreated after a resize")
	}
	if b := g.gpuFrame.Bounds(); b.Dx() != 12 || b.Dy() != 6 {
		t.Fatalf("recreated texture has bounds %v, want 12x6", b)
	}
}

// TestDrawFallbackOnOddStride verifies a non-tightly-packed frame (stride !=
// width*4) takes the one-shot NewImageFromImage path instead of WritePixels,
// which would panic on a wrong-length pixel slice.
func TestDrawFallbackOnOddStride(t *testing.T) {
	f := &image.RGBA{
		Pix:    make([]byte, 4*2*4), // 2 rows of 4 px, but...
		Stride: 20,                  // ...stride (20) != width*4 (16)
		Rect:   image.Rect(0, 0, 4, 2),
	}
	g := &ebitenGame{frame: f}
	screen := ebiten.NewImage(4, 2)

	// Must not panic and must not build the persistent texture.
	g.Draw(screen)
	if g.gpuFrame != nil {
		t.Fatal("odd-stride frame should use the fallback path, not the persistent texture")
	}

	// A tightly-packed frame of the same size then uses the persistent path.
	g.frame = image.NewRGBA(image.Rect(0, 0, 4, 2))
	g.Draw(screen)
	if g.gpuFrame == nil {
		t.Fatal("tightly-packed frame should populate the persistent texture")
	}
}
