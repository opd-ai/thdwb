package main

import (
	"fmt"
	"image"
	"strings"

	"github.com/danfragoso/thdwb/bun"

	gg "github.com/danfragoso/thdwb/gg"
	hotdog "github.com/danfragoso/thdwb/hotdog"
	ketchup "github.com/danfragoso/thdwb/ketchup"
	mustard "github.com/danfragoso/thdwb/mustard"
	profiler "github.com/danfragoso/thdwb/profiler"
	sauce "github.com/danfragoso/thdwb/sauce"
)

func loadDocument(windowContext *hotdog.WindowContext, link string) {
	URL := sauce.ParseURL(link)

	if URL.Scheme == "" && URL.Host == "" {
		if !strings.HasPrefix(URL.Path, "/") {
			URL.Path = "/" + URL.Path
		}

		if strings.HasSuffix(windowContext.ActiveDocument.URL.String(), "/") {
			URL.Path = strings.TrimSuffix(windowContext.ActiveDocument.URL.Path, "/") + URL.Path
		}

		URL = sauce.ParseURL(windowContext.ActiveDocument.URL.Scheme + "://" + windowContext.ActiveDocument.URL.Host + URL.Path)
	}

	resource := sauce.GetResource(URL, windowContext)
	rawDocument := string(resource.Body)

	switch strings.Split(resource.ContentType, ";")[0] {
	case "text/plain", "text/xml", "application/json":
		windowContext.ActiveDocument = ketchup.ParsePlainText(rawDocument)
	default:
		windowContext.ActiveDocument = ketchup.ParseHTMLDocument(rawDocument)
	}

	windowContext.ActiveDocument.URL = resource.URL
	windowContext.ActiveDocument.ContentType = resource.ContentType

	windowContext.ActiveDocument.Title = bun.GetPageTitle(windowContext.ActiveDocument.DOM) + " - THDWB"
	windowContext.Window.SetTitle(windowContext.ActiveDocument.Title)

	windowContext.Window.RemoveStaticOverlay("debugOverlay")

	if windowContext.History.PageCount() == 0 || windowContext.History.Last().String() != resource.URL.String() {
		windowContext.History.Push(resource.URL)
	}
}

func loadDocumentFromUrl(windowContext *hotdog.WindowContext, statusLabel *mustard.LabelWidget, urlInput *mustard.InputWidget, viewPort *mustard.CanvasWidget) {
	statusLabel.SetContent("Loading: " + urlInput.GetValue())
	statusLabel.RequestRepaint()

	loadDocument(windowContext, urlInput.GetValue())
	viewPort.SetOffset(0)
	viewPort.SetDrawingRepaint(true)
	viewPort.RequestRepaint()
	urlInput.SetValue(windowContext.ActiveDocument.URL.String())
}

func treeNodeFromDOM(node *hotdog.NodeDOM) *mustard.TreeWidgetNode {
	nodeString := fmt.Sprintf(node.Element)
	xPath := node.GetXPath()
	treeNode := mustard.CreateTreeWidgetNode(nodeString, xPath)
	treeNode.Open()
	for _, childNode := range node.Children {
		treeNode.AddNode(treeNodeFromDOM((childNode)))
	}
	return treeNode
}

func createStatusLabel(perf *profiler.Profiler) string {
	return "Loaded; " +
		"Render took: " + perf.GetProfile("render").GetElapsedTime().String() + "; "
}

func processPointerPositionEvent(windowContext *hotdog.WindowContext, x, y float64) {
	y -= float64(windowContext.Viewport.GetOffset())
	selectedElement := windowContext.ActiveDocument.DOM.CalcPointIntersection(x, y)

	if windowContext.ActiveDocument.SelectedElement != selectedElement {
		windowContext.ActiveDocument.SelectedElement = selectedElement

		if windowContext.ActiveDocument.SelectedElement != nil && windowContext.ActiveDocument.SelectedElement.Element == "a" {
			windowContext.Window.SetCursor(mustard.PointerCursor)
			windowContext.StatusLabel.SetContent(windowContext.ActiveDocument.SelectedElement.Attr("href"))
		} else {
			windowContext.Window.SetCursor(mustard.DefaultCursor)
			windowContext.StatusLabel.SetContent(createStatusLabel(windowContext.Profiler))
		}

		if windowContext.ActiveDocument.DebugFlag &&
			windowContext.ActiveDocument.SelectedElement != nil &&
			windowContext.ActiveDocument.SelectedElement.Element != "html" {

			if windowContext.ActiveDocument.DebugWindow != nil {
				windowContext.ActiveDocument.DebugTree.SelectNodeByValue(windowContext.ActiveDocument.SelectedElement.GetXPath())
				windowContext.ActiveDocument.DebugTree.RequestRepaint()
			}

			showDebugOverlay(windowContext)
		}

		windowContext.StatusLabel.RequestRepaint()
	}
}

func printNodeDebug(node *hotdog.NodeDOM) {
	rect := fmt.Sprintf("{%.2f, %.2f, %.2f, %.2f}", node.RenderBox.Top, node.RenderBox.Left, node.RenderBox.Width, node.RenderBox.Height)
	fmt.Printf("%s [\n %s\n]\n\n", node.Element, rect)
}

func showDebugOverlay(windowContext *hotdog.WindowContext) {
	windowContext.Window.RemoveStaticOverlay("debugOverlay")

	debugEl := windowContext.ActiveDocument.SelectedElement
	top, left, _, height := debugEl.RenderBox.GetRect()
	ctx := gg.NewContext(int(windowContext.ActiveDocument.DOM.RenderBox.Width), int(height+20))
	paintDebugRect(ctx, debugEl)

	overlay := mustard.CreateStaticOverlay("debugOverlay", ctx, image.Point{
		int(left), int(top+windowContext.Viewport.GetTop()) + windowContext.Viewport.GetOffset(),
	})

	windowContext.Window.AddStaticOverlay(overlay)
}

func paintDebugRect(ctx *gg.Context, node *hotdog.NodeDOM) {
	debugString := node.Element + " {" + fmt.Sprint(node.RenderBox.Top, node.RenderBox.Left, node.RenderBox.Width, node.RenderBox.Height) + "}"
	ctx.DrawRectangle(0, 0, node.RenderBox.Width, node.RenderBox.Height)
	ctx.SetRGBA(.2, .8, .4, .3)
	ctx.Fill()

	w, h := ctx.MeasureString(debugString)

	if node.RenderBox.Width < w {
		ctx.DrawRectangle(0, node.RenderBox.Height, w+4, h+4)
		ctx.SetRGB(1, 1, 0)
		ctx.Fill()

		ctx.SetRGB(0, 0, 0)
		ctx.DrawString(debugString, 2, node.RenderBox.Height+h)
		ctx.Fill()
	} else {
		ctx.DrawRectangle(node.RenderBox.Width-w-2, node.RenderBox.Height, w+4, h+4)
		ctx.SetRGB(1, 1, 0)
		ctx.Fill()

		ctx.SetRGB(0, 0, 0)
		ctx.DrawString(debugString, node.RenderBox.Width-w, node.RenderBox.Height+h)
		ctx.Fill()
	}
}
