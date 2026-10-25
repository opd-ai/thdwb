package main

import (
	"fmt"
	"image"
	"runtime"

	assets "github.com/danfragoso/thdwb/assets"
	gg "github.com/danfragoso/thdwb/gg"
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

	app := mustard.CreateNewApp(nil, "THDWB Modal Example")
	window, err := mustard.CreateNewWindow(nil, "Modal Example", settings.WindowWidth, settings.WindowHeight, settings.HiDPI)
	if err != nil {
		fmt.Printf("Failed to create window: %v\n", err)
		return
	}

	wb := window.(*mustard.WindowBase)
	windowContext.Window = wb

	// Create root frame
	rootFrame := mustard.CreateFrame(mustard.VerticalFrame)

	// Title
	title := mustard.CreateLabelWidget("Modal Dialog Example")
	title.SetFontSize(24)
	title.SetFontColor("#333333")
	title.SetWidth(400)

	// Open modal button
	openModalButton := mustard.CreateButtonWidget("Open Modal", []byte{})
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
	confirmButton := mustard.CreateButtonWidget("Confirm", []byte{})
	confirmButton.SetWidth(120)
	confirmButton.SetHeight(35)
	confirmButton.SetBackgroundColor("#F44336")
	confirmButton.SetFontColor("#FFFFFF")
	confirmButton.SetFontSize(14)

	// Cancel button
	cancelButton := mustard.CreateButtonWidget("Cancel", []byte{})
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

	// Create overlay from modal frame buffer
	modalCtx := gg.NewContext(400, 200)
	overlay := mustard.CreateOverlay("modal", modalCtx.Image().(*image.RGBA), image.Point{X: 200, Y: 200})
	wb.AddStaticOverlay(overlay)

	// Button handlers
	wb.RegisterButton(openModalButton, func() {
		modalOverlay.SetVisible(true)
		modalOverlay.RequestRepaint()
		wb.RequestReflow()
	})

	wb.RegisterButton(confirmButton, func() {
		modalOverlay.SetVisible(false)
		statusLabel.SetContent("Action confirmed!")
		statusLabel.SetFontColor("#4CAF50")
		statusLabel.RequestRepaint()
	})

	wb.RegisterButton(cancelButton, func() {
		modalOverlay.SetVisible(false)
		statusLabel.SetContent("Action cancelled")
		statusLabel.SetFontColor("#FF9800")
		statusLabel.RequestRepaint()
	})

	// Add widgets to frame
	rootFrame.AttachWidget(title)
	rootFrame.AttachWidget(openModalButton)
	rootFrame.AttachWidget(statusLabel)

	wb.SetRootFrame(rootFrame)
	app.AddWindow(window)
	window.Show()

	// Run the app
	app.Run(func() {
		// App runs until window is closed
	})
}
