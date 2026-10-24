//go:build cgo

package mustard

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"strings"

	gg "github.com/danfragoso/thdwb/gg"
	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

// cgoDriver implements Driver using GLFW and OpenGL.
type cgoDriver struct{}

var _ Driver = (*cgoDriver)(nil)

func (d *cgoDriver) NewWindow(title string, width, height int, hiDPI bool) (Window, error) {
	setGLFWHints()
	glfw.Init()
	glw, err := glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GLFW window: %w", err)
	}
	glw.SetSizeLimits(300, 200, glfw.DontCare, glfw.DontCare)

	xscale, yscale := float32(1), float32(1)
	if hiDPI {
		xscale, yscale = glw.GetContentScale()
	}

	w := &cgoWindow{
		glw:         glw,
		width:       int(float32(width) / xscale),
		height:      int(float32(height) / yscale),
		hiDPI:       hiDPI,
		visible:     true,
		active:      true,
		defaultCursor: glfw.CreateStandardCursor(glfw.ArrowCursor),
		pointerCursor: glfw.CreateStandardCursor(glfw.HandCursor),
	}
	w.RecreateContext()
	glw.MakeContextCurrent()
	w.backend = createGLBackend()
	w.addEvents()
	w.generateTexture()
	return w, nil
}

func (d *cgoDriver) NewRenderer(w Window) (Renderer, error) {
	cw, ok := w.(*cgoWindow)
	if !ok {
		return nil, &errInvalidWindow{}
	}
	return &cgoRenderer{window: cw}, nil
}

func (d *cgoDriver) NewApp(name string) AppDriver {
	return &cgoApp{name: name}
}

var _ AppDriver = (*cgoApp)(nil)

// cgoWindow implements Window using GLFW.
type cgoWindow struct {
	glw         *glfw.Window
	context     *gg.Context
	backend     *glBackend
	frameBuffer *image.RGBA

	title       string
	width       int
	height      int
	hiDPI       bool
	visible     bool
	active      bool
	activeSet   bool

	defaultCursor *glfw.Cursor
	pointerCursor *glfw.Cursor
	cursor        Cursor

	registeredTrees   []*TreeWidget
	registeredButtons []*ButtonWidget
	registeredInputs  []*InputWidget
	activeInput       *InputWidget
	rootFrame         *Frame

	cursorX float64
	cursorY float64

	pointerPositionEventListeners []func(float64, float64)
	pointerClickEventListeners    []func(float64, float64, MustardKey)
	keyboardInputEventListeners   []func(KeyboardInput)
	textInputEventListeners       []func(rune)
	scrollEventListeners          []func(int)
	clickEventListeners           []func(MustardKey)
	activeChangeListeners         []func(bool)

	overlays         []*Overlay
	hasActiveOverlay bool

	staticOverlays   []*Overlay
	hasStaticOverlay bool

	contextMenu *contextMenu

	needsReflow     bool
	selectedWidget  Widget
}

func (w *cgoWindow) Show() {
	w.visible = true
	w.glw.Show()
}

func (w *cgoWindow) Hide() {
	w.visible = false
	w.glw.Hide()
}

func (w *cgoWindow) Close() {
	w.glw.SetShouldClose(true)
}

func (w *cgoWindow) IsClosed() bool {
	return w.glw.ShouldClose()
}

func (w *cgoWindow) SetTitle(title string) {
	w.title = title
	w.glw.SetTitle(title)
}

func (w *cgoWindow) Size() (int, int) {
	return w.width, w.height
}

func (w *cgoWindow) SetSize(width, height int) {
	w.glw.SetSize(width, height)
}

func (w *cgoWindow) PollEvents() {
	glfw.PollEvents()
}

func (w *cgoWindow) ShouldClose() bool {
	return w.glw.ShouldClose()
}

func (w *cgoWindow) GetCursorPosition() (float64, float64) {
	return w.cursorX, w.cursorY
}

