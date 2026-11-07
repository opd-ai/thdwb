package mustard

import (
	"testing"

	gg "github.com/danfragoso/thdwb/gg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateImageWidgetUndecodableDegrades (audit C-18) feeds CreateImageWidget
// bytes that no image decoder can read. log.Fatal there used to exit the whole
// browser process on a content error; the constructor must instead return a
// usable widget backed by a placeholder image.
func TestCreateImageWidgetUndecodableDegrades(t *testing.T) {
	w := CreateImageWidget([]byte("this is definitely not a PNG"))
	require.NotNil(t, w, "undecodable bytes must not kill the process or return nil")
	assert.NotNil(t, w.img, "the widget must carry a placeholder image for draw()")

	// draw() must survive the placeholder path end to end.
	win := &WindowBase{title: "w", width: 20, height: 20, visible: true}
	win.context = gg.NewContext(20, 20)
	w.SetWindow(win)
	w.computedBox.SetCoords(0, 0, 10, 10)
	require.NotPanics(t, w.draw)
}

// TestCreateImageWidgetValidStillDecodes guards the happy path.
func TestCreateImageWidgetValidStillDecodes(t *testing.T) {
	w := CreateImageWidget(makeTestPNG(t))
	require.NotNil(t, w)
	require.NotNil(t, w.img)
	b := w.img.Bounds()
	assert.Equal(t, 8, b.Dx())
	assert.Equal(t, 8, b.Dy())
}
