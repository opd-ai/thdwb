package mustard

import (
	"image"
	"testing"

	gg "github.com/danfragoso/thdwb/gg"
)

func TestCreateButtonWidget(t *testing.T) {
	// Create minimal button without GLFW initialization
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
	if btn == nil {
		t.Fatal("CreateButtonWidget returned nil")
	}
	if btn.GetContent() != "Test" {
		t.Errorf("Expected content 'Test', got '%s'", btn.GetContent())
	}
}

func TestButtonWidgetSetters(t *testing.T) {
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
}

func TestCreateInputWidget(t *testing.T) {
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
	if input == nil {
		t.Fatal("CreateInputWidget returned nil")
	}
	input.SetWidth(200)
	input.SetHeight(40)
	input.SetFontSize(14)
	input.SetFontColor("#0000FF")
	input.SetBackgroundColor("#FFFF00")
	input.SetValue("test value")

	if input.GetValue() != "test value" {
		t.Errorf("Expected value 'test value', got '%s'", input.GetValue())
	}
}

func TestCreateLabelWidget(t *testing.T) {
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
	if label == nil {
		t.Fatal("CreateLabelWidget returned nil")
	}
	label.SetWidth(300)
	label.SetFontSize(18)
	label.SetFontColor("#FF00FF")
}

func TestCreateFrame(t *testing.T) {
	frame := &Frame{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   frameWidget,
			backgroundColor: "#fff",
			font:         nil,
		},
		orientation: VerticalFrame,
	}
	if frame == nil {
		t.Fatal("CreateFrame returned nil")
	}
	frame.SetWidth(400)
	frame.SetHeight(300)
	frame.SetBackgroundColor("#00FFFF")

	child := &LabelWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   labelWidget,
			backgroundColor: "#fff",
			font:         nil,
		},
		content:   "Child",
		fontSize:  16,
		fontColor: "#000",
	}
	frame.AttachWidget(child)

	if len(frame.Widgets()) != 1 {
		t.Errorf("Expected 1 child widget, got %d", len(frame.Widgets()))
	}

	detached := frame.DetachWidget(child)
	if detached != child {
		t.Error("Detached widget doesn't match")
	}
	if len(frame.Widgets()) != 0 {
		t.Errorf("Expected 0 child widgets after detach, got %d", len(frame.Widgets()))
	}
}

func TestCreateCanvasWidget(t *testing.T) {
	canvas := &CanvasWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   canvasWidget,
			backgroundColor: "#fff",
			font:         nil,
		},
		context:        gg.NewContext(0, 0),
		drawingContext: gg.NewContext(0, 0),
	}
	if canvas == nil {
		t.Fatal("CreateCanvasWidget returned nil")
	}
	canvas.SetWidth(500)
	canvas.SetHeight(400)
	canvas.EnableScrolling()
	canvas.DisableScrolling()
	canvas.SetOffset(100)
	if canvas.GetOffset() != 100 {
		t.Errorf("Expected offset 100, got %d", canvas.GetOffset())
	}
	ctx := canvas.GetContext()
	if ctx == nil {
		t.Error("GetContext returned nil")
	}
}

func TestCreateTreeWidget(t *testing.T) {
	tree := &TreeWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   treeWidget,
			backgroundColor: "#fff",
			font:         nil,
		},
		fontSize:  16,
		fontColor: "#000",
		nodes:     []*TreeWidgetNode{},
	}
	if tree == nil {
		t.Fatal("CreateTreeWidget returned nil")
	}
	node := CreateTreeWidgetNode("key", "value")
	tree.AddNode(node)
	if len(tree.nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(tree.nodes))
	}
	tree.RemoveNodes()
	if len(tree.nodes) != 0 {
		t.Errorf("Expected 0 nodes after RemoveNodes, got %d", len(tree.nodes))
	}
}

func TestCreateTreeWidgetNode(t *testing.T) {
	node := CreateTreeWidgetNode("key", "value")
	if node == nil {
		t.Fatal("CreateTreeWidgetNode returned nil")
	}
	if node.Key != "key" || node.Value != "value" {
		t.Errorf("Expected key='key', value='value', got key='%s', value='%s'", node.Key, node.Value)
	}

	child := CreateTreeWidgetNode("child_key", "child_value")
	node.AddNode(child)
	if len(node.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(node.Children))
	}
	if node.Children[0].Parent != node {
		t.Error("Child parent not set correctly")
	}

	node.Toggle()
	if !node.isOpen {
		t.Error("Toggle didn't open node")
	}
	node.Toggle()
	if node.isOpen {
		t.Error("Toggle didn't close node")
	}
	node.Close()
	if node.isOpen {
		t.Error("Close didn't close node")
	}
	node.Open()
	if !node.isOpen {
		t.Error("Open didn't open node")
	}
}

