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

	app := mustard.CreateNewApp("THDWB Modal Example")
	window := mustard.CreateNewWindow("Modal Example", settings.WindowWidth, settings.WindowHeight, settings.HiDPI)
	if window == nil {
		fmt.Println("Failed to create window")
		return
	}

	windowContext.Window = window

	// Create root frame
	rootFrame := mustard.CreateFrame(mustard.VerticalFrame)

	// Title
	title := mustard.CreateLabelWidget("Modal Dialog Example")
	title.SetFontSize(24)
	title.SetFontColor("#333333")
	title.SetWidth(400)

	// Open modal button
	openModalButton := mustard.CreateButtonWidget("Open Modal")
	openModalButton.SetWidth(200)
	openModalButton.SetHeight(40)
	openModalButton.SetBackgroundColor("#FF9800")
	openModalButton.SetFontColor("#FFFFFF")
	openModalButton.SetFontSize(16)

	// Status label
	statusLabel := mustard.CreateLabelWidget("Click 'Open Modal' to show a modal dialog")
	statusLabel.SetFontSize(14)
	statusLabel.SetFontColor("#333333")

	// Create modal overlay
	modalOverlay := mustard.CreateFrame(mustard.VerticalFrame)
	modalOverlay.SetWidth(400)
	modalOverlay.SetHeight(200)
	modalOverlay.SetBackgroundColor("#FFFFFF")
	modalOverlay.SetVisible(false)

	// Modal title
	modalTitle := mustard.CreateLabelWidget("Confirm Action")
	modalTitle.SetFontSize(20)
	modalTitle.SetFontColor("#333333")
	modalTitle.SetWidth(350)

	// Modal message
	modalMessage := mustard.CreateLabelWidget("Are you sure you want to proceed?")
	modalMessage.SetFontSize(14)
	modalMessage.SetFontColor("#666666")
	modalMessage.SetWidth(350)

	// Modal buttons frame
	modalButtons := mustard.CreateFrame(mustard.HorizontalFrame)

	// Confirm button
	confirmButton := mustard.CreateButtonWidget("Confirm")
	confirmButton.SetWidth(120)
	confirmButton.SetHeight(35)
	confirmButton.SetBackgroundColor("#F44336")
	confirmButton.SetFontColor("#FFFFFF")
	confirmButton.SetFontSize(14)

	// Cancel button
	cancelButton := mustard.CreateButtonWidget("Cancel")
	cancelButton.SetWidth(120)
	cancelButton.SetHeight(35)
	cancelButton.SetBackgroundColor("#9E9E9E")
	cancelButton.SetFontColor("#FFFFFF")
	cancelButton.SetFontSize(14)

	modalButtons.AttachWidget(confirmButton)
	modalButtons.AttachWidget(cancelButton)

	modalOverlay.AttachWidget(modalTitle)
	modalOverlay.AttachWidget(modalMessage)
	modalOverlay.AttachWidget(modalButtons)

	// Add modal to window as overlay
	window.AddOverlay(modalOverlay)

	// Button handlers
	openModalButton.SetClickHandler(func() {
		modalOverlay.SetVisible(true)
		modalOverlay.RequestRepaint()
		window.RequestReflow()
	})

	confirmButton.SetClickHandler(func() {
		modalOverlay.SetVisible(false)
		statusLabel.SetContent("Action confirmed!")
		statusLabel.SetFontColor("#4CAF50")
		statusLabel.RequestRepaint()
	})

	cancelButton.SetClickHandler(func() {
		modalOverlay.SetVisible(false)
		statusLabel.SetContent("Action cancelled")
		statusLabel.SetFontColor("#FF9800")
		statusLabel.RequestRepaint()
	})

	// Add widgets to frame
	rootFrame.AttachWidget(title)
	rootFrame.AttachWidget(openModalButton)
	rootFrame.AttachWidget(statusLabel)

	window.SetRootFrame(rootFrame)
	app.AddWindow(window)
	window.Show()

	// Run the app
	app.Run(func() {
		// App runs until window is closed
	})
}