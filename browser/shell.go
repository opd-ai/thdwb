package main

import (
	"fmt"

	bun "github.com/danfragoso/thdwb/bun"
	gg "github.com/danfragoso/thdwb/gg"
	hotdog "github.com/danfragoso/thdwb/hotdog"
	mustard "github.com/danfragoso/thdwb/mustard"
	profiler "github.com/danfragoso/thdwb/profiler"
)

// BrowserShell manages multiple window contexts (tabs) in a single browser instance.
type BrowserShell struct {
	app            *mustard.App
	windowContexts []*hotdog.WindowContext
	activeTabIndex int
	settings       *hotdog.Settings
	buildInfo      *hotdog.BuildInfo
	profiler       *profiler.Profiler
	tabBar         *mustard.Frame
	contentArea    *mustard.Frame
}

// NewBrowserShell creates a new browser shell with tabbed interface.
func NewBrowserShell(settings *hotdog.Settings, buildInfo *hotdog.BuildInfo, profiler *profiler.Profiler) *BrowserShell {
	app := mustard.CreateNewApp("THDWB")
	window := mustard.CreateNewWindow("THDWB", settings.WindowWidth, settings.WindowHeight, settings.HiDPI)
	window.EnableContextMenus()
	app.AddWindow(window)

	shell := &BrowserShell{
		app:            app,
		windowContexts: make([]*hotdog.WindowContext, 0),
		activeTabIndex: -1,
		settings:       settings,
		buildInfo:      buildInfo,
		profiler:       profiler,
	}

	// Create the main window UI with tab bar and content area
	shell.createMainUI(window)

	// Create initial tab
	shell.NewTab(settings.Homepage)

	return shell
}

// createMainUI sets up the tab bar and content area in the main window.
func (shell *BrowserShell) createMainUI(window *mustard.Window) {
	rootFrame := mustard.CreateFrame(mustard.VerticalFrame)

	// Create tab bar
	shell.tabBar = mustard.CreateFrame(mustard.HorizontalFrame)
	shell.tabBar.SetHeight(32)
	shell.tabBar.SetBackgroundColor("#eee")
	rootFrame.AttachWidget(shell.tabBar)

	// Create content area (where tab content renders)
	shell.contentArea = mustard.CreateFrame(mustard.VerticalFrame)
	rootFrame.AttachWidget(shell.contentArea)

	window.SetRootFrame(rootFrame)
	window.Show()
}

// NewTab creates a new tab with the given URL.
func (shell *BrowserShell) NewTab(url string) *hotdog.WindowContext {
	window := shell.app.Windows()[0] // Main window
	windowContext := hotdog.NewWindowContext(shell.settings, shell.buildInfo, shell.profiler)
	windowContext.Window = window

	// Create tab UI
	tabButton := mustard.CreateButtonWidget("New Tab", nil)
	tabButton.SetPadding(8)
	shell.tabBar.AttachWidget(tabButton)

	tabIndex := len(shell.windowContexts)
	shell.windowContexts = append(shell.windowContexts, windowContext)

	// Store tab button reference for later updates
	windowContext.SetInputState("tabButton", tabButton)

	// Set up tab click handler
	tabIdx := tabIndex
	window.RegisterButton(tabButton, func() {
		shell.SwitchTab(tabIdx)
	})

	// Create tab content (navigation bar + viewport)
	shell.createTabContent(windowContext, tabIndex)

	// Switch to new tab if it's the first one
	if shell.activeTabIndex == -1 {
		shell.SwitchTab(tabIndex)
	}

	// Load the URL
	loadDocument(windowContext, url)
	windowContext.Window.SetTitle(windowContext.ActiveDocument.Title)

	// Update tab button text with page title
	if tab := windowContext.ActiveDocument.Title; tab != "" {
		tabButton.SetContent(tab)
		tabButton.RequestRepaint()
	}

	return windowContext
}