func (w *cgoWindow) SetCursor(cursor Cursor) {
	w.cursor = cursor
	if glwc, ok := cursor.(*glfw.Cursor); ok {
		w.glw.SetCursor(glwc)
	}
}

func (w *cgoWindow) resize(width, height int, xscale, yscale float32) {
	if width <= 0 || height <= 0 {
		return
	}
	if xscale <= 0 {
		xscale = 1
	}
	if yscale <= 0 {
		yscale = 1
	}
	swidth := int(float32(width) / xscale)
	sheight := int(float32(height) / yscale)
	if swidth <= 0 || sheight <= 0 || (w.width == swidth && w.height == sheight) {
		return
	}
	w.width, w.height = swidth, sheight
	w.RecreateContext()
	w.frameBuffer = nil
	w.needsReflow = true
}

func (w *cgoWindow) RecreateContext() {
	w.context = gg.NewContext(w.width, w.height)
}

func (w *cgoWindow) addEvents() {
	w.glw.SetSizeCallback(func(glw *glfw.Window, width, height int) {
		xscale, yscale := float32(1), float32(1)
		if w.hiDPI {
			xscale, yscale = glw.GetContentScale()
		}
		w.resize(width, height, xscale, yscale)
	})

	w.glw.SetCursorPosCallback(func(glw *glfw.Window, x, y float64) {
		xscale, yscale := float32(1), float32(1)
		if w.hiDPI {
			xscale, yscale = glw.GetContentScale()
		}

		w.cursorX, w.cursorY = x/float64(xscale), y/float64(yscale)
		w.ProcessPointerPosition()
	})

	w.glw.SetCharCallback(func(glw *glfw.Window, char rune) {
		w.ProcessTextInput(char)
		if w.activeInput != nil {
			inputVal, cursorPos := w.activeInput.value, w.activeInput.cursorPosition

			w.activeInput.value = inputVal[:len(inputVal)+cursorPos] + string(char) + inputVal[len(inputVal)+cursorPos:]
			w.activeInput.needsRepaint = true
		}
	})

	w.glw.SetKeyCallback(func(glw *glfw.Window, key glfw.Key, sc int, action glfw.Action, mods glfw.ModifierKey) {
		if input, ok := keyboardInputFromGLFW(key, sc, action, mods); ok {
			w.ProcessKeyboardInput(input)
		}
		switch key {
		case glfw.KeyBackspace:
			if action == glfw.Repeat || action == glfw.Release {
				if w.activeInput != nil && len(w.activeInput.value) > 0 {
					if w.activeInput.cursorPosition == 0 {
						w.activeInput.value = w.activeInput.value[:len(w.activeInput.value)-1]
					} else {
						inputVal, cursorPos := w.activeInput.value, w.activeInput.cursorPosition

						if cursorPos+len(inputVal) > 0 {
							w.activeInput.value = inputVal[:len(inputVal)+cursorPos-1] + inputVal[len(inputVal)+cursorPos:]
						}
					}
					w.activeInput.needsRepaint = true
				}
			}
			break
		case glfw.KeyEscape:
			if action == glfw.Release {
				w.DestroyContextMenu()
			}
			break
		case glfw.KeyUp:
			if action == glfw.Release || action == glfw.Repeat {
				w.ProcessArrowKeys("up")
			}
			break
		case glfw.KeyDown:
			if action == glfw.Release || action == glfw.Repeat {
				w.ProcessArrowKeys("down")
			}
			break
		case glfw.KeyLeft:
			if action == glfw.Release || action == glfw.Repeat {
				w.ProcessArrowKeys("left")
			}
			break
		case glfw.KeyRight:
			if action == glfw.Release || action == glfw.Repeat {
				w.ProcessArrowKeys("right")
			}
			break
		case glfw.KeyV:
			if action == glfw.Release && mods == glfw.ModControl {
				if w.activeInput != nil {
					if w.activeInput.cursorPosition == 0 {
						w.activeInput.value = w.activeInput.value + glfw.GetClipboardString()
					} else {
						inputVal, cursorPos := w.activeInput.value, w.activeInput.cursorPosition
						w.activeInput.value = inputVal[:len(inputVal)+cursorPos] + glfw.GetClipboardString() + inputVal[len(inputVal)+cursorPos:]
					}
					w.activeInput.needsRepaint = true
				}
			}
			break
		case glfw.KeyEnter:
			if action == glfw.Release {
				w.ProcessReturnKey()
			}
			break
		}
	})

	w.glw.SetMouseButtonCallback(func(glw *glfw.Window, button glfw.MouseButton, action glfw.Action, mod glfw.ModifierKey) {
		if action == glfw.Release {
			w.ProcessPointerClick(button)
		}
	})

	w.glw.SetScrollCallback(func(glw *glfw.Window, x, y float64) {
		w.ProcessScroll(x, y)
	})
}

