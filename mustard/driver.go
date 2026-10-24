// Package mustard provides the immediate-mode UI widget toolkit for thdwb.
package mustard

import "image"

// Driver is the platform-specific rendering and windowing backend.
// It is implemented separately for CGo (GLFW/OpenGL) and pure Go (Ebitengine) builds.
type Driver interface {
	// NewWindow creates a new platform window.
	NewWindow(title string, width, height int, hiDPI bool) (Window, error)
	// NewRenderer creates a new renderer for the given window.
	NewRenderer(w Window) (Renderer, error)
	// NewApp creates a new application.
	NewApp(name string) AppDriver
}

// Window is the platform-agnostic window interface used by mustard.
type Window interface {
	Show()
	Hide()
	Close()
	IsClosed() bool
	SetTitle(string)
	Size() (int, int)
	SetSize(int, int)
	PollEvents()
	ShouldClose() bool
	GetCursorPosition() (float64, float64)
	SetCursor(cursor Cursor)
}

// Renderer is the platform-agnostic rendering interface used by mustard.
type Renderer interface {
	Clear()
	DrawImage(img image.Image, x, y float64)
	SetColor(r, g, b, a float32)
	DrawRectangle(x, y, width, height float64)
	Present()
	Size() (int, int)
}

// AppDriver is the platform application interface.
type AppDriver interface {
	Run(callback func())
	AddWindow(w Window)
	DestroyWindow(w Window)
}

// Cursor is a platform-agnostic cursor.
type Cursor interface{}
