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
		WindowWidth:  800,
		WindowHeight: 600,
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

	app := mustard.CreateNewApp(mustard.NewDriver(), "THDWB Button Example")
	window, err := mustard.CreateNewWindow(mustard.NewDriver(), "Button Example", settings.WindowWidth, settings.WindowHeight, settings.HiDPI)
	if err != nil {
		fmt.Printf("Failed to create window: %v\n", err)
		return
	}

	wb := window.(*mustard.WindowBase)
	windowContext.Window = wb

	// Create root frame
	rootFrame := mustard.CreateFrame(mustard.VerticalFrame)

	// Create a button (need to pass an asset for icon, use empty)
	button := mustard.CreateButtonWidget("Click Me!", []byte{})
	button.SetWidth(200)
	button.SetHeight(50)
	button.SetBackgroundColor("#4CAF50")
	button.SetFontColor("#FFFFFF")
	button.SetFontSize(18)

	// Add click handler
	clickCount := 0
	wb.RegisterButton(button, func() {
		clickCount++
		button.SetContent(fmt.Sprintf("Clicked %d times!", clickCount))
		button.RequestRepaint()
	})

	// Create a label to show status
	statusLabel := mustard.CreateLabelWidget("Button not clicked yet")
	statusLabel.SetFontSize(16)
	statusLabel.SetFontColor("#333333")

	// Add widgets to frame
	rootFrame.AttachWidget(button)
	rootFrame.AttachWidget(statusLabel)

	wb.SetRootFrame(rootFrame)
	app.AddWindow(window)
	window.Show()

	// Run the app
	app.Run(func() {
		// App runs until window is closed
	})
}
