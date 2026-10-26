package mustard

import (
	"bytes"
	"image"
	"image/png"
	"testing"
	"time"

	assets "github.com/danfragoso/thdwb/assets"
	gg "github.com/danfragoso/thdwb/gg"
)

func newFontFace(t *testing.T) interface{} {
	t.Helper()
	face, err := loadFontFace(assets.OpenSans(400), 20)
	if err != nil {
		t.Fatalf("loadFontFace: %v", err)
	}
	return face
}

func makeTestPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// ---- App ----

func TestAppRunWithNilDriverLoops(t *testing.T) {
	// CreateNewApp substitutes a default driver for nil, so build the struct
	// directly to exercise the driver-less fallback loop.
	app := &App{name: "loop"}
	app.AddWindow(&WindowBase{visible: true})
	done := make(chan struct{})
	go func() {
		defer close(done)
		app.Run(func() {})
	}()
	select {
	case <-done:
		t.Fatal("Run returned for a nil-driver app; expected the event loop to keep running")
	case <-time.After(60 * time.Millisecond):
	}
}

func TestAppRunWithDriverDelegatesToAppDriver(t *testing.T) {
	driven := false
	app := CreateNewApp(&captureAppDriver{run: func() { driven = true }}, "d")
	app.Run(func() {})
	if !driven {
		t.Fatal("expected the app driver's Run to be called with a driver present")
	}
}

func TestAppDestroyWindowDropsActiveAndReactivates(t *testing.T) {
	app := CreateNewApp(nil, "x")
	first := &WindowBase{visible: true}
	second := &WindowBase{visible: true}
	app.AddWindow(first)
	app.AddWindow(second)
	if err := app.ActivateWindow(first); err != nil {
		t.Fatal(err)
	}
	app.DestroyWindow(first)
	if len(app.Windows()) != 1 || app.Windows()[0] != second {
		t.Fatalf("windows after destroy = %v, want [second]", app.Windows())
	}
	if app.activeWindow != second {
		t.Fatal("expected the remaining window to become active")
	}
}

func TestAppDestroyWindowRemovesNonActive(t *testing.T) {
	app := CreateNewApp(nil, "x")
	w := &WindowBase{visible: true}
	app.AddWindow(w)
	app.DestroyWindow(w)
	if len(app.Windows()) != 0 {
		t.Fatalf("windows after destroy = %d, want 0", len(app.Windows()))
	}
	if app.activeWindow != nil {
		t.Fatal("activeWindow should be nil after destroying the only window")
	}
}

func TestAppRootFrame(t *testing.T) {
	app := CreateNewApp(nil, "x")
	window := &WindowBase{visible: true}
	app.AddWindow(window)
	if app.rootFrame(window) != nil {
		t.Fatal("expected nil root frame for an unframed window")
	}
	window.rootFrame = CreateFrame(VerticalFrame)
	if app.rootFrame(window) != window.rootFrame {
		t.Fatal("expected the window's root frame to be returned")
	}
}

func TestAppRedrawWidgetsOnlyRedrawsDirty(t *testing.T) {
	window := newTestWindow(t, 50, 50)
	parent := CreateFrame(VerticalFrame)
	setWidgetWindow(parent, window)
	parent.SetVisible(true)
	parent.computedBox.SetCoords(0, 0, 50, 50)
	parent.needsRepaint = false

	child := CreateFrame(VerticalFrame)
	child.SetVisible(true)
	child.computedBox.SetCoords(0, 0, 50, 50)
	child.needsRepaint = true
	parent.AttachWidget(child)
	setWidgetWindow(parent, window)

	// redrawWidgets alone only triggers draws; the render pass (redraw +
	// composite) is what clears the repaint flag on the dirty child.
	window.rootFrame = parent
	window.needsReflow = false
	window.render()
	if child.NeedsRepaint() {
		t.Fatal("expected the dirty child to be drawn and cleared of its repaint flag")
	}
}

// ---- Window render pass ----

func TestRenderCreatesContextAndHandlesNilRootFrame(t *testing.T) {
	w := &WindowBase{width: 20, height: 20}
	if buf := w.render(); buf == nil {
		t.Fatal("render returned a nil frame buffer")
	}
	if w.context == nil {
		t.Fatal("render did not create a context")
	}

	w2 := &WindowBase{width: 30, height: 30, context: gg.NewContext(30, 30)}
	if buf := w2.render(); buf == nil {
		t.Fatal("render with an existing context returned a nil buffer")
	}
}

func TestRenderReflowsAndRedrawsBothPaths(t *testing.T) {
	window := newTestWindow(t, 60, 60)

	// needsReflow false -> redrawWidgets path (root frame is dirty).
	window.render()

	// needsReflow true -> drawRootFrame path.
	window.needsReflow = true
	window.render()
	if window.needsReflow {
		t.Fatal("render did not clear the reflow flag")
	}
}

