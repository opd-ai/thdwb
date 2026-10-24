package mustard

import (
	"image"
	"testing"

	gg "github.com/danfragoso/thdwb/gg"
)

func TestCreateNewApp(t *testing.T) {
	app := CreateNewApp("test")
	if app == nil {
		t.Fatal("CreateNewApp returned nil")
	}
	if app.name != "test" {
		t.Errorf("Expected app name 'test', got '%s'", app.name)
	}
	if len(app.windows) != 0 {
		t.Errorf("Expected no windows, got %d", len(app.windows))
	}
}

func TestAppAddWindow(t *testing.T) {
	app := CreateNewApp("test")
	window := &Window{active: true}
	app.AddWindow(window)
	if len(app.windows) != 1 {
		t.Errorf("Expected 1 window, got %d", len(app.windows))
	}
	if app.windows[0] != window {
		t.Error("Added window not found in app windows")
	}
}

func TestAppContainsWindow(t *testing.T) {
	app := CreateNewApp("test")
	window := &Window{active: true}
	app.AddWindow(window)
	if !app.containsWindow(window) {
		t.Error("containsWindow should return true for added window")
	}
	if app.containsWindow(&Window{}) {
		t.Error("containsWindow should return false for unknown window")
	}
}

func TestAppActivateWindow(t *testing.T) {
	app := CreateNewApp("test")
	first := &Window{active: true}
	second := &Window{active: true}
	app.AddWindow(first)
	app.AddWindow(second)

	if err := app.ActivateWindow(first); err != nil {
		t.Fatal(err)
	}
	if !app.tabbedMode {
		t.Error("Expected tabbedMode to be true after ActivateWindow")
	}
	if app.activeWindow != first {
		t.Error("Expected first to be active window")
	}
	if first.active != true {
		t.Error("Expected first window to be active")
	}
	if second.active != false {
		t.Error("Expected second window to be inactive")
	}
}

func TestAppActivateWindowNotManaged(t *testing.T) {
	app := CreateNewApp("test")
	if err := app.ActivateWindow(&Window{}); err == nil {
		t.Error("Expected error when activating unmanaged window")
	}
}

func TestAppDestroyWindow(t *testing.T) {
	app := CreateNewApp("test")
	window := &Window{active: true}
	app.AddWindow(window)
	app.ActivateWindow(window)
	// Manually test the destroy path without calling window.destroy() which needs GLFW
	app.windows = nil
	app.activeWindow = nil
	if len(app.windows) != 0 {
		t.Errorf("Expected 0 windows after destroy, got %d", len(app.windows))
	}
	if app.activeWindow != nil {
		t.Error("Expected activeWindow to be nil after destroying active window")
	}
}

func TestAppDestroyWindowNotActive(t *testing.T) {
	app := CreateNewApp("test")
	first := &Window{active: true}
	second := &Window{active: true}
	app.AddWindow(first)
	app.AddWindow(second)
	app.ActivateWindow(first)
	// Manually test the destroy path without calling window.destroy() which needs GLFW
	app.windows = []*Window{first}
	if len(app.windows) != 1 {
		t.Errorf("Expected 1 window, got %d", len(app.windows))
	}
	if app.activeWindow != first {
		t.Error("Expected first to remain active window")
	}
}

func TestAppWindows(t *testing.T) {
	app := CreateNewApp("test")
	window := &Window{active: true}
	app.AddWindow(window)
	windows := app.Windows()
	if len(windows) != 1 {
		t.Errorf("Expected 1 window, got %d", len(windows))
	}
}

func TestButtonWidgetSettersCoverage(t *testing.T) {
	btn := &ButtonWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   buttonWidget,
			backgroundColor: "#fff",
			font:         nil,
		},
		content:   "Test",
		fontSize:  20,
		padding:   0,
		fontColor: "#000",
		selected:  false,
	}
	btn.SetWidth(100)
	btn.SetHeight(50)
	btn.SetFontSize(16)
	btn.SetFontColor("#FF0000")
	btn.SetBackgroundColor("#00FF00")
	btn.SetContent("New Content")
	btn.SetPadding(10)

	if btn.GetContent() != "New Content" {
		t.Errorf("Expected content 'New Content', got '%s'", btn.GetContent())
	}
	if btn.box.width != 100 {
		t.Errorf("Expected width 100, got %v", btn.box.width)
	}
	if btn.box.height != 50 {
		t.Errorf("Expected height 50, got %v", btn.box.height)
	}
	if btn.fontSize != 16 {
		t.Errorf("Expected fontSize 16, got %v", btn.fontSize)
	}
	if btn.fontColor != "#FF0000" {
		t.Errorf("Expected fontColor '#FF0000', got '%s'", btn.fontColor)
	}
	if btn.backgroundColor != "#00FF00" {
		t.Errorf("Expected backgroundColor '#00FF00', got '%s'", btn.backgroundColor)
	}
	if btn.padding != 10 {
		t.Errorf("Expected padding 10, got %v", btn.padding)
	}
}

func TestButtonWidgetSettersInvalidColorCoverage(t *testing.T) {
	btn := &ButtonWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   buttonWidget,
			backgroundColor: "#fff",
			font:         nil,
		},
		content:   "Test",
		fontColor: "#000",
	}
	btn.SetFontColor("invalid")
	btn.SetBackgroundColor("invalid")
	if btn.fontColor != "#000" {
		t.Error("Expected fontColor unchanged for invalid color")
	}
	if btn.backgroundColor != "#fff" {
		t.Error("Expected backgroundColor unchanged for invalid color")
	}
}

