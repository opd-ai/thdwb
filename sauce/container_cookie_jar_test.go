package sauce

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestContainerCookieJarSetAndGet(t *testing.T) {
	jar := NewContainerCookieJar()
	u, _ := url.Parse("https://example.com/page")

	cookies := []*http.Cookie{
		{Name: "session", Value: "abc123", Domain: "example.com", Path: "/"},
	}
	jar.SetCookies("c1", u, cookies)

	got := jar.Cookies("c1", u)
	if len(got) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(got))
	}
	if got[0].Name != "session" || got[0].Value != "abc123" {
		t.Fatalf("cookie = %+v, want session=abc123", got[0])
	}
}

func TestContainerCookieJarIsolation(t *testing.T) {
	jar := NewContainerCookieJar()
	u, _ := url.Parse("https://example.com/page")

	jar.SetCookies("c1", u, []*http.Cookie{{Name: "a", Value: "1", Domain: "example.com", Path: "/"}})
	jar.SetCookies("c2", u, []*http.Cookie{{Name: "b", Value: "2", Domain: "example.com", Path: "/"}})

	c1 := jar.Cookies("c1", u)
	if len(c1) != 1 || c1[0].Name != "a" {
		t.Fatalf("c1 cookies = %+v, want [a=1]", c1)
	}

	c2 := jar.Cookies("c2", u)
	if len(c2) != 1 || c2[0].Name != "b" {
		t.Fatalf("c2 cookies = %+v, want [b=2]", c2)
	}

	// Default container (empty ID) should be isolated too.
	jar.SetCookies("", u, []*http.Cookie{{Name: "default", Value: "d", Domain: "example.com", Path: "/"}})
	def := jar.Cookies("", u)
	if len(def) != 1 || def[0].Name != "default" {
		t.Fatalf("default cookies = %+v, want [default=d]", def)
	}
}

func TestContainerCookieJarClearContainer(t *testing.T) {
	jar := NewContainerCookieJar()
	u, _ := url.Parse("https://example.com/page")

	jar.SetCookies("c1", u, []*http.Cookie{{Name: "a", Value: "1", Domain: "example.com", Path: "/"}})
	jar.SetCookies("c2", u, []*http.Cookie{{Name: "b", Value: "2", Domain: "example.com", Path: "/"}})

	jar.ClearContainer("c1")

	if got := jar.Cookies("c1", u); len(got) != 0 {
		t.Fatalf("c1 should be empty after clear, got %d cookies", len(got))
	}
	if got := jar.Cookies("c2", u); len(got) != 1 {
		t.Fatalf("c2 should still have 1 cookie, got %d", len(got))
	}
}

func TestContainerCookieJarContainerCount(t *testing.T) {
	jar := NewContainerCookieJar()
	u1, _ := url.Parse("https://example.com/page")
	u2, _ := url.Parse("https://other.com/page")

	jar.SetCookies("c1", u1, []*http.Cookie{{Name: "a", Value: "1", Domain: "example.com", Path: "/"}})
	jar.SetCookies("c1", u2, []*http.Cookie{{Name: "b", Value: "2", Domain: "other.com", Path: "/"}})
	jar.SetCookies("c2", u1, []*http.Cookie{{Name: "c", Value: "3", Domain: "example.com", Path: "/"}})

	if got := jar.ContainerCount(); got != 2 {
		t.Fatalf("container count = %d, want 2", got)
	}
}

func TestContainerCookieJarSameSite(t *testing.T) {
	jar := NewContainerCookieJar()
	u, _ := url.Parse("https://example.com/page")
	initiator, _ := url.Parse("https://example.com/")

	// SameSite=Lax cookie should be sent for same-site top-level GET.
	jar.SetCookies("c1", u, []*http.Cookie{
		{Name: "lax", Value: "1", Domain: "example.com", Path: "/", SameSite: http.SameSiteLaxMode},
	})
	got := jar.CookiesForRequest("c1", u, initiator, true, http.MethodGet)
	if len(got) != 1 || got[0].Name != "lax" {
		t.Fatalf("expected lax cookie for same-site GET, got %+v", got)
	}

	// SameSite=Lax cookies are also sent on cross-site top-level GET (per spec).
	other, _ := url.Parse("https://evil.com/page")
	got = jar.CookiesForRequest("c1", u, other, true, http.MethodGet)
	if len(got) != 1 || got[0].Name != "lax" {
		t.Fatalf("expected lax cookie for cross-site top-level GET, got %+v", got)
	}

	// SameSite=Strict cookie should NOT be sent for cross-site request.
	jar2 := NewContainerCookieJar()
	jar2.SetCookies("c1", u, []*http.Cookie{
		{Name: "strict", Value: "1", Domain: "example.com", Path: "/", SameSite: http.SameSiteStrictMode},
	})
	got = jar2.CookiesForRequest("c1", u, other, true, http.MethodGet)
	if len(got) != 0 {
		t.Fatalf("expected no strict cookie for cross-site, got %+v", got)
	}

	// SameSite=Strict cookie should be sent for same-site request.
	got = jar2.CookiesForRequest("c1", u, initiator, true, http.MethodGet)
	if len(got) != 1 || got[0].Name != "strict" {
		t.Fatalf("expected strict cookie for same-site GET, got %+v", got)
	}
}

func TestContainerCookieJarExpiration(t *testing.T) {
	jar := NewContainerCookieJar()
	u, _ := url.Parse("https://example.com/page")

	// Expired cookie should not be returned.
	jar.SetCookies("c1", u, []*http.Cookie{
		{Name: "expired", Value: "1", Domain: "example.com", Path: "/", Expires: time.Now().Add(-time.Hour)},
	})
	got := jar.Cookies("c1", u)
	if len(got) != 0 {
		t.Fatalf("expected no expired cookies, got %+v", got)
	}
}
