package mustard

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// ebitenDriver implements Driver using Ebitengine (pure Go, no CGo).
type ebitenDriver struct{}

var _ Driver = (*ebitenDriver)(nil)

// NewDriver returns a new ebitenDriver (pure Go Ebitengine backend).
func NewDriver() Driver {
	return &ebitenDriver{}
}

func (d *ebitenDriver) NewWindow(title string, width, height int, hiDPI bool) (Window, error) {
	w := &WindowBase{
		title:   title,
		width:   width,
		height:  height,
		visible: true,
	}
	return w, nil
}

func (d *ebitenDriver) NewRenderer(w Window) (Renderer, error) {
	return &ebitenRenderer{window: w}, nil
}

func (d *ebitenDriver) NewApp(name string) AppDriver {
	return &ebitenApp{name: name}
}

// ebitenRenderer implements Renderer using Ebitengine. The window-level paint
// path (see WindowBase.render) performs the actual widget compositing; this
// renderer exposes the platform-agnostic primitive interface for callers that
// draw directly onto the Ebitengine surface.
type ebitenRenderer struct {
	window    Window
	screen    *ebiten.Image
	current   color.Color
	fillCache map[fillKey]*ebiten.Image
}

// fillKey identifies a cached solid-color fill source by color and size.
type fillKey struct {
	r, g, b, a uint8
	w, h       int
}

func (r *ebitenRenderer) Clear() {
	if r.screen != nil {
		r.screen.Fill(color.Transparent)
	}
}

func (r *ebitenRenderer) DrawImage(img image.Image, x, y float64) {
	if r.screen == nil || img == nil {
		return
	}
	opts := drawImageOptionsAt(x, y)
	r.screen.DrawImage(ebiten.NewImageFromImage(img), &opts)
}

func (r *ebitenRenderer) SetColor(red, green, blue, alpha float32) {
	r.current = color.NRGBA{
		R: uint8(clampf01(red) * 255),
		G: uint8(clampf01(green) * 255),
		B: uint8(clampf01(blue) * 255),
		A: uint8(clampf01(alpha) * 255),
	}
}

func (r *ebitenRenderer) DrawRectangle(x, y, width, height float64) {
	if r.screen == nil {
		return
	}
	w, h := int(width), int(height)
	if w <= 0 || h <= 0 {
		return
	}
	fill := r.current.(color.NRGBA)
	key := fillKey{fill.R, fill.G, fill.B, fill.A, w, h}
	if r.fillCache == nil {
		r.fillCache = make(map[fillKey]*ebiten.Image)
	}
	src, ok := r.fillCache[key]
	if !ok {
		src = ebiten.NewImage(w, h)
		for dy := 0; dy < h; dy++ {
			for dx := 0; dx < w; dx++ {
				src.Set(dx, dy, fill)
			}
		}
		r.fillCache[key] = src
	}
	opts := drawImageOptionsAt(x, y)
	r.screen.DrawImage(src, &opts)
}

func (r *ebitenRenderer) Present() {}

func (r *ebitenRenderer) Size() (int, int) {
	if r.screen != nil {
		b := r.screen.Bounds()
		return b.Dx(), b.Dy()
	}
	return r.window.Size()
}

// drawImageOptionsAt returns draw options translating the source to (x, y).
func drawImageOptionsAt(x, y float64) ebiten.DrawImageOptions {
	var opts ebiten.DrawImageOptions
	opts.GeoM.Translate(x, y)
	return opts
}

// clampf01 clamps a float to the [0,1] range for color components.
func clampf01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// ebitenApp implements AppDriver using Ebitengine. It renders the active,
// visible window each frame by blitting its composited frame buffer to the
// Ebitengine screen.
type ebitenApp struct {
	name     string
	windows  []Window
	callback func()
}

func (a *ebitenApp) NewWindow(title string, width, height int, hiDPI bool) (Window, error) {
	w := &WindowBase{
		title:   title,
		width:   width,
		height:  height,
		visible: true,
	}
	return w, nil
}

func (a *ebitenApp) Run(callback func()) {
	a.callback = callback
	if len(a.windows) == 0 {
		w, _ := a.NewWindow(a.name, 640, 480, false)
		a.AddWindow(w)
	}
	ws, hs := a.windows[0].Size()
	ebiten.SetWindowSize(ws, hs)
	ebiten.SetWindowTitle(a.name)
	_ = ebiten.RunGame(&ebitenGame{app: a})
}

func (a *ebitenApp) AddWindow(w Window) {
	a.windows = append(a.windows, w)
}

func (a *ebitenApp) DestroyWindow(w Window) {
	for i, candidate := range a.windows {
		if candidate == w {
			a.windows = append(a.windows[:i], a.windows[i+1:]...)
			break
		}
	}
}

// activeVisibleWindow returns the window to render: the first visible window
// that is explicitly active (tabbed mode) or, failing that, the first visible
// window (single-window mode).
func (a *ebitenApp) activeVisibleWindow() *WindowBase {
	for _, w := range a.windows {
		if wb, ok := w.(*WindowBase); ok && wb.visible && !wb.IsClosed() && wb.IsActive() {
			return wb
		}
	}
	for _, w := range a.windows {
		if wb, ok := w.(*WindowBase); ok && wb.visible && !wb.IsClosed() {
			return wb
		}
	}
	return nil
}

// ebitenGame is the Ebitengine game loop that drives thdwb rendering.
type ebitenGame struct {
	app   *ebitenApp
	frame *image.RGBA
}

func (g *ebitenGame) Update() error {
	if g.app.callback != nil {
		g.app.callback()
	}
	active := g.app.activeVisibleWindow()
	if active == nil {
		g.frame = nil
		return nil
	}
	g.frame = active.render()
	return nil
}

func (g *ebitenGame) Draw(screen *ebiten.Image) {
	if g.frame == nil {
		return
	}
	screen.DrawImage(ebiten.NewImageFromImage(g.frame), nil)
}

func (g *ebitenGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	if len(g.app.windows) > 0 {
		return g.app.windows[0].Size()
	}
	return 640, 480
}
