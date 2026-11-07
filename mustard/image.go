package mustard

import (
	"image"
	"log"

	gg "github.com/danfragoso/thdwb/gg"
)

// CreateImageWidget - Creates and returns a new Image Widget
func CreateImageWidget(path []byte) *ImageWidget {
	var widgets []Widget

	img, err := gg.LoadAsset(path)
	if err != nil {
		// An undecodable image is a content error, not a programming error:
		// log.Fatal would exit the whole browser. Degrade to a 1x1
		// transparent placeholder so draw() keeps working (audit C-18).
		log.Printf("mustard: undecodable image asset: %v", err)
		img = image.NewRGBA(image.Rect(0, 0, 1, 1))
	}

	return &ImageWidget{
		baseWidget: baseWidget{
			needsRepaint: true,
			visible:      true,
			widgets:      widgets,

			widgetType: imageWidget,

			backgroundColor: "#fff",
		},

		// path: path,
		img: img,
	}
}

// SetWidth - Sets the label width
func (label *ImageWidget) SetWidth(width float64) {
	label.box.width = width
	label.fixedWidth = true
	label.RequestReflow()
}

// SetHeight - Sets the label height
func (label *ImageWidget) SetHeight(height float64) {
	label.box.height = height
	label.fixedHeight = true
	label.RequestReflow()
}

func (im *ImageWidget) draw() {
	if im.window == nil || im.window.context == nil {
		return
	}
	context, ok := im.window.context.(*gg.Context)
	if !ok {
		return
	}
	top, left, _, _ := im.computedBox.GetCoords()
	context.DrawImage(im.img, int(left)+15, int(top)+3)

	copyWidgetToBuffer(im, context.Image())
}