func TestRenderCompositesOverlays(t *testing.T) {
	window := newTestWindow(t, 60, 60)
	static := &Overlay{buffer: image.NewRGBA(image.Rect(0, 0, 10, 10)), position: image.Point{2, 2}}
	dynamic := &Overlay{buffer: image.NewRGBA(image.Rect(0, 0, 10, 10)), position: image.Point{5, 5}}
	window.AddStaticOverlay(static)
	window.AddOverlay(dynamic)
	// An overlay with a nil buffer must be skipped, not crash.
	window.AddOverlay(&Overlay{position: image.Point{1, 1}})

	if buf := window.render(); buf == nil {
		t.Fatal("render with overlays returned a nil buffer")
	}
}

// ---- Widget draw methods (headless) ----

func TestButtonDrawSelectedAndUnselected(t *testing.T) {
	window := newTestWindow(t, 100, 100)
	button := &ButtonWidget{
		baseWidget: baseWidget{window: window, needsRepaint: true, font: newFontFace(t)},
		content:    "Hi",
		fontSize:   20,
		fontColor:  "#000",
		padding:    2,
	}
	button.computedBox.SetCoords(10, 10, 80, 40)
	button.draw()
	button.selected = true
	button.draw()
	if button.Buffer() == nil {
		t.Fatal("button draw did not produce a buffer")
	}
}

func TestTextAndLabelDraw(t *testing.T) {
	window := newTestWindow(t, 100, 100)
	text := &TextWidget{
		baseWidget:    baseWidget{window: window, needsRepaint: true, backgroundColor: "#fff", font: newFontFace(t)},
		content:       "Hello",
		fontSize:      20,
		fontColor:     "#000",
	}
	text.computedBox.SetCoords(0, 0, 100, 40)
	text.draw()

	label := &LabelWidget{
		baseWidget:    baseWidget{window: window, needsRepaint: true, backgroundColor: "#fff", font: newFontFace(t)},
		content:       "World",
		fontSize:      20,
		fontColor:     "#000",
	}
	label.computedBox.SetCoords(0, 0, 100, 40)
	label.draw()
}

func TestImageDraw(t *testing.T) {
	window := newTestWindow(t, 60, 60)
	img, _ := gg.LoadAsset(makeTestPNG(t))
	im := &ImageWidget{
		baseWidget: baseWidget{window: window, needsRepaint: true},
		img:        img,
	}
	im.computedBox.SetCoords(0, 0, 40, 40)
	im.draw()
	if im.Buffer() == nil {
		t.Fatal("image draw did not produce a buffer")
	}
}

func TestCreateImageWidgetConstructor(t *testing.T) {
	w := CreateImageWidget(makeTestPNG(t))
	if w == nil || w.widgetType != imageWidget {
		t.Fatal("CreateImageWidget returned an unexpected widget")
	}
}

func TestInputDrawInactiveAndActiveOverflow(t *testing.T) {
	window := newTestWindow(t, 120, 60)
	input := &InputWidget{
		baseWidget: baseWidget{window: window, needsRepaint: true, font: newFontFace(t)},
		value:      "",
		fontSize:   20,
		fontColor:  "#000",
	}
	input.computedBox.SetCoords(0, 0, 100, 40)
	input.draw()

	// Active with a value wider than the box exercises the anchored draw path.
	input.active = true
	input.value = "a pretty long value that should overflow the box width"
	input.draw()
	if input.Buffer() == nil {
		t.Fatal("input draw did not produce a buffer")
	}
}

func TestScrollBarDrawWithAndWithoutThumb(t *testing.T) {
	window := newTestWindow(t, 40, 200)
	sb := &ScrollBarWidget{
		baseWidget:    baseWidget{window: window, needsRepaint: true, backgroundColor: "#fff"},
		thumbColor:    "#000",
		orientation:   VerticalScrollBar,
	}
	sb.computedBox.SetCoords(0, 0, 12, 100)

	// scrollerSize > height -> thumb drawn.
	sb.scrollerSize = 200
	sb.scrollerOffset = 10
	sb.draw()

	// scrollerSize <= height -> no thumb.
	sb2 := &ScrollBarWidget{
		baseWidget:   baseWidget{window: window, needsRepaint: true, backgroundColor: "#fff"},
		thumbColor:   "#000",
		orientation:  HorizontalScrollBar,
	}
	sb2.computedBox.SetCoords(0, 0, 100, 12)
	sb2.scrollerSize = 200
	sb2.scrollerOffset = 20
	sb2.draw()
}

// captureAppDriver is a minimal Driver for exercising App.Run's driver path.
type captureAppDriver struct {
	run func()
}

func (c *captureAppDriver) NewWindow(title string, width, height int, hiDPI bool) (Window, error) {
	return &WindowBase{title: title, width: width, height: height, visible: true}, nil
}

func (c *captureAppDriver) NewRenderer(w Window) (Renderer, error) { return nil, nil }

func (c *captureAppDriver) NewApp(name string) AppDriver {
	return &captureAppDriverInner{run: c.run, name: name}
}

type captureAppDriverInner struct {
	name string
	run  func()
}

func (c *captureAppDriverInner) Run(callback func()) {
	if c.run != nil {
		c.run()
	}
}

func (c *captureAppDriverInner) AddWindow(w Window) {}

func (c *captureAppDriverInner) DestroyWindow(w Window) {}
