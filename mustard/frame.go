package mustard

import (
	"log"

	gg "github.com/danfragoso/thdwb/gg"
)

// CreateFrame - Creates and returns a new Frame
func CreateFrame(orientation FrameOrientation) *Frame {
	var widgets []Widget

	return &Frame{
		baseWidget: baseWidget{
			widgetType: frameWidget,

			needsRepaint: true,
			visible:      true,
			widgets:      widgets,

			backgroundColor: "#fff",
		},

		orientation: orientation,
	}
}

// SetBackgroundColor - Sets the frame background color
func (frame *Frame) SetBackgroundColor(backgroundColor string) {
	if len(backgroundColor) > 0 && string(backgroundColor[0]) == "#" {
		frame.backgroundColor = backgroundColor
		frame.needsRepaint = true
	}
}

// SetWidth - Sets the frame width
func (frame *Frame) SetWidth(width float64) {
	frame.box.width = width
	frame.fixedWidth = true
	frame.RequestReflow()
}

// SetHeight - Sets the frame height
func (frame *Frame) SetHeight(height float64) {
	frame.box.height = height
	frame.fixedHeight = true
	frame.RequestReflow()
}

// GetHeight returns the frame height
func (frame *Frame) GetHeight() float64 {
	return frame.box.height
}

// SetVisible sets the visibility of the frame and its children.
func (frame *Frame) SetVisible(visible bool) {
	frame.baseWidget.SetVisible(visible)
}

// RemoveWidget removes a widget from the frame at the given index.
func (frame *Frame) RemoveWidget(index int) {
	if index >= 0 && index < len(frame.widgets) {
		frame.DetachWidget(frame.widgets[index])
	}
}

func drawRootFrame(window *WindowBase) {
	window.rootFrame.computedBox.SetCoords(0, 0, float64(window.width), float64(window.height))

	window.rootFrame.draw()
}

func (frame *Frame) draw() {
	// Recover from panics during rendering so a single bad frame drops the
	// frame instead of crashing the whole browser process.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("render panic in frame draw: %v", r)
		}
	}()

	if !frame.GetVisible() {
		return
	}
	if frame.window == nil {
		return
	}
	top, left, width, height := frame.computedBox.GetCoords()
	window := frame.window
	context, ok := window.context.(*gg.Context)
	if !ok {
		return
	}
	context.SetHexColor(frame.backgroundColor)
	context.DrawRectangle(float64(left), float64(top), float64(width), float64(height))
	context.Fill()

	copyWidgetToBuffer(frame, context.Image())

	// Collect visible children once so the layout array and the draw loop
	// iterate over the identical set of widgets (prevents index-out-of-range
	// when hidden children are interleaved with visible ones).
	var visibleChildren []Widget
	for _, child := range frame.Widgets() {
		if child.BaseWidget().GetVisible() {
			visibleChildren = append(visibleChildren, child)
		}
	}

	if len(visibleChildren) > 0 {
		var coreWidgets []*baseWidget
		for _, child := range visibleChildren {
			coreWidgets = append(coreWidgets, child.BaseWidget())
		}
		childrenLayout := calculateChildrenWidgetsLayout(coreWidgets, top, left, width, height, frame.orientation)

		for i, child := range visibleChildren {
			child.ComputedBox().SetCoords(childrenLayout[i].box.GetCoords())
			child.draw()
		}
	}
}
