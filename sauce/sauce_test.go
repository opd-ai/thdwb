package sauce

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = rw.Write([]byte(`OK!`))
	}))

	defer server.Close()

	url, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	resource := GetResource(url, nil, nil)
	var testResource *Resource

	assert.IsType(t, testResource, resource, "Expecting: Resource Struct")
	assert.Equal(t, "OK!", resource.Body, "Expecting: OK!")
}

func TestGetImage(t *testing.T) {
	// This is a base64-encoded empty PNG, from https://stackoverflow.com/a/31035384
	emptyImage := "iVBORw0KGgoAAAANSUhEUgAAAGQAAABkCAYA"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoded, err := base64.RawStdEncoding.DecodeString(emptyImage)
		require.NoError(t, err)

		_, _ = w.Write(decoded)
	}))
	defer srv.Close()

	testCases := []struct {
		name      string
		url       string
		expectLen int
	}{
		{"base64 encoded", "data:image/png;base64," + emptyImage, 27},
		{"malformed base64 encoded 1", "data:image/png;base64:" + emptyImage, 0},
		{"malformed base64 encoded 2", "data:image/png;base64", 0},
		{"short URL", "http://foo", 0},
		{"regular URL", srv.URL + "/this-is-a-relatively-long-url", 27},
		{"not an HTTP URL", "//foo-bar-fnord-asdf-bla-bla", 0},
		{"empty URL", "", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := url.Parse(tc.url)
			require.NoError(t, err)

			var data []byte

			assert.NotPanics(t, func() {
				data, _ = GetImage(u)
			})

			assert.Len(t, data, tc.expectLen)
		})
	}
}

func TestOriginCookieJarMatchesSecureAndSameSiteRules(t *testing.T) {
	jar := NewOriginCookieJar()

	httpsURL, err := url.Parse("https://example.com/path")
	require.NoError(t, err)

	jar.SetCookies(httpsURL, []*http.Cookie{
		{Name: "secure", Value: "1", Secure: true, Path: "/path"},
		{Name: "same_site_none", Value: "1", SameSite: http.SameSiteNoneMode, Secure: true, Path: "/"},
		{Name: "allowed", Value: "2", Path: "/"},
	})

	cookies := jar.Cookies(httpsURL)
	require.Len(t, cookies, 3)
	assert.Contains(t, []string{cookies[0].Name, cookies[1].Name, cookies[2].Name}, "allowed")
	assert.Contains(t, []string{cookies[0].Name, cookies[1].Name, cookies[2].Name}, "secure")
	assert.Contains(t, []string{cookies[0].Name, cookies[1].Name, cookies[2].Name}, "same_site_none")

	httpURL, err := url.Parse("http://example.com/path")
	require.NoError(t, err)
	assert.Empty(t, jar.Cookies(httpURL))

	badPathURL, err := url.Parse("https://example.com/other")
	require.NoError(t, err)
	pathCookies := jar.Cookies(badPathURL)
	require.Len(t, pathCookies, 2)
	assert.NotContains(t, []string{pathCookies[0].Name, pathCookies[1].Name}, "secure")

	crossDomainURL, err := url.Parse("https://api.example.com/path")
	require.NoError(t, err)
	jar.SetCookies(crossDomainURL, []*http.Cookie{{Name: "subdomain", Value: "1", Domain: "example.com", Path: "/"}})
	assert.Len(t, jar.Cookies(crossDomainURL), 1)
	assert.Equal(t, "subdomain", jar.Cookies(crossDomainURL)[0].Name)
}