// createTabContent creates the navigation bar and viewport for a tab.
func (shell *BrowserShell) createTabContent(windowContext *hotdog.WindowContext, tabIndex int) {
	// Navigation bar
	appBar, statusLabel, menuButton, nextButton, previousButton, reloadButton, urlInput := createMainBar(windowContext.Window, windowContext)

	// Viewport
	scrollBar := mustard.CreateScrollBarWidget(mustard.VerticalScrollBar)
	scrollBar.SetTrackColor("#ccc")
	scrollBar.SetThumbColor("#aaa")
	scrollBar.SetWidth(12)

	viewPort := mustard.CreateCanvasWidget(func(canvas *mustard.CanvasWidget) {
		go func() {
			windowContext.Profiler.Start("render")
			ctxBounds := canvas.GetContext().Image().Bounds()
			drawingContext := gg.NewContext(ctxBounds.Max.X, ctxBounds.Max.Y)

			err := bun.RenderDocument(drawingContext, windowContext.ActiveDocument, shell.settings.ExperimentalLayout)
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

	// Set up navigation callbacks
	urlInput.SetReturnCallback(func() {
		loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
	})

	windowContext.Window.RegisterButton(menuButton, func() {
		windowContext.Window.AddContextMenuEntry("Home", func() {
			urlInput.SetValue("thdwb://homepage/")
			loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
		})

		windowContext.Window.AddContextMenuEntry("History", func() {
			urlInput.SetValue("thdwb://history/")
			loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
		})

		windowContext.Window.AddContextMenuEntry("About", func() {
			urlInput.SetValue("thdwb://about/")
			loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
		})

		windowContext.Window.AddContextMenuEntry("New Tab", func() {
			shell.NewTab(shell.settings.Homepage)
		})

		windowContext.Window.DrawContextMenu()
	})

	windowContext.Window.RegisterButton(reloadButton, func() {
		loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
	})

	windowContext.Window.RegisterButton(nextButton, func() {
		if len(windowContext.History.NextPages()) > 0 {
			windowContext.History.PopNext()
			urlInput.SetValue(windowContext.History.Last().String())
			loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
		}
	})

	windowContext.Window.RegisterButton(previousButton, func() {
		if windowContext.History.PageCount() > 1 {
			windowContext.History.Pop()
			urlInput.SetValue(windowContext.History.Last().String())
			loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
		}
	})
	// Store tab-specific widgets for cleanup
	windowContext.SetInputState("menuButton", menuButton)
	windowContext.SetInputState("reloadButton", reloadButton)
	windowContext.SetInputState("nextButton", nextButton)
	windowContext.SetInputState("previousButton", previousButton)
	windowContext.SetInputState("urlInput", urlInput)
	// Input handling - only process for active tab
	windowContext.Window.AttachPointerPositionEventListener(func(pointerX, pointerY float64) {
		if shell.activeTabIndex == tabIndex && viewPort.IsPointInside(pointerX, pointerY) {
			offset := float64(appBar.GetHeight())
			processPointerPositionEvent(windowContext, pointerX, pointerY-offset)
		} else {
			windowContext.ActiveDocument.SelectedElement = nil
		}
	})

	windowContext.Window.AttachScrollEventListener(func(direction int) {
		if shell.activeTabIndex != tabIndex {
			return
		}
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

	windowContext.Window.AttachClickEventListener(func(key mustard.MustardKey) {
		if shell.activeTabIndex != tabIndex {
			return
		}
		if viewPort.IsPointInside(windowContext.Window.GetCursorPosition()) {
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
					windowContext.Window.AddContextMenuEntry("Back", func() {
						previousButton.Click()
					})
					windowContext.Window.AddContextMenuEntry("Reload", func() {
						loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
					})
					windowContext.Window.AddContextMenuEntry("History", func() {
						urlInput.SetValue("thdwb://history")
						loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
					})
					windowContext.Window.AddContextMenuEntry("Home", func() {
						urlInput.SetValue("thdwb://homepage")
						loadDocumentFromUrl(windowContext, statusLabel, urlInput, viewPort)
					})

					windowContext.Window.DrawContextMenu()
				}
			}
		}
	})

	// Layout
	viewArea := mustard.CreateFrame(mustard.VerticalFrame)
	viewArea.AttachWidget(viewPort)
	viewArea.AttachWidget(scrollBar)

	tabContentFrame := mustard.CreateFrame(mustard.VerticalFrame)
	tabContentFrame.AttachWidget(appBar)
	tabContentFrame.AttachWidget(viewArea)

	// Store tab content frame in window context for switching
	windowContext.SetInputState("tabContentFrame", tabContentFrame)
	windowContext.SetInputState("tabIndex", tabIndex)
}

// SwitchTab switches to the tab at the given index.
func (shell *BrowserShell) SwitchTab(index int) {
	if index < 0 || index >= len(shell.windowContexts) {
		return
	}

	// Hide previous tab content
	if shell.activeTabIndex >= 0 && shell.activeTabIndex < len(shell.windowContexts) {
		prevContext := shell.windowContexts[shell.activeTabIndex]
		if frame, ok := prevContext.GetInputState("tabContentFrame"); ok {
			if tabFrame, ok := frame.(*mustard.Frame); ok {
				tabFrame.SetVisible(false)
			}
		}
	}

	// Show new tab content
	newContext := shell.windowContexts[index]
	if frame, ok := newContext.GetInputState("tabContentFrame"); ok {
		if tabFrame, ok := frame.(*mustard.Frame); ok {
			tabFrame.SetVisible(true)
			// Reparent to content area
			shell.contentArea.AttachWidget(tabFrame)
			newContext.Window.RequestReflow()
		}
	}

	shell.activeTabIndex = index
	fmt.Printf("Switched to tab %d\n", index)
}

// CloseTab closes the tab at the given index.
func (shell *BrowserShell) CloseTab(index int) {
	if index < 0 || index >= len(shell.windowContexts) {
		return
	}

	windowContext := shell.windowContexts[index]

	// Unregister tab-specific buttons and inputs
	if btn, ok := windowContext.GetInputState("menuButton"); ok {
		if button, ok := btn.(*mustard.ButtonWidget); ok {
			windowContext.Window.UnregisterButton(button)
		}
	}
	if btn, ok := windowContext.GetInputState("reloadButton"); ok {
		if button, ok := btn.(*mustard.ButtonWidget); ok {
			windowContext.Window.UnregisterButton(button)
		}
	}
	if btn, ok := windowContext.GetInputState("nextButton"); ok {
		if button, ok := btn.(*mustard.ButtonWidget); ok {
			windowContext.Window.UnregisterButton(button)
		}
	}
	if btn, ok := windowContext.GetInputState("previousButton"); ok {
		if button, ok := btn.(*mustard.ButtonWidget); ok {
			windowContext.Window.UnregisterButton(button)
		}
	}
	if input, ok := windowContext.GetInputState("urlInput"); ok {
		if inputWidget, ok := input.(*mustard.InputWidget); ok {
			windowContext.Window.UnregisterInput(inputWidget)
		}
	}

	// Detach tab content frame from content area
	if frame, ok := windowContext.GetInputState("tabContentFrame"); ok {
		if tabFrame, ok := frame.(*mustard.Frame); ok {
			shell.contentArea.DetachWidget(tabFrame)
		}
	}

	windowContext.Destroy()

	// Remove from slice
	shell.windowContexts = append(shell.windowContexts[:index], shell.windowContexts[index+1:]...)

	// Remove tab button
	if index < len(shell.tabBar.Widgets()) {
		shell.tabBar.RemoveWidget(index)
	}

	// Adjust active tab index
	if shell.activeTabIndex == index {
		if len(shell.windowContexts) > 0 {
			newIndex := index
			if newIndex >= len(shell.windowContexts) {
				newIndex = len(shell.windowContexts) - 1
			}
			shell.SwitchTab(newIndex)
		} else {
			shell.activeTabIndex = -1
		}
	} else if shell.activeTabIndex > index {
		shell.activeTabIndex--
	}
}

// Run starts the browser shell event loop.
func (shell *BrowserShell) Run() {
	shell.app.Run(func() {})
}
