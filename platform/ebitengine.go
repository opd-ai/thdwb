//go:build !cgo

package platform

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ebitenApp implements App using Ebitengine.
type ebitenApp struct {
	name     string
	windows  []*ebitenWindow
	callback func()
}

// ebitenWindow implements Window using Ebitengine.
type ebitenWindow struct {
	title   string
	width   int
	height  int
	visible bool
	closed  bool
	cursor  Cursor

	// Mobile-specific state
	orientation  Orientation
	lifecycle    LifecycleState
	touchPoints  map[ebiten.TouchID]TouchPoint

	needsReflow bool
}

// Orientation represents the device orientation.
type Orientation int

const (
	OrientationPortrait Orientation = iota
	OrientationLandscapeLeft
	OrientationLandscapeRight
	OrientationPortraitUpsideDown
)

// LifecycleState represents the application lifecycle state.
type LifecycleState int

const (
	LifecycleForeground LifecycleState = iota
	LifecycleBackground
	LifecyclePaused
)

// TouchPoint represents a single touch point.
type TouchPoint struct {
	ID      ebiten.TouchID
	X, Y    float64
	Pressed bool
}

// ebitenRenderer implements Renderer using Ebitengine.
type ebitenRenderer struct {
	window *ebitenWindow
	screen *ebiten.Image
}

// ebitenInput implements Input using Ebitengine.
type ebitenInput struct {
	window *ebitenWindow
}

var _ App = (*ebitenApp)(nil)
var _ Window = (*ebitenWindow)(nil)
var _ Renderer = (*ebitenRenderer)(nil)
var _ Input = (*ebitenInput)(nil)

func newApp(name string) App {
	return &ebitenApp{name: name}
}

func newWindow(title string, width, height int, hiDPI bool) (Window, error) {
	return &ebitenWindow{
		title:   title,
		width:   width,
		height:  height,
		visible: true,
	}, nil
}

func newRenderer(w Window) (Renderer, error) {
	ew, ok := w.(*ebitenWindow)
	if !ok {
		return nil, &errInvalidWindow{}
	}
	return &ebitenRenderer{window: ew}, nil
}

func (a *ebitenApp) Run(callback func()) {
	a.callback = callback
	if len(a.windows) == 0 {
		w, _ := newWindow(a.name, 640, 480, false)
		a.AddWindow(w)
	}
	for _, w := range a.windows {
		w.visible = true
	}
	ebiten.SetWindowSize(a.windows[0].width, a.windows[0].height)
	ebiten.SetWindowTitle(a.name)
	ebiten.RunGame(&ebitenGame{app: a})
}

func (a *ebitenApp) AddWindow(w Window) {
	ew, ok := w.(*ebitenWindow)
	if !ok {
		return
	}
	a.windows = append(a.windows, ew)
}

func (a *ebitenApp) DestroyWindow(w Window) {
	ew, ok := w.(*ebitenWindow)
	if !ok {
		return
	}
	ew.closed = true
	for i, candidate := range a.windows {
		if candidate == ew {
			a.windows = append(a.windows[:i], a.windows[i+1:]...)
			break
		}
	}
}

func (w *ebitenWindow) Show() {
	w.visible = true
}

func (w *ebitenWindow) Hide() {
	w.visible = false
}

func (w *ebitenWindow) Close() {
	w.closed = true
}

func (w *ebitenWindow) IsClosed() bool {
	return w.closed
}

func (w *ebitenWindow) SetTitle(title string) {
	w.title = title
	ebiten.SetWindowTitle(title)
}

func (w *ebitenWindow) Size() (int, int) {
	return w.width, w.height
}

func (w *ebitenWindow) SetSize(width, height int) {
	w.width = width
	w.height = height
	ebiten.SetWindowSize(width, height)
}

func (w *ebitenWindow) PollEvents() {
	// Ebitengine handles events internally via the game loop.
}

func (w *ebitenWindow) ShouldClose() bool {
	return w.closed || inpututil.IsWindowBeingClosed()
}

func (w *ebitenWindow) GetCursorPosition() (float64, float64) {
	x, y := ebiten.CursorPosition()
	return float64(x), float64(y)
}

func (w *ebitenWindow) SetCursor(cursor Cursor) {
	w.cursor = cursor
}

func (r *ebitenRenderer) Clear() {
	if r.screen != nil {
		r.screen.Fill(color.RGBA{0, 0, 0, 0})
	}
}

func (r *ebitenRenderer) DrawImage(img Image, x, y float64) {
	_ = img
	_ = x
	_ = y
}

