package main

import (
	"fmt"
	"runtime"

	assets "github.com/danfragoso/thdwb/assets"
	hotdog "github.com/danfragoso/thdwb/hotdog"
	mustard "github.com/danfragoso/thdwb/mustard"
	profiler "github.com/danfragoso/thdwb/profiler"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func main() {
	runtime.LockOSThread()

	if err := glfw.Init(); err != nil {
		fmt.Printf("Failed to initialize GLFW: %v\n", err)
		return
	}
	defer glfw.Terminate()

	if err := gl.Init(); err != nil {
		fmt.Printf("Failed to initialize OpenGL: %v\n", err)
		return
	}

	mustard.SetGLFWHints()

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

	app := mustard.CreateNewApp("THDWB Button Example")
	window := mustard.CreateNewWindow("Button Example", settings.WindowWidth, settings.WindowHeight, settings.HiDPI)
	if window == nil {
		fmt.Println("Failed to create window")
		return
	}

	windowContext.Window = window

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
	window.RegisterButton(button, func() {
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

	window.SetRootFrame(rootFrame)
	app.AddWindow(window)
	window.Show()

	// Run the app
	app.Run(func() {
		// App runs until window is closed
	})
}
