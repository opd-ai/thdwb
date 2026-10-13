package bun

import (
	gg "github.com/danfragoso/thdwb/gg"
	hotdog "github.com/danfragoso/thdwb/hotdog"
)

func paintBlockElement(ctx *gg.Context, node *hotdog.NodeDOM) {
	// Text nodes don't have backgrounds or borders, just render their content
	if node.Type == hotdog.NodeTypeText {
		paintTextWithinBounds(ctx, node, node.GetTextContent(), 1, 1.5)
		return
	}

	ctx.DrawRectangle(node.RenderBox.Left, node.RenderBox.Top, node.RenderBox.Width, node.RenderBox.Height)
	ctx.SetRGBA(node.Style.BackgroundColor.R, node.Style.BackgroundColor.G, node.Style.BackgroundColor.B, node.Style.BackgroundColor.A)
	ctx.Fill()

	paintTextWithinBounds(ctx, node, node.Content, 1, 1.5)
}

func calculateBlockLayout(ctx *gg.Context, node *hotdog.NodeDOM, childIdx int) {
	// Text nodes are inline by nature, but if they're rendered as block, handle them
	if node.Type == hotdog.NodeTypeText {
		textContent := node.GetTextContent()
		ctx.SetFont(sansSerif[node.Style.FontWeight], node.Style.FontSize)
		if node.Style.Width == 0 {
			node.RenderBox.Width = node.Parent.RenderBox.Width
		}
		if node.Style.Height == 0 {
			node.RenderBox.Height = ctx.MeasureStringWrapped(textContent, node.RenderBox.Width, 1.5) + 2 + ctx.FontHeight()*.5
		}

		if childIdx > 0 {
			prev := node.Parent.Children[childIdx-1]
			node.RenderBox.Top = prev.RenderBox.Top + prev.RenderBox.Height
		} else {
			node.RenderBox.Top = node.Parent.RenderBox.Top
		}
		return
	}

	if node.Style.Width == 0 {
		node.RenderBox.Width = node.Parent.RenderBox.Width
	}

	if node.Style.Height == 0 {
		ctx.SetFont(sansSerif[node.Style.FontWeight], node.Style.FontSize)
		node.RenderBox.Height = ctx.MeasureStringWrapped(node.Content, node.RenderBox.Width, 1.5) + 2 + ctx.FontHeight()*.5
	}

	if childIdx > 0 {
		prev := node.Parent.Children[childIdx-1]

		node.RenderBox.Top = prev.RenderBox.Top + prev.RenderBox.Height
	} else {
		node.RenderBox.Top = node.Parent.RenderBox.Top
	}
}
