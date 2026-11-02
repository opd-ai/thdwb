package mustard

import (
	gg "github.com/danfragoso/thdwb/gg"
)

// CreateImageWidget - Creates and returns a new Image Widget
func CreateCanvasWidget(renderer func(*CanvasWidget)) *CanvasWidget {
	var widgets []Widget

	return &CanvasWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			visible:      true,
			widgets:      widgets,

			widgetType: canvasWidget,

			backgroundColor: "#fff",
		},

		context:        gg.NewContext(0, 0),
		drawingContext: gg.NewContext(0, 0),
		renderer:       renderer,
		drawingRepaint: true,
	}
}

// SetWidth - Sets the label width
func (canvas *CanvasWidget) SetWidth(width float64) {
	canvas.box.width = width
	canvas.fixedWidth = true
	canvas.RequestReflow()
}

// SetHeight - Sets the label height
func (canvas *CanvasWidget) SetHeight(height float64) {
	canvas.box.height = height
	canvas.fixedHeight = true
	canvas.RequestReflow()
}

func (canvas *CanvasWidget) EnableScrolling() {
	canvas.scrollable = true
}

func (canvas *CanvasWidget) DisableScrolling() {
	canvas.scrollable = false
	canvas.SetOffset(0)
}

func (canvas *CanvasWidget) SetOffset(offset int) {
	if offset < 0 {
		offset = 0
	}
	if maxOffset, known := canvas.maxOffset(); known && offset > maxOffset {
		offset = maxOffset
	}
	if canvas.offset == offset {
		return
	}
	canvas.offset = offset
	canvas.needsRepaint = true
}

func (canvas *CanvasWidget) GetOffset() int {
	return canvas.offset
}

func (canvas *CanvasWidget) GetContext() *gg.Context {
	if canvas.drawingContext == nil {
		return nil
	}
	ctx, ok := canvas.drawingContext.(*gg.Context)
	if !ok {
		return nil
	}
	return ctx
}

func (canvas *CanvasWidget) SetContext(ctx *gg.Context) {
	canvas.drawingContext = ctx
}

func (cavas *CanvasWidget) SetDrawingRepaint(repaint bool) {
	cavas.drawingRepaint = repaint
}

func (ctx *CanvasWidget) draw() {
	if ctx.window == nil || ctx.window.context == nil {
		return
	}
	windowContext, ok := ctx.window.context.(*gg.Context)
	if !ok {
		return
	}
	canvasContext, ok := ctx.context.(*gg.Context)
	if !ok {
		return
	}
	top, left, width, height := ctx.computedBox.GetCoords()
	ctx.ensureContexts(int(width), int(height))

	if ctx.drawingRepaint {
		ctx.renderer(ctx)
		ctx.drawingRepaint = false
	}

	canvasContext.SetHexColor(ctx.backgroundColor)
	canvasContext.Clear()
	if drawingCtx, ok := ctx.drawingContext.(*gg.Context); ok {
		canvasContext.DrawImage(drawingCtx.Image(), 0, -ctx.offset)
	}
	windowContext.DrawImage(canvasContext.Image(), int(left), int(top))
	copyWidgetToBuffer(ctx, canvasContext.Image())
}

func (canvas *CanvasWidget) maxOffset() (int, bool) {
	if canvas.drawingContext == nil {
		return 0, false
	}
	drawingCtx, ok := canvas.drawingContext.(*gg.Context)
	if !ok {
		return 0, false
	}
	if drawingCtx.Height() == 0 {
		return 0, false
	}
	viewportHeight := int(canvas.computedBox.height)
	if viewportHeight <= 0 && canvas.context != nil {
		if ctx, ok := canvas.context.(*gg.Context); ok {
			viewportHeight = ctx.Height()
		}
	}
	if viewportHeight <= 0 {
		return 0, false
	}
	if drawingCtx.Height() <= viewportHeight {
		return 0, true
	}
	return drawingCtx.Height() - viewportHeight, true
}

func (canvas *CanvasWidget) ensureContexts(width, height int) {
	if canvas.context == nil {
		return
	}
	context, ok := canvas.context.(*gg.Context)
	if !ok {
		return
	}
	currentSize := context.Image().Bounds().Size()
	if currentSize.X != width || currentSize.Y != height {
		canvas.context = gg.NewContext(width, height)
		canvas.drawingContext = gg.NewContext(width, 12000)
		canvas.drawingRepaint = true
	}
	canvas.SetOffset(canvas.offset)
}

func createCtxScrollBar(ctx *CanvasWidget) {
	if ctx.window == nil || ctx.window.context == nil {
		return
	}
	top, _, width, height := ctx.computedBox.GetCoords()
	context, ok := ctx.window.context.(*gg.Context)
	if !ok {
		return
	}

	// Scroll Track
	context.SetHexColor("#c1c1c1")
	context.DrawRectangle(float64(width-12), float64(top), 12, float64(height))
	context.Fill()

	// Scroll Arrow
	context.SetHexColor("#ff0000")
	context.DrawRectangle(float64(width-12), 30, 10, 10)
	context.Fill()

	// Scroll Thumb
	context.SetHexColor("#565656")
	context.DrawRectangle(float64(width-12), float64(top), 12, 200)
	context.Fill()
}
