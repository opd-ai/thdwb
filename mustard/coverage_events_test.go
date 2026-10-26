package mustard

import (
	"image"
	"testing"

	gg "github.com/danfragoso/thdwb/gg"
)

// newTestWindow builds a headless WindowBase with a real gg context, a visible
// root frame sized to the window, and the cursor centered on that frame.
func newTestWindow(t *testing.T, width, height int) *WindowBase {
	t.Helper()
	window := &WindowBase{
		width:   width,
		height:  height,
		visible: true,
		context: gg.NewContext(width, height),
	}
	frame := CreateFrame(VerticalFrame)
	frame.SetVisible(true)
	setWidgetWindow(frame, window)
	frame.computedBox.SetCoords(0, 0, float64(width), float64(height))
	window.rootFrame = frame
	window.cursorX = float64(width / 2)
	window.cursorY = float64(height / 2)
	return window
}

// ---- ProcessPointerPosition ----

func TestProcessPointerPositionSelectsAndFiresEvents(t *testing.T) {
	window := newTestWindow(t, 100, 80)

	window.AttachPointerPositionEventListener(func(x, y float64) {
		if x != 50 || y != 40 {
			t.Errorf("pointer position = (%v,%v), want (50,40)", x, y)
		}
	})
	window.ProcessPointerPosition()
	if window.selectedWidget == nil {
		t.Fatal("expected a widget to be selected under the cursor")
	}
	if window.cursor != DefaultCursor {
		t.Errorf("cursor = %v, want DefaultCursor for frame selection", window.cursor)
	}
}

func TestProcessPointerPositionWithOverlayRoutesToContextMenu(t *testing.T) {
	window := newTestWindow(t, 100, 80)
	window.EnableContextMenus()
	window.AddContextMenuEntry("Item", func() {})
	// DrawContextMenu positions the single entry at the cursor and installs
	// the overlay. PointIntersects uses strict comparisons, so step the cursor
	// slightly below the entry's top to land strictly inside its bounds.
	window.cursorX, window.cursorY = 20, 20
	window.DrawContextMenu()
	if !window.hasActiveOverlay {
		t.Fatal("expected the context menu overlay to be active")
	}
	window.cursorY = float64(window.contextMenu.overlay.top) + 10

	window.ProcessPointerPosition()
	if window.contextMenu.selectedEntry == nil {
		t.Fatal("expected the context menu entry to be selected under the cursor")
	}
	if window.cursor != PointerCursor {
		t.Errorf("cursor = %v, want PointerCursor after entry selection", window.cursor)
	}
}

// ---- ProcessPointerClick ----

func TestProcessPointerClickLeftFiresCallbacks(t *testing.T) {
	window := newTestWindow(t, 100, 80)
	called := 0
	window.AttachClickEventListener(func(k MustardKey) {
		called++
		if k != MouseLeft {
			t.Errorf("click key = %v, want MouseLeft", k)
		}
	})
	window.AttachPointerClickEventListener(func(x, y float64, k MustardKey) {
		if x != 50 || y != 40 || k != MouseLeft {
			t.Errorf("pointer click = (%v,%v,%v), want (50,40,MouseLeft)", x, y, k)
		}
	})
	window.ProcessPointerClick(MouseLeft)
	if called != 1 {
		t.Fatalf("click listener fired %d times, want 1", called)
	}
}

func TestProcessPointerClickRightFiresCallbacks(t *testing.T) {
	window := newTestWindow(t, 100, 80)
	var key MustardKey
	window.AttachClickEventListener(func(k MustardKey) { key = k })
	window.ProcessPointerClick(MouseRight)
	if key != MouseRight {
		t.Errorf("click key = %v, want MouseRight", key)
	}
}

func TestProcessPointerClickWithOverlayRunsContextMenuClick(t *testing.T) {
	window := newTestWindow(t, 100, 80)
	window.EnableContextMenus()
	actionRan := false
	window.AddContextMenuEntry("Do it", func() { actionRan = true })
	overlay := &Overlay{ref: "ctx", top: 0, left: 0, width: 100, height: 80, buffer: image.NewRGBA(image.Rect(0, 0, 100, 80))}
	window.AddOverlay(overlay)
	window.contextMenu.overlay = overlay
	window.contextMenu.selectedEntry = window.contextMenu.entries[0]

	window.ProcessPointerClick(MouseLeft)
	if !actionRan {
		t.Fatal("expected the selected entry action to run")
	}
	if len(window.contextMenu.entries) != 0 {
		t.Fatal("expected the context menu to be destroyed after click")
	}
	if window.hasActiveOverlay {
		t.Fatal("expected the overlay to be removed after destroy")
	}
}