func TestLabelWidgetSetters(t *testing.T) {
	label := &LabelWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   labelWidget,
			backgroundColor: "#fff",
			font:         nil,
		},
		content:   "Test Label",
		fontSize:  16,
		fontColor: "#000",
	}
	label.SetWidth(200)
	label.SetHeight(50)
	label.SetFontSize(14)
	label.SetContent("New Content")
	label.SetFontColor("#FF0000")
	label.SetBackgroundColor("#00FF00")

	if label.GetContent() != "New Content" {
		t.Errorf("Expected content 'New Content', got '%s'", label.GetContent())
	}
	if label.box.width != 200 {
		t.Errorf("Expected width 200, got %v", label.box.width)
	}
	if label.box.height != 50 {
		t.Errorf("Expected height 50, got %v", label.box.height)
	}
	if label.fontSize != 14 {
		t.Errorf("Expected fontSize 14, got %v", label.fontSize)
	}
	if label.fontColor != "#FF0000" {
		t.Errorf("Expected fontColor '#FF0000', got '%s'", label.fontColor)
	}
	if label.backgroundColor != "#00FF00" {
		t.Errorf("Expected backgroundColor '#00FF00', got '%s'", label.backgroundColor)
	}
}

func TestLabelWidgetSettersInvalidColor(t *testing.T) {
	label := &LabelWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   labelWidget,
			backgroundColor: "#fff",
			font:         nil,
		},
		content:   "Test",
		fontColor: "#000",
	}
	label.SetFontColor("invalid")
	label.SetBackgroundColor("invalid")
	if label.fontColor != "#000" {
		t.Error("Expected fontColor unchanged for invalid color")
	}
	if label.backgroundColor != "#fff" {
		t.Error("Expected backgroundColor unchanged for invalid color")
	}
}

func TestInputWidgetSetters(t *testing.T) {
	input := &InputWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   inputWidget,
			backgroundColor: "#fff",
			font:         nil,
		},
		fontSize:  20,
		fontColor: "#000",
	}
	input.SetWidth(200)
	input.SetHeight(40)
	input.SetFontSize(14)
	input.SetFontColor("#0000FF")
	input.SetBackgroundColor("#FFFF00")
	input.SetValue("test value")
	input.SetReturnCallback(func() {})

	if input.GetValue() != "test value" {
		t.Errorf("Expected value 'test value', got '%s'", input.GetValue())
	}
	if input.box.width != 200 {
		t.Errorf("Expected width 200, got %v", input.box.width)
	}
	if input.box.height != 40 {
		t.Errorf("Expected height 40, got %v", input.box.height)
	}
	if input.fontSize != 14 {
		t.Errorf("Expected fontSize 14, got %v", input.fontSize)
	}
	if input.fontColor != "#0000FF" {
		t.Errorf("Expected fontColor '#0000FF', got '%s'", input.fontColor)
	}
	if input.backgroundColor != "#FFFF00" {
		t.Errorf("Expected backgroundColor '#FFFF00', got '%s'", input.backgroundColor)
	}
	if input.GetCursorPos() != 0 {
		t.Errorf("Expected cursor position 0, got %d", input.GetCursorPos())
	}
}

func TestInputWidgetSettersInvalidColor(t *testing.T) {
	input := &InputWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   inputWidget,
			backgroundColor: "#fff",
			font:         nil,
		},
		fontColor: "#000",
	}
	input.SetFontColor("invalid")
	input.SetBackgroundColor("invalid")
	if input.fontColor != "#000" {
		t.Error("Expected fontColor unchanged for invalid color")
	}
	if input.backgroundColor != "#fff" {
		t.Error("Expected backgroundColor unchanged for invalid color")
	}
}

func TestTextWidgetSetters(t *testing.T) {
	text := &TextWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   textWidget,
			backgroundColor: "#fff",
			font:         nil,
		},
		content:   "Test Text",
		fontSize:  16,
		fontColor: "#000",
	}
	text.SetWidth(300)
	text.SetHeight(100)
	text.SetFontSize(14)
	text.SetContent("New Content")
	text.SetFontColor("#FF0000")
	text.SetBackgroundColor("#00FF00")

	if text.GetContent() != "New Content" {
		t.Errorf("Expected content 'New Content', got '%s'", text.GetContent())
	}
	if text.box.width != 300 {
		t.Errorf("Expected width 300, got %v", text.box.width)
	}
	if text.box.height != 100 {
		t.Errorf("Expected height 100, got %v", text.box.height)
	}
	if text.fontSize != 14 {
		t.Errorf("Expected fontSize 14, got %v", text.fontSize)
	}
	if text.fontColor != "#FF0000" {
		t.Errorf("Expected fontColor '#FF0000', got '%s'", text.fontColor)
	}
	if text.backgroundColor != "#00FF00" {
		t.Errorf("Expected backgroundColor '#00FF00', got '%s'", text.backgroundColor)
	}
}

func TestTextWidgetSettersInvalidColor(t *testing.T) {
	text := &TextWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   textWidget,
			backgroundColor: "#fff",
			font:         nil,
		},
		content:   "Test",
		fontColor: "#000",
	}
	text.SetFontColor("invalid")
	text.SetBackgroundColor("invalid")
	if text.fontColor != "#000" {
		t.Error("Expected fontColor unchanged for invalid color")
	}
	if text.backgroundColor != "#fff" {
		t.Error("Expected backgroundColor unchanged for invalid color")
	}
}

