package mustard

import (
	"math"
)

// keyboardInputFromDriver converts platform-agnostic input to KeyboardInput.
func keyboardInputFromDriver(key string, action KeyboardAction, control, shift, alt, super bool) KeyboardInput {
	_ = key
	_ = action
	_ = control
	_ = shift
	_ = alt
	_ = super
	return KeyboardInput{
		Key:     key,
		Action:  action,
		Control: control,
		Shift:   shift,
		Alt:     alt,
		Super:   super,
	}
}

func (window *WindowBase) ProcessKeyboardInput(input KeyboardInput) {
	for _, listener := range window.keyboardInputEventListeners {
		listener(input)
	}
}

func (window *WindowBase) ProcessTextInput(char rune) {
	for _, listener := range window.textInputEventListeners {
		listener(char)
	}
}

func (window *WindowBase) ProcessPointerPosition() {
	if window.hasActiveOverlay {
		window.ProcessContextMenu()
	} else {
		window.selectWidget(window.rootFrame)
		if window.selectedWidget != nil &&
			window.selectedWidget.BaseWidget().widgetType == frameWidget {
			window.SetCursor(DefaultCursor)
		}

		window.ProcessButtons()
		window.ProcessTrees()
		window.ProcessInputs()
		window.firePointerPositionEvents()
	}
}

func (window *WindowBase) selectWidget(widget Widget) {
	if widget.BaseWidget().IsPointInside(window.cursorX, window.cursorY) {
		window.selectedWidget = widget

		for _, childWidget := range widget.Widgets() {
			window.selectWidget(childWidget)
		}
	}
}

func (window *WindowBase) ProcessPointerClick(button MustardKey) {
	if window.hasActiveOverlay {
		window.ProcessContextMenuClick()
	} else {
		switch button {
		case MouseLeft:
			window.ProcessButtonClick()
			window.ProcessTreeClick()
			window.ProcessInputActivation()
			window.fireClickEvents(MouseLeft)
			window.firePointerClickEvents(MouseLeft)
		case MouseRight:
			window.fireClickEvents(MouseRight)
			window.firePointerClickEvents(MouseRight)
		}
	}
}

func (window *WindowBase) ProcessScroll(x, y float64) {
	if y != 0 && window.rootFrame != nil {
		scrollWidgetAt(window.rootFrame, window.cursorX, window.cursorY, -int(math.Round(y*40)))
	}
	window.fireScrollEvents(x, y)
}

func scrollWidgetAt(widget Widget, x, y float64, delta int) bool {
	if widget == nil || !widget.BaseWidget().IsPointInside(x, y) {
		return false
	}
	children := widget.Widgets()
	for i := len(children) - 1; i >= 0; i-- {
		if scrollWidgetAt(children[i], x, y, delta) {
			return true
		}
	}
	canvas, ok := widget.(*CanvasWidget)
	if !ok || !canvas.scrollable {
		return false
	}
	offset := canvas.offset
	canvas.SetOffset(offset + delta)
	return canvas.offset != offset
}

func (window *WindowBase) ProcessReturnKey() {
	if window.activeInput != nil && window.activeInput.active {
		window.activeInput.active = false

		window.activeInput.needsRepaint = true
		window.activeInput.returnCallback()
		window.activeInput = nil
	}
}

func (window *WindowBase) ProcessArrowKeys(arrowKey string) {
	if window.activeInput != nil && window.activeInput.active {
		if arrowKey == "left" && (window.activeInput.cursorPosition+len(window.activeInput.value)) > 0 {
			window.activeInput.cursorPosition--
			window.activeInput.cursorDirection = true
			window.activeInput.needsRepaint = true
		} else if arrowKey == "right" && window.activeInput.cursorPosition < 0 {
			window.activeInput.cursorPosition++
			window.activeInput.cursorDirection = false
			window.activeInput.needsRepaint = true
		}
	}
}

