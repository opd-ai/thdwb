//go:build !android && !ios

package webview

import (
	"github.com/danfragoso/thdwb/platform"
)

// stubWebView implements WebView for desktop platforms without native WebView
// bindings. It delegates to the desktopWebView implementation which integrates
// with the thdwb rendering engine (hotdog) for actual content rendering.
type stubWebView struct {
	desktop *desktopWebView
}

var _ WebView = (*stubWebView)(nil)

func newWebView(config Config) (WebView, error) {
	dwv, err := newDesktopWebView(config)
	if err != nil {
		return nil, err
	}
	return &stubWebView{desktop: dwv.(*desktopWebView)}, nil
}

func (w *stubWebView) LoadURL(url string) error {
	return w.desktop.LoadURL(url)
}

func (w *stubWebView) LoadHTML(html, baseURL string) error {
	return w.desktop.LoadHTML(html, baseURL)
}

func (w *stubWebView) SetSize(width, height int) {
	w.desktop.SetSize(width, height)
}

func (w *stubWebView) Size() (int, int) {
	return w.desktop.Size()
}

func (w *stubWebView) Show() {
	w.desktop.Show()
}

func (w *stubWebView) Hide() {
	w.desktop.Hide()
}

func (w *stubWebView) Destroy() {
	w.desktop.Destroy()
}

func (w *stubWebView) EvaluateJS(script string) (string, error) {
	return w.desktop.EvaluateJS(script)
}

func (w *stubWebView) SetDelegate(delegate Delegate) {
	w.desktop.SetDelegate(delegate)
}

// Compile-time check that platform.Window is used.
var _ = platform.Window(nil)
