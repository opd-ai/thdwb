package sauce

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests pin down the P0 audit recommendation #2: the sauce transport
// must work against a real non-localhost host, and transport failures must
// not be swallowed as a "success".

// newTLSLoopbackServer starts a self-signed TLS HTTP server on 127.0.0.1.
// The self-signed cert is deliberately NOT trusted, so a fetch against it
// must fail with a TLS validation error (proof the request reached the
// wire) rather than a transport-configuration error (proof the transport
// was mis-built before any connection was attempted).
func newTLSLoopbackServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	cert, key := makeTestServerCertificate(t, "127.0.0.1", time.Now().Add(time.Hour))
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{cert.Raw}, PrivateKey: key}}}
	srv.StartTLS()
	return srv, srv.Close
}

// TestFetchExternalPageDoesNotDoubleWrapTransport is the regression test for
// the P0#2 "configurable *http.Transport" failure. The double-wrap bug made
// newVerifiedTransport fail its *http.Transport assertion for every HTTPS
// request, breaking every non-localhost host. The guard unwraps an
// already-wrapped client to its base before re-wrapping, so a fetch against
// a reachable TLS server must fail only at TLS validation (untrusted
// self-signed cert), never at transport configuration.
func TestFetchExternalPageDoesNotDoubleWrapTransport(t *testing.T) {
	srv, shutdown := newTLSLoopbackServer(t)
	defer shutdown()

	u, err := url.Parse(srv.URL + "/probe")
	require.NoError(t, err)

	// A client whose Transport is already an *originCookieTransport, as
	// produced by a prior fetchExternalPageWithOptions call. The guard must
	// reuse its base transport rather than nest a second wrapper.
	prewrapped := &http.Client{Transport: &originCookieTransport{
		jar:  NewOriginCookieJar(),
		base: http.DefaultTransport,
		hsts: NewHSTSStore(),
	}}

	resource := fetchExternalPageWithOptions(prewrapped, u, "GET", nil, nil)

	assert.Equal(t, -1, resource.Code, "untrusted-cert failure must set the -1 sentinel Code")
	assert.NotContains(t, resource.Body, "configurable *http.Transport",
		"double-wrap bug: transport configuration failed before any connection")
	assert.NotContains(t, resource.Body, "configure HTTPS transport",
		"double-wrap bug: transport configuration failed before any connection")
}

// TestRequestClientUnwrapsExistingTransport asserts the guard directly:
// wrapping a client that already carries an *originCookieTransport reuses
// the underlying base transport instead of nesting wrappers.
func TestRequestClientUnwrapsExistingTransport(t *testing.T) {
	base := &http.Transport{}
	existing := &originCookieTransport{
		jar:  NewOriginCookieJar(),
		base: base,
		hsts: NewHSTSStore(),
	}
	client := &http.Client{Jar: NewOriginCookieJar(), Transport: existing}

	u, err := url.Parse("https://example.test/")
	require.NoError(t, err)

	cloned := requestClient(client, u, "")
	require.NotNil(t, cloned)
	require.NotNil(t, cloned.Transport)

	wrapped, ok := cloned.Transport.(*originCookieTransport)
	require.True(t, ok, "transport must be a single originCookieTransport, not nested")
	assert.Same(t, base, wrapped.base, "underlying *http.Transport must be reused, not wrapped")
}

// TestFetchExternalPageTransportErrorSurfaces asserts the "stop swallowing
// fetch errors" half of P0#2: a failed transport leaves the -1 sentinel
// Code and an error page that names the underlying cause, never a success.
func TestFetchExternalPageTransportErrorSurfaces(t *testing.T) {
	// 127.0.0.1:1 has no listener: a deterministic connection-refused, no
	// network egress, no DNS. Loopback is not upgraded to HTTPS by
	// upgradeHTTPURL, so this stays a plain HTTP fetch.
	u, err := url.Parse("http://127.0.0.1:1/nothing")
	require.NoError(t, err)

	resource := fetchExternalPageWithOptions(nil, u, "GET", nil, nil)

	assert.Equal(t, -1, resource.Code, "connection failure must set the -1 sentinel Code")
	assert.Contains(t, resource.Body, "Error!", "error page must be populated")
	assert.Contains(t, resource.Body, "connection refused", "underlying error must be surfaced")
}

// TestFetchExternalPageRealHostHTTPS is an opt-in integration test that
// re-verifies the transport against a real non-localhost host, per the
// audit. Gated on THDWB_NETWORK=1 so the default CI run (go test -race
// work, no network allowance) stays green and fast.
func TestFetchExternalPageRealHostHTTPS(t *testing.T) {
	if os.Getenv("THDWB_NETWORK") != "1" {
		t.Skip("set THDWB_NETWORK=1 to run the real-host integration test")
	}

	u, err := url.Parse("https://example.com/")
	require.NoError(t, err)

	resource := fetchExternalPageWithOptions(nil, u, "GET", nil, nil)

	assert.Equal(t, 200, resource.Code, "real-host HTTPS fetch must succeed")
	assert.NotEmpty(t, resource.Body, "real-host HTTPS fetch must return a body")
	assert.Contains(t, strings.ToLower(resource.Body), "example domain",
		"body should be the example.com page")
}