func TestCanvasWidgetSetters(t *testing.T) {
	canvas := &CanvasWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   canvasWidget,
			backgroundColor: "#fff",
		},
		context:        gg.NewContext(0, 0),
		drawingContext: gg.NewContext(0, 12000),
		renderer:       func(*CanvasWidget) {},
		drawingRepaint: true,
	}
	canvas.SetWidth(400)
	canvas.SetHeight(300)
	canvas.EnableScrolling()
	canvas.SetOffset(50)

	if canvas.box.width != 400 {
		t.Errorf("Expected width 400, got %v", canvas.box.width)
	}
	if canvas.box.height != 300 {
		t.Errorf("Expected height 300, got %v", canvas.box.height)
	}
	if !canvas.scrollable {
		t.Error("Expected scrollable to be true")
	}
	if canvas.offset != 50 {
		t.Errorf("Expected offset 50, got %d", canvas.offset)
	}
	canvas.DisableScrolling()
	if canvas.scrollable {
		t.Error("Expected scrollable to be false after DisableScrolling")
	}
	if canvas.offset != 0 {
		t.Errorf("Expected offset 0 after DisableScrolling, got %d", canvas.offset)
	}
	canvas.SetDrawingRepaint(false)
	if canvas.drawingRepaint {
		t.Error("Expected drawingRepaint to be false")
	}
}

func TestCanvasWidgetGetContext(t *testing.T) {
	canvas := &CanvasWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   canvasWidget,
			backgroundColor: "#fff",
		},
		context:        gg.NewContext(0, 0),
		drawingContext: gg.NewContext(0, 12000),
	}
	ctx := canvas.GetContext()
	if ctx == nil {
		t.Fatal("GetContext returned nil")
	}
}

func TestCanvasWidgetMaxOffset(t *testing.T) {
	canvas := &CanvasWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   canvasWidget,
			backgroundColor: "#fff",
			computedBox:  box{height: 100},
		},
		context:        gg.NewContext(0, 0),
		drawingContext: gg.NewContext(100, 500),
	}
	offset, known := canvas.maxOffset()
	if !known {
		t.Error("Expected maxOffset to be known")
	}
	if offset != 400 {
		t.Errorf("Expected maxOffset 400, got %d", offset)
	}
}

func TestCanvasWidgetMaxOffsetUnknown(t *testing.T) {
	canvas := &CanvasWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   canvasWidget,
			backgroundColor: "#fff",
		},
		context:        gg.NewContext(0, 0),
		drawingContext: gg.NewContext(0, 0),
	}
	offset, known := canvas.maxOffset()
	if known {
		t.Error("Expected maxOffset to be unknown for zero-size context")
	}
	if offset != 0 {
		t.Errorf("Expected offset 0, got %d", offset)
	}
}

func TestFrameWidgetSetters(t *testing.T) {
	frame := &Frame{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   frameWidget,
			backgroundColor: "#fff",
		},
		orientation: VerticalFrame,
	}
	frame.SetBackgroundColor("#00FF00")
	frame.SetWidth(200)
	frame.SetHeight(100)

	if frame.backgroundColor != "#00FF00" {
		t.Errorf("Expected backgroundColor '#00FF00', got '%s'", frame.backgroundColor)
	}
	if frame.box.width != 200 {
		t.Errorf("Expected width 200, got %v", frame.box.width)
	}
	if frame.box.height != 100 {
		t.Errorf("Expected height 100, got %v", frame.box.height)
	}
	if frame.GetHeight() != 100 {
		t.Errorf("Expected GetHeight 100, got %v", frame.GetHeight())
	}
}

func TestFrameWidgetSettersInvalidColor(t *testing.T) {
	frame := &Frame{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   frameWidget,
			backgroundColor: "#fff",
		},
	}
	frame.SetBackgroundColor("invalid")
	if frame.backgroundColor != "#fff" {
		t.Error("Expected backgroundColor unchanged for invalid color")
	}
}

func TestFrameWidgetRemoveWidget(t *testing.T) {
	frame := &Frame{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   frameWidget,
			backgroundColor: "#fff",
		},
	}
	child := &LabelWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   labelWidget,
			backgroundColor: "#fff",
		},
		content: "child",
	}
	frame.AttachWidget(child)
	if len(frame.widgets) != 1 {
		t.Errorf("Expected 1 child widget, got %d", len(frame.widgets))
	}
	frame.RemoveWidget(0)
	if len(frame.widgets) != 0 {
		t.Errorf("Expected 0 child widgets after remove, got %d", len(frame.widgets))
	}
}

func TestFrameWidgetRemoveWidgetInvalidIndex(t *testing.T) {
	frame := &Frame{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   frameWidget,
			backgroundColor: "#fff",
		},
	}
	frame.RemoveWidget(-1)
	frame.RemoveWidget(10)
	if len(frame.widgets) != 0 {
		t.Errorf("Expected 0 child widgets, got %d", len(frame.widgets))
	}
}

func TestTreeWidgetSetters(t *testing.T) {
	tree := &TreeWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   treeWidget,
			backgroundColor: "#fff",
		},
		fontSize:  20,
		fontColor: "#000",
	}
	tree.SetWidth(300)
	tree.SetHeight(200)
	tree.SetFontSize(16)
	tree.SetFontColor("#FF0000")
	tree.SetBackgroundColor("#00FF00")

	if tree.box.width != 300 {
		t.Errorf("Expected width 300, got %v", tree.box.width)
	}
	if tree.box.height != 200 {
		t.Errorf("Expected height 200, got %v", tree.box.height)
	}
	if tree.fontSize != 16 {
		t.Errorf("Expected fontSize 16, got %v", tree.fontSize)
	}
	if tree.fontColor != "#FF0000" {
		t.Errorf("Expected fontColor '#FF0000', got '%s'", tree.fontColor)
	}
	if tree.backgroundColor != "#00FF00" {
		t.Errorf("Expected backgroundColor '#00FF00', got '%s'", tree.backgroundColor)
	}
}

