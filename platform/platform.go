// Package platform provides a cross-platform abstraction layer for rendering,
// windowing, and input, decoupling the browser render stack from CGo dependencies.
package platform

// Renderer is the platform-agnostic rendering interface.
type Renderer interface {
	// Clear clears the current render target.
	Clear()
	// DrawImage draws an image at the given position.
	DrawImage(img Image, x, y float64)
	// SetColor sets the current drawing color.
	SetColor(r, g, b, a float32)
	// DrawRectangle draws a filled rectangle.
	DrawRectangle(x, y, width, height float64)
	// Present swaps buffers and presents the rendered frame.
	Present()
	// Size returns the current render target dimensions.
	Size() (width, height int)
}

// Window is the platform-agnostic window interface.
type Window interface {
	// Show displays the window.
	Show()
	// Hide hides the window.
	Hide()
	// Close closes the window.
	Close()
	// IsClosed returns true if the window has been closed.
	IsClosed() bool
	// SetTitle sets the window title.
	SetTitle(title string)
	// Size returns the window dimensions.
	Size() (width, height int)
	// SetSize sets the window dimensions.
	SetSize(width, height int)
	// PollEvents processes pending input events.
	PollEvents()
	// ShouldClose returns true if the window should close.
	ShouldClose() bool
}

// Input provides platform-agnostic input state.
type Input interface {
	// CursorPosition returns the current cursor position.
	CursorPosition() (x, y float64)
	// IsKeyPressed returns true if the given key is currently pressed.
	IsKeyPressed(key Key) bool
	// IsMouseButtonPressed returns true if the given mouse button is pressed.
	IsMouseButtonPressed(button MouseButton) bool
}

// Key represents a keyboard key.
type Key int

const (
	KeyUnknown Key = iota
	KeyBackspace
	KeyDelete
	KeyEnter
	KeyEscape
	KeyTab
	KeyArrowLeft
	KeyArrowRight
	KeyArrowUp
	KeyArrowDown
	KeyA
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
	KeyG
	KeyH
	KeyI
	KeyJ
	KeyK
	KeyL
	KeyM
	KeyN
	KeyO
	KeyP
	KeyQ
	KeyR
	KeyS
	KeyT
	KeyU
	KeyV
	KeyW
	KeyX
	KeyY
	KeyZ
	Key0
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9
	KeySpace
	KeyControl
	KeyShift
	KeyAlt
	KeySuper
)

// MouseButton represents a mouse button.
type MouseButton int

const (
	MouseButtonLeft MouseButton = iota
	MouseButtonRight
)

// Image is a platform-agnostic image interface.
type Image interface {
	// Bounds returns the image bounds.
	Bounds() (x0, y0, x1, y1 int)
	// RGBA returns the pixel at (x, y).
	RGBAAt(x, y int) (r, g, b, a uint32)
}

// App is the platform application interface.
type App interface {
	// Run starts the main loop, calling callback each frame.
	Run(callback func())
	// AddWindow adds a window to the application.
	AddWindow(w Window)
	// DestroyWindow removes and destroys a window.
	DestroyWindow(w Window)
}