// ---- ProcessScroll ----

func TestProcessScrollWithoutRootFrameStillFiresEvent(t *testing.T) {
	window := newTestWindow(t, 100, 80)
	window.rootFrame = nil
	var dirs []int
	window.AttachScrollEventListener(func(d int) { dirs = append(dirs, d) })
	window.ProcessScroll(0, -1)
	if len(dirs) != 1 || dirs[0] != -1 {
		t.Fatalf("scroll directions = %v, want [-1]", dirs)
	}
}

func TestProcessScrollWithRootFrameNoCanvas(t *testing.T) {
	window := newTestWindow(t, 100, 80)
	var dirs []int
	window.AttachScrollEventListener(func(d int) { dirs = append(dirs, d) })
	window.ProcessScroll(0, 2)
	if len(dirs) != 1 || dirs[0] != 2 {
		t.Fatalf("scroll directions = %v, want [2]", dirs)
	}
	if window.needsReflow {
		t.Fatal("scroll without a canvas must not trigger a reflow")
	}
}

// ---- keyboard / text ----

func TestProcessKeyboardInputAndTextInput(t *testing.T) {
	window := newTestWindow(t, 100, 80)
	var gotKey KeyboardInput
	window.AttachKeyboardInputEventListener(func(k KeyboardInput) { gotKey = k })
	window.ProcessKeyboardInput(KeyboardInput{Key: "a", Action: KeyboardPress})
	if gotKey.Key != "a" || gotKey.Action != KeyboardPress {
		t.Fatalf("keyboard input = %+v, want key=a action=press", gotKey)
	}

	var gotRune rune
	window.AttachTextInputEventListener(func(r rune) { gotRune = r })
	window.ProcessTextInput('z')
	if gotRune != 'z' {
		t.Fatalf("text input = %q, want z", gotRune)
	}
}

func TestKeyboardInputFromDriver(t *testing.T) {
	in := keyboardInputFromDriver("q", KeyboardRepeat, true, false, true, false)
	if in.Key != "q" || in.Action != KeyboardRepeat || !in.Control || in.Shift || !in.Alt || in.Super {
		t.Fatalf("keyboardInputFromDriver = %+v, want q/repeat/ctrl+alt", in)
	}
}

// ---- input activation / return / arrows ----

func TestProcessInputActivationAndReturn(t *testing.T) {
	window := newTestWindow(t, 100, 80)
	input := &InputWidget{
		baseWidget:   baseWidget{visible: true},
		returnCallback: func() {},
	}
	input.computedBox.SetCoords(10, 10, 80, 60)
	window.RegisterInput(input)
	window.RegisterButton(&ButtonWidget{baseWidget: baseWidget{visible: true}}, func() {})

	// Position cursor over the input so ProcessInputs marks it selected.
	window.cursorX, window.cursorY = 50, 40
	window.ProcessInputs()
	if !input.selected {
		t.Fatal("expected input to be selected under the cursor")
	}
	window.ProcessInputActivation()
	if !input.active || window.activeInput != input {
		t.Fatal("expected the input to become active")
	}

	fired := false
	input.returnCallback = func() { fired = true }
	window.ProcessReturnKey()
	if !fired {
		t.Fatal("expected the return callback to fire")
	}
	if input.active || window.activeInput != nil {
		t.Fatal("expected the input to be deactivated after return")
	}

	// ProcessInputActivation with no selected input deactivates the active one.
	input.selected = false
	input.active = true
	window.activeInput = input
	window.ProcessInputActivation()
	if input.active {
		t.Fatal("expected the active input to deactivate when nothing is selected")
	}
}

