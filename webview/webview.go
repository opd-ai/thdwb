// Package webview provides an embeddable native WebView layer for in-app
// browsing and hybrid app workflows, enabling sandboxed execution with shared
// browser policy enforcement across Android, iOS, and desktop platforms.
package webview

import (
	"github.com/danfragoso/thdwb/platform"
)

// WebView is an embeddable browser view that renders web content within a
// native application window.
type WebView interface {
	// LoadURL loads the specified URL into the WebView.
	LoadURL(url string) error
	// LoadHTML loads the specified HTML content into the WebView.
	LoadHTML(html, baseURL string) error
	// SetSize sets the WebView dimensions.
	SetSize(width, height int)
	// Size returns the WebView dimensions.
	Size() (int, int)
	// Show displays the WebView.
	Show()
	// Hide hides the WebView.
	Hide()
	// Destroy releases all resources associated with the WebView.
	Destroy()
	// EvaluateJS executes JavaScript in the WebView context.
	EvaluateJS(script string) (string, error)
	// SetDelegate sets the delegate for WebView events.
	SetDelegate(delegate Delegate)
}

// Delegate handles WebView events.
type Delegate interface {
	// OnPageStarted is called when a page starts loading.
	OnPageStarted(url string)
	// OnPageFinished is called when a page finishes loading.
	OnPageFinished(url string)
	// OnPageError is called when a page fails to load.
	OnPageError(url string, err error)
	// OnPermissionRequest is called when a page requests a permission.
	OnPermissionRequest(permission string) bool
}

// Config holds WebView configuration options.
type Config struct {
	// Width and height of the WebView.
	Width, Height int
	// UserAgent sets the user agent string.
	UserAgent string
	// JavaScriptEnabled enables or disables JavaScript execution.
	JavaScriptEnabled bool
	// SandboxEnabled enables sandboxed execution.
	SandboxEnabled bool
}

// NewWebView creates a new WebView using the platform's native implementation.
func NewWebView(config Config) (WebView, error) {
	return newWebView(config)
}

// Embed embeds a WebView into a platform window.
func Embed(w platform.Window, config Config) (WebView, error) {
	webView, err := NewWebView(config)
	if err != nil {
		return nil, err
	}
	webView.SetSize(config.Width, config.Height)
	return webView, nil
}
