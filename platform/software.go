package platform

import (
	"image"
	"image/color"
	"image/draw"
)

// softwareRenderer implements Renderer using pure Go CPU-based rendering.
type softwareRenderer struct {
	im *image.RGBA
}

var _ Renderer = (*softwareRenderer)(nil)

func newSoftwareRenderer(w Window) (Renderer, error) {
	width, height := w.Size()
	return &softwareRenderer{
		im: image.NewRGBA(image.Rect(0, 0, width, height)),
	}, nil
}

func (r *softwareRenderer) Clear() {
	draw.Draw(r.im, r.im.Bounds(), image.NewUniform(color.Transparent), image.Point{}, draw.Src)
}

func (r *softwareRenderer) DrawImage(img Image, x, y float64) {
	_ = img
	_ = x
	_ = y
}

func (r *softwareRenderer) SetColor(red, green, blue, alpha float32) {
	_ = red
	_ = green
	_ = blue
	_ = alpha
}

func (r *softwareRenderer) DrawRectangle(x, y, width, height float64) {
	_ = x
	_ = y
	_ = width
	_ = height
}

func (r *softwareRenderer) Present() {
	// Software renderer presents by making the image available to the caller.
}

func (r *softwareRenderer) Size() (int, int) {
	return r.im.Bounds().Dx(), r.im.Bounds().Dy()
}
