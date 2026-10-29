//go:build android || ios

// Package main is the thdwb mobile browser entry point for Android and iOS.
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
		HiDPI:        true,
	}

	profilerInstance := profiler.CreateProfiler()
	buildInfo := &assets.BuildInfo{
		GitRevision: "mobile",
		GitBranch:   "main",
		HostInfo:    runtime.GOOS + "/" + runtime.GOARCH,
		BuildTime:   "now",
	}

	// Create a new window context
	windowContext := hotdog.NewWindowContext(settings, buildInfo, profilerInstance)
	defer windowContext.Destroy()

	app := mustard.CreateNewApp(mustard.NewDriver(), "THDWB Mobile Browser")

	window, err := mustard.CreateNewWindow(mustard.NewDriver(), "THDWB", settings.WindowWidth, settings.WindowHeight, settings.HiDPI)
	if err != nil {
		fmt.Printf("Failed to create window: %v\n", err)
		return
	}

	wb := window.(*mustard.WindowBase)
	windowContext.Window = wb

	// Create root frame
	rootFrame := mustard.CreateFrame(mustard.VerticalFrame)

	title := mustard.CreateLabelWidget("THDWB Mobile Browser")
	title.SetFontSize(24)
	title.SetFontColor("#333333")
	title.SetWidth(300)
	rootFrame.AttachWidget(title)

	wb.SetRootFrame(rootFrame)
	app.AddWindow(window)
	window.Show()

	// Run the app
	app.Run(func() {
		// App runs until window is closed
	})
}
