package mustard

import (
	"testing"
)

// TestEbitenDriverConstructors exercises the pure-Go driver's factory methods
// without touching the Ebitengine runtime.
func TestEbitenDriverConstructors(t *testing.T) {
	d := NewDriver()
	if d == nil {
		t.Fatal("NewDriver returned nil")
	}

	w, err := d.NewWindow("t", 640, 480, false)
	if err != nil || w == nil {
		t.Fatalf("NewWindow = %v, %v", w, err)
	}
	wb, ok := w.(*WindowBase)
	if !ok || wb.width != 640 || wb.height != 480 || !wb.visible {
		t.Fatalf("NewWindow window = %+v, want 640x480 visible", wb)
	}

	renderer, err := d.NewRenderer(w)
	if err != nil || renderer == nil {
		t.Fatalf("NewRenderer = %v, %v", renderer, err)
	}

	app := d.NewApp("app")
	if app == nil {
		t.Fatal("NewApp returned nil")
	}
}

// TestEbitenRendererNilScreenGuards ensures the renderer is a safe no-op before
// a screen is attached.
func TestEbitenRendererNilScreenGuards(t *testing.T) {
	window := &WindowBase{width: 640, height: 480, visible: true}
	r := &ebitenRenderer{window: window}

	r.Clear()
	r.DrawImage(nil, 0, 0)
	r.SetColor(1.5, -1, 2, 0.5)
	if r.current == nil {
		t.Fatal("SetColor did not store a color")
	}
	r.DrawRectangle(0, 0, 10, 10) // nil screen -> early return
	if w, h := r.Size(); w != 640 || h != 480 {
		t.Fatalf("Size = %dx%d, want the window size 640x480", w, h)
	}
}

func TestClampf01(t *testing.T) {
	if clampf01(-1) != 0 || clampf01(2) != 1 || clampf01(0.25) != 0.25 {
		t.Fatal("clampf01 did not clamp to the [0,1] range")
	}
}

func TestDrawImageOptionsAt(t *testing.T) {
	opts := drawImageOptionsAt(3, 4)
	x, y := opts.GeoM.Apply(0, 0)
	if x != 3 || y != 4 {
		t.Fatalf("options translation = (%v,%v), want (3,4)", x, y)
	}
}

func TestEbitenAppWindowManagement(t *testing.T) {
	a := &ebitenApp{name: "app"}
	if len(a.windows) != 0 {
		t.Fatal("new ebitenApp should have no windows")
	}

	w1, _ := a.NewWindow("one", 100, 100, false)
	w2, _ := a.NewWindow("two", 100, 100, false)
	a.AddWindow(w1)
	a.AddWindow(w2)
	if len(a.windows) != 2 {
		t.Fatalf("windows = %d, want 2", len(a.windows))
	}

	a.DestroyWindow(w1)
	if len(a.windows) != 1 || a.windows[0] != w2 {
		t.Fatalf("windows after destroy = %v, want [w2]", a.windows)
	}
}

func TestEbitenAppActiveVisibleWindowSelection(t *testing.T) {
	a := &ebitenApp{}
	// No windows -> nil.
	if a.activeVisibleWindow() != nil {
		t.Fatal("expected nil with no windows")
	}

	invisible := &WindowBase{width: 10, height: 10, visible: false}
	hidden := &WindowBase{width: 10, height: 10, visible: true}
	a.windows = []Window{invisible, hidden}
	if got := a.activeVisibleWindow(); got != hidden {
		t.Fatalf("activeVisibleWindow = %v, want the visible window", got)
	}
}

func TestEbitenAppActiveVisibleWindowPrefersActive(t *testing.T) {
	a := &ebitenApp{}
	first := &WindowBase{width: 10, height: 10, visible: true}
	second := &WindowBase{width: 10, height: 10, visible: true}
	// Mark second as explicitly active (tabbed mode) via setActive.
	first.setActive(false)
	second.setActive(true)
	a.windows = []Window{first, second}
	if got := a.activeVisibleWindow(); got != second {
		t.Fatalf("activeVisibleWindow = %v, want the explicitly-active window", got)
	}
}

func TestEbitenGameUpdateAndLayout(t *testing.T) {
	a := &ebitenApp{}
	w, _ := a.NewWindow("w", 80, 60, false)
	a.AddWindow(w)

	g := &ebitenGame{app: a}
	if err := g.Update(); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if g.frame == nil {
		t.Fatal("Update did not produce a frame for the visible window")
	}

	if w, h := g.Layout(0, 0); w != 80 || h != 60 {
		t.Fatalf("Layout = %dx%d, want 80x60", w, h)
	}

	// No windows -> default layout.
	empty := &ebitenGame{app: &ebitenApp{}}
	if w, h := empty.Layout(0, 0); w != 640 || h != 480 {
		t.Fatalf("empty Layout = %dx%d, want 640x480", w, h)
	}
}

func TestEbitenGameUpdateNilWindowAndDrawNilFrame(t *testing.T) {
	g := &ebitenGame{app: &ebitenApp{}}
	// No windows -> Update sets frame to nil and returns nil error.
	if err := g.Update(); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if g.frame != nil {
		t.Fatal("expected a nil frame when there is no visible window")
	}
	// Draw with a nil frame must be a safe no-op (nil screen is never dereferenced).
	g.Draw(nil)
}