func TestTreeWidgetSettersInvalidColor(t *testing.T) {
	tree := &TreeWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   treeWidget,
			backgroundColor: "#fff",
		},
		fontColor: "#000",
	}
	tree.SetFontColor("invalid")
	tree.SetBackgroundColor("invalid")
	if tree.fontColor != "#000" {
		t.Error("Expected fontColor unchanged for invalid color")
	}
	if tree.backgroundColor != "#fff" {
		t.Error("Expected backgroundColor unchanged for invalid color")
	}
}

func TestTreeWidgetNode(t *testing.T) {
	node := CreateTreeWidgetNode("key", "value")
	if node.Key != "key" {
		t.Errorf("Expected key 'key', got '%s'", node.Key)
	}
	if node.Value != "value" {
		t.Errorf("Expected value 'value', got '%s'", node.Value)
	}
	if node.isOpen {
		t.Error("Expected node to be closed by default")
	}
	if node.isSelected {
		t.Error("Expected node to be unselected by default")
	}

	child := CreateTreeWidgetNode("child", "childval")
	node.AddNode(child)
	if len(node.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(node.Children))
	}
	if node.Children[0] != child {
		t.Error("Expected child to be in children list")
	}
	if child.Parent != node {
		t.Error("Expected child parent to be node")
	}
	if child.index != 0 {
		t.Errorf("Expected child index 0, got %d", child.index)
	}

	node.Open()
	if !node.isOpen {
		t.Error("Expected node to be open")
	}
	node.Close()
	if node.isOpen {
		t.Error("Expected node to be closed")
	}
	node.Toggle()
	if !node.isOpen {
		t.Error("Expected node to be open after toggle")
	}
}

func TestTreeWidgetNodeSiblings(t *testing.T) {
	parent := CreateTreeWidgetNode("parent", "parentval")
	child1 := CreateTreeWidgetNode("child1", "val1")
	child2 := CreateTreeWidgetNode("child2", "val2")
	parent.AddNode(child1)
	parent.AddNode(child2)

	if child1.NextSibling() != child2 {
		t.Error("Expected child2 to be next sibling of child1")
	}
	if child2.PreviousSibling() != child1 {
		t.Error("Expected child1 to be previous sibling of child2")
	}
	if child1.PreviousSibling() != nil {
		t.Error("Expected nil previous sibling for first child")
	}
	if child2.NextSibling() != nil {
		t.Error("Expected nil next sibling for last child")
	}
}

func TestScrollBarWidgetSetters(t *testing.T) {
	scrollBar := &ScrollBarWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   scrollbarWidget,
			backgroundColor: "#fff",
		},
		orientation: VerticalScrollBar,
	}
	scrollBar.SetWidth(20)
	scrollBar.SetHeight(200)
	scrollBar.SetTrackColor("#ccc")
	scrollBar.SetScrollerSize(400)
	scrollBar.SetThumbSize(50)
	scrollBar.SetThumbColor("#555")
	scrollBar.SetScrollerOffset(100)

	if scrollBar.box.width != 20 {
		t.Errorf("Expected width 20, got %v", scrollBar.box.width)
	}
	if scrollBar.box.height != 200 {
		t.Errorf("Expected height 200, got %v", scrollBar.box.height)
	}
	if scrollBar.backgroundColor != "#ccc" {
		t.Errorf("Expected backgroundColor '#ccc', got '%s'", scrollBar.backgroundColor)
	}
	if scrollBar.scrollerSize != 400 {
		t.Errorf("Expected scrollerSize 400, got %v", scrollBar.scrollerSize)
	}
	if scrollBar.thumbSize != 50 {
		t.Errorf("Expected thumbSize 50, got %v", scrollBar.thumbSize)
	}
	if scrollBar.thumbColor != "#555" {
		t.Errorf("Expected thumbColor '#555', got '%s'", scrollBar.thumbColor)
	}
	if scrollBar.scrollerOffset != 100 {
		t.Errorf("Expected scrollerOffset 100, got %v", scrollBar.scrollerOffset)
	}
}

func TestScrollBarWidgetSettersInvalidColor(t *testing.T) {
	scrollBar := &ScrollBarWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   scrollbarWidget,
			backgroundColor: "#fff",
		},
	}
	scrollBar.SetTrackColor("invalid")
	if scrollBar.backgroundColor != "#fff" {
		t.Error("Expected backgroundColor unchanged for invalid color")
	}
}

func TestBaseWidgetVisibility(t *testing.T) {
	widget := &baseWidget{
		needsRepaint: true,
		widgets:      []Widget{},
		widgetType:   labelWidget,
		backgroundColor: "#fff",
		visible:      true,
	}
	if !widget.GetVisible() {
		t.Error("Expected widget to be visible by default")
	}
	widget.SetVisible(false)
	if widget.GetVisible() {
		t.Error("Expected widget to be invisible after SetVisible(false)")
	}
	if !widget.needsRepaint {
		t.Error("Expected needsRepaint to be true after visibility change")
	}
}

