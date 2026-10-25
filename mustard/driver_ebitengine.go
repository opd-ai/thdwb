package mustard

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// ebitenDriver implements Driver using Ebitengine.
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

// ebitenRenderer implements Renderer using Ebitengine.
type ebitenRenderer struct {
	window Window
	screen *ebiten.Image
}

func (r *ebitenRenderer) Clear() {
	if r.screen != nil {
		r.screen.Fill(color.RGBA{0, 0, 0, 0})
	}
}

func (r *ebitenRenderer) DrawImage(img image.Image, x, y float64) {
	_ = img
	_ = x
	_ = y
}

func (r *ebitenRenderer) SetColor(rf, g, b, a float32) {
	_ = rf
	_ = g
	_ = b
	_ = a
}

func (r *ebitenRenderer) DrawRectangle(x, y, width, height float64) {
	_ = x
	_ = y
	_ = width
	_ = height
}
func (r *ebitenRenderer) Present() {}
func (r *ebitenRenderer) Size() (int, int) {
	if r.screen != nil {
		b := r.screen.Bounds()
		return b.Dx(), b.Dy()
	}
	return r.window.Size()
}

// ebitenApp implements App using Ebitengine.
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
	w := a.windows[0]
	ws, hs := w.Size()
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

type ebitenGame struct {
	app *ebitenApp
}

func (g *ebitenGame) Update() error {
	if g.app.callback != nil {
		g.app.callback()
	}
	return nil
}

func (g *ebitenGame) Draw(screen *ebiten.Image) {
	for _, w := range g.app.windows {
		if w.IsClosed() {
			continue
		}
		_ = w
	}
}

func (g *ebitenGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	if len(g.app.windows) > 0 {
		return g.app.windows[0].Size()
	}
	return 640, 480
}