func TestOriginCookieJarAppliesSameSiteRequestContext(t *testing.T) {
	jar := NewOriginCookieJar()
	target, err := url.Parse("https://api.example.com/data")
	require.NoError(t, err)
	site, err := url.Parse("https://www.example.com")
	require.NoError(t, err)
	crossSite, err := url.Parse("https://attacker.test")
	require.NoError(t, err)

	jar.SetCookies(target, []*http.Cookie{
		{Name: "strict", Value: "1", SameSite: http.SameSiteStrictMode, Path: "/"},
		{Name: "lax", Value: "1", SameSite: http.SameSiteLaxMode, Path: "/"},
		{Name: "default", Value: "1", Path: "/"},
		{Name: "none", Value: "1", SameSite: http.SameSiteNoneMode, Secure: true, Path: "/"},
	})

	require.Len(t, jar.CookiesForRequest(target, site, false, http.MethodPost), 4)
	assertCookieNames(t, jar.CookiesForRequest(target, crossSite, false, http.MethodGet), "none")
	assertCookieNames(t, jar.CookiesForRequest(target, crossSite, true, http.MethodGet), "lax", "default", "none")
	assertCookieNames(t, jar.CookiesForRequest(target, crossSite, true, http.MethodPost), "none")
}

func TestOriginCookieTransportFiltersCrossSiteCookies(t *testing.T) {
	received := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Get("Cookie")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	target, err := url.Parse(server.URL + "/resource")
	require.NoError(t, err)
	initiator, err := url.Parse("https://attacker.test")
	require.NoError(t, err)
	jar := NewOriginCookieJar()
	jar.SetCookies(target, []*http.Cookie{{Name: "lax", Value: "1", SameSite: http.SameSiteLaxMode, Path: "/"}})

	client := requestClient(&http.Client{Jar: jar}, target, initiator.String())
	response, err := client.Get(target.String())
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	assert.Empty(t, <-received)
}

func TestOriginCookieJarDeletesExpiredCookies(t *testing.T) {
	jar := NewOriginCookieJar()
	target, err := url.Parse("https://example.com/path")
	require.NoError(t, err)
	jar.SetCookies(target, []*http.Cookie{{Name: "session", Value: "active", Path: "/"}})
	jar.SetCookies(target, []*http.Cookie{{Name: "session", Path: "/", MaxAge: -1}})
	assert.Empty(t, jar.Cookies(target))
}

func TestOriginCookieJarValidatesCookieDomainAndDefaultPath(t *testing.T) {
	jar := NewOriginCookieJar()
	target, err := url.Parse("https://shop.example.com/catalog/item")
	require.NoError(t, err)
	jar.SetCookies(target, []*http.Cookie{
		{Name: "public_suffix", Value: "bad", Domain: "com", Path: "/"},
		{Name: "catalog", Value: "good", Domain: "example.com", Path: "invalid"},
	})

	inScope, err := url.Parse("https://shop.example.com/catalog/child")
	require.NoError(t, err)
	assertCookieNames(t, jar.Cookies(inScope), "catalog")
	outOfScope, err := url.Parse("https://shop.example.com/catalogue")
	require.NoError(t, err)
	assert.Empty(t, jar.Cookies(outOfScope))
}

func TestHSTSStoreRequiresSecureResponseAndAppliesSubdomains(t *testing.T) {
	store := NewHSTSStore()
	store.ProcessHeader("example.com", "max-age=3600; includeSubDomains", false)
	assert.False(t, store.ShouldUpgrade("example.com"))

	store.ProcessHeader("example.com", "max-age=3600; includeSubDomains", true)
	assert.True(t, store.ShouldUpgrade("example.com"))
	assert.True(t, store.ShouldUpgrade("shop.example.com"))
	assert.False(t, store.ShouldUpgrade("notexample.com"))

	store.ProcessHeader("example.com", "max-age=0", true)
	assert.False(t, store.ShouldUpgrade("example.com"))
	assert.False(t, store.ShouldUpgrade("shop.example.com"))
}

func TestHSTSStoreRejectsInvalidPolicies(t *testing.T) {
	store := NewHSTSStore()
	for _, header := range []string{"includeSubDomains", "max-age=-1", "max-age=invalid"} {
		store.ProcessHeader("example.com", header, true)
	}
	store.ProcessHeader("com", "max-age=3600", true)
	store.ProcessHeader("127.0.0.1", "max-age=3600", true)
	assert.False(t, store.ShouldUpgrade("example.com"))
	assert.False(t, store.ShouldUpgrade("127.0.0.1"))
}