func (window *WindowBase) ProcessTrees() {
	x, y := window.cursorX, window.cursorY

	for _, tree := range window.registeredTrees {
		if x > float64(tree.computedBox.left) &&
			x < float64(tree.computedBox.left+tree.computedBox.width) &&
			y > float64(tree.computedBox.top) &&
			y < float64(tree.computedBox.top+tree.computedBox.height) {
			tree.selected = true
			tree.needsRepaint = true
			break
		} else {
			tree.selected = false
			tree.needsRepaint = true
		}
	}
}

func (window *WindowBase) ProcessButtons() {
	x, y := window.cursorX, window.cursorY

	for _, button := range window.registeredButtons {
		if x > float64(button.computedBox.left)+button.padding &&
			x < float64(button.computedBox.left+button.computedBox.width)-button.padding &&
			y > float64(button.computedBox.top)+button.padding &&
			y < float64(button.computedBox.top+button.computedBox.height)-button.padding {
			button.selected = true
			button.needsRepaint = true
			window.SetCursor(button.cursor)
			break
		} else {
			button.selected = false
			button.needsRepaint = true
		}
	}
}

func (window *WindowBase) ProcessInputs() {
	x, y := window.cursorX, window.cursorY

	for _, input := range window.registeredInputs {
		if x > float64(input.computedBox.left)+input.padding &&
			x < float64(input.computedBox.left+input.computedBox.width)-input.padding &&
			y > float64(input.computedBox.top)+input.padding &&
			y < float64(input.computedBox.top+input.computedBox.height)-input.padding {
			input.selected = true
			input.needsRepaint = true
			window.SetCursor(input.cursor)
			break
		} else {
			input.selected = false
			input.needsRepaint = true
		}
	}
}

func (window *WindowBase) ProcessInputActivation() {
	for _, input := range window.registeredInputs {
		if input.selected {
			window.activeInput = input
			input.active = true
			input.needsRepaint = true
			return
		}

		if input.active {
			input.active = false
			input.needsRepaint = true
			window.activeInput = nil
		}
	}
}

func (window *WindowBase) ProcessButtonClick() {
	for _, button := range window.registeredButtons {
		if button.selected {
			button.onClick()
			button.needsRepaint = true
			return
		}
	}
}

func (window *WindowBase) ProcessTreeClick() {
	for _, tree := range window.registeredTrees {
		if tree.selected {
			tree.Click()
			tree.needsRepaint = true
			return
		}
	}
}

func (window *WindowBase) firePointerPositionEvents() {
	for _, eventCallback := range window.pointerPositionEventListeners {
		eventCallback(window.cursorX, window.cursorY)
	}
}

func (window *WindowBase) fireScrollEvents(x, y float64) {
	for _, eventCallback := range window.scrollEventListeners {
		eventCallback(int(y))
	}
}

func (window *WindowBase) fireClickEvents(key MustardKey) {
	for _, eventCallback := range window.clickEventListeners {
		eventCallback(key)
	}
}

func (window *WindowBase) firePointerClickEvents(button MustardKey) {
	for _, eventCallback := range window.pointerClickEventListeners {
		eventCallback(window.cursorX, window.cursorY, button)
	}
}

func (window *WindowBase) ProcessContextMenu() {
	if len(window.contextMenu.entries) > 0 {
		x, y := window.GetCursorPosition()

		if x > window.contextMenu.overlay.left &&
			x < window.contextMenu.overlay.left+window.contextMenu.overlay.width &&
			y > window.contextMenu.overlay.top &&
			y < window.contextMenu.overlay.top+window.contextMenu.overlay.height {

			for _, entry := range window.contextMenu.entries {
				if entry.PointIntersects(x, y) {
					if entry != window.contextMenu.selectedEntry {
						window.SelectEntry(entry)
					}

					break
				}
			}
		} else {
			window.DeselectEntries()
		}
	}
}

func (window *WindowBase) ProcessContextMenuClick() {
	if window.contextMenu.selectedEntry != nil {
		window.contextMenu.selectedEntry.action()
	}

	window.DestroyContextMenu()
}
