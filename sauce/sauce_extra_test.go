package sauce

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danfragoso/thdwb/assets"
)

func TestHistoryNavigation(t *testing.T) {
	parse := func(s string) *url.URL {
		u, err := url.Parse(s)
		require.NoError(t, err)
		return u
	}
	h := &History{}
	assert.Nil(t, h.Last())
	assert.Empty(t, h.NextPages())
	assert.Empty(t, h.AllPages())
	assert.Zero(t, h.PageCount())

	// Pop/PopNext on empty history are no-ops.
	h.Pop()
	h.PopNext()

	h.Push(parse("https://a.test/1"))
	h.Push(parse("https://a.test/2"))
	assert.Equal(t, 2, h.PageCount())
	assert.Equal(t, "https://a.test/2", h.Last().String())
	assert.Len(t, h.AllPages(), 2)

	// Pop moves 2 to next; PopNext restores it.
	h.Pop()
	assert.Len(t, h.NextPages(), 1)
	assert.Equal(t, "https://a.test/1", h.Last().String())
	h.PopNext()
	assert.Empty(t, h.NextPages())
	assert.Equal(t, "https://a.test/2", h.Last().String())

	// Push clears forward history.
	h.Pop()
	require.Len(t, h.NextPages(), 1)
	h.Push(parse("https://a.test/3"))
	assert.Empty(t, h.NextPages())
	assert.Equal(t, "https://a.test/3", h.Last().String())
}

func TestFetchInternalPages(t *testing.T) {
	history := &History{}
	history.Push(mustParseURL(t, "https://a.test/"))
	buildInfo := &assets.BuildInfo{GitRevision: "abc123", GitBranch: "main", HostInfo: "host", BuildTime: "now"}

	internal := func(host string) *Resource {
		u, err := url.Parse("thdwb://" + host)
		require.NoError(t, err)
		return fetchInternalPage(u, history, buildInfo)
	}

	homepage := internal("homepage")
	assert.NotEmpty(t, homepage.Body)
	assert.True(t, strings.Contains(homepage.Body, "<"))

	historyPage := internal("history")
	assert.Contains(t, historyPage.Body, "https://a.test/")

	aboutPage := internal("about")
	assert.Contains(t, aboutPage.Body, "abc123")

	other := internal("somethingelse")
	assert.NotEmpty(t, other.Body)

	// GetResource file scheme renders the file browser without network.
	fileURL, err := url.Parse("file:///some/path")
	require.NoError(t, err)
	fileRes := GetResource(fileURL, nil, nil)
	assert.Equal(t, "file", fileRes.URL.Scheme)
	assert.Contains(t, fileRes.Body, "some/path")
}

func TestGetResourceEmptySchemeDefaultsToHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("scheme-test"))
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL + "/empty-scheme")
	require.NoError(t, err)
	u.Scheme = "" // simulate a schemeless URL
	resource := GetResource(u, nil, nil)
	assert.Equal(t, "scheme-test", resource.Body)
}

func TestFetchExternalPageWithOptionsCachesGET(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = fmt.Fprintf(w, "hit-%d", calls)
	}))
	defer srv.Close()

	// Unique path so the global cache never matches a previous test.
	first, err := url.Parse(srv.URL + "/cache-check-1")
	require.NoError(t, err)
	resource := fetchExternalPageWithOptions(nil, first, "GET", nil, nil)
	assert.Equal(t, "hit-1", resource.Body)
	assert.Equal(t, 200, resource.Code)

	resource = fetchExternalPageWithOptions(nil, first, "GET", nil, nil)
	assert.Equal(t, "hit-1", resource.Body, "second call should hit the cache")
	assert.Equal(t, 1, calls)

	// POST with body bypasses the cache.
	second, err := url.Parse(srv.URL + "/cache-check-2")
	require.NoError(t, err)
	resource = fetchExternalPageWithOptions(nil, second, "POST", strings.NewReader("x"), map[string]string{"Content-Type": "text/plain"})
	assert.Equal(t, "hit-2", resource.Body)
}

