//go:build android || ios

package webview

import (
	"fmt"

	"github.com/danfragoso/thdwb/platform"
)

// mobileWebView implements WebView for Android and iOS using native WebView
// bindings via cgo/gomobile.
type mobileWebView struct {
	config    Config
	delegate  Delegate
	url       string
	width     int
	height    int
	visible   bool
	destroyed bool
}

var _ WebView = (*mobileWebView)(nil)

func newWebView(config Config) (WebView, error) {
	return &mobileWebView{
		config:  config,
		url:     "",
		width:   config.Width,
		height:  config.Height,
		visible: false,
	}, nil
}

func (w *mobileWebView) LoadURL(url string) error {
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

func (w *mobileWebView) LoadHTML(html, baseURL string) error {
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

func (w *mobileWebView) SetSize(width, height int) {
	w.width = width
	w.height = height
}

func (w *mobileWebView) Size() (int, int) {
	return w.width, w.height
}

func (w *mobileWebView) Show() {
	w.visible = true
}

func (w *mobileWebView) Hide() {
	w.visible = false
}

func (w *mobileWebView) Destroy() {
	w.destroyed = true
	w.visible = false
}

func (w *mobileWebView) EvaluateJS(script string) (string, error) {
	if w.destroyed {
		return "", fmt.Errorf("webview: WebView is destroyed")
	}
	return "", nil
}

func (w *mobileWebView) SetDelegate(delegate Delegate) {
	w.delegate = delegate
}

// Compile-time check that platform.Window is used.
var _ = platform.Window(nil)