func (w *cgoWindow) generateTexture() {
	gl.DeleteTextures(1, &w.backend.texture)
	w.frameBuffer = w.context.Image().(*image.RGBA)

	if w.rootFrame != nil {
		compositeAll(w.frameBuffer, w.rootFrame)
	}

	if w.hasStaticOverlay {
		nBuffer := image.NewRGBA(w.frameBuffer.Bounds())
		draw.Draw(nBuffer, w.frameBuffer.Bounds(), w.frameBuffer, image.ZP, draw.Over)
		w.frameBuffer = nBuffer

		for _, overlay := range w.staticOverlays {
			draw.Draw(w.frameBuffer, overlay.buffer.Bounds().Add(overlay.position), overlay.buffer, image.ZP, draw.Over)
		}
	}

	if w.hasActiveOverlay {
		nBuffer := image.NewRGBA(w.frameBuffer.Bounds())
		draw.Draw(nBuffer, w.frameBuffer.Bounds(), w.frameBuffer, image.ZP, draw.Over)
		w.frameBuffer = nBuffer

		for _, overlay := range w.overlays {
			draw.Draw(w.frameBuffer, overlay.buffer.Bounds().Add(overlay.position), overlay.buffer, image.ZP, draw.Over)
		}
	}

	gl.GenTextures(1, &w.backend.texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, w.backend.texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)

	gl.TexImage2D(
		gl.TEXTURE_2D, 0, gl.RGBA,
		int32(w.frameBuffer.Rect.Size().X), int32(w.frameBuffer.Rect.Size().Y),
		0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(w.frameBuffer.Pix),
	)
}

func (w *cgoWindow) ProcessPointerPosition() {
	w.selectWidget(w.rootFrame)
	if w.selectedWidget != nil &&
		w.selectedWidget.BaseWidget().widgetType == frameWidget {
		w.SetCursor(DefaultCursor)
	}

	w.ProcessButtons()
	w.ProcessTrees()
	w.ProcessInputs()
	w.firePointerPositionEvents()
}

func (w *cgoWindow) selectWidget(widget Widget) {
	if widget.BaseWidget().IsPointInside(w.cursorX, w.cursorY) {
		w.selectedWidget = widget

		for _, childWidget := range widget.Widgets() {
			w.selectWidget(childWidget)
		}
	}
}

func (w *cgoWindow) ProcessButtons() {
	x, y := w.cursorX, w.cursorY

	for _, button := range w.registeredButtons {
		if x > float64(button.computedBox.left)+button.padding &&
			x < float64(button.computedBox.left+button.computedBox.width)-button.padding &&
			y > float64(button.computedBox.top)+button.padding &&
			y < float64(button.computedBox.top+button.computedBox.height)-button.padding {
			button.selected = true
			button.needsRepaint = true
			w.SetCursor(button.cursor)
			break
		} else {
			button.selected = false
			button.needsRepaint = true
		}
	}
}

