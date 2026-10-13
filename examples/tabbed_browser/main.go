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
		WindowWidth:  1024,
		WindowHeight: 768,
		HiDPI:        false,
	}

	profilerInstance := profiler.CreateProfiler()
	buildInfo := &hotdog.BuildInfo{
		GitRevision: "example",
		GitBranch:   "main",
		HostInfo:    "example",
		BuildTime:   "now",
	}

	// Create the app with tabbed mode
	app := mustard.CreateNewApp("THDWB Tabbed Browser")

	// Create first tab
	tab1Settings := &hotdog.Settings{
		WindowWidth:  1024,
		WindowHeight: 768,
		HiDPI:        false,
	}
	tab1Context := hotdog.NewWindowContext(tab1Settings, buildInfo, profilerInstance)
	defer tab1Context.Destroy()

	tab1Window := mustard.CreateNewWindow("Tab 1 - Home", settings.WindowWidth, settings.WindowHeight, settings.HiDPI)
	if tab1Window == nil {
		fmt.Println("Failed to create tab 1 window")
		return
	}
	tab1Context.Window = tab1Window

	// Create root frame for tab 1
	tab1RootFrame := mustard.CreateFrame(mustard.VerticalFrame)

	// Tab 1 content
	tab1Title := mustard.CreateLabelWidget("Welcome to THDWB Tabbed Browser")
	tab1Title.SetFontSize(24)
	tab1Title.SetFontColor("#333333")
	tab1Title.SetWidth(500)

	tab1Desc := mustard.CreateLabelWidget("This is Tab 1. Click the buttons below to interact.")
	tab1Desc.SetFontSize(14)
	tab1Desc.SetFontColor("#666666")
	tab1Desc.SetWidth(500)

	tab1Button := mustard.CreateButtonWidget("Click Me from Tab 1")
	tab1Button.SetWidth(200)
	tab1Button.SetHeight(40)
	tab1Button.SetBackgroundColor("#4CAF50")
	tab1Button.SetFontColor("#FFFFFF")
	tab1Button.SetFontSize(16)

	tab1Status := mustard.CreateLabelWidget("")
	tab1Status.SetFontSize(14)
	tab1Status.SetFontColor("#333333")

	clickCount1 := 0
	tab1Button.SetClickHandler(func() {
		clickCount1++
		tab1Status.SetContent(fmt.Sprintf("Tab 1 button clicked %d times!", clickCount1))
		tab1Status.RequestRepaint()
	})

	tab1RootFrame.AttachWidget(tab1Title)
	tab1RootFrame.AttachWidget(tab1Desc)
	tab1RootFrame.AttachWidget(tab1Button)
	tab1RootFrame.AttachWidget(tab1Status)

	tab1Window.SetRootFrame(tab1RootFrame)
	app.AddWindow(tab1Window)

	// Create second tab
	tab2Settings := &hotdog.Settings{
		WindowWidth:  1024,
		WindowHeight: 768,
		HiDPI:        false,
	}
	tab2Context := hotdog.NewWindowContext(tab2Settings, buildInfo, profilerInstance)
	defer tab2Context.Destroy()

	tab2Window := mustard.CreateNewWindow("Tab 2 - Form", settings.WindowWidth, settings.WindowHeight, settings.HiDPI)
	if tab2Window == nil {
		fmt.Println("Failed to create tab 2 window")
		return
	}
	tab2Context.Window = tab2Window

	// Create root frame for tab 2
	tab2RootFrame := mustard.CreateFrame(mustard.VerticalFrame)

	// Tab 2 content
	tab2Title := mustard.CreateLabelWidget("Tab 2 - Form Example")
	tab2Title.SetFontSize(24)
	tab2Title.SetFontColor("#333333")
	tab2Title.SetWidth(500)

	tab2Desc := mustard.CreateLabelWidget("This is Tab 2 with a form. Fill in the fields and submit.")
	tab2Desc.SetFontSize(14)
	tab2Desc.SetFontColor("#666666")
	tab2Desc.SetWidth(500)

	// Username field
	usernameLabel := mustard.CreateLabelWidget("Username:")
	usernameLabel.SetFontSize(14)
	usernameLabel.SetFontColor("#333333")

	usernameInput := mustard.CreateInputWidget()
	usernameInput.SetWidth(300)
	usernameInput.SetHeight(35)
	usernameInput.SetPlaceholder("Enter username")

	// Email field
	emailLabel := mustard.CreateLabelWidget("Email:")
	emailLabel.SetFontSize(14)
	emailLabel.SetFontColor("#333333")

	emailInput := mustard.CreateInputWidget()
	emailInput.SetWidth(300)
	emailInput.SetHeight(35)
	emailInput.SetPlaceholder("Enter email")

	// Submit button
	submitButton := mustard.CreateButtonWidget("Submit Form")
	submitButton.SetWidth(200)
	submitButton.SetHeight(40)
	submitButton.SetBackgroundColor("#2196F3")
	submitButton.SetFontColor("#FFFFFF")
	submitButton.SetFontSize(16)

	tab2Status := mustard.CreateLabelWidget("")
	tab2Status.SetFontSize(14)
	tab2Status.SetFontColor("#333333")

	submitButton.SetClickHandler(func() {
		username := usernameInput.GetValue()
		email := emailInput.GetValue()
		if username == "" || email == "" {
			tab2Status.SetContent("Please fill in all fields")
			tab2Status.SetFontColor("#FF0000")
		} else {
			tab2Status.SetContent(fmt.Sprintf("Submitted: %s (%s)", username, email))
			tab2Status.SetFontColor("#4CAF50")
		}
		tab2Status.RequestRepaint()
	})

	tab2RootFrame.AttachWidget(tab2Title)
	tab2RootFrame.AttachWidget(tab2Desc)
	tab2RootFrame.AttachWidget(usernameLabel)
	tab2RootFrame.AttachWidget(usernameInput)
	tab2RootFrame.AttachWidget(emailLabel)
	tab2RootFrame.AttachWidget(emailInput)
	tab2RootFrame.AttachWidget(submitButton)
	tab2RootFrame.AttachWidget(tab2Status)

	tab2Window.SetRootFrame(tab2RootFrame)
	app.AddWindow(tab2Window)

	// Create third tab
	tab3Settings := &hotdog.Settings{
		WindowWidth:  1024,
		WindowHeight: 768,
		HiDPI:        false,
	}
	tab3Context := hotdog.NewWindowContext(tab3Settings, buildInfo, profilerInstance)
	defer tab3Context.Destroy()

	tab3Window := mustard.CreateNewWindow("Tab 3 - Modal", settings.WindowWidth, settings.WindowHeight, settings.HiDPI)
	if tab3Window == nil {
		fmt.Println("Failed to create tab 3 window")
		return
	}
	tab3Context.Window = tab3Window

	// Create root frame for tab 3
	tab3RootFrame := mustard.CreateFrame(mustard.VerticalFrame)

	// Tab 3 content
	tab3Title := mustard.CreateLabelWidget("Tab 3 - Modal Dialog")
	tab3Title.SetFontSize(24)
	tab3Title.SetFontColor("#333333")
	tab3Title.SetWidth(500)

	tab3Desc := mustard.CreateLabelWidget("This is Tab 3 with a modal dialog example.")
	tab3Desc.SetFontSize(14)
	tab3Desc.SetFontColor("#666666")
	tab3Desc.SetWidth(500)

	openModalButton := mustard.CreateButtonWidget("Open Modal")
	openModalButton.SetWidth(200)
	openModalButton.SetHeight(40)
	openModalButton.SetBackgroundColor("#FF9800")
	openModalButton.SetFontColor("#FFFFFF")
	openModalButton.SetFontSize(16)

	tab3Status := mustard.CreateLabelWidget("Click 'Open Modal' to show a modal dialog")
	tab3Status.SetFontSize(14)
	tab3Status.SetFontColor("#333333")

	// Create modal overlay
	modalOverlay := mustard.CreateFrame(mustard.VerticalFrame)
	modalOverlay.SetWidth(400)
	modalOverlay.SetHeight(200)
	modalOverlay.SetBackgroundColor("#FFFFFF")
	modalOverlay.SetVisible(false)

	modalTitle := mustard.CreateLabelWidget("Confirm Action")
	modalTitle.SetFontSize(20)
	modalTitle.SetFontColor("#333333")
	modalTitle.SetWidth(350)

	modalMessage := mustard.CreateLabelWidget("Are you sure you want to proceed?")
	modalMessage.SetFontSize(14)
	modalMessage.SetFontColor("#666666")
	modalMessage.SetWidth(350)

	modalButtons := mustard.CreateFrame(mustard.HorizontalFrame)

	confirmButton := mustard.CreateButtonWidget("Confirm")
	confirmButton.SetWidth(120)
	confirmButton.SetHeight(35)
	confirmButton.SetBackgroundColor("#F44336")
	confirmButton.SetFontColor("#FFFFFF")
	confirmButton.SetFontSize(14)

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

	tab3Window.AddOverlay(modalOverlay)

	openModalButton.SetClickHandler(func() {
		modalOverlay.SetVisible(true)
		modalOverlay.RequestRepaint()
		tab3Window.RequestReflow()
	})

	confirmButton.SetClickHandler(func() {
		modalOverlay.SetVisible(false)
		tab3Status.SetContent("Action confirmed!")
		tab3Status.SetFontColor("#4CAF50")
		tab3Status.RequestRepaint()
	})

	cancelButton.SetClickHandler(func() {
		modalOverlay.SetVisible(false)
		tab3Status.SetContent("Action cancelled")
		tab3Status.SetFontColor("#FF9800")
		tab3Status.RequestRepaint()
	})

	tab3RootFrame.AttachWidget(tab3Title)
	tab3RootFrame.AttachWidget(tab3Desc)
	tab3RootFrame.AttachWidget(openModalButton)
	tab3RootFrame.AttachWidget(tab3Status)

	tab3Window.SetRootFrame(tab3RootFrame)
	app.AddWindow(tab3Window)

	// Activate first tab
	if err := app.ActivateWindow(tab1Window); err != nil {
		fmt.Printf("Failed to activate first tab: %v\n", err)
		return
	}

	// Show all windows
	tab1Window.Show()
	tab2Window.Show()
	tab3Window.Show()

	// Add tab switching hotkeys (1, 2, 3 keys)
	tab1Window.AttachKeyboardInputEventListener(func(input mustard.KeyboardInput) {
		if input.Action == mustard.KeyboardRelease {
			switch input.Key {
			case "1":
				app.ActivateWindow(tab1Window)
				fmt.Println("Switched to Tab 1")
			case "2":
				app.ActivateWindow(tab2Window)
				fmt.Println("Switched to Tab 2")
			case "3":
				app.ActivateWindow(tab3Window)
				fmt.Println("Switched to Tab 3")
			}
		}
	})

	tab2Window.AttachKeyboardInputEventListener(func(input mustard.KeyboardInput) {
		if input.Action == mustard.KeyboardRelease {
			switch input.Key {
			case "1":
				app.ActivateWindow(tab1Window)
				fmt.Println("Switched to Tab 1")
			case "2":
				app.ActivateWindow(tab2Window)
				fmt.Println("Switched to Tab 2")
			case "3":
				app.ActivateWindow(tab3Window)
				fmt.Println("Switched to Tab 3")
			}
		}
	})

	tab3Window.AttachKeyboardInputEventListener(func(input mustard.KeyboardInput) {
		if input.Action == mustard.KeyboardRelease {
			switch input.Key {
			case "1":
				app.ActivateWindow(tab1Window)
				fmt.Println("Switched to Tab 1")
			case "2":
				app.ActivateWindow(tab2Window)
				fmt.Println("Switched to Tab 2")
			case "3":
				app.ActivateWindow(tab3Window)
				fmt.Println("Switched to Tab 3")
			}
		}
	})

	fmt.Println("Tabbed Browser Example")
	fmt.Println("Press 1, 2, or 3 to switch tabs")
	fmt.Println("Close window to exit")

	// Run the app
	app.Run(func() {
		// App runs until window is closed
	})
}