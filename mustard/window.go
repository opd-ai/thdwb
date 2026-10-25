package mustard

import (
	"image"
	"image/draw"

	gg "github.com/danfragoso/thdwb/gg"
)

func CreateNewWindow(driver Driver, title string, width, height int, hiDPI bool) (Window, error) {
	if driver == nil {
		driver = NewDriver()
	}
	return driver.NewWindow(title, width, height, hiDPI)
}

func (window *WindowBase) Show() {
	window.needsReflow = true
	window.visible = true
}

func (window *WindowBase) Hide() {
	window.visible = false
}

func (window *WindowBase) Close() {
}

func (window *WindowBase) IsClosed() bool {
	return false
}

func (window *WindowBase) SetTitle(title string) {
	window.title = title
}

func (window *WindowBase) Size() (int, int) {
	return window.width, window.height
}

func (window *WindowBase) SetSize(width, height int) {
	window.width = width
	window.height = height
}

func (window *WindowBase) PollEvents() {
}

func (window *WindowBase) ShouldClose() bool {
	return false
}

func (window *WindowBase) GetCursorPosition() (float64, float64) {
	return window.cursorX, window.cursorY
}

func (window *WindowBase) SetCursor(cursor Cursor) {
	window.cursor = cursor
}

// AttachActiveChangeListener registers a callback and immediately reports the current state.
func (window *WindowBase) AttachActiveChangeListener(callback func(bool)) {
	if callback == nil {
		return
	}
	window.activeChangeListeners = append(window.activeChangeListeners, callback)
	callback(window.IsActive())
}

// IsActive reports whether this window is active; new unmanaged windows default to active.
func (window *WindowBase) IsActive() bool {
	return !window.activeSet || window.active
}

func (window *WindowBase) setActive(active bool) {
	wasActive := window.IsActive()
	window.activeSet = true
	if wasActive == active {
		window.active = active
		return
	}
	window.active = active
	for _, listener := range append([]func(bool){}, window.activeChangeListeners...) {
		listener(active)
	}
}

// SetRootFrame - Sets the window root frame
func (window *WindowBase) SetRootFrame(frame *Frame) {
	window.rootFrame = frame
}

// GetSize returns the window dimensions.
func (window *WindowBase) GetSize() (int, int) {
	return window.width, window.height
}

func (window *WindowBase) RequestReflow() {
	window.needsReflow = true
}

func (window *WindowBase) resize(width, height int, xscale, yscale float32) {
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
	if swidth <= 0 || sheight <= 0 || (window.width == swidth && window.height == sheight) {
		return
	}
	window.width, window.height = swidth, sheight
	window.context = gg.NewContext(window.width, window.height)
	window.frameBuffer = nil
	window.needsReflow = true
}

func compositeWidget(buffer *image.RGBA, widget Widget) {
	if widget.NeedsRepaint() {
		top, left, width, height := widget.ComputedBox().GetCoords()

		draw.Draw(buffer, image.Rectangle{
			image.Point{int(left), int(top)}, image.Point{int(left + width), int(top + height)},
		}, widget.Buffer(), image.Point{}, draw.Over)

		widget.SetNeedsRepaint(false)
	}
}

func compositeAll(buffer *image.RGBA, widget Widget) {
	compositeWidget(buffer, widget)

	for _, childWidget := range widget.Widgets() {
		compositeAll(buffer, childWidget)
	}
}

func (window *WindowBase) RegisterTree(tree *TreeWidget) {
	window.registeredTrees = append(window.registeredTrees, tree)
}

func (window *WindowBase) RegisterButton(button *ButtonWidget, callback func()) {
	button.onClick = callback
	window.registeredButtons = append(window.registeredButtons, button)
}

func (window *WindowBase) UnregisterButton(button *ButtonWidget) {
	for i, btn := range window.registeredButtons {
		if btn == button {
			window.registeredButtons = append(window.registeredButtons[:i], window.registeredButtons[i+1:]...)
			break
		}
	}
}

func (window *WindowBase) RegisterInput(input *InputWidget) {
	window.registeredInputs = append(window.registeredInputs, input)
}

func (window *WindowBase) UnregisterInput(input *InputWidget) {
	for i, in := range window.registeredInputs {
		if in == input {
			window.registeredInputs = append(window.registeredInputs[:i], window.registeredInputs[i+1:]...)
			break
		}
	}
}

func (window *WindowBase) AttachPointerPositionEventListener(callback func(pointerX, pointerY float64)) {
	window.pointerPositionEventListeners = append(window.pointerPositionEventListeners, callback)
}

func (window *WindowBase) AttachScrollEventListener(callback func(direction int)) {
	window.scrollEventListeners = append(window.scrollEventListeners, callback)
}

func (window *WindowBase) AttachClickEventListener(callback func(MustardKey)) {
	window.clickEventListeners = append(window.clickEventListeners, callback)
}

func (window *WindowBase) AttachPointerClickEventListener(callback func(float64, float64, MustardKey)) {
	window.pointerClickEventListeners = append(window.pointerClickEventListeners, callback)
}

func (window *WindowBase) AttachKeyboardInputEventListener(callback func(KeyboardInput)) {
	window.keyboardInputEventListeners = append(window.keyboardInputEventListeners, callback)
}

func (window *WindowBase) AttachTextInputEventListener(callback func(rune)) {
	window.textInputEventListeners = append(window.textInputEventListeners, callback)
}

func (window *WindowBase) AddOverlay(overlay *Overlay) {
	window.overlays = append(
		window.overlays,
		overlay,
	)
	window.hasActiveOverlay = true
}

func (window *WindowBase) RemoveOverlay(overlay *Overlay) {
	var nOverlays []*Overlay
	for _, o := range window.overlays {
		if o != overlay {
			nOverlays = append(nOverlays, o)
		}
	}
	window.overlays = nOverlays
	if len(window.overlays) == 0 {
		window.hasActiveOverlay = false
	}
}

func (window *WindowBase) AddStaticOverlay(overlay *Overlay) {
	window.staticOverlays = append(
		window.staticOverlays,
		overlay,
	)
	window.hasStaticOverlay = true
}

func (window *WindowBase) RemoveStaticOverlay(overlay *Overlay) {
	var nOverlays []*Overlay
	for _, o := range window.staticOverlays {
		if o != overlay {
			nOverlays = append(nOverlays, o)
		}
	}
	window.staticOverlays = nOverlays
	if len(window.staticOverlays) == 0 {
		window.hasStaticOverlay = false
	}
}

func CreateOverlay(ref string, buffer *image.RGBA, position image.Point) *Overlay {
	return &Overlay{
		ref:    ref,
		active: true,
		top:    float64(position.Y),
		left:   float64(position.X),
		width:  float64(buffer.Rect.Max.X),
		height: float64(buffer.Rect.Max.Y),
		position: position,
		buffer:   buffer,
	}
}