func (w *cgoWindow) ProcessTrees() {
	x, y := w.cursorX, w.cursorY

	for _, tree := range w.registeredTrees {
		if x > float64(tree.computedBox.left) &&
			x < float64(tree.computedBox.left+tree.computedBox.width) &&
			y > float64(tree.computedBox.top) &&
			y < float64(tree.computedBox.top+tree.computedBox.height) {
			tree.selected = true
			tree.needsRepaint = true
			if glwc, ok := tree.cursor.(*glfw.Cursor); ok {
			w.glw.SetCursor(glwc)
		}
			break
		} else {
			tree.selected = false
			tree.needsRepaint = true
		}
	}
}

func (w *cgoWindow) ProcessInputs() {
	x, y := w.cursorX, w.cursorY

	for _, input := range w.registeredInputs {
		if x > float64(input.computedBox.left)+input.padding &&
			x < float64(input.computedBox.left+input.computedBox.width)-input.padding &&
			y > float64(input.computedBox.top)+input.padding &&
			y < float64(input.computedBox.top+input.computedBox.height)-input.padding {
			input.selected = true
			input.needsRepaint = true
			w.SetCursor(input.cursor)
			break
		} else {
			input.selected = false
			input.needsRepaint = true
		}
	}
}

func (w *cgoWindow) ProcessInputActivation() {
	for _, input := range w.registeredInputs {
		if input.selected == true {
			w.activeInput = input
			input.active = true
			input.needsRepaint = true
			return
		}

		if input.active {
			input.active = false
			input.needsRepaint = true
			w.activeInput = nil
		}
	}
}

func (w *cgoWindow) ProcessButtonClick() {
	for _, button := range w.registeredButtons {
		if button.selected == true {
			button.onClick()
			button.needsRepaint = true
			return
		}
	}
}

func (w *cgoWindow) ProcessTreeClick() {
	for _, tree := range w.registeredTrees {
		if tree.selected == true {
			tree.Click()
			tree.needsRepaint = true
			return
		}
	}
}

func (w *cgoWindow) firePointerPositionEvents() {
	for _, eventCallback := range w.pointerPositionEventListeners {
		eventCallback(w.cursorX, w.cursorY)
	}
}

func (w *cgoWindow) fireScrollEvents(x, y float64) {
	for _, eventCallback := range w.scrollEventListeners {
		eventCallback(int(y))
	}
}

func (w *cgoWindow) fireClickEvents(key MustardKey) {
	for _, eventCallback := range w.clickEventListeners {
		eventCallback(key)
	}
}

func (w *cgoWindow) firePointerClickEvents(button MustardKey) {
	for _, eventCallback := range w.pointerClickEventListeners {
		eventCallback(w.cursorX, w.cursorY, button)
	}
}

func (w *cgoWindow) ProcessContextMenu() {
	if len(w.contextMenu.entries) > 0 {
		x, y := w.GetCursorPosition()

		if x > w.contextMenu.overlay.left &&
			x < w.contextMenu.overlay.left+w.contextMenu.overlay.width &&
			y > w.contextMenu.overlay.top &&
			y < w.contextMenu.overlay.top+w.contextMenu.overlay.height {

			for _, entry := range w.contextMenu.entries {
				if entry.PointIntersects(x, y) {
					if entry != w.contextMenu.selectedEntry {
						w.contextMenu.selectedEntry = entry
						w.refreshContextMenu()
						w.SetCursor(PointerCursor)
					}

					break
				}
			}
		} else {
			if w.contextMenu.selectedEntry != nil {
				w.contextMenu.selectedEntry = nil
				w.refreshContextMenu()
				w.SetCursor(DefaultCursor)
			}
		}
	}
}

func (w *cgoWindow) ProcessContextMenuClick() {
	if w.contextMenu.selectedEntry != nil {
		w.contextMenu.selectedEntry.action()
	}

	w.DestroyContextMenu()
}

func (w *cgoWindow) ProcessScroll(x, y float64) {
	if y != 0 && w.rootFrame != nil {
		scrollWidgetAt(w.rootFrame, w.cursorX, w.cursorY, -int(math.Round(y*40)))
	}
	w.fireScrollEvents(x, y)
}

