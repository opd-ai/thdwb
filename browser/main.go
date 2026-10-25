// Package main is the thdwb browser entry point.
// It creates a window using the platform driver and runs the app event loop.
package main

import (
	"flag"
	"fmt"
	"runtime"

	assets "github.com/danfragoso/thdwb/assets"
	hotdog "github.com/danfragoso/thdwb/hotdog"
	mustard "github.com/danfragoso/thdwb/mustard"
	profiler "github.com/danfragoso/thdwb/profiler"
)

// Build metadata injected at build time via -ldflags (see Makefile).
var (
	gitRevision = "dev"
	gitBranch   = "dev"
	hostInfo    = runtime.GOOS + "/" + runtime.GOARCH
	buildTime   = "unknown"
)

func main() {
	runtime.LockOSThread()

	settingsPath := flag.String("settings", "", "path to a settings JSON file")
	flag.Parse()

	var settings *hotdog.Settings
	if *settingsPath != "" {
		settings = hotdog.LoadSettings(*settingsPath)
	} else {
		settings = &hotdog.Settings{
			Homepage:     "thdwb://homepage/",
			WindowWidth:  900,
			WindowHeight: 600,
			HiDPI:        true,
		}
	}

	profilerInstance := profiler.CreateProfiler()
	buildInfo := &assets.BuildInfo{
		GitRevision: gitRevision,
		GitBranch:   gitBranch,
		HostInfo:    hostInfo,
		BuildTime:   buildTime,
	}

	// Create a new window context
	windowContext := hotdog.NewWindowContext(settings, buildInfo, profilerInstance)
	defer windowContext.Destroy()

	app := mustard.CreateNewApp(mustard.NewDriver(), "THDWB Browser")
	window, err := mustard.CreateNewWindow(mustard.NewDriver(), "THDWB Browser", settings.WindowWidth, settings.WindowHeight, settings.HiDPI)
	if err != nil {
		fmt.Printf("Failed to create window: %v\n", err)
		return
	}

	wb, ok := window.(*mustard.WindowBase)
	if !ok {
		fmt.Printf("Window is not WindowBase: %T\n", window)
		return
	}
	windowContext.Window = wb

	// Create root frame
	rootFrame := mustard.CreateFrame(mustard.VerticalFrame)

	// Title label
	title := mustard.CreateLabelWidget("THDWB - The Hotdog Web Browser")
	title.SetFontSize(24)
	title.SetFontColor("#333333")
	title.SetWidth(400)

	// Status label
	statusLabel := mustard.CreateLabelWidget("Browser ready")
	statusLabel.SetFontSize(14)
	statusLabel.SetFontColor("#333333")

	// Add widgets to frame
	rootFrame.AttachWidget(title)
	rootFrame.AttachWidget(statusLabel)

	wb.SetRootFrame(rootFrame)
	app.AddWindow(window)
	window.Show()

	// Run the app
	app.Run(func() {
		// App runs until window is closed
	})
}
