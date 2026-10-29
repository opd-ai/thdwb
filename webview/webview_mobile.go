//go:build android || ios

package webview

import (
	"fmt"

	"github.com/danfragoso/thdwb/hotdog"
	"github.com/danfragoso/thdwb/platform"
)

// mobileWebView implements WebView for Android and iOS using native WebView
// bindings via cgo/gomobile. It bridges the WebView API with the thdwb
// rendering engine (hotdog) for content rendering.
type mobileWebView struct {
	config    Config
	delegate  Delegate
	url       string
	width     int
	height    int
	visible   bool
	destroyed bool

	window   platform.Window
	renderer platform.Renderer
	wc       *hotdog.WindowContext
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
	}
	if w.wc != nil {
		if err := w.wc.LoadURL(url); err != nil {
			if w.delegate != nil {
				w.delegate.OnPageError(url, err)
			}
			return err
		}
	}
	if w.delegate != nil {
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
	}
	if w.wc != nil {
		if err := w.wc.LoadHTML(html, baseURL); err != nil {
			if w.delegate != nil {
				w.delegate.OnPageError(baseURL, err)
			}
			return err
		}
	}
	if w.delegate != nil {
		w.delegate.OnPageFinished(baseURL)
	}
	return nil
}

func (w *mobileWebView) SetSize(width, height int) {
	w.width = width
	w.height = height
	if w.window != nil {
		w.window.SetSize(width, height)
	}
}

func (w *mobileWebView) Size() (int, int) {
	return w.width, w.height
}

func (w *mobileWebView) Show() {
	w.visible = true
	if w.window != nil {
		w.window.Show()
	}
}

func (w *mobileWebView) Hide() {
	w.visible = false
	if w.window != nil {
		w.window.Hide()
	}
}

func (w *mobileWebView) Destroy() {
	w.destroyed = true
	w.visible = false
	if w.wc != nil {
		w.wc.Destroy()
		w.wc = nil
	}
	if w.window != nil {
		w.window.Close()
	}
}

func (w *mobileWebView) EvaluateJS(script string) (string, error) {
	if w.destroyed {
		return "", fmt.Errorf("webview: WebView is destroyed")
	}
	if w.wc == nil {
		return "", fmt.Errorf("webview: no window context available")
	}
	val, err := w.wc.RunJS(script)
	if err != nil {
		return "", err
	}
	return val.String(), nil
}

func (w *mobileWebView) SetDelegate(delegate Delegate) {
	w.delegate = delegate
}

// BindWindow binds the webview to a platform window and hotdog window context
// for actual content rendering.
func (w *mobileWebView) BindWindow(window platform.Window, wc *hotdog.WindowContext) {
	w.window = window
	w.wc = wc
}

// Compile-time check that platform.Window is used.
var _ = platform.Window(nil)