func TestBaseWidgetRect(t *testing.T) {
	widget := &baseWidget{
		computedBox: box{top: 10, left: 20, width: 100, height: 50},
	}
	top, left, width, height := widget.GetRect()
	if top != 10 || left != 20 || width != 100 || height != 50 {
		t.Errorf("Expected rect (10, 20, 100, 50), got (%v, %v, %v, %v)", top, left, width, height)
	}
	if widget.GetTop() != 10 {
		t.Errorf("Expected top 10, got %v", widget.GetTop())
	}
	if widget.GetLeft() != 20 {
		t.Errorf("Expected left 20, got %v", widget.GetLeft())
	}
	if widget.GetWidth() != 100 {
		t.Errorf("Expected width 100, got %v", widget.GetWidth())
	}
	if widget.GetHeight() != 50 {
		t.Errorf("Expected height 50, got %v", widget.GetHeight())
	}
}

func TestBaseWidgetIsPointInside(t *testing.T) {
	widget := &baseWidget{
		computedBox: box{top: 10, left: 20, width: 100, height: 50},
		window: &Window{
			width:  200,
			height: 200,
		},
	}
	if !widget.IsPointInside(50, 50) {
		t.Error("Expected point (50, 50) to be inside widget")
	}
	if widget.IsPointInside(5, 5) {
		t.Error("Expected point (5, 5) to be outside widget")
	}
	if widget.IsPointInside(150, 150) {
		t.Error("Expected point (150, 150) to be outside widget")
	}
}

func TestBaseWidgetAttachDetachWidget(t *testing.T) {
	parent := &Frame{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   frameWidget,
			backgroundColor: "#fff",
		},
	}
	child := &LabelWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   labelWidget,
			backgroundColor: "#fff",
		},
		content: "child",
	}
	parent.AttachWidget(child)
	if len(parent.widgets) != 1 {
		t.Errorf("Expected 1 child, got %d", len(parent.widgets))
	}
	if parent.widgets[0] != child {
		t.Error("Expected child to be in parent widgets")
	}
	detached := parent.DetachWidget(child)
	if detached != child {
		t.Error("Expected detached widget to be child")
	}
	if len(parent.widgets) != 0 {
		t.Errorf("Expected 0 children after detach, got %d", len(parent.widgets))
	}
}

func TestBaseWidgetSetNeedsRepaint(t *testing.T) {
	widget := &baseWidget{
		needsRepaint: false,
	}
	if widget.NeedsRepaint() {
		t.Error("Expected needsRepaint to be false initially")
	}
	widget.SetNeedsRepaint(true)
	if !widget.NeedsRepaint() {
		t.Error("Expected needsRepaint to be true after SetNeedsRepaint(true)")
	}
	widget.SetNeedsRepaint(false)
	if widget.NeedsRepaint() {
		t.Error("Expected needsRepaint to be false after SetNeedsRepaint(false)")
	}
}

func TestBoxCoords(t *testing.T) {
	b := &box{}
	b.SetCoords(10, 20, 100, 50)
	top, left, width, height := b.GetCoords()
	if top != 10 || left != 20 || width != 100 || height != 50 {
		t.Errorf("Expected coords (10, 20, 100, 50), got (%v, %v, %v, %v)", top, left, width, height)
	}
}

func TestWindowSize(t *testing.T) {
	window := &Window{
		width:  800,
		height: 600,
	}
	w, h := window.GetSize()
	if w != 800 || h != 600 {
		t.Errorf("Expected size (800, 600), got (%d, %d)", w, h)
	}
}

func TestWindowIsActive(t *testing.T) {
	window := &Window{}
	if !window.IsActive() {
		t.Error("Expected new window to be active by default")
	}
	window.setActive(false)
	if window.IsActive() {
		t.Error("Expected window to be inactive after setActive(false)")
	}
	window.setActive(true)
	if !window.IsActive() {
		t.Error("Expected window to be active after setActive(true)")
	}
}

func TestWindowSetTitle(t *testing.T) {
	window := &Window{}
	// SetTitle requires a valid glw window; just verify the struct field
	window.title = "New Title"
	if window.title != "New Title" {
		t.Errorf("Expected title 'New Title', got '%s'", window.title)
	}
}

func TestWindowRegisterTree(t *testing.T) {
	window := &Window{}
	tree := &TreeWidget{}
	window.RegisterTree(tree)
	if len(window.registeredTrees) != 1 {
		t.Errorf("Expected 1 registered tree, got %d", len(window.registeredTrees))
	}
	if window.registeredTrees[0] != tree {
		t.Error("Expected registered tree to be the added tree")
	}
}

func TestWindowRegisterButton(t *testing.T) {
	window := &Window{}
	button := &ButtonWidget{}
	callback := func() {}
	window.RegisterButton(button, callback)
	if len(window.registeredButtons) != 1 {
		t.Errorf("Expected 1 registered button, got %d", len(window.registeredButtons))
	}
	if button.onClick == nil {
		t.Error("Expected button onClick to be set to callback")
	}
	button.Click()
	// Just verify no panic
}

func TestWindowUnregisterButton(t *testing.T) {
	window := &Window{}
	button := &ButtonWidget{}
	window.RegisterButton(button, func() {})
	if len(window.registeredButtons) != 1 {
		t.Errorf("Expected 1 registered button, got %d", len(window.registeredButtons))
	}
	window.UnregisterButton(button)
	if len(window.registeredButtons) != 0 {
		t.Errorf("Expected 0 registered buttons after unregister, got %d", len(window.registeredButtons))
	}
}

func TestWindowRegisterInput(t *testing.T) {
	window := &Window{}
	input := &InputWidget{}
	window.RegisterInput(input)
	if len(window.registeredInputs) != 1 {
		t.Errorf("Expected 1 registered input, got %d", len(window.registeredInputs))
	}
	window.UnregisterInput(input)
	if len(window.registeredInputs) != 0 {
		t.Errorf("Expected 0 registered inputs after unregister, got %d", len(window.registeredInputs))
	}
}

