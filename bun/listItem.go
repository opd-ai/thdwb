package bun

import (
	gg "github.com/danfragoso/thdwb/gg"
	hotdog "github.com/danfragoso/thdwb/hotdog"
)

func paintListItemElement(ctx *gg.Context, node *hotdog.NodeDOM) {
	if node.RenderBox == nil || node.Style == nil || node.Style.BackgroundColor == nil {
		return
	}

	ctx.DrawRectangle(node.RenderBox.Left, node.RenderBox.Top, node.RenderBox.Width, node.RenderBox.Height)
	ctx.SetRGBA(node.Style.BackgroundColor.R, node.Style.BackgroundColor.G, node.Style.BackgroundColor.B, node.Style.BackgroundColor.A)
	ctx.Fill()

	ctx.DrawCircle(node.RenderBox.Left-15, node.RenderBox.Top+node.Style.FontSize/2, 3)
	color := node.Style.Color
	if color == nil {
		color = &hotdog.ColorRGBA{R: 0, G: 0, B: 0, A: 1}
	}
	ctx.SetRGBA(color.R, color.G, color.B, color.A)
	ctx.SetFont(getFontFace(node.Style.FontWeight), node.Style.FontSize)
	ctx.DrawStringWrapped(node.Content, node.RenderBox.Left, node.RenderBox.Top+1, 0, 0, node.RenderBox.Width, 1.5, gg.AlignLeft)
	ctx.Fill()
}

func calculateListItemLayout(ctx *gg.Context, node *hotdog.NodeDOM, childIdx int) {
	if node.Style == nil || node.RenderBox == nil || node.Parent == nil || node.Parent.RenderBox == nil {
		return
	}

	if node.Style.Width == 0 {
		if w := node.Parent.RenderBox.Width - 30; w > 0 {
			node.RenderBox.Width = w
		} else {
			node.RenderBox.Width = 0
		}
	}

	if node.Style.Height == 0 && len(node.Content) > 0 {
		ctx.SetFont(getFontFace(node.Style.FontWeight), node.Style.FontSize)
		node.RenderBox.Height = ctx.MeasureStringWrapped(node.Content, node.RenderBox.Width, 1.5) + 2 + ctx.FontHeight()*.5
	}

	node.RenderBox.Top = resolveBlockTop(node, childIdx)

	node.RenderBox.Left = 30
}
