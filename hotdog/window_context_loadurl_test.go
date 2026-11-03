package hotdog

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danfragoso/thdwb/assets"
	profiler "github.com/danfragoso/thdwb/profiler"
)

// newWindowContextForLoadURL builds a fresh window context for LoadURL tests.
func newWindowContextForLoadURL(t *testing.T) *WindowContext {
	t.Helper()
	settings := &Settings{}
	buildInfo := &assets.BuildInfo{}
	prof := profiler.CreateProfiler()
	return NewWindowContext(settings, buildInfo, prof)
}

// TestLoadURLSurfacesTransportError asserts the "stop swallowing fetch
// errors" half of P0#2 at the navigation API level: an unreachable host
// must make LoadURL return a non-nil error, not a silent success. 127.0.0.1:1
// has no listener, so the failure is a deterministic connection-refused with
// no network egress and no DNS. Loopback is not upgraded to HTTPS, so this
// is a plain HTTP fetch.
func TestLoadURLSurfacesTransportError(t *testing.T) {
	wc := newWindowContextForLoadURL(t)
	err := wc.LoadURL("http://127.0.0.1:1/unreachable")
	require.Error(t, err, "LoadURL must surface transport failures, not swallow them")
	assert.Contains(t, err.Error(), "transport error", "error must name the transport failure")
}

// TestLoadURLSurfacesHTTPErrorStatus asserts that a real non-2xx response is
// surfaced as an HTTP error rather than loaded as a page.
func TestLoadURLSurfacesHTTPErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	wc := newWindowContextForLoadURL(t)
	err := wc.LoadURL(srv.URL + "/missing")
	require.Error(t, err, "LoadURL must surface non-2xx responses as errors")
	assert.Contains(t, err.Error(), "HTTP 404", "error must carry the HTTP status")
}

// TestLoadURLLoadsLocalhostPage asserts the happy path: LoadURL fetches a
// localhost page and populates the active document, so the navigation
// pipeline works end-to-end on a local host.
func TestLoadURLLoadsLocalhostPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><head><title>thdwb-test</title></head><body><h1>hello</h1></body></html>"))
	}))
	defer srv.Close()

	wc := newWindowContextForLoadURL(t)
	require.NoError(t, wc.LoadURL(srv.URL+"/page"))
	require.NotNil(t, wc.ActiveDocument)
	require.NotNil(t, wc.ActiveDocument.URL)
	assert.Contains(t, wc.ActiveDocument.URL.String(), "/page", "document URL must reflect the fetched page")
	assert.NotEmpty(t, wc.Documents, "the loaded page must be recorded in the document list")
}

// TestLoadURLRealHost is an opt-in integration test that re-verifies the
// navigation pipeline against a real non-localhost host, per the audit.
// Gated on THDWB_NETWORK=1 so the default CI run stays green and fast.
func TestLoadURLRealHost(t *testing.T) {
	if os.Getenv("THDWB_NETWORK") != "1" {
		t.Skip("set THDWB_NETWORK=1 to run the real-host integration test")
	}

	wc := newWindowContextForLoadURL(t)
	require.NoError(t, wc.LoadURL("https://example.com/"))
	require.NotNil(t, wc.ActiveDocument)
	require.NotNil(t, wc.ActiveDocument.URL)
	assert.Contains(t, wc.ActiveDocument.URL.Hostname(), "example.com",
		"loaded document should be the example.com page")
}