func TestUpgradeHTTPURLRequiresHTTPSExceptLoopback(t *testing.T) {
	testCases := []struct {
		input string
		want  string
	}{
		{"http://example.com/path", "https://example.com/path"},
		{"http://example.com:80/path", "https://example.com:443/path"},
		{"http://example.com:8080/path", "https://example.com:8080/path"},
		{"http://localhost/path", "http://localhost/path"},
		{"http://127.0.0.1/path", "http://127.0.0.1/path"},
		{"https://example.com/path", "https://example.com/path"},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			requestURL, err := url.Parse(tc.input)
			require.NoError(t, err)
			upgradeHTTPURL(requestURL, NewHSTSStore())
			assert.Equal(t, tc.want, requestURL.String())
		})
	}
}

func TestHSTSUpgradesLoopbackWhenExplicitlySet(t *testing.T) {
	store := NewHSTSStore()
	store.ProcessHeader("localhost", "max-age=3600", true)
	requestURL, err := url.Parse("http://localhost/resource")
	require.NoError(t, err)
	upgradeHTTPURL(requestURL, store)
	assert.Equal(t, "https://localhost/resource", requestURL.String())
}

func TestHTTPTransportUpgradesURLAndCachesHSTS(t *testing.T) {
	store := NewHSTSStore()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=3600")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	store.ProcessHeader("example.com", "max-age=3600", true)
	dialer := &net.Dialer{}
	base := &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, server.Listener.Addr().String())
		},
	}
	tofu, err := NewTOFUStore("")
	require.NoError(t, err)
	require.NoError(t, tofu.TrustCertificate("example.com", server.Certificate()))
	transport := &originCookieTransport{
		hsts: store, base: base, tofu: tofu,
	}
	request, err := http.NewRequest(http.MethodGet, "http://example.com:"+serverURL.Port()+"/resource", nil)
	require.NoError(t, err)
	response, err := (&http.Client{Transport: transport}).Do(request)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	assert.Equal(t, "https://example.com:"+serverURL.Port()+"/resource", response.Request.URL.String())
	assert.True(t, store.ShouldUpgrade("example.com"))
}

func assertCookieNames(t *testing.T, cookies []*http.Cookie, want ...string) {
	t.Helper()
	names := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		names = append(names, cookie.Name)
	}
	assert.ElementsMatch(t, want, names)
}

func TestWebRequestObserverRegistration(t *testing.T) {
	called := false
	obs := func(details WebRequestDetails) (WebRequestDetails, error) {
		called = true
		assert.Equal(t, "GET", details.Method)
		return details, nil
	}
	id := RegisterWebRequestObserver(obs)
	require.GreaterOrEqual(t, id, 0)

	// Verify observer is invoked
	_, err := notifyWebRequestObservers(WebRequestDetails{Method: "GET"})
	require.NoError(t, err)
	require.True(t, called)

	// Unregister and verify it's removed
	UnregisterWebRequestObserver(id)
	called = false
	_, err = notifyWebRequestObservers(WebRequestDetails{Method: "GET"})
	require.NoError(t, err)
	require.False(t, called)
}

func TestWebRequestObserverModifyHeaders(t *testing.T) {
	obs := func(details WebRequestDetails) (WebRequestDetails, error) {
		if details.Headers == nil {
			details.Headers = make(map[string]string)
		}
		details.Headers["X-Custom"] = "value"
		return details, nil
	}
	id := RegisterWebRequestObserver(obs)
	defer UnregisterWebRequestObserver(id)

	details, err := notifyWebRequestObservers(WebRequestDetails{Method: "GET"})
	require.NoError(t, err)
	require.Equal(t, "value", details.Headers["X-Custom"])
}

func TestWebRequestObserverBlock(t *testing.T) {
	obs := func(details WebRequestDetails) (WebRequestDetails, error) {
		return details, fmt.Errorf("blocked")
	}
	id := RegisterWebRequestObserver(obs)
	defer UnregisterWebRequestObserver(id)

	_, err := notifyWebRequestObservers(WebRequestDetails{Method: "GET"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "blocked")
}
