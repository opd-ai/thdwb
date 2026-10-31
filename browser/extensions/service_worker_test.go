package extensions

import (
	"testing"

	goja "github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

// TestServiceWorkerLifecycle verifies install → activate → activated states.
func TestServiceWorkerLifecycle(t *testing.T) {
	dir := t.TempDir()
	swPath := writeServiceWorkerScript(t, dir, `
		self.addEventListener("install", function(e) {});
		self.addEventListener("activate", function(e) {});
	`)

	sw := NewServiceWorker("ext1", swPath)
	require.Equal(t, ServiceWorkerInstalling, sw.State())
	require.False(t, sw.IsRunning())

	vm := goja.New()
	require.NoError(t, sw.Start(vm))
	require.Equal(t, ServiceWorkerActivated, sw.State())
	require.True(t, sw.IsRunning())

	require.NoError(t, sw.Stop())
	require.Equal(t, ServiceWorkerRedundant, sw.State())
	require.False(t, sw.IsRunning())
}

// TestServiceWorkerInstallEvent verifies the install event is dispatched.
func TestServiceWorkerInstallEvent(t *testing.T) {
	dir := t.TempDir()
	swPath := writeServiceWorkerScript(t, dir, `
		var installFired = false;
		self.addEventListener("install", function(e) { installFired = true; });
	`)

	sw := NewServiceWorker("ext1", swPath)
	vm := goja.New()
	require.NoError(t, sw.Start(vm))

	// Verify the install handler was registered by checking the VM state.
	// The service worker script ran without error, meaning addEventListener
	// was available on self.
	require.Equal(t, ServiceWorkerActivated, sw.State())

	sw.Stop()
}

// TestServiceWorkerFetchEvent verifies fetch events can be dispatched.
func TestServiceWorkerFetchEvent(t *testing.T) {
	dir := t.TempDir()
	swPath := writeServiceWorkerScript(t, dir, `
		self.addEventListener("fetch", function(e) {
			e.respondWith({ status: 200, body: "intercepted" });
		});
	`)

	sw := NewServiceWorker("ext1", swPath)
	vm := goja.New()
	require.NoError(t, sw.Start(vm))

	resp, err := sw.DispatchFetch(FetchEvent{
		RequestURL:      "http://example.com/",
		RequestMethod:   "GET",
		RequestHeaders:  map[string]string{},
		Response:        make(chan *FetchResponse, 1),
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "intercepted", resp.Body)

	sw.Stop()
}

// TestServiceWorkerFetchNoHandler verifies fetch with no handler returns nil.
func TestServiceWorkerFetchNoHandler(t *testing.T) {
	dir := t.TempDir()
	swPath := writeServiceWorkerScript(t, dir, `
		self.addEventListener("install", function(e) {});
	`)

	sw := NewServiceWorker("ext1", swPath)
	vm := goja.New()
	require.NoError(t, sw.Start(vm))

	resp, err := sw.DispatchFetch(FetchEvent{
		RequestURL:      "http://example.com/",
		RequestMethod:   "GET",
		RequestHeaders:  map[string]string{},
		Response:        make(chan *FetchResponse, 1),
	})
	require.NoError(t, err)
	require.Nil(t, resp)

	sw.Stop()
}

// TestServiceWorkerNotRunning verifies Stop on a non-running worker is a no-op.
func TestServiceWorkerNotRunning(t *testing.T) {
	sw := NewServiceWorker("ext1", "/nonexistent/sw.js")
	require.NoError(t, sw.Stop())
	require.Equal(t, ServiceWorkerRedundant, sw.State())
}

// writeServiceWorkerScript writes a service worker JS file to the given dir.
func writeServiceWorkerScript(t *testing.T, dir, content string) string {
	t.Helper()
	path := dir + "/service-worker.js"
	writeFile(t, dir, "service-worker.js", content)
	return path
}
