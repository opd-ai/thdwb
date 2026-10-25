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

	app := mustard.CreateNewApp(nil, "THDWB Form Example")
	window, err := mustard.CreateNewWindow(nil, "Form Example", settings.WindowWidth, settings.WindowHeight, settings.HiDPI)
	if err != nil {
		fmt.Printf("Failed to create window: %v\n", err)
		return
	}

	wb := window.(*mustard.WindowBase)
	windowContext.Window = wb

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
	usernameInput.SetValue("")

	// Password field
	passwordLabel := mustard.CreateLabelWidget("Password:")
	passwordLabel.SetFontSize(14)
	passwordLabel.SetFontColor("#333333")

	passwordInput := mustard.CreateInputWidget()
	passwordInput.SetWidth(300)
	passwordInput.SetHeight(35)
	passwordInput.SetValue("")

	// Submit button
	submitButton := mustard.CreateButtonWidget("Login", []byte{})
	submitButton.SetWidth(300)
	submitButton.SetHeight(40)
	submitButton.SetBackgroundColor("#2196F3")
	submitButton.SetFontColor("#FFFFFF")
	submitButton.SetFontSize(16)

	// Status label
	statusLabel := mustard.CreateLabelWidget("")
	statusLabel.SetFontSize(14)
	statusLabel.SetFontColor("#FF0000")

	wb.RegisterButton(submitButton, func() {
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

	wb.SetRootFrame(rootFrame)
	app.AddWindow(window)
	window.Show()

	// Run the app
	app.Run(func() {
		// App runs until window is closed
	})
}