func TestFetchExternalPageWithOptionsInvalidRequest(t *testing.T) {
	// Opaque URL that String() renders with a control character: url.Parse
	// accepts it but http.NewRequest rejects it.
	bad := &url.URL{Scheme: "http", Opaque: "\r"}
	resource := fetchExternalPageWithOptions(nil, bad, "GET", nil, nil)
	assert.Contains(t, resource.Body, "Error")
}

func TestCORSRequestHelpers(t *testing.T) {
	sawOrigin := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") == "https://initiator.test" {
			sawOrigin = true
		}
		_, _ = w.Write([]byte("cors-ok"))
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL + "/cors-check")
	require.NoError(t, err)
	resource := MakeCORSRequestWithClient(nil, u, "https://initiator.test", "POST", strings.NewReader("body"), nil)
	assert.Equal(t, "cors-ok", resource.Body)
	assert.True(t, sawOrigin, "Origin header should be sent")

	// Deprecated helpers route through the default client.
	u2, err := url.Parse(srv.URL + "/cors-check-2")
	require.NoError(t, err)
	resource = MakeCORSRequest(u2, "https://initiator.test", "GET", nil, nil)
	assert.Equal(t, "cors-ok", resource.Body)

	resource = FetchExternalPageWithOptions(u2, "GET", nil, nil)
	assert.Equal(t, "cors-ok", resource.Body)
}

func TestParseURL(t *testing.T) {
	good := ParseURL("https://example.test/page?q=1")
	assert.Equal(t, "example.test", good.Hostname())

	bad := ParseURL("http://bad host with spaces.example")
	assert.Equal(t, "thdwb", bad.Scheme)
	assert.Equal(t, "error", bad.Host)
}

func TestLoadErrorPageAndLog(t *testing.T) {
	page := loadErrorPage("something broke")
	assert.Contains(t, page, "something broke")
	assert.Contains(t, page, "<h1>")

	// Log prints to stdout; just exercise it.
	Log("sauce", "hello")
}

func TestCookieJarEdgeCases(t *testing.T) {
	// prepareCookie edge cases.
	u, err := url.Parse("https://example.test/a/b/c")
	require.NoError(t, err)
	candidate, ok := prepareCookie(nil, &http.Cookie{Name: "x"})
	assert.False(t, ok)
	assert.Nil(t, candidate)
	candidate, ok = prepareCookie(u, nil)
	assert.False(t, ok)
	assert.Nil(t, candidate)
	_, ok = prepareCookie(u, &http.Cookie{Name: ""})
	assert.False(t, ok)

	// SameSite=None without Secure is rejected.
	_, ok = prepareCookie(u, &http.Cookie{Name: "x", SameSite: http.SameSiteNoneMode})
	assert.False(t, ok)

	// Pathless cookie gets the directory default path.
	candidate, ok = prepareCookie(u, &http.Cookie{Name: "x"})
	require.True(t, ok)
	assert.Equal(t, "/a/b", candidate.Path)

	// Domain that is not a suffix of host is rejected.
	_, ok = prepareCookie(u, &http.Cookie{Name: "x", Domain: "other.test"})
	assert.False(t, ok)

	// defaultCookiePath edge cases.
	assert.Equal(t, "/", defaultCookiePath(""))
	assert.Equal(t, "/", defaultCookiePath("relative"))
	assert.Equal(t, "/", defaultCookiePath("/root"))
	assert.Equal(t, "/a/b", defaultCookiePath("/a/b/c"))
	assert.Equal(t, "/a", defaultCookiePath("/a/b"))

	// cloneCookie.
	assert.Nil(t, cloneCookie(nil))
	clone := cloneCookie(&http.Cookie{Name: "n"})
	assert.Equal(t, "n", clone.Name)

	// cookieIdentity.
	assert.False(t, cookieIdentity(nil, &http.Cookie{Name: "x"}))
	assert.False(t, cookieIdentity(&http.Cookie{Name: "x"}, nil))
	assert.True(t, cookieIdentity(&http.Cookie{Name: "x", Domain: "d", Path: "/"}, &http.Cookie{Name: "x", Domain: "d", Path: "/"}))
	assert.False(t, cookieIdentity(&http.Cookie{Name: "x"}, &http.Cookie{Name: "y"}))

	// SetCookies on nil URL is a no-op; upsert replaces same identity.
	jar := NewOriginCookieJar()
	jar.SetCookies(nil, []*http.Cookie{{Name: "x"}})
	origin, err := url.Parse("https://cookie-edge.test/")
	require.NoError(t, err)
	jar.SetCookies(origin, []*http.Cookie{{Name: "session", Value: "one"}})
	jar.SetCookies(origin, []*http.Cookie{{Name: "session", Value: "two"}})
	cookies := jar.Cookies(origin)
	require.Len(t, cookies, 1)
	assert.Equal(t, "two", cookies[0].Value)

	// Expiring a cookie removes it.
	jar.SetCookies(origin, []*http.Cookie{{Name: "session", Value: "gone", Expires: time.Now().Add(-time.Minute)}})
	assert.Empty(t, jar.Cookies(origin))
}

