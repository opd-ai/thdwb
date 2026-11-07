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

	// Skip painting zero-dimension boxes to avoid degenerate draw calls.
	if node.RenderBox == nil || node.RenderBox.Width <= 0 || node.RenderBox.Height <= 0 {
		return
	}

	// Default to white background if no style is set.
	bg := &hotdog.ColorRGBA{R: 1, G: 1, B: 1, A: 1}
	if node.Style != nil && node.Style.BackgroundColor != nil {
		bg = node.Style.BackgroundColor
	}

	ctx.SetRGBA(bg.R, bg.G, bg.B, bg.A)
	ctx.DrawRectangle(node.RenderBox.Left, node.RenderBox.Top, node.RenderBox.Width, node.RenderBox.Height)
	ctx.Fill()

	paintTextWithinBounds(ctx, node, node.Content, 1, 1.5)
}

// resolveBlockTop computes the vertical origin of a block at index childIdx
// within its parent: it stacks below the previous sibling when one exists,
// otherwise it aligns with the parent's top. A nil parent (the layout root on
// body-less/frameset documents) or a missing sibling box yields 0 rather than
// dereferencing nil.
func resolveBlockTop(node *hotdog.NodeDOM, childIdx int) float64 {
	if node.Parent == nil || node.Parent.RenderBox == nil {
		return 0
	}
	if childIdx > 0 && childIdx <= len(node.Parent.Children) {
		if prev := node.Parent.Children[childIdx-1]; prev != nil && prev.RenderBox != nil {
			return prev.RenderBox.Top + prev.RenderBox.Height
		}
	}
	return node.Parent.RenderBox.Top
}

func calculateBlockLayout(ctx *gg.Context, node *hotdog.NodeDOM, childIdx int) {
	// Text nodes are inline by nature, but if they're rendered as block, handle them
	if node.Type == hotdog.NodeTypeText {
		textContent := node.GetTextContent()
		ctx.SetFont(getFontFace(node.Style.FontWeight), node.Style.FontSize)
		if node.Style.Width == 0 {
			parentWidth := float64(ctx.Width())
			if node.Parent != nil && node.Parent.RenderBox != nil && node.Parent.RenderBox.Width > 0 {
				parentWidth = node.Parent.RenderBox.Width
			}
			node.RenderBox.Width = parentWidth
		}
		if node.Style.Height == 0 {
			node.RenderBox.Height = ctx.MeasureStringWrapped(textContent, node.RenderBox.Width, 1.5) + 2 + ctx.FontHeight()*.5
		}

		node.RenderBox.Top = resolveBlockTop(node, childIdx)
		return
	}

	if node.Style.Width == 0 {
		parentWidth := float64(ctx.Width())
		if node.Parent != nil && node.Parent.RenderBox != nil && node.Parent.RenderBox.Width > 0 {
			parentWidth = node.Parent.RenderBox.Width
		}
		node.RenderBox.Width = parentWidth
	}

	if node.Style.Height == 0 {
		ctx.SetFont(getFontFace(node.Style.FontWeight), node.Style.FontSize)
		node.RenderBox.Height = ctx.MeasureStringWrapped(node.Content, node.RenderBox.Width, 1.5) + 2 + ctx.FontHeight()*.5
	}

	node.RenderBox.Top = resolveBlockTop(node, childIdx)
}
