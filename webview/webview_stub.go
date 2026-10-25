//go:build !android && !ios

package webview

import (
	"fmt"

	"github.com/danfragoso/thdwb/platform"
)

// stubWebView implements WebView for desktop platforms without native WebView
// bindings. It provides a placeholder that can be extended with platform-specific
// WebView implementations (WebKitGTK on Linux, WebKit on macOS, WebView2 on Windows).
type stubWebView struct {
	config    Config
	delegate  Delegate
	url       string
	width     int
	height    int
	visible   bool
	destroyed bool
}

var _ WebView = (*stubWebView)(nil)

func newWebView(config Config) (WebView, error) {
	return &stubWebView{
		config:  config,
		url:     "",
		width:   config.Width,
		height:  config.Height,
		visible: false,
	}, nil
}

func (w *stubWebView) LoadURL(url string) error {
	if w.destroyed {
		return fmt.Errorf("webview: WebView is destroyed")
	}
	w.url = url
	if w.delegate != nil {
		w.delegate.OnPageStarted(url)
		w.delegate.OnPageFinished(url)
	}
	return nil
}

func (w *stubWebView) LoadHTML(html, baseURL string) error {
	if w.destroyed {
		return fmt.Errorf("webview: WebView is destroyed")
	}
	w.url = baseURL
	if w.delegate != nil {
		w.delegate.OnPageStarted(baseURL)
		w.delegate.OnPageFinished(baseURL)
	}
	return nil
}

func (w *stubWebView) SetSize(width, height int) {
	w.width = width
	w.height = height
}

func (w *stubWebView) Size() (int, int) {
	return w.width, w.height
}

func (w *stubWebView) Show() {
	w.visible = true
}

func (w *stubWebView) Hide() {
	w.visible = false
}

func (w *stubWebView) Destroy() {
	w.destroyed = true
	w.visible = false
}

func (w *stubWebView) EvaluateJS(script string) (string, error) {
	if w.destroyed {
		return "", fmt.Errorf("webview: WebView is destroyed")
	}
	return "", nil
}

func (w *stubWebView) SetDelegate(delegate Delegate) {
	w.delegate = delegate
}

// Compile-time check that platform.Window is used.
var _ = platform.Window(nil)
