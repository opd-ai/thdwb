package platform

// Command represents a single GPU command in the platform-agnostic command buffer.
type Command interface {
	// Execute applies the command to the given renderer.
	Execute(r Renderer)
}

// ClearCommand clears the render target with the given color.
type ClearCommand struct {
	R, G, B, A float32
}

func (c ClearCommand) Execute(r Renderer) {
	r.SetColor(c.R, c.G, c.B, c.A)
	r.Clear()
}

// DrawRectangleCommand draws a filled rectangle.
type DrawRectangleCommand struct {
	X, Y, Width, Height float64
	R, G, B, A          float32
}

func (c DrawRectangleCommand) Execute(r Renderer) {
	r.SetColor(c.R, c.G, c.B, c.A)
	r.DrawRectangle(c.X, c.Y, c.Width, c.Height)
}

// DrawImageCommand draws an image at the given position.
type DrawImageCommand struct {
	Img  Image
	X, Y float64
}

func (c DrawImageCommand) Execute(r Renderer) {
	r.DrawImage(c.Img, c.X, c.Y)
}

// CommandBuffer is a platform-agnostic sequence of GPU commands.
type CommandBuffer struct {
	commands []Command
}

// NewCommandBuffer creates an empty command buffer.
func NewCommandBuffer() *CommandBuffer {
	return &CommandBuffer{}
}

// Clear adds a clear command to the buffer.
func (cb *CommandBuffer) Clear(r, g, b, a float32) {
	cb.commands = append(cb.commands, ClearCommand{R: r, G: g, B: b, A: a})
}

// DrawRectangle adds a filled rectangle command to the buffer.
func (cb *CommandBuffer) DrawRectangle(x, y, width, height float64, r, g, b, a float32) {
	cb.commands = append(cb.commands, DrawRectangleCommand{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
		R:      r,
		G:      g,
		B:      b,
		A:      a,
	})
}

// DrawImage adds an image draw command to the buffer.
func (cb *CommandBuffer) DrawImage(img Image, x, y float64) {
	cb.commands = append(cb.commands, DrawImageCommand{Img: img, X: x, Y: y})
}

// Execute replays all commands in the buffer through the given renderer.
func (cb *CommandBuffer) Execute(r Renderer) {
	for _, cmd := range cb.commands {
		cmd.Execute(r)
	}
}

// Len returns the number of commands in the buffer.
func (cb *CommandBuffer) Len() int {
	return len(cb.commands)
}

// Reset clears all commands from the buffer.
func (cb *CommandBuffer) Reset() {
	cb.commands = nil
}
