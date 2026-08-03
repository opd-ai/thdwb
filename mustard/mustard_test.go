//go:build ignore

package mustard

import (
	"os"
	"runtime"
	"strconv"
	"testing"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func TestMustard(t *testing.T) {
	// Skip test in headless/CI environments
	if runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" {
		t.Skip("Skipping GUI test in headless environment (no DISPLAY)")
	}
	// Also skip in common CI environments even if DISPLAY is set
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" || os.Getenv("GITLAB_CI") != "" {
		t.Skip("Skipping GUI test in CI environment")
	}

	runtime.LockOSThread()
	if err := glfw.Init(); err != nil {
		t.Skipf("Failed to initialize GLFW: %v", err)
	}
	defer glfw.Terminate()

	// Try to create a test window to verify display works
	testWindow, err := glfw.CreateWindow(100, 100, "test", nil, nil)
	if err != nil {
		t.Skipf("Cannot create GLFW window (no working display): %v", err)
	}
	testWindow.Destroy()

	if err := gl.Init(); err != nil {
		t.Skipf("Failed to initialize OpenGL: %v", err)
	}

	SetGLFWHints()

	app := CreateNewApp("THDWB")
	window := CreateNewWindow("THDWB", 600, 600, false)
	if window == nil {
		t.Skip("Failed to create window (likely no display available)")
	}
	rootFrame := CreateFrame(HorizontalFrame)

	appBar := CreateFrame(VerticalFrame)

	titleBar := CreateLabelWidget("THDWB - nil")
	titleBar.SetFontColor("#fff")

	appBar.SetHeight(28)
	appBar.AttachWidget(titleBar)
	appBar.SetBackgroundColor("#5f6368")

	rootFrame.AttachWidget(appBar)

	viewPort := CreateCanvasWidget(func(ctx *CanvasWidget) {})

	rootFrame.AttachWidget(viewPort)

	statusBar := CreateFrame(HorizontalFrame)
	statusBar.SetBackgroundColor("#babcbe")
	statusBar.SetHeight(20)

	statusLabel := CreateLabelWidget("Processed Events:")
	statusLabel.SetFontSize(16)
	frameEvents := 0

	rootFrame.AttachWidget(statusBar)
	statusBar.AttachWidget(statusLabel)

	window.SetRootFrame(rootFrame)

	app.AddWindow(window)

	window.Show()

	// Run for a few frames then exit
	frameCount := 0
	app.Run(func() {
		frameCount++
		statusLabel.SetContent("Processed Events: " + strconv.Itoa(frameEvents) + "; Resolution: " + strconv.Itoa(window.width) + "X" + strconv.Itoa(window.height))
		if frameCount >= 3 {
			window.glw.SetShouldClose(true)
		}
	})
}
