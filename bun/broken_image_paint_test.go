package bun

import (
	"image"
	"net/http"
	"net/url"
	"testing"

	gg "github.com/danfragoso/thdwb/gg"
	"github.com/danfragoso/thdwb/hotdog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPaintInlineBrokenImageDoesNotPanic (audit R-13) verifies an <img> whose
// fetch fails degrades to the placeholder. paintInlineElement used to discard
// the placeholder decode error with `im, _, _ =`, so if the stand-in ever
// yielded a nil image it was passed straight to DrawImage.
func TestPaintInlineBrokenImageDoesNotPanic(t *testing.T) {
	pageURL, err := url.Parse("https://example.com/index.html")
	require.NoError(t, err)
	doc := &hotdog.Document{URL: pageURL}
	headers := make(http.Header)
	headers.Set("Content-Security-Policy", "img-src 'self'")
	doc.ApplyContentSecurityPolicyHeaders(headers)

	node := &hotdog.NodeDOM{
		Type:       hotdog.NodeTypeElement,
		Element:    "img",
		Attributes: []*hotdog.Attribute{{Name: "src", Value: "https://evil.example/image.png"}},
		Document:   doc,
		Style: &hotdog.Stylesheet{
			Display:        "inline",
			BackgroundColor: &hotdog.ColorRGBA{R: 1, G: 1, B: 1, A: 1},
		},
		RenderBox: &hotdog.RenderBox{Left: 0, Top: 0, Width: 32, Height: 32},
	}

	ctx := gg.NewContext(100, 100)
	assert.NotPanics(t, func() { paintInlineElement(ctx, node) })
}

// TestDrawImageNilImagePanics documents the precise hazard the fix guards:
// gg.Context.DrawImage dereferences src.Bounds(), so a nil image panics. The
// production guard returns before this call whenever the placeholder decode
// yields nil, which is why the broken-image path above never reaches it.
func TestDrawImageNilImagePanics(t *testing.T) {
	ctx := gg.NewContext(10, 10)
	var im image.Image // nil interface
	assert.Panics(t, func() { ctx.DrawImage(im, 0, 0) })
}
