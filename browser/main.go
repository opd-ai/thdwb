// Package main is the thdwb browser entry point.
// It creates a window using the platform driver and runs the app event loop.
package main

import (
	"flag"
	"fmt"
	"runtime"

	"github.com/danfragoso/thdwb/browser/shell"
	assets "github.com/danfragoso/thdwb/assets"
	hotdog "github.com/danfragoso/thdwb/hotdog"
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
			Homepage:     "thdwb://newtab/",
			WindowWidth:  1024,
			WindowHeight: 768,
			HiDPI:        true,
		}
	}

	buildInfo := &assets.BuildInfo{
		GitRevision: gitRevision,
		GitBranch:   gitBranch,
		HostInfo:    hostInfo,
		BuildTime:   buildTime,
	}

	// Create browser shell
	browserShell := shell.NewBrowserShell(settings, buildInfo)

	// Initialize the shell (creates window and UI)
	if err := browserShell.Init(); err != nil {
		fmt.Printf("Failed to initialize browser shell: %v\n", err)
		return
	}

	// Show and run
	browserShell.Show()
	browserShell.Run()
}