func (w *cgoWindow) ProcessReturnKey() {
	if w.activeInput != nil && w.activeInput.active == true {
		w.activeInput.active = false

		w.activeInput.needsRepaint = true
		w.activeInput.returnCallback()
		w.activeInput = nil
	}
}

func (w *cgoWindow) ProcessArrowKeys(arrowKey string) {
	if w.activeInput != nil && w.activeInput.active == true {
		if arrowKey == "left" && (w.activeInput.cursorPosition+len(w.activeInput.value)) > 0 {
			w.activeInput.cursorPosition--
			w.activeInput.cursorDirection = true
			w.activeInput.needsRepaint = true
		} else if arrowKey == "right" && w.activeInput.cursorPosition < 0 {
			w.activeInput.cursorPosition++
			w.activeInput.cursorDirection = false
			w.activeInput.needsRepaint = true
		}
	}
}

func (w *cgoWindow) ProcessTextInput(char rune) {
	for _, listener := range w.textInputEventListeners {
		listener(char)
	}
}

func (w *cgoWindow) ProcessKeyboardInput(input KeyboardInput) {
	for _, listener := range w.keyboardInputEventListeners {
		listener(input)
	}
}

func (w *cgoWindow) destroy() {
	w.visible = false
	w.glw.Destroy()

	w.glw = nil
	w.context = nil
	w.backend = nil
	w.frameBuffer = nil

	w.defaultCursor = nil
	w.pointerCursor = nil

	w.registeredButtons = nil
	w.registeredInputs = nil
	w.activeInput = nil

	w.rootFrame = nil
}

func (w *cgoWindow) setActive(active bool) {
	wasActive := w.IsActive()
	w.activeSet = true
	if wasActive == active {
		w.active = active
		return
	}
	w.active = active
	for _, listener := range append([]func(bool){}, w.activeChangeListeners...) {
		listener(active)
	}
}

func (w *cgoWindow) IsActive() bool {
	return !w.activeSet || w.active
}

func setGLFWHints() {
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 2)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
}

func drawRootFrameCGO(window *cgoWindow) {
	window.rootFrame.computedBox.SetCoords(0, 0, float64(window.width), float64(window.height))

	window.rootFrame.draw()
}

func (w *cgoWindow) drawRootFrame() {
	w.rootFrame.computedBox.SetCoords(0, 0, float64(w.width), float64(w.height))

	w.rootFrame.draw()
}


func (w *cgoWindow) ProcessPointerClick(button glfw.MouseButton) {
	if w.hasActiveOverlay {
		w.ProcessContextMenuClick()
	} else {
		if button == glfw.MouseButtonLeft {
		w.ProcessButtonClick()
		w.ProcessTreeClick()
		w.ProcessInputActivation()
		w.fireClickEvents(MouseLeft)
		w.firePointerClickEvents(MouseLeft)
		} else if button == glfw.MouseButtonRight {
		w.fireClickEvents(MouseRight)
		w.firePointerClickEvents(MouseRight)
		}
	}
}

func (w *cgoWindow) DestroyContextMenu() {
	w.RemoveOverlay(w.contextMenu.overlay)
	w.contextMenu.entries = nil
	w.contextMenu.selectedEntry = nil
	w.SetCursor(DefaultCursor)
}

func (w *cgoWindow) RemoveOverlay(overlay *Overlay) {
	var nOverlays []*Overlay
	for _, o := range w.overlays {
		if o != overlay {
			nOverlays = append(nOverlays, o)
		}
	}
	w.overlays = nOverlays
}