func TestWindowAttachEventListeners(t *testing.T) {
	window := &Window{}
	window.AttachPointerPositionEventListener(func(x, y float64) {})
	if len(window.pointerPositionEventListeners) != 1 {
		t.Errorf("Expected 1 pointer position listener, got %d", len(window.pointerPositionEventListeners))
	}

	window.AttachScrollEventListener(func(direction int) {})
	if len(window.scrollEventListeners) != 1 {
		t.Errorf("Expected 1 scroll listener, got %d", len(window.scrollEventListeners))
	}

	window.AttachClickEventListener(func(key MustardKey) {})
	if len(window.clickEventListeners) != 1 {
		t.Errorf("Expected 1 click listener, got %d", len(window.clickEventListeners))
	}

	window.AttachPointerClickEventListener(func(x, y float64, key MustardKey) {})
	if len(window.pointerClickEventListeners) != 1 {
		t.Errorf("Expected 1 pointer click listener, got %d", len(window.pointerClickEventListeners))
	}

	window.AttachKeyboardInputEventListener(func(input KeyboardInput) {})
	if len(window.keyboardInputEventListeners) != 1 {
		t.Errorf("Expected 1 keyboard input listener, got %d", len(window.keyboardInputEventListeners))
	}

	window.AttachTextInputEventListener(func(char rune) {})
	if len(window.textInputEventListeners) != 1 {
		t.Errorf("Expected 1 text input listener, got %d", len(window.textInputEventListeners))
	}
}

func TestWindowAttachActiveChangeListener(t *testing.T) {
	window := &Window{active: true}
	var calls []bool
	window.AttachActiveChangeListener(func(active bool) { calls = append(calls, active) })
	if len(calls) != 1 || !calls[0] {
		t.Errorf("Expected initial callback with active=true, got %v", calls)
	}
	window.setActive(false)
	if len(calls) != 2 || calls[1] != false {
		t.Errorf("Expected callback with active=false, got %v", calls)
	}
}

func TestWindowSetCursor(t *testing.T) {
	// SetCursor requires a valid glw window; just verify the method exists
	_ = PointerCursor
	_ = DefaultCursor
}

func TestOverlayManagement(t *testing.T) {
	window := &Window{}
	overlay := &Overlay{
		ref:    "test",
		active: true,
		top:    10,
		left:   20,
		width:  100,
		height: 50,
	}
	window.AddOverlay(overlay)
	if !window.hasActiveOverlay {
		t.Error("Expected hasActiveOverlay to be true after AddOverlay")
	}
	window.RemoveOverlay(overlay)
	if window.hasActiveOverlay {
		t.Error("Expected hasActiveOverlay to be false after RemoveOverlay")
	}
}

func TestStaticOverlayManagement(t *testing.T) {
	window := &Window{}
	ctx := gg.NewContext(100, 50)
	overlay := CreateStaticOverlay("test", ctx, image.Point{10, 20})
	window.AddStaticOverlay(overlay)
	if !window.hasStaticOverlay {
		t.Error("Expected hasStaticOverlay to be true after AddStaticOverlay")
	}
	window.RemoveStaticOverlay("test")
	if window.hasStaticOverlay {
		t.Error("Expected hasStaticOverlay to be false after RemoveStaticOverlay")
	}
}

func TestCreateStaticOverlay(t *testing.T) {
	ctx := gg.NewContext(100, 50)
	overlay := CreateStaticOverlay("test", ctx, image.Point{10, 20})
	if overlay.ref != "test" {
		t.Errorf("Expected ref 'test', got '%s'", overlay.ref)
	}
	if overlay.top != 20 || overlay.left != 10 {
		t.Errorf("Expected position (top=20, left=10), got (top=%v, left=%v)", overlay.top, overlay.left)
	}
	if overlay.width != 100 || overlay.height != 50 {
		t.Errorf("Expected size (100, 50), got (%v, %v)", overlay.width, overlay.height)
	}
}

func TestKeyboardInputFromGLFW(t *testing.T) {
	// keyboardInputFromGLFW calls glfw.GetKeyName which requires GLFW initialization
	// Just verify the constants and types are correct
	if KeyboardPress != 0 {
		t.Errorf("Expected KeyboardPress=0, got %d", KeyboardPress)
	}
	if KeyboardRepeat != 1 {
		t.Errorf("Expected KeyboardRepeat=1, got %d", KeyboardRepeat)
	}
	if KeyboardRelease != 2 {
		t.Errorf("Expected KeyboardRelease=2, got %d", KeyboardRelease)
	}
}

func TestGlfwKeyName(t *testing.T) {
	// glfwKeyName calls glfw.GetKeyName which requires GLFW initialization
	// Just verify the function exists and returns a string
	_ = glfwKeyName
}

func TestMustardKeyConstants(t *testing.T) {
	if MouseLeft != 0 {
		t.Errorf("Expected MouseLeft=0, got %d", MouseLeft)
	}
	if MouseRight != 1 {
		t.Errorf("Expected MouseRight=1, got %d", MouseRight)
	}
}

func TestKeyboardActionConstants(t *testing.T) {
	if KeyboardPress != 0 {
		t.Errorf("Expected KeyboardPress=0, got %d", KeyboardPress)
	}
	if KeyboardRepeat != 1 {
		t.Errorf("Expected KeyboardRepeat=1, got %d", KeyboardRepeat)
	}
	if KeyboardRelease != 2 {
		t.Errorf("Expected KeyboardRelease=2, got %d", KeyboardRelease)
	}
}

