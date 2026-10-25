package main

import (
	"fmt"
	"runtime"

	assets "github.com/danfragoso/thdwb/assets"
	hotdog "github.com/danfragoso/thdwb/hotdog"
	mustard "github.com/danfragoso/thdwb/mustard"
	profiler "github.com/danfragoso/thdwb/profiler"
)

func main() {
	runtime.LockOSThread()

	settings := &hotdog.Settings{
		WindowWidth:  1024,
		WindowHeight: 768,
		HiDPI:        false,
	}

	profilerInstance := profiler.CreateProfiler()
	buildInfo := &assets.BuildInfo{
		GitRevision: "example",
		GitBranch:   "main",
		HostInfo:    "example",
		BuildTime:   "now",
	}

	// Create a new window context
	windowContext := hotdog.NewWindowContext(settings, buildInfo, profilerInstance)
	defer windowContext.Destroy()

	app := mustard.CreateNewApp(mustard.NewDriver(), "THDWB Tabbed Browser")

	// Create multiple windows (tabs)
	window1, err := mustard.CreateNewWindow(mustard.NewDriver(), "Tab 1", settings.WindowWidth, settings.WindowHeight, settings.HiDPI)
	if err != nil {
		fmt.Printf("Failed to create window 1: %v\n", err)
		return
	}

	window2, err := mustard.CreateNewWindow(mustard.NewDriver(), "Tab 2", settings.WindowWidth, settings.WindowHeight, settings.HiDPI)
	if err != nil {
		fmt.Printf("Failed to create window 2: %v\n", err)
		return
	}

	wb1 := window1.(*mustard.WindowBase)
	wb2 := window2.(*mustard.WindowBase)
	windowContext.Window = wb1

	// Create root frame for window 1
	rootFrame1 := mustard.CreateFrame(mustard.VerticalFrame)
	title1 := mustard.CreateLabelWidget("Tab 1 - Home")
	title1.SetFontSize(24)
	title1.SetFontColor("#333333")
	title1.SetWidth(400)
	rootFrame1.AttachWidget(title1)
	wb1.SetRootFrame(rootFrame1)

	// Create root frame for window 2
	rootFrame2 := mustard.CreateFrame(mustard.VerticalFrame)
	title2 := mustard.CreateLabelWidget("Tab 2 - Settings")
	title2.SetFontSize(24)
	title2.SetFontColor("#333333")
	title2.SetWidth(400)
	rootFrame2.AttachWidget(title2)
	wb2.SetRootFrame(rootFrame2)

	// Add windows to app
	app.AddWindow(window1)
	app.AddWindow(window2)

	// Activate first window
	if err := app.ActivateWindow(window1); err != nil {
		fmt.Printf("Failed to activate window: %v\n", err)
		return
	}

	window1.Show()
	window2.Show()

	// Run the app
	app.Run(func() {
		// App runs until windows are closed
	})
}