func (r *ebitenRenderer) SetColor(r, g, b, a float32) {
	_ = r
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

func (r *ebitenRenderer) Present() {
	// Ebitengine presents automatically at end of frame.
}

func (r *ebitenRenderer) Size() (int, int) {
	if r.screen != nil {
		b := r.screen.Bounds()
		return b.Dx(), b.Dy()
	}
	return r.window.Size()
}

func (r *ebitenRenderer) GPUInfo() GPUInfo {
	return GPUInfo{
		Backend: GPUBackendUnknown, // Ebitengine abstracts the backend
	}
}

func (i *ebitenInput) CursorPosition() (float64, float64) {
	x, y := ebiten.CursorPosition()
	return float64(x), float64(y)
}

func (i *ebitenInput) IsKeyPressed(key Key) bool {
	return ebiten.IsKeyPressed(ebitenKey(key))
}

func (i *ebitenInput) IsMouseButtonPressed(button MouseButton) bool {
	return ebiten.IsMouseButtonPressed(ebitenMouseButton(button))
}

func (i *ebitenInput) TouchPoints() []TouchPoint {
	ids := ebiten.TouchIDs()
	points := make([]TouchPoint, 0, len(ids))
	for _, id := range ids {
		x, y := ebiten.TouchPosition(id)
		points = append(points, TouchPoint{
			ID:      id,
			X:       float64(x),
			Y:       float64(y),
			Pressed: true,
		})
	}
	return points
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
		if !w.visible || w.closed {
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

func ebitenKey(k Key) ebiten.Key {
	switch k {
	case KeyBackspace:
		return ebiten.KeyBackspace
	case KeyDelete:
		return ebiten.KeyDelete
	case KeyEnter:
		return ebiten.KeyEnter
	case KeyEscape:
		return ebiten.KeyEscape
	case KeyTab:
		return ebiten.KeyTab
	case KeyArrowLeft:
		return ebiten.KeyArrowLeft
	case KeyArrowRight:
		return ebiten.KeyArrowRight
	case KeyArrowUp:
		return ebiten.KeyArrowUp
	case KeyArrowDown:
		return ebiten.KeyArrowDown
	case KeyControl:
		return ebiten.KeyControl
	case KeyShift:
		return ebiten.KeyShift
	case KeyAlt:
		return ebiten.KeyAlt
	case KeySuper:
		return ebiten.KeySuper
	case KeySpace:
		return ebiten.KeySpace
	case KeyA:
		return ebiten.KeyA
	case KeyB:
		return ebiten.KeyB
	case KeyC:
		return ebiten.KeyC
	case KeyD:
		return ebiten.KeyD
	case KeyE:
		return ebiten.KeyE
	case KeyF:
		return ebiten.KeyF
	case KeyG:
		return ebiten.KeyG
	case KeyH:
		return ebiten.KeyH
	case KeyI:
		return ebiten.KeyI
	case KeyJ:
		return ebiten.KeyJ
	case KeyK:
		return ebiten.KeyK
	case KeyL:
		return ebiten.KeyL
	case KeyM:
		return ebiten.KeyM
	case KeyN:
		return ebiten.KeyN
	case KeyO:
		return ebiten.KeyO
	case KeyP:
		return ebiten.KeyP
	case KeyQ:
		return ebiten.KeyQ
	case KeyR:
		return ebiten.KeyR
	case KeyS:
		return ebiten.KeyS
	case KeyT:
		return ebiten.KeyT
	case KeyU:
		return ebiten.KeyU
	case KeyV:
		return ebiten.KeyV
	case KeyW:
		return ebiten.KeyW
	case KeyX:
		return ebiten.KeyX
	case KeyY:
		return ebiten.KeyY
	case KeyZ:
		return ebiten.KeyZ
	case Key0:
		return ebiten.Key0
	case Key1:
		return ebiten.Key1
	case Key2:
		return ebiten.Key2
	case Key3:
		return ebiten.Key3
	case Key4:
		return ebiten.Key4
	case Key5:
		return ebiten.Key5
	case Key6:
		return ebiten.Key6
	case Key7:
		return ebiten.Key7
	case Key8:
		return ebiten.Key8
	case Key9:
		return ebiten.Key9
	default:
		return ebiten.KeyUnknown
	}
}

func ebitenMouseButton(b MouseButton) ebiten.MouseButton {
	switch b {
	case MouseButtonLeft:
		return ebiten.MouseButtonLeft
	case MouseButtonRight:
		return ebiten.MouseButtonRight
	default:
		return ebiten.MouseButtonLeft
	}
}

type errInvalidWindow struct{}

func (e *errInvalidWindow) Error() string {
	return "platform: invalid window type for ebitengine renderer"
}