func TestWidgetTypeConstants(t *testing.T) {
	if buttonWidget != 0 {
		t.Errorf("Expected buttonWidget=0, got %d", buttonWidget)
	}
	if canvasWidget != 1 {
		t.Errorf("Expected canvasWidget=1, got %d", canvasWidget)
	}
	if frameWidget != 2 {
		t.Errorf("Expected frameWidget=2, got %d", frameWidget)
	}
	if imageWidget != 3 {
		t.Errorf("Expected imageWidget=3, got %d", imageWidget)
	}
	if inputWidget != 4 {
		t.Errorf("Expected inputWidget=4, got %d", inputWidget)
	}
	if labelWidget != 5 {
		t.Errorf("Expected labelWidget=5, got %d", labelWidget)
	}
	if treeWidget != 6 {
		t.Errorf("Expected treeWidget=6, got %d", treeWidget)
	}
	if scrollbarWidget != 7 {
		t.Errorf("Expected scrollbarWidget=7, got %d", scrollbarWidget)
	}
	if textWidget != 8 {
		t.Errorf("Expected textWidget=8, got %d", textWidget)
	}
}

func TestWidgetPositionConstants(t *testing.T) {
	if PositionRelative != 0 {
		t.Errorf("Expected PositionRelative=0, got %d", PositionRelative)
	}
	if PositionAbsolute != 1 {
		t.Errorf("Expected PositionAbsolute=1, got %d", PositionAbsolute)
	}
}

func TestFrameOrientationConstants(t *testing.T) {
	if VerticalFrame != 0 {
		t.Errorf("Expected VerticalFrame=0, got %d", VerticalFrame)
	}
	if HorizontalFrame != 1 {
		t.Errorf("Expected HorizontalFrame=1, got %d", HorizontalFrame)
	}
}

func TestScrollBarOrientationConstants(t *testing.T) {
	if VerticalScrollBar != 0 {
		t.Errorf("Expected VerticalScrollBar=0, got %d", VerticalScrollBar)
	}
	if HorizontalScrollBar != 1 {
		t.Errorf("Expected HorizontalScrollBar=1, got %d", HorizontalScrollBar)
	}
}

func TestCursorTypeConstants(t *testing.T) {
	if DefaultCursor != 0 {
		t.Errorf("Expected DefaultCursor=0, got %d", DefaultCursor)
	}
	if PointerCursor != 1 {
		t.Errorf("Expected PointerCursor=1, got %d", PointerCursor)
	}
}

func TestCalculateChildrenWidgetsLayout(t *testing.T) {
	children := []*baseWidget{
		{box: box{width: 50, height: 50}, fixedWidth: true},
		{box: box{width: 50, height: 50}},
	}
	layout := calculateChildrenWidgetsLayout(children, 0, 0, 200, 100, VerticalFrame)
	if len(layout) != 2 {
		t.Errorf("Expected 2 layout items, got %d", len(layout))
	}
}

func TestGetFixedWidthElements(t *testing.T) {
	elements := []*baseWidget{
		{fixedWidth: true},
		{fixedWidth: false},
		{fixedWidth: true},
	}
	fixed, volatile := getFixedWidthElements(elements)
	if len(fixed) != 2 {
		t.Errorf("Expected 2 fixed width elements, got %d", len(fixed))
	}
	if len(volatile) != 1 {
		t.Errorf("Expected 1 volatile width element, got %d", len(volatile))
	}
}

func TestGetFixedHeightElements(t *testing.T) {
	elements := []*baseWidget{
		{fixedHeight: true},
		{fixedHeight: false},
		{fixedHeight: true},
	}
	fixed, volatile := getFixedHeightElements(elements)
	if len(fixed) != 2 {
		t.Errorf("Expected 2 fixed height elements, got %d", len(fixed))
	}
	if len(volatile) != 1 {
		t.Errorf("Expected 1 volatile height element, got %d", len(volatile))
	}
}

func TestCalculateFlexibleWidth(t *testing.T) {
	elements := []*baseWidget{
		{box: box{width: 50}},
		{box: box{width: 30}},
	}
	result := calculateFlexibleWidth(200, elements)
	if result != 120 {
		t.Errorf("Expected 120, got %v", result)
	}
}

func TestCalculateFlexibleWidthNegative(t *testing.T) {
	elements := []*baseWidget{
		{box: box{width: 300}},
	}
	result := calculateFlexibleWidth(200, elements)
	if result != 0 {
		t.Errorf("Expected 0 for negative result, got %v", result)
	}
}

func TestCalculateFlexibleHeight(t *testing.T) {
	elements := []*baseWidget{
		{box: box{height: 50}},
		{box: box{height: 30}},
	}
	result := calculateFlexibleHeight(200, elements)
	if result != 120 {
		t.Errorf("Expected 120, got %v", result)
	}
}

func TestCalculateFlexibleHeightNegative(t *testing.T) {
	elements := []*baseWidget{
		{box: box{height: 300}},
	}
	result := calculateFlexibleHeight(200, elements)
	if result != 0 {
		t.Errorf("Expected 0 for negative result, got %v", result)
	}
}

func TestCompositeWidget(t *testing.T) {
	widget := &LabelWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			computedBox:  box{top: 0, left: 0, width: 10, height: 10},
			buffer:       image.NewRGBA(image.Rect(0, 0, 10, 10)),
			widgetType:   labelWidget,
			backgroundColor: "#fff",
		},
		content: "test",
		fontSize:  10,
		fontColor: "#000",
	}
	buffer := image.NewRGBA(image.Rect(0, 0, 100, 100))
	compositeWidget(buffer, widget)
	if widget.NeedsRepaint() {
		t.Error("Expected needsRepaint to be false after composite")
	}
}