func (w *cgoWindow) refreshContextMenu() {
	ctx := gg.NewContextForRGBA(w.contextMenu.overlay.buffer)
	menuWidth := float64(w.contextMenu.overlay.buffer.Rect.Max.X)
	textLeft := 4.
	ctx.SetHexColor("#eee")
	ctx.Clear()

	for idx, entry := range w.contextMenu.entries {
		if w.contextMenu.selectedEntry == entry {
			ctx.DrawRectangle(0, float64(idx*20), menuWidth, 20)
			ctx.SetHexColor("#ccc")
			ctx.Fill()
		}

		ctx.SetHexColor("#222")
		ctx.DrawString(prepEntry(ctx, entry.entryText, menuWidth-textLeft), textLeft, 16+float64(idx*20))
		ctx.Fill()
	}
}


func keyboardInputFromGLFW(key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) (KeyboardInput, bool) {
	input := KeyboardInput{Key: glfwKeyName(key, scancode)}
	switch action {
	case glfw.Press:
		input.Action = KeyboardPress
	case glfw.Repeat:
		input.Action = KeyboardRepeat
	case glfw.Release:
		input.Action = KeyboardRelease
	default:
		return KeyboardInput{}, false
	}
	input.Control = mods&glfw.ModControl != 0
	input.Shift = mods&glfw.ModShift != 0
	input.Alt = mods&glfw.ModAlt != 0
	input.Super = mods&glfw.ModSuper != 0
	return input, true
}

func glfwKeyName(key glfw.Key, scancode int) string {
	switch key {
	case glfw.KeyBackspace:
		return "Backspace"
	case glfw.KeyDelete:
		return "Delete"
	case glfw.KeyEnter:
		return "Enter"
	case glfw.KeyEscape:
		return "Escape"
	case glfw.KeyTab:
		return "Tab"
	case glfw.KeyLeft:
		return "ArrowLeft"
	case glfw.KeyRight:
		return "ArrowRight"
	case glfw.KeyUp:
		return "ArrowUp"
	case glfw.KeyDown:
		return "ArrowDown"
	default:
		if name := glfw.GetKeyName(key, scancode); name != "" {
			return name
		}
		return "Unidentified"
	}
}

// cgoRenderer implements Renderer using OpenGL.
type cgoRenderer struct {
	window *cgoWindow
}

