package mustard

import (
	"image"
	"testing"

	gg "github.com/danfragoso/thdwb/gg"
)

func BenchmarkCreateNewApp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CreateNewApp(nil, "bench")
	}
}

func BenchmarkCreateFrame(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CreateFrame(VerticalFrame)
	}
}

func BenchmarkWidgetSetters(b *testing.B) {
	label := &LabelWidget{
		baseWidget: baseWidget{
			widgetType:      labelWidget,
			backgroundColor: "#fff",
		},
		content:   "test",
		fontSize:  16,
		fontColor: "#000",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		label.SetWidth(100)
		label.SetHeight(50)
		label.SetFontSize(14)
		label.SetContent("new content")
		label.SetFontColor("#FF0000")
		label.SetBackgroundColor("#00FF00")
	}
}

func BenchmarkBaseWidgetVisibility(b *testing.B) {
	widget := &baseWidget{
		widgetType:      labelWidget,
		backgroundColor: "#fff",
		visible:         true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		widget.SetVisible(true)
		widget.SetVisible(false)
		_ = widget.GetVisible()
	}
}

func BenchmarkBaseWidgetRect(b *testing.B) {
	widget := &baseWidget{
		computedBox: box{top: 10, left: 20, width: 100, height: 50},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		widget.GetRect()
		widget.GetTop()
		widget.GetLeft()
		widget.GetWidth()
		widget.GetHeight()
	}
}

func BenchmarkBaseWidgetIsPointInside(b *testing.B) {
	widget := &baseWidget{
		computedBox: box{top: 10, left: 20, width: 100, height: 50},
		window: &WindowBase{
			width:  200,
			height: 200,
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		widget.IsPointInside(50, 50)
		widget.IsPointInside(5, 5)
		widget.IsPointInside(150, 150)
	}
}

func BenchmarkCalculateFlexibleWidth(b *testing.B) {
	elements := []*baseWidget{
		{box: box{width: 50}},
		{box: box{width: 30}},
		{box: box{width: 20}},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateFlexibleWidth(200, elements)
	}
}

func BenchmarkCalculateFlexibleHeight(b *testing.B) {
	elements := []*baseWidget{
		{box: box{height: 50}},
		{box: box{height: 30}},
		{box: box{height: 20}},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateFlexibleHeight(200, elements)
	}
}

func BenchmarkGetCoreWidgets(b *testing.B) {
	widgets := []Widget{
		&LabelWidget{
			baseWidget: baseWidget{
				visible:         true,
				widgetType:      labelWidget,
				backgroundColor: "#fff",
			},
			content:   "a",
			fontSize:  10,
			fontColor: "#000",
		},
		&LabelWidget{
			baseWidget: baseWidget{
				visible:         false,
				widgetType:      labelWidget,
				backgroundColor: "#fff",
			},
			content:   "b",
			fontSize:  10,
			fontColor: "#000",
		},
		&LabelWidget{
			baseWidget: baseWidget{
				visible:         true,
				widgetType:      labelWidget,
				backgroundColor: "#fff",
			},
			content:   "c",
			fontSize:  10,
			fontColor: "#000",
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getCoreWidgets(widgets)
	}
}

func BenchmarkCompositeWidget(b *testing.B) {
	widget := &LabelWidget{
		baseWidget: baseWidget{
			needsRepaint:    true,
			computedBox:     box{top: 0, left: 0, width: 10, height: 10},
			buffer:          image.NewRGBA(image.Rect(0, 0, 10, 10)),
			widgetType:      labelWidget,
			backgroundColor: "#fff",
		},
		content:   "test",
		fontSize:  10,
		fontColor: "#000",
	}
	buffer := image.NewRGBA(image.Rect(0, 0, 100, 100))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compositeWidget(buffer, widget)
	}
}

func BenchmarkGGContext(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctx := gg.NewContext(100, 100)
		ctx.SetHexColor("#ff0000")
		ctx.DrawRectangle(0, 0, 100, 100)
		ctx.Fill()
		_ = ctx.Image()
	}
}