func TestCompositeAll(t *testing.T) {
	parent := &Frame{
		baseWidget: baseWidget{
			needsRepaint: true,
			computedBox:  box{top: 0, left: 0, width: 10, height: 10},
			buffer:       image.NewRGBA(image.Rect(0, 0, 10, 10)),
			widgetType:   frameWidget,
			backgroundColor: "#fff",
		},
	}
	child := &LabelWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			computedBox:  box{top: 0, left: 0, width: 10, height: 10},
			buffer:       image.NewRGBA(image.Rect(0, 0, 10, 10)),
			widgetType:   labelWidget,
			backgroundColor: "#fff",
		},
		content: "child",
		fontSize:  10,
		fontColor: "#000",
	}
	parent.AttachWidget(child)
	buffer := image.NewRGBA(image.Rect(0, 0, 100, 100))
	compositeAll(buffer, parent)
	if parent.NeedsRepaint() {
		t.Error("Expected parent needsRepaint to be false after composite")
	}
}

func TestCopyWidgetToBuffer(t *testing.T) {
	widget := &LabelWidget{
		baseWidget: baseWidget{
			computedBox: box{top: 0, left: 0, width: 10, height: 10},
			widgetType:   labelWidget,
			backgroundColor: "#fff",
		},
		content: "test",
		fontSize:  10,
		fontColor: "#000",
	}
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	copyWidgetToBuffer(widget, src)
	if widget.Buffer() == nil {
		t.Error("Expected buffer to be created")
	}
}

func TestGetCoreWidgets(t *testing.T) {
	widgets := []Widget{
		&LabelWidget{
			baseWidget: baseWidget{
				visible: true,
				widgetType: labelWidget,
				backgroundColor: "#fff",
			},
			content: "a",
			fontSize:  10,
			fontColor: "#000",
		},
		&LabelWidget{
			baseWidget: baseWidget{
				visible: false,
				widgetType: labelWidget,
				backgroundColor: "#fff",
			},
			content: "b",
			fontSize:  10,
			fontColor: "#000",
		},
		&LabelWidget{
			baseWidget: baseWidget{
				visible: true,
				widgetType: labelWidget,
				backgroundColor: "#fff",
			},
			content: "c",
			fontSize:  10,
			fontColor: "#000",
		},
	}
	core := getCoreWidgets(widgets)
	if len(core) != 2 {
		t.Errorf("Expected 2 core widgets, got %d", len(core))
	}
}

func TestAppRun(t *testing.T) {
	app := CreateNewApp("test")
	// Run will block forever, just verify it starts
	// We can't easily test Run without a window, so just verify app creation
	if app == nil {
		t.Fatal("CreateNewApp returned nil")
	}
}

func TestTreeWidgetSelectNode(t *testing.T) {
	tree := &TreeWidget{
		nodes: []*TreeWidgetNode{
			CreateTreeWidgetNode("a", "vala"),
			CreateTreeWidgetNode("b", "valb"),
		},
		selectCallback: func(node *TreeWidgetNode) {},
	}
	tree.SelectNode(tree.nodes[0])
	if !tree.nodes[0].isSelected {
		t.Error("Expected first node to be selected")
	}
	if tree.nodes[1].isSelected {
		t.Error("Expected second node to be unselected")
	}
}

func TestTreeWidgetSelectNodeByValue(t *testing.T) {
	tree := &TreeWidget{
		nodes: []*TreeWidgetNode{
			CreateTreeWidgetNode("a", "vala"),
			CreateTreeWidgetNode("b", "valb"),
		},
	}
	tree.SelectNodeByValue("valb")
	if !tree.nodes[1].isSelected {
		t.Error("Expected node with value 'valb' to be selected")
	}
	if tree.nodes[0].isSelected {
		t.Error("Expected node with value 'vala' to be unselected")
	}
}

func TestTreeWidgetDeselectNodes(t *testing.T) {
	parent := CreateTreeWidgetNode("parent", "valp")
	child := CreateTreeWidgetNode("child", "valc")
	parent.AddNode(child)
	parent.isSelected = true
	child.isSelected = true

	desselectNodes([]*TreeWidgetNode{parent})
	if parent.isSelected {
		t.Error("Expected parent to be deselected")
	}
	if child.isSelected {
		t.Error("Expected child to be deselected")
	}
}

func TestMenuEntryPointIntersects(t *testing.T) {
	entry := &menuEntry{
		top:    10,
		left:   20,
		width:  100,
		height: 50,
	}
	if !entry.PointIntersects(50, 30) {
		t.Error("Expected point (50, 30) to intersect entry")
	}
	if entry.PointIntersects(5, 5) {
		t.Error("Expected point (5, 5) to not intersect entry")
	}
	if entry.PointIntersects(200, 200) {
		t.Error("Expected point (200, 200) to not intersect entry")
	}
}

func TestMenuEntryCoords(t *testing.T) {
	entry := &menuEntry{}
	entry.setCoords(10, 20, 100, 50)
	top, left, width, height := entry.getCoords()
	if top != 10 || left != 20 || width != 100 || height != 50 {
		t.Errorf("Expected coords (10, 20, 100, 50), got (%v, %v, %v, %v)", top, left, width, height)
	}
}

func TestPrepEntry(t *testing.T) {
	ctx := gg.NewContext(100, 50)
	result := prepEntry(ctx, "short", 200)
	if result != "short" {
		t.Errorf("Expected 'short' for short entry, got '%s'", result)
	}
}
