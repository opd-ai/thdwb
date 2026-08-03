package main

import (
	"flag"
	"runtime"

	bun "github.com/danfragoso/thdwb/bun"
	gg "github.com/danfragoso/thdwb/gg"
	hotdog "github.com/danfragoso/thdwb/hotdog"
	mustard "github.com/danfragoso/thdwb/mustard"
	profiler "github.com/danfragoso/thdwb/profiler"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func main() {
	runtime.LockOSThread()
	glfw.Init()
	gl.Init()

	mustard.SetGLFWHints()

	defaultPath := "./settings.json"
	settingsPath := flag.String("settings", defaultPath, "This flag sets the location for the browser settings file.")
	flag.Parse()

	settings := hotdog.LoadSettings(*settingsPath)

	profilerInstance := profiler.CreateProfiler()
	buildInfo := &hotdog.BuildInfo{
		GitRevision: gitRevision,
		GitBranch:   gitBranch,
		HostInfo:    hostInfo,
		BuildTime:   buildTime,
	}

	// Create a new window context for this tab/window
	windowContext := hotdog.NewWindowContext(settings, buildInfo, profilerInstance)

	app := mustard.CreateNewApp("THDWB")
	window := mustard.CreateNewWindow("THDWB", settings.WindowWidth, settings.WindowHeight, settings.HiDPI)
	window.EnableContextMenus()
	windowContext.Window = window

	rootFrame := mustard.CreateFrame(mustard.HorizontalFrame)

	appBar, statusLabel, menuButton, nextButton, previousButton, reloadButton, urlInput := createMainBar(window, windowContext)
	rootFrame.AttachWidget(appBar)

	loadDocument(windowContext, settings.Homepage)
	urlInput.SetValue(windowContext.ActiveDocument.URL.String())

	scrollBar := mustard.CreateScrollBarWidget(mustard.VerticalScrollBar)
	scrollBar.SetTrackColor("#ccc")
	scrollBar.SetThumbColor("#aaa")
	scrollBar.SetWidth(12)

	viewPort := mustard.CreateCanvasWidget(func(canvas *mustard.CanvasWidget) {
		go func() {
			windowContext.Profiler.Start("render")
			ctxBounds := canvas.GetContext().Image().Bounds()
			drawingContext := gg.NewContext(ctxBounds.Max.X, ctxBounds.Max.Y)

			err := bun.RenderDocument(drawingContext, windowContext.ActiveDocument, settings.ExperimentalLayout)
			if err != nil {
				hotdog.Log("render", "Can't render page: "+err.Error())
			}

			canvas.SetContext(drawingContext)
			canvas.RequestRepaint()
			windowContext.Profiler.Stop("render")

			statusLabel.SetContent(createStatusLabel(windowContext.Profiler))
			statusLabel.RequestRepaint()
			canvas.RequestRepaint()

			scrollBar.SetScrollerOffset(0)

			body, err := windowContext.ActiveDocument.DOM.FindChildByName("body")
			if err != nil {
				hotdog.Log("render", "can't find body element: "+err.Error())
				return
			}

			scrollBar.SetScrollerSize(body.RenderBox.Height)
			scrollBar.RequestReflow()
		}()
	})

	windowContext.Viewport = viewPort
	windowContext.StatusLabel = statusLabel

	urlInput.SetReturnCallback(func() {
		loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
	})

	window.RegisterButton(menuButton, func() {
		window.AddContextMenuEntry("Home", func() {
			urlInput.SetValue("thdwb://homepage/")
			loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
		})

		window.AddContextMenuEntry("History", func() {
			urlInput.SetValue("thdwb://history/")
			loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
		})

		window.AddContextMenuEntry("About", func() {
			urlInput.SetValue("thdwb://about/")
			loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
		})

		if windowContext.ActiveDocument.DebugFlag {
			window.AddContextMenuEntry("Disable debug mode", func() {
				windowContext.Window.RemoveStaticOverlay("debugOverlay")
				windowContext.ActiveDocument.DebugFlag = false

				if windowContext.ActiveDocument.DebugWindow != nil {
					app.DestroyWindow(windowContext.ActiveDocument.DebugWindow)
					windowContext.ActiveDocument.DebugWindow = nil
					windowContext.ActiveDocument.DebugTree = nil
				}
			})
		} else {
			window.AddContextMenuEntry("Enable debug mode", func() {
				windowContext.ActiveDocument.DebugFlag = true
			})
		}

		if windowContext.ActiveDocument.DebugFlag {
			if windowContext.ActiveDocument.DebugWindow != nil {
				window.AddContextMenuEntry("Hide Tree", func() {
					app.DestroyWindow(windowContext.ActiveDocument.DebugWindow)
					windowContext.ActiveDocument.DebugWindow = nil
					windowContext.ActiveDocument.DebugTree = nil
				})
			} else {
				window.AddContextMenuEntry("Show Tree", func() {
					tree := mustard.CreateTreeWidget()

					windowContext.ActiveDocument.DebugWindow = mustard.CreateNewWindow("HTML tree view", 600, 800, true)
					windowContext.ActiveDocument.DebugTree = tree

					rFrame := mustard.CreateFrame(mustard.HorizontalFrame)
					tree.SetFontSize(14)
					rFrame.AttachWidget(tree)

					windowContext.ActiveDocument.DebugWindow.RegisterTree(tree)
					windowContext.ActiveDocument.DebugWindow.SetRootFrame(rFrame)
					windowContext.ActiveDocument.DebugWindow.Show()

					app.AddWindow(windowContext.ActiveDocument.DebugWindow)

					treeNodeDOM := treeNodeFromDOM(windowContext.ActiveDocument.DOM)
					tree.SetSelectCallback(func(selectedNode *mustard.TreeWidgetNode) {
						if windowContext.ActiveDocument.DebugFlag {
							child, _ := windowContext.ActiveDocument.DOM.FindByXPath(selectedNode.Value)
							windowContext.ActiveDocument.SelectedElement = child
							showDebugOverlay(windowContext)
						}
					})

					tree.RemoveNodes()
					tree.AddNode(treeNodeDOM)
					tree.RequestRepaint()
				})
			}
		}

		window.DrawContextMenu()
	})

	window.RegisterButton(reloadButton, func() {
		loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
	})

	window.RegisterButton(nextButton, func() {
		if len(windowContext.History.NextPages()) > 0 {
			windowContext.History.PopNext()
			urlInput.SetValue(windowContext.History.Last().String())
			loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
		}
	})

	window.RegisterButton(previousButton, func() {
		if windowContext.History.PageCount() > 1 {
			windowContext.History.Pop()
			urlInput.SetValue(windowContext.History.Last().String())
			loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
		}
	})

	window.AttachPointerPositionEventListener(func(pointerX, pointerY float64) {
		if viewPort.IsPointInside(pointerX, pointerY) {
			offset := float64(appBar.GetHeight())
			processPointerPositionEvent(windowContext, pointerX, pointerY-offset)
		} else {
			windowContext.ActiveDocument.SelectedElement = nil
		}
	})

	window.AttachScrollEventListener(func(direction int) {
		scrollStep := 20

		body, err := windowContext.ActiveDocument.DOM.FindChildByName("body")
		if err != nil {
			hotdog.Log("render", "Can't find body element: "+err.Error())
			return
		}

		if direction > 0 {
			if viewPort.GetOffset() < 0 {
				viewPort.SetOffset(viewPort.GetOffset() + scrollStep)
			}
		} else {
			documentOffset := viewPort.GetOffset() + int(body.RenderBox.Height)

			if float64(documentOffset) >= viewPort.GetHeight() {
				viewPort.SetOffset(viewPort.GetOffset() - scrollStep)
			}
		}

		scrollBar.SetScrollerOffset(float64(viewPort.GetOffset()))
		scrollBar.SetScrollerSize(body.RenderBox.Height)
		scrollBar.RequestReflow()

		windowContext.Viewport.SetDrawingRepaint(false)
		viewPort.RequestRepaint()

		windowContext.Window.RemoveStaticOverlay("debugOverlay")
	})

	window.AttachClickEventListener(func(key mustard.MustardKey) {
		if viewPort.IsPointInside(window.GetCursorPosition()) {
			if key == mustard.MouseLeft {
				if windowContext.ActiveDocument.SelectedElement != nil {
					if windowContext.ActiveDocument.SelectedElement.Element == "a" {
						href := windowContext.ActiveDocument.SelectedElement.Attr("href")
						urlInput.SetValue(href)
						loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
					}
				}
			} else {
				if windowContext.ActiveDocument.SelectedElement != nil {
					window.AddContextMenuEntry("Back", func() {
						previousButton.Click()
					})
					window.AddContextMenuEntry("Reload", func() {
						loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
					})
					window.AddContextMenuEntry("History", func() {
						urlInput.SetValue("thdwb://history")
						loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
					})
					window.AddContextMenuEntry("Home", func() {
						urlInput.SetValue("thdwb://homepage")
						loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
					})

					window.DrawContextMenu()
				}
			}
		}
	})

	viewArea := mustard.CreateFrame(mustard.VerticalFrame)
	viewArea.AttachWidget(viewPort)
	viewArea.AttachWidget(scrollBar)

	rootFrame.AttachWidget(viewArea)

	window.SetRootFrame(rootFrame)
	window.Show()

	app.AddWindow(window)
	app.Run(func() {})
}