func TestHSTSStoreEdgeCases(t *testing.T) {
	var store *HSTSStore
	store.ProcessHeader("example.test", "max-age=100", true) // nil store no-op
	assert.False(t, store.ShouldUpgrade("example.test"))

	s := NewHSTSStore()
	// Invalid hosts are ignored.
	for _, host := range []string{"", "127.0.0.1", "not a host"} {
		s.ProcessHeader(host, "max-age=100", true)
	}
	// Insecure responses are ignored.
	s.ProcessHeader("hsts-edge.test", "max-age=100", false)
	assert.False(t, s.ShouldUpgrade("hsts-edge.test"))

	// No max-age -> no policy.
	s.ProcessHeader("hsts-edge.test", "includeSubdomains", true)
	assert.False(t, s.ShouldUpgrade("hsts-edge.test"))

	// Negative or non-numeric max-age -> rejected.
	s.ProcessHeader("hsts-edge.test", "max-age=-1", true)
	s.ProcessHeader("hsts-edge.test", "max-age=abc", true)
	s.ProcessHeader("hsts-edge.test", "max-age=", true)
	assert.False(t, s.ShouldUpgrade("hsts-edge.test"))

	// max-age=0 removes an existing policy.
	s.ProcessHeader("hsts-edge.test", "max-age=60; includeSubdomains", true)
	assert.True(t, s.ShouldUpgrade("hsts-edge.test"))
	s.ProcessHeader("hsts-edge.test", "max-age=0", true)
	assert.False(t, s.ShouldUpgrade("hsts-edge.test"))

	// Expired entries are cleaned up by ShouldUpgrade.
	s.ProcessHeader("hsts-edge.test", "max-age=1", true)
	time.Sleep(1100 * time.Millisecond)
	assert.False(t, s.ShouldUpgrade("hsts-edge.test"))
}

func TestOriginKeyAndURLCloning(t *testing.T) {
	assert.Equal(t, "", originKey(nil))
	https, err := url.Parse("https://example.test")
	require.NoError(t, err)
	assert.Equal(t, "https://example.test:443", originKey(https))
	http8080, err := url.Parse("http://example.test:8080/x")
	require.NoError(t, err)
	assert.Equal(t, "http://example.test:8080", originKey(http8080))

	assert.Nil(t, cloneURL(nil))
	src, err := url.Parse("https://example.test/p?q=1")
	require.NoError(t, err)
	cloned := cloneURL(src)
	assert.Equal(t, "q=1", cloned.RawQuery)
}

