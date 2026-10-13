package bun

import (
	"fmt"

	gg "github.com/danfragoso/thdwb/gg"
	hotdog "github.com/danfragoso/thdwb/hotdog"
)

func RenderDocument(ctx *gg.Context, document *hotdog.Document, experimentalLayout bool) error {
	if !experimentalLayout {
		body, err := document.DOM.FindChildByName("body")
		if err != nil || body == nil {
			// No explicit body element, use the document root (html) as fallback
			body = document.DOM
		}

		document.DOM.RenderBox.Width = float64(ctx.Width())
		document.DOM.RenderBox.Height = float64(ctx.Height())

		ctx.SetRGB(1, 1, 1)
		ctx.Clear()

		layoutDOM(ctx, body, 0)
		paintTree(ctx, body)
	} else {
		html, err := document.DOM.FindChildByName("html")
		if err != nil {
			return err
		}

		// Create flex nodes for the original DOM
		createFlexNodes(html)

		renderTree := createRenderTree(html)
		if renderTree == nil {
			return nil
		}
		renderTree.RenderBox.Width = float64(ctx.Width())
		renderTree.RenderBox.Height = float64(ctx.Height())

		ctx.SetRGB(1, 1, 1)
		ctx.Clear()

		layoutNode(ctx, renderTree)
		paintTree(ctx, renderTree)

		renderTree.Print(0)
	}

	return nil
}

// CanvasContext exposes the drawing context used by a canvas renderer.
type CanvasContext interface {
	GetContext() *gg.Context
}

// RenderDocumentToCanvas draws a document into a mustard-compatible canvas context.
func RenderDocumentToCanvas(canvas CanvasContext, document *hotdog.Document, experimentalLayout bool) error {
	if canvas == nil {
		return fmt.Errorf("canvas has no drawing context")
	}
	context := canvas.GetContext()
	if context == nil {
		return fmt.Errorf("canvas has no drawing context")
	}
	return RenderDocument(context, document, experimentalLayout)
}

func getNodeContent(NodeDOM *hotdog.NodeDOM) string {
	return NodeDOM.Content
}

func getElementName(NodeDOM *hotdog.NodeDOM) string {
	return NodeDOM.Element
}

func getNodeChildren(NodeDOM *hotdog.NodeDOM) []*hotdog.NodeDOM {
	return NodeDOM.Children
}

func walkDOM(TreeDOM *hotdog.NodeDOM, d string) {
	fmt.Println(d, getElementName(TreeDOM))
	nodeChildren := getNodeChildren(TreeDOM)

	for i := 0; i < len(nodeChildren); i++ {
		walkDOM(nodeChildren[i], d+"-")
	}
}

func layoutDOM(ctx *gg.Context, node *hotdog.NodeDOM, childIdx int) {
	nodeChildren := getNodeChildren(node)

	node.RenderBox = &hotdog.RenderBox{}
	calculateNode(ctx, node, childIdx)

	for i := 0; i < len(nodeChildren); i++ {
		layoutDOM(ctx, nodeChildren[i], i)
		node.RenderBox.Height += nodeChildren[i].RenderBox.Height
	}
}

func paintNode(ctx *gg.Context, node *hotdog.NodeDOM) {
	switch node.Style.Display {
	case "block":
		paintBlockElement(ctx, node)
	case "inline":
		paintInlineElement(ctx, node)
	case "list-item":
		paintListItemElement(ctx, node)
	}
}

func paintTree(ctx *gg.Context, node *hotdog.NodeDOM) {
	if node == nil {
		return
	}
	paintNode(ctx, node)
	for _, child := range node.Children {
		paintTree(ctx, child)
	}
	paintFocusRing(ctx, node)
}

func paintFocusRing(ctx *gg.Context, node *hotdog.NodeDOM) {
	if node == nil || node.RenderBox == nil || node.RenderBox.Width <= 0 || node.RenderBox.Height <= 0 {
		return
	}
	source := node
	if node.RenderBox.Node != nil {
		source = node.RenderBox.Node
	}
	if source.WindowCtx == nil || source.WindowCtx.GetFocusedElement() != source {
		return
	}
	box := node.RenderBox
	ctx.Push()
	ctx.SetHexColor("#1a73e8")
	ctx.SetLineWidth(2)
	ctx.DrawRectangle(box.Left-2, box.Top-2, box.Width+4, box.Height+4)
	ctx.Stroke()
	ctx.Pop()
}

func calculateNode(ctx *gg.Context, node *hotdog.NodeDOM, postion int) {
	switch node.Style.Display {
	case "block":
		calculateBlockLayout(ctx, node, postion)
	case "inline":
		calculateInlineLayout(ctx, node, postion)
	case "list-item":
		calculateListItemLayout(ctx, node, postion)
	}
}

func GetPageTitle(TreeDOM *hotdog.NodeDOM) string {
	nodeChildren := getNodeChildren(TreeDOM)
	pageTitle := "Sem Titulo"

	if getElementName(TreeDOM) == "title" {
		return getNodeContent(TreeDOM)
	}

	for i := 0; i < len(nodeChildren); i++ {
		nPageTitle := GetPageTitle(nodeChildren[i])

		if nPageTitle != "Sem Titulo" {
			pageTitle = nPageTitle
		}
	}

	return pageTitle
}
