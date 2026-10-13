//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"runtime"

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
	buildInfo := &hotdog.BuildInfo{
		GitRevision: "example",
		GitBranch:   "main",
		HostInfo:    "example",
		BuildTime:   "now",
	}

	// Create a new window context
	windowContext := hotdog.NewWindowContext(settings, buildInfo, profilerInstance)
	defer windowContext.Destroy()

	app := mustard.CreateNewApp("THDWB Form Example")
	window := mustard.CreateNewWindow("Form Example", settings.WindowWidth, settings.WindowHeight, settings.HiDPI)
	if window == nil {
		fmt.Println("Failed to create window")
		return
	}

	windowContext.Window = window

	// Create root frame
	rootFrame := mustard.CreateFrame(mustard.VerticalFrame)

	// Title
	title := mustard.CreateLabelWidget("Login Form")
	title.SetFontSize(24)
	title.SetFontColor("#333333")
	title.SetWidth(300)

	// Username field
	usernameLabel := mustard.CreateLabelWidget("Username:")
	usernameLabel.SetFontSize(14)
	usernameLabel.SetFontColor("#333333")

	usernameInput := mustard.CreateInputWidget()
	usernameInput.SetWidth(300)
	usernameInput.SetHeight(35)
	usernameInput.SetPlaceholder("Enter username")

	// Password field
	passwordLabel := mustard.CreateLabelWidget("Password:")
	passwordLabel.SetFontSize(14)
	passwordLabel.SetFontColor("#333333")

	passwordInput := mustard.CreateInputWidget()
	passwordInput.SetWidth(300)
	passwordInput.SetHeight(35)
	passwordInput.SetPlaceholder("Enter password")
	passwordInput.SetPasswordMode(true)

	// Submit button
	submitButton := mustard.CreateButtonWidget("Login")
	submitButton.SetWidth(300)
	submitButton.SetHeight(40)
	submitButton.SetBackgroundColor("#2196F3")
	submitButton.SetFontColor("#FFFFFF")
	submitButton.SetFontSize(16)

	// Status label
	statusLabel := mustard.CreateLabelWidget("")
	statusLabel.SetFontSize(14)
	statusLabel.SetFontColor("#FF0000")

	submitButton.SetClickHandler(func() {
		username := usernameInput.GetValue()
		password := passwordInput.GetValue()
		if username == "" || password == "" {
			statusLabel.SetContent("Please fill in all fields")
			statusLabel.SetFontColor("#FF0000")
		} else {
			statusLabel.SetContent(fmt.Sprintf("Welcome, %s!", username))
			statusLabel.SetFontColor("#4CAF50")
		}
		statusLabel.RequestRepaint()
	})

	// Add widgets to frame
	rootFrame.AttachWidget(title)
	rootFrame.AttachWidget(usernameLabel)
	rootFrame.AttachWidget(usernameInput)
	rootFrame.AttachWidget(passwordLabel)
	rootFrame.AttachWidget(passwordInput)
	rootFrame.AttachWidget(submitButton)
	rootFrame.AttachWidget(statusLabel)

	window.SetRootFrame(rootFrame)
	app.AddWindow(window)
	window.Show()

	// Run the app
	app.Run(func() {
		// App runs until window is closed
	})
}