func TestRequestClientVariants(t *testing.T) {
	assert.Nil(t, requestClient(nil, mustParseURL(t, "https://x.test"), ""))
	plain := mustParseURL(t, "https://x.test")
	// Nil request URL: returns the original client untouched.
	same := &http.Client{}
	assert.Same(t, same, requestClient(same, nil, ""))
	// Non-nil URL: returns a new client with a transport and no jar.
	cloned := requestClient(same, plain, "")
	require.NotNil(t, cloned)
	assert.NotSame(t, same, cloned)
	assert.Nil(t, cloned.Jar)
	assert.NotNil(t, cloned.Transport)
	// A client without an OriginCookieJar gets a nil jar transport.
	c := requestClient(&http.Client{Jar: NewOriginCookieJar()}, plain, "https://init.test")
	require.NotNil(t, c)
	assert.Nil(t, c.Jar)
	assert.NotNil(t, c.Transport)
}

func TestNewVerifiedTransportErrors(t *testing.T) {
	store, err := NewTOFUStore("")
	require.NoError(t, err)
	_, err = newVerifiedTransport(nil, store, "example.test")
	assert.Error(t, err)
	_, err = newVerifiedTransport(&fakeTransport{}, store, "example.test")
	assert.Error(t, err)
}

type fakeTransport struct{}

func (fakeTransport) RoundTrip(*http.Request) (*http.Response, error) { return nil, nil }

// makeTestServerCertificate is like makeTestCertificate but also returns the
// private key so the certificate can serve TLS connections.
func makeTestServerCertificate(t *testing.T, hostname string, notAfter time.Time) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: hostname},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if ip := net.ParseIP(hostname); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{hostname}
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	certificate, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return certificate, key
}

func TestOriginCookieTransportHTTPSWithTOFU(t *testing.T) {
	certificate, key := makeTestServerCertificate(t, "127.0.0.1", time.Now().Add(time.Hour))
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("secure-ok"))
	}))
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{certificate.Raw}, PrivateKey: key}}}
	srv.StartTLS()
	defer srv.Close()

	store, err := NewTOFUStore("")
	require.NoError(t, err)
	require.NoError(t, store.TrustCertificate("127.0.0.1", certificate))

	transport := &originCookieTransport{
		jar:  NewOriginCookieJar(),
		base: http.DefaultTransport,
		tofu: store,
		hsts: NewHSTSStore(),
	}
	req, err := http.NewRequest("GET", srv.URL+"/secure", nil)
	require.NoError(t, err)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "secure-ok", string(body))

	// The per-host transport is cached.
	cached, err := transport.httpsTransport("127.0.0.1")
	require.NoError(t, err)
	assert.NotNil(t, cached)
}

func TestOriginCookieTransportHTTPAndResponseProcessing(t *testing.T) {
	setCookie := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") == "sid=abc" {
			setCookie = true
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "sid", Value: "abc"})
		w.Header().Set("Strict-Transport-Security", "max-age=60")
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL + "/page")
	require.NoError(t, err)
	transport := &originCookieTransport{
		jar:       NewOriginCookieJar(),
		base:      http.DefaultTransport,
		hsts:      NewHSTSStore(),
		initiator: u,
		topLevel:  true,
	}
	// Pre-seed a cookie so the response processing round trip is observable.
	transport.jar.SetCookies(u, []*http.Cookie{{Name: "sid", Value: "abc"}})

	req, err := http.NewRequest("GET", srv.URL+"/page", nil)
	require.NoError(t, err)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.True(t, setCookie, "jar cookie should be attached to the request")

	// OPTIONS preflight skips cookie attachment but still works.
	optsReq, err := http.NewRequest("OPTIONS", srv.URL+"/page", nil)
	require.NoError(t, err)
	optsResp, err := transport.RoundTrip(optsReq)
	require.NoError(t, err)
	_ = optsResp.Body.Close()

	// HSTS header from an http response is not recorded (only https is).
	assert.False(t, transport.hsts.ShouldUpgrade(u.Hostname()))
}

func mustParseURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	require.NoError(t, err)
	return u
}
