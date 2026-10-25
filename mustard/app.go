package mustard

import (
	"fmt"
	"time"
)

// App manages windows and the main event loop.
type App struct {
	name         string
	windows      []Window
	activeWindow Window
	tabbedMode   bool
	driver       Driver
}

// CreateNewApp creates a new application using the given driver.
func CreateNewApp(driver Driver, name string) *App {
	if driver == nil {
		driver = NewDriver()
	}
	return &App{name: name, driver: driver}
}

func (app *App) Run(callback func()) {
	if app.driver != nil {
		appDriver := app.driver.NewApp(app.name)
		appDriver.Run(callback)
		return
	}
	for {
		for _, window := range app.windows {
			if app.tabbedMode && window != app.activeWindow {
				continue
			}
			if wb, ok := window.(*WindowBase); ok && wb.visible && !wb.IsClosed() {
				callback()
			}
		}
		time.Sleep(time.Millisecond)
	}
}

func (app *App) AddWindow(window Window) {
	app.windows = append(app.windows, window)
	if app.tabbedMode {
		if wb, ok := window.(*WindowBase); ok {
			wb.setActive(false)
		}
	}

	if frame := app.rootFrame(window); frame != nil {
		setWidgetWindow(frame, window)
	}
}

func (app *App) rootFrame(window Window) *Frame {
	if wb, ok := window.(*WindowBase); ok {
		return wb.rootFrame
	}
	return nil
}

// ActivateWindow opts the app into tab-style processing and activates one managed window.
func (app *App) ActivateWindow(window Window) error {
	if window == nil || !app.containsWindow(window) {
		return fmt.Errorf("window is not managed by this app")
	}
	app.tabbedMode = true
	app.activeWindow = window
	for _, appWindow := range app.windows {
		if wb, ok := appWindow.(*WindowBase); ok {
			wb.setActive(appWindow == window)
		}
	}
	return nil
}

func (app *App) containsWindow(window Window) bool {
	for _, candidate := range app.windows {
		if candidate == window {
			return true
		}
	}
	return false
}

// Windows returns the list of windows managed by this app.
func (app *App) Windows() []Window {
	return app.windows
}

func (app *App) DestroyWindow(window Window) {
	wasActive := app.activeWindow == window
	var nWindows []Window

	for _, appWindow := range app.windows {
		if appWindow != window {
			nWindows = append(nWindows, appWindow)
		}
	}

	app.windows = nWindows
	if wb, ok := window.(*WindowBase); ok {
		wb.setActive(false)
	}
	if wasActive {
		app.activeWindow = nil
		if len(app.windows) > 0 {
			_ = app.ActivateWindow(app.windows[0])
		}
	}
}

func setWidgetWindow(widget Widget, window Window) {
	if wb, ok := window.(*WindowBase); ok {
		widget.SetWindow(wb)
	}

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