func TestCreateTextWidget(t *testing.T) {
	text := &TextWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   textWidget,
			backgroundColor: "#fff",
			font:         nil,
		},
		content:   "Test text",
		fontSize:  16,
		fontColor: "#000",
	}
	if text == nil {
		t.Fatal("CreateTextWidget returned nil")
	}
	text.SetFontSize(20)
	text.SetFontColor("#123456")
}

func TestCreateImageWidget(t *testing.T) {
	imgWidget := &ImageWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   imageWidget,
			backgroundColor: "#fff",
			font:         nil,
		},
		img: nil,
	}
	if imgWidget == nil {
		t.Fatal("CreateImageWidget returned nil")
	}
	imgWidget.SetWidth(100)
	imgWidget.SetHeight(100)
}

func TestCreateScrollBarWidget(t *testing.T) {
	scrollbar := &ScrollBarWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   scrollbarWidget,
			backgroundColor: "#fff",
			font:         nil,
		},
		orientation: VerticalScrollBar,
	}
	if scrollbar == nil {
		t.Fatal("CreateScrollBarWidget returned nil")
	}
	scrollbar.SetWidth(20)
	scrollbar.SetHeight(200)
	scrollbar.SetThumbSize(50)
	scrollbar.SetThumbColor("#FF0000")
}

func TestBaseWidgetMethods(t *testing.T) {
	btn := &ButtonWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   buttonWidget,
			backgroundColor: "#fff",
			font:         nil,
			box:          box{},
			computedBox:  box{top: 0, left: 0, width: 100, height: 50},
			visible:      true,
		},
		content:   "Test",
		fontSize:  20,
		padding:   0,
		fontColor: "#000",
		selected:  false,
	}
	btn.SetWidth(100)
	btn.SetHeight(50)

	// Test ComputedBox
	box := btn.ComputedBox()
	if box == nil {
		t.Error("ComputedBox returned nil")
	}

	// Test GetRect
	top, left, width, height := btn.GetRect()
	if width != 100 || height != 50 {
		t.Errorf("Expected width=100, height=50, got width=%f, height=%f", width, height)
	}

	// Test GetTop, GetLeft, GetWidth, GetHeight
	if btn.GetTop() != top || btn.GetLeft() != left || btn.GetWidth() != width || btn.GetHeight() != height {
		t.Error("GetTop/GetLeft/GetWidth/GetHeight don't match GetRect")
	}

	// Test NeedsRepaint/SetNeedsRepaint
	btn.SetNeedsRepaint(true)
	if !btn.NeedsRepaint() {
		t.Error("NeedsRepaint should be true after SetNeedsRepaint(true)")
	}
	btn.SetNeedsRepaint(false)
	if btn.NeedsRepaint() {
		t.Error("NeedsRepaint should be false after SetNeedsRepaint(false)")
	}

	// Test SetVisible/GetVisible
	btn.SetVisible(false)
	if btn.GetVisible() {
		t.Error("GetVisible should be false after SetVisible(false)")
	}
	btn.SetVisible(true)
	if !btn.GetVisible() {
		t.Error("GetVisible should be true after SetVisible(true)")
	}
}

func TestWidgetBuffer(t *testing.T) {
	btn := &ButtonWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			widgets:      []Widget{},
			widgetType:   buttonWidget,
			backgroundColor: "#fff",
			font:         nil,
			buffer:       image.NewRGBA(image.Rect(0, 0, 100, 50)),
		},
		content:   "Test",
		fontSize:  20,
		padding:   0,
		fontColor: "#000",
		selected:  false,
	}
	btn.SetWidth(100)
	btn.SetHeight(50)

	buffer := btn.Buffer()
	if buffer == nil {
		t.Error("Buffer returned nil")
	}

	newBuffer := image.NewRGBA(image.Rect(0, 0, 200, 100))
	btn.SetBuffer(newBuffer)
	if btn.Buffer() != newBuffer {
		t.Error("SetBuffer didn't update buffer")
	}
}

func TestWidgetWindow(t *testing.T) {
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

	// Create a minimal window for testing
	window := &WindowBase{
		width:  800,
		height: 600,
	}

	btn.SetWindow(window)
	if btn.BaseWidget().GetWindow() != window {
		t.Error("GetWindow didn't return the set window")
	}
}

func TestRequestReflow(t *testing.T) {
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

	window := &WindowBase{
		width:  800,
		height: 600,
	}
	btn.SetWindow(window)

	btn.RequestReflow()
	if !window.needsReflow {
		t.Error("RequestReflow didn't set window.needsReflow")
	}
}

func TestRequestRepaint(t *testing.T) {
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
	btn.RequestRepaint()
	if !btn.NeedsRepaint() {
		t.Error("RequestRepaint didn't set needsRepaint")
	}
}