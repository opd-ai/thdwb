package mustard

import (
	"image"
	"testing"

	gg "github.com/danfragoso/thdwb/gg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGameUpdateContainsCallbackPanic (audit C-12) proves a panic from the
// per-frame app callback -- which runs on the Ebitengine game goroutine
// outside Frame.draw's recover -- drops the frame instead of aborting the
// process. Before the fix, probe P20 killed the binary outright.
func TestGameUpdateContainsCallbackPanic(t *testing.T) {
	a := &ebitenApp{name: "app"}
	g := &ebitenGame{app: a}
	a.callback = func() { panic("logic exploded") }

	require.NotPanics(t, func() {
		err := g.Update()
		assert.NoError(t, err, "a paint panic must not terminate the game loop")
	})
	assert.Nil(t, g.frame, "the frame must be dropped on panic")

	// The loop stays alive: a later clean update still works.
	a.callback = func() {}
	require.NotPanics(t, func() { assert.NoError(t, g.Update()) })
}

// panickingBufferWidget simulates a leaf widget that blows up while being
// composited. compositeWidget/compositeAll have no recover of their own, so
// Update's recover is the only thing between such a panic and the process.
type panickingBufferWidget struct {
	baseWidget
}

func (w *panickingBufferWidget) Buffer() *image.RGBA {
	panic("widget buffer exploded")
}

func (w *panickingBufferWidget) draw() {}

// TestGameUpdateContainsCompositePanic drives a panic through the real render
// pass: the root frame is a visible widget whose child panics in Buffer().
func TestGameUpdateContainsCompositePanic(t *testing.T) {
	a := &ebitenApp{name: "app"}
	w := &WindowBase{title: "w", width: 10, height: 10, visible: true}
	w.context = gg.NewContext(10, 10)

	root := CreateFrame(VerticalFrame)
	root.SetVisible(true)
	child := &panickingBufferWidget{}
	child.baseWidget.visible = true
	child.baseWidget.needsRepaint = true
	child.baseWidget.computedBox.SetCoords(0, 0, 5, 5)
	root.AttachWidget(child)
	w.rootFrame = root

	a.windows = append(a.windows, w)
	g := &ebitenGame{app: a}

	require.NotPanics(t, func() {
		err := g.Update()
		assert.NoError(t, err)
	})
}

// TestGameDrawContainsPanic covers Draw: a nil screen with a non-nil frame
// panics inside DrawImage; the recover must contain it.
func TestGameDrawContainsPanic(t *testing.T) {
	g := &ebitenGame{app: &ebitenApp{name: "app"}}
	g.frame = image.NewRGBA(image.Rect(0, 0, 4, 4))

	require.NotPanics(t, func() { g.Draw(nil) })

	g.frame = nil
	require.NotPanics(t, func() { g.Draw(nil) })
}
