package mustard

import (
	"image"
	"image/draw"

	gg "github.com/danfragoso/thdwb/gg"
)

// render composites the window's current widget tree into its frame buffer and
// returns that buffer.
//
// Feature: Window paint pass
// Purpose: Run the reflow/redraw cycle and composite every widget (and overlay)
// into the window's frame buffer so the platform driver can blit it to screen.
// Dependencies: gg (2D context), frame.go (drawRootFrame), window.go (compositeAll)
// Limitations: GPU-independent by design; the platform driver is responsible for
// presenting the returned buffer. The buffer is the context's backing image and
// is reused across frames until the window is resized.
func (window *WindowBase) render() *image.RGBA {
	if window.context == nil {
		window.context = gg.NewContext(window.width, window.height)
		window.needsReflow = true
	}

	context := window.context.(*gg.Context)
	window.frameBuffer = context.Image().(*image.RGBA)

	if window.rootFrame == nil {
		return window.frameBuffer
	}

	if window.needsReflow {
		drawRootFrame(window)
		window.needsReflow = false
	} else {
		redrawWidgets(window.rootFrame)
	}

	compositeAll(window.frameBuffer, window.rootFrame)

	compositeOverlays(window, window.frameBuffer, window.staticOverlays)
	compositeOverlays(window, window.frameBuffer, window.overlays)

	return window.frameBuffer
}

// compositeOverlays blits each overlay's buffer onto frame at its position,
// copying the current frame first so a static overlay never bleeds into the
// live context image.
func compositeOverlays(window *WindowBase, frame *image.RGBA, overlays []*Overlay) {
	if len(overlays) == 0 {
		return
	}
	nBuffer := image.NewRGBA(frame.Bounds())
	draw.Draw(nBuffer, frame.Bounds(), frame, image.Point{}, draw.Over)
	window.frameBuffer = nBuffer
	for _, overlay := range overlays {
		if overlay.buffer == nil {
			continue
		}
		draw.Draw(window.frameBuffer, overlay.buffer.Bounds().Add(overlay.position), overlay.buffer, image.Point{}, draw.Over)
	}
}
