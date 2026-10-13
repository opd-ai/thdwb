package mustard

import "fmt"

func CreateNewApp(name string) *App {
	return &App{name: name}
}

func (app *App) Run(callback func()) {
	for {
		for _, window := range app.windows {
			if app.tabbedMode && window != app.activeWindow {
				continue
			}
			if window.visible && !window.glw.ShouldClose() {
				window.processFrame()
			}
		}

		callback()
	}
}

func (app *App) AddWindow(window *Window) {
	app.windows = append(app.windows, window)
	if app.tabbedMode {
		window.setActive(false)
	}

	if window.rootFrame != nil {
		setWidgetWindow(window.rootFrame, window)
	}
}

// ActivateWindow opts the app into tab-style processing and activates one managed window.
func (app *App) ActivateWindow(window *Window) error {
	if window == nil || !app.containsWindow(window) {
		return fmt.Errorf("window is not managed by this app")
	}
	app.tabbedMode = true
	app.activeWindow = window
	for _, appWindow := range app.windows {
		appWindow.setActive(appWindow == window)
	}
	return nil
}

func (app *App) containsWindow(window *Window) bool {
	for _, candidate := range app.windows {
		if candidate == window {
			return true
		}
	}
	return false
}

// Windows returns the list of windows managed by this app.
func (app *App) Windows() []*Window {
	return app.windows
}

func (app *App) DestroyWindow(window *Window) {
	wasActive := app.activeWindow == window
	var nWindows []*Window

	for _, appWindow := range app.windows {
		if appWindow != window {
			nWindows = append(nWindows, appWindow)
		}
	}

	app.windows = nWindows
	window.setActive(false)
	window.destroy()
	if wasActive {
		app.activeWindow = nil
		if len(app.windows) > 0 {
			_ = app.ActivateWindow(app.windows[0])
		}
	}
}

func setWidgetWindow(widget Widget, window *Window) {
	widget.SetWindow(window)

	for _, childWidget := range widget.Widgets() {
		setWidgetWindow(childWidget, window)
	}
}

func redrawWidgets(widget Widget) {
	if widget.NeedsRepaint() {
		widget.draw()
	} else {
		for _, childWidget := range widget.Widgets() {
			redrawWidgets(childWidget)
		}
	}
}