func (r *cgoRenderer) Clear() {
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

func (r *cgoRenderer) DrawImage(img image.Image, x, y float64) {
	_ = img
	_ = x
	_ = y
}

func (r *cgoRenderer) SetColor(red, green, blue, alpha float32) {
	_ = red
	_ = green
	_ = blue
	_ = alpha
}

func (r *cgoRenderer) DrawRectangle(x, y, width, height float64) {
	_ = x
	_ = y
	_ = width
	_ = height
}

func (r *cgoRenderer) Present() {
	r.window.glw.SwapBuffers()
}

func (r *cgoRenderer) Size() (int, int) {
	return r.window.Size()
}

// cgoApp implements App using GLFW.
type cgoApp struct {
	name         string
	windows      []Window
	activeWindow Window
	tabbedMode   bool
}

func (a *cgoApp) Run(callback func()) {
	for {
		for _, window := range a.windows {
			if a.tabbedMode && window != a.activeWindow {
				continue
			}
			if w, ok := window.(*cgoWindow); ok && w.visible && !w.IsClosed() {
				w.glw.MakeContextCurrent()
				w.glw.SwapBuffers()

				if w.needsReflow {
					w.drawRootFrame()
					w.needsReflow = false
				} else {
					redrawWidgets(w.rootFrame)
				}

				w.generateTexture()

				xscale, yscale := float32(1), float32(1)
				if w.hiDPI {
					xscale, yscale = w.glw.GetContentScale()
				}

				gl.Viewport(0, 0, int32(float32(w.width)*xscale), int32(float32(w.height)*yscale))
				gl.DrawArrays(gl.TRIANGLES, 0, 6)

				glfw.PollEvents()
			}
		}

		callback()
	}
}

func (a *cgoApp) AddWindow(w Window) {
	a.windows = append(a.windows, w)
	if a.tabbedMode {
		if cw, ok := w.(*cgoWindow); ok {
			cw.setActive(false)
		}
	}
	if cw, ok := w.(*cgoWindow); ok && cw.rootFrame != nil {
		setWidgetWindow(cw.rootFrame, cw)
	}
}

func (a *cgoApp) DestroyWindow(w Window) {
	wasActive := a.activeWindow == w
	var nWindows []Window
	for _, appWindow := range a.windows {
		if appWindow != w {
			nWindows = append(nWindows, appWindow)
		}
	}
	a.windows = nWindows
	if cw, ok := w.(*cgoWindow); ok {
		cw.setActive(false)
		cw.destroy()
	}
	if wasActive {
		a.activeWindow = nil
		if len(a.windows) > 0 {
			a.ActivateWindow(a.windows[0])
		}
	}
}

// ActivateWindow opts the app into tab-style processing and activates one managed window.
func (a *cgoApp) ActivateWindow(window Window) error {
	if window == nil || !a.containsWindow(window) {
		return fmt.Errorf("window is not managed by this app")
	}
	a.tabbedMode = true
	a.activeWindow = window
	for _, appWindow := range a.windows {
		if cw, ok := appWindow.(*cgoWindow); ok {
			cw.setActive(appWindow == window)
		}
	}
	return nil
}

func (a *cgoApp) containsWindow(window Window) bool {
	for _, candidate := range a.windows {
		if candidate == window {
			return true
		}
	}
	return false
}

func (a *cgoApp) Windows() []Window {
	return a.windows
}

type glBackend struct {
	program uint32

	vao uint32
	vbo uint32

	texture uint32
	quad    []float32
}

type errInvalidWindow struct{}

func (e *errInvalidWindow) Error() string {
	return "mustard: invalid window type for cgo driver"
}

var quad = []float32{
	1.0, 1.0, 1.0, 1.0, 0.0,
	-1.0, 1.0, 1.0, 0.0, 0.0,
	1.0, -1.0, 1.0, 1.0, 1.0,

	-1.0, 1.0, 1.0, 0.0, 0.0,
	-1.0, -1.0, 1.0, 0.0, 1.0,
	1.0, -1.0, 1.0, 1.0, 1.0,
}

func createNewGLProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}

	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	program := gl.CreateProgram()

	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return program, nil
}

func createGLBackend() *glBackend {
	program, err := createNewGLProgram(vertexShader, fragmentShader)
	if err != nil {
		panic(err)
	}

	gl.UseProgram(program)
	textureUniform := gl.GetUniformLocation(program, gl.Str("tex\x00"))
	gl.Uniform1i(textureUniform, 0)
	gl.BindFragDataLocation(program, 0, gl.Str("outputColor\x00"))

	backend := &glBackend{program: program}

	// Init OpenGL vao
	gl.GenVertexArrays(1, &backend.vao)
	gl.BindVertexArray(backend.vao)

	// Init OpenGL vbo
	gl.GenBuffers(1, &backend.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, backend.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(quad)*4, gl.Ptr(quad), gl.STATIC_DRAW)

	vertAttrib := uint32(gl.GetAttribLocation(backend.program, gl.Str("vert\x00")))
	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointer(vertAttrib, 3, gl.FLOAT, false, 5*4, gl.PtrOffset(0))

	texCoordAttrib := uint32(gl.GetAttribLocation(backend.program, gl.Str("vertTexCoord\x00")))
	gl.EnableVertexAttribArray(texCoordAttrib)
	gl.VertexAttribPointer(texCoordAttrib, 2, gl.FLOAT, false, 5*4, gl.PtrOffset(3*4))

	return backend
}

const vertexShader = `
#version 330

in vec3 vert;
in vec2 vertTexCoord;

out vec2 fragTexCoord;

void main() {
	fragTexCoord = vertTexCoord;
	gl_Position = vec4(vert, 1.0);
}
` + "\x00"

const fragmentShader = `
#version 330

uniform sampler2D tex;

in vec2 fragTexCoord;

out vec4 outputColor;

void main() {
	outputColor = texture(tex, fragTexCoord);
}
` + "\x00"

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}