func TestProcessArrowKeysCoverage(t *testing.T) {
	window := newTestWindow(t, 100, 80)
	input := &InputWidget{baseWidget: baseWidget{visible: true}, value: "abcd", cursorPosition: 2, returnCallback: func() {}}
	window.activeInput = input
	input.active = true

	window.ProcessArrowKeys("left")
	if input.cursorPosition != 1 || !input.cursorDirection || !input.needsRepaint {
		t.Fatalf("after left: pos=%d dir=%v repaint=%v", input.cursorPosition, input.cursorDirection, input.needsRepaint)
	}
	input.cursorPosition = -1
	window.ProcessArrowKeys("right")
	if input.cursorPosition != 0 || input.cursorDirection {
		t.Fatalf("after right: pos=%d dir=%v", input.cursorPosition, input.cursorDirection)
	}
	// No-op key should leave state unchanged.
	window.ProcessArrowKeys("up")
	if input.cursorPosition != 0 {
		t.Fatalf("up changed cursor position to %d", input.cursorPosition)
	}
	// No active input should be a no-op.
	window.activeInput = nil
	window.ProcessArrowKeys("left")
}

// ---- button processing & clicks ----

func TestProcessButtonsAndButtonClick(t *testing.T) {
	window := newTestWindow(t, 100, 80)
	var clicked bool
	button := &ButtonWidget{
		baseWidget: baseWidget{visible: true, cursor: PointerCursor},
		padding:    4,
	}
	button.computedBox.SetCoords(10, 10, 80, 60)
	window.RegisterButton(button, func() { clicked = true })

	window.cursorX, window.cursorY = 50, 40
	window.ProcessButtons()
	if !button.selected || window.cursor != PointerCursor {
		t.Fatal("expected the button to be selected with pointer cursor")
	}

	window.ProcessButtonClick()
	if !clicked {
		t.Fatal("expected the button click callback to fire")
	}
	if !button.needsRepaint {
		t.Fatal("expected the button to request a repaint after click")
	}

	// Moving off the button deselects it.
	window.cursorX, window.cursorY = 2, 2
	window.ProcessButtons()
	if button.selected {
		t.Fatal("expected the button to deselect when the cursor leaves")
	}
}

// ---- tree processing & clicks ----

func TestProcessTreesAndTreeClick(t *testing.T) {
	window := newTestWindow(t, 100, 80)
	clicked := false
	tree := &TreeWidget{
		baseWidget: baseWidget{visible: true},
		fontSize:   20,
	}
	tree.window = window
	tree.computedBox.SetCoords(0, 0, 100, 80)
	node := CreateTreeWidgetNode("n", "v")
	node.box.SetCoords(10, 10, 80, 24)
	tree.AddNode(node)
	tree.SetSelectCallback(func(*TreeWidgetNode) { clicked = true })
	window.RegisterTree(tree)

	window.cursorX, window.cursorY = 50, 20
	window.ProcessTrees()
	if !tree.selected {
		t.Fatal("expected the tree to be selected under the cursor")
	}

	// Click through fireNodeEvents: y within node height, x > left+25 (no toggle).
	tree.Click()
	if !clicked {
		t.Fatal("expected the tree select callback to fire on click")
	}

	// Off-tree deselects.
	window.cursorX, window.cursorY = 2, 70
	tree.computedBox.SetCoords(30, 30, 40, 20)
	window.ProcessTrees()
	if tree.selected {
		t.Fatal("expected the tree to deselect when the cursor leaves")
	}
}

func TestTreeClickTogglesNodeWhenClickingIconColumn(t *testing.T) {
	window := newTestWindow(t, 100, 80)
	tree := &TreeWidget{baseWidget: baseWidget{visible: true}, fontSize: 20}
	tree.window = window
	tree.computedBox.SetCoords(0, 0, 100, 80)
	node := CreateTreeWidgetNode("n", "v")
	node.box.SetCoords(10, 10, 80, 24)
	tree.AddNode(node)
	tree.SetSelectCallback(func(*TreeWidgetNode) {})

	// x within the icon column (< left+25) triggers a toggle.
	window.cursorX, window.cursorY = 12, 20
	tree.Click()
	if !node.isOpen {
		t.Fatal("expected the node to toggle open when clicking the icon column")
	}
}

// ---- selectWidget ----

func TestSelectWidgetRecursesIntoChildren(t *testing.T) {
	window := newTestWindow(t, 100, 80)
	window.cursorX, window.cursorY = 50, 40

	frame := window.rootFrame
	child := CreateFrame(VerticalFrame)
	child.SetVisible(true)
	child.computedBox.SetCoords(0, 0, 100, 80)
	setWidgetWindow(child, window)
	frame.AttachWidget(child)

	window.selectWidget(frame)
	if window.selectedWidget != child {
		t.Fatal("expected selectWidget to recurse into the child under the cursor")
	}
}
