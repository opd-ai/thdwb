package gg

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/draw"
)

// TestMaskedDrawStringPixelParity (audit R-14) proves the bounded scratch
// layer introduced for masked text draws is pixel-identical to the old
// context-sized scratch. The reference path below replicates the pre-fix code
// (full-size RGBA + DrawMask over the whole context); both must produce the
// same destination pixels.
func TestMaskedDrawStringPixelParity(t *testing.T) {
	const W, H = 400, 300

	mask := image.NewAlpha(image.Rect(0, 0, W, H))
	// A diagonal band clip so parts of the text fall outside the mask.
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			if (x+y)%3 != 0 {
				mask.SetAlpha(x, y, color.Alpha{A: 255})
			}
		}
	}

	ctxA := NewContext(W, H) // current bounded-scratch path
	ctxB := NewContext(W, H) // reference: manual full-size path

	require.NoError(t, ctxA.SetMask(cloneAlpha(mask)))
	require.NoError(t, ctxB.SetMask(cloneAlpha(mask)))

	texts := []struct {
		s       string
		x, y    float64
		wrapped float64
	}{
		{"Hello world, this line is clipped mid-way through the band", 10, 30, 0},
		{"short", 350, 100, 0},
		{"near edge", -5, 250, 0},
		{"wide text that overflows the context boundary entirely to the right side", 100, 200, 0},
	}
	for _, tc := range texts {
		ctxA.SetHexColor("#334455")
		ctxB.SetHexColor("#334455")
		ctxA.DrawString(tc.s, tc.x, tc.y)
		// Reference: replicate the OLD masked behavior exactly -- draw into a
		// full-context scratch, then composite the whole context.
		im := image.NewRGBA(image.Rect(0, 0, W, H))
		refB := NewContext(W, H)
		refB.SetHexColor("#334455")
		refB.drawString(im, tc.s, tc.x, tc.y)
		draw.DrawMask(ctxB.im, ctxB.im.Bounds(), im, image.Point{}, ctxB.mask, image.Point{}, draw.Over)
	}

	assert.True(t, imageEqual(ctxA.im, ctxB.im),
		"bounded scratch must be pixel-identical to the full-size reference")
}

func cloneAlpha(a *image.Alpha) *image.Alpha {
	out := image.NewAlpha(a.Rect)
	copy(out.Pix, a.Pix)
	out.Stride = a.Stride
	return out
}

// TestScratchFallsBackForTransformedContext: a non-identity matrix must take
// the safe full-size path (no panic, still composites).
func TestScratchFallsBackForTransformedContext(t *testing.T) {
	ctx := NewContext(200, 200)
	ctx.SetHexColor("#000000")
	mask := image.NewAlpha(image.Rect(0, 0, 200, 200))
	for i := range mask.Pix {
		mask.Pix[i] = 255
	}
	require.NoError(t, ctx.SetMask(mask))

	ctx.Rotate(0.5)
	assert.NotPanics(t, func() { ctx.DrawString("rotated text", 50, 50) })
}

// TestScratchSmallerThanContext documents the win: the scratch allocation for
// a short masked string is bounded by the glyph footprint, not the context.
func TestScratchSmallerThanContext(t *testing.T) {
	ctx := NewContext(1024, 768)
	mask := image.NewAlpha(image.Rect(0, 0, 1024, 768))
	for i := range mask.Pix {
		mask.Pix[i] = 255
	}
	require.NoError(t, ctx.SetMask(mask))

	r, ok := ctx.glyphScratch("hello", 10, 20, 30, 13)
	require.True(t, ok, "short text on a big context must take the bounded path")
	assert.Less(t, (r.Bounds().Dx() * r.Bounds().Dy()), 1024*768/4)
}

func imageEqual(a, b *image.RGBA) bool {
	if a.Rect != b.Rect {
		return false
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return false
		}
	}
	return true
}
