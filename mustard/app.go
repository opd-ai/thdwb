package mustard

func CreateNewApp(name string) *App {
	return &App{name: name}
}

func (app *App) Run(callback func()) {
	for {
		for _, window := range app.windows {
			if window.visible && !window.glw.ShouldClose() {
				window.processFrame()
			}
		}

		callback()
	}
}

func (app *App) AddWindow(window *Window) {
	app.windows = append(app.windows, window)

	setWidgetWindow(window.rootFrame, window)
}

// Windows returns the list of windows managed by this app.
func (app *App) Windows() []*Window {
	return app.windows
}

func (app *App) DestroyWindow(window *Window) {
	var nWindows []*Window

	for _, appWindow := range app.windows {
		if appWindow != window {
			nWindows = append(nWindows, appWindow)
		}
	}

	app.windows = nWindows
	window.destroy()
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
