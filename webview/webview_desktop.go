//go:build !android && !ios

package webview

import (
	"fmt"

	"github.com/danfragoso/thdwb/hotdog"
	"github.com/danfragoso/thdwb/platform"
)

// desktopWebView implements WebView for desktop platforms using the thdwb
// rendering engine (hotdog) for content rendering. It bridges the WebView
// API with the platform's windowing and rendering abstractions.
type desktopWebView struct {
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

var _ WebView = (*desktopWebView)(nil)

func newDesktopWebView(config Config) (WebView, error) {
	return &desktopWebView{
		config:  config,
		url:     "",
		width:   config.Width,
		height:  config.Height,
		visible: false,
	}, nil
}

func (w *desktopWebView) LoadURL(url string) error {
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

func (w *desktopWebView) LoadHTML(html, baseURL string) error {
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

func (w *desktopWebView) SetSize(width, height int) {
	w.width = width
	w.height = height
	if w.window != nil {
		w.window.SetSize(width, height)
	}
}

func (w *desktopWebView) Size() (int, int) {
	return w.width, w.height
}

func (w *desktopWebView) Show() {
	w.visible = true
	if w.window != nil {
		w.window.Show()
	}
}

func (w *desktopWebView) Hide() {
	w.visible = false
	if w.window != nil {
		w.window.Hide()
	}
}

func (w *desktopWebView) Destroy() {
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

func (w *desktopWebView) EvaluateJS(script string) (string, error) {
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

func (w *desktopWebView) SetDelegate(delegate Delegate) {
	w.delegate = delegate
}

// BindWindow binds the webview to a platform window and hotdog window context
// for actual content rendering.
func (w *desktopWebView) BindWindow(window platform.Window, wc *hotdog.WindowContext) {
	w.window = window
	w.wc = wc
	if wc != nil {
		wc.Window = nil // will be set by the platform window integration
	}
}

// Compile-time check that platform.Window is used.
var _ = platform.Window(nil)