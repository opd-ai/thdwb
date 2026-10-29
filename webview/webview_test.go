// Package webview_test provides unit tests for the webview package.
package webview

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// testDelegate records delegate events for assertion.
type testDelegate struct {
	started   []string
	finished  []string
	errors    []struct{ url string; err error }
	permissions []string
}

func (d *testDelegate) OnPageStarted(url string) {
	d.started = append(d.started, url)
}

func (d *testDelegate) OnPageFinished(url string) {
	d.finished = append(d.finished, url)
}

func (d *testDelegate) OnPageError(url string, err error) {
	d.errors = append(d.errors, struct{ url string; err error }{url, err})
}

func (d *testDelegate) OnPermissionRequest(permission string) bool {
	d.permissions = append(d.permissions, permission)
	return true
}

func TestNewWebViewStub(t *testing.T) {
	wv, err := NewWebView(Config{
		Width:             800,
		Height:            600,
		UserAgent:         "thdwb/1.0",
		JavaScriptEnabled: true,
		SandboxEnabled:    true,
	})
	require.NoError(t, err)
	require.NotNil(t, wv)

	w, h := wv.Size()
	require.Equal(t, 800, w)
	require.Equal(t, 600, h)
}

func TestStubWebViewLoadURL(t *testing.T) {
	wv, err := NewWebView(Config{})
	require.NoError(t, err)

	delegate := &testDelegate{}
	wv.SetDelegate(delegate)

	err = wv.LoadURL("https://example.com")
	require.NoError(t, err)
	require.Contains(t, delegate.started, "https://example.com")
	require.Contains(t, delegate.finished, "https://example.com")
}

func TestStubWebViewLoadURLDestroyed(t *testing.T) {
	wv, err := NewWebView(Config{})
	require.NoError(t, err)
	wv.Destroy()
	err = wv.LoadURL("https://example.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "destroyed")
}

func TestStubWebViewLoadHTML(t *testing.T) {
	wv, err := NewWebView(Config{})
	require.NoError(t, err)

	delegate := &testDelegate{}
	wv.SetDelegate(delegate)

	err = wv.LoadHTML("<html><body>Hello</body></html>", "https://example.com")
	require.NoError(t, err)
	require.Contains(t, delegate.started, "https://example.com")
	require.Contains(t, delegate.finished, "https://example.com")
}

func TestStubWebViewSetSize(t *testing.T) {
	wv, err := NewWebView(Config{})
	require.NoError(t, err)
	wv.SetSize(800, 600)
	w, h := wv.Size()
	require.Equal(t, 800, w)
	require.Equal(t, 600, h)
}

func TestStubWebViewShowHide(t *testing.T) {
	wv, err := NewWebView(Config{})
	require.NoError(t, err)
	wv.Show()
	wv.Hide()
	// Show/Hide should not error on stub implementation
}

func TestStubWebViewDestroy(t *testing.T) {
	wv, err := NewWebView(Config{})
	require.NoError(t, err)
	wv.Destroy()

	// Operations after destroy should fail.
	err = wv.LoadURL("https://example.com")
	require.Error(t, err)

	_, err = wv.EvaluateJS("1+1")
	require.Error(t, err)
}

func TestStubWebViewEvaluateJS(t *testing.T) {
	wv, err := NewWebView(Config{})
	require.NoError(t, err)
	// Without a bound window context, EvaluateJS should return an error.
	_, err = wv.EvaluateJS("1 + 1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no window context available")
}

func TestStubWebViewSetDelegate(t *testing.T) {
	wv, err := NewWebView(Config{})
	require.NoError(t, err)
	delegate := &testDelegate{}
	wv.SetDelegate(delegate)
	// Delegate should be set without error; verify via LoadURL.
	err = wv.LoadURL("https://example.com")
	require.NoError(t, err)
	require.Contains(t, delegate.started, "https://example.com")
}

func TestEmbed(t *testing.T) {
	// Embed should return a WebView with the configured size.
	wv, err := Embed(nil, Config{Width: 400, Height: 300})
	require.NoError(t, err)
	require.NotNil(t, wv)

	w, h := wv.Size()
	require.Equal(t, 400, w)
	require.Equal(t, 300, h)
}

func TestConfigDefaults(t *testing.T) {
	// Zero-value Config should be usable.
	wv, err := NewWebView(Config{})
	require.NoError(t, err)
	require.NotNil(t, wv)

	w, h := wv.Size()
	require.Equal(t, 0, w)
	require.Equal(t, 0, h)
}

func TestWebViewInterfaceCompileTime(t *testing.T) {
	// Verify stubWebView implements WebView.
	var _ WebView = (*stubWebView)(nil)

	// Verify Delegate interface is satisfied by testDelegate.
	var _ Delegate = (*testDelegate)(nil)
}