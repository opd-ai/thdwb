package sauce

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`OK!`))
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

		w.Write(decoded)
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
