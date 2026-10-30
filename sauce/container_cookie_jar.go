package sauce

import (
	"net/http"
	"net/url"
	"sync"
	"time"
)

// DefaultContainerID is the sentinel container ID used for the default
// (non-container) browsing context.
const DefaultContainerID = ""

// ContainerCookieJar implements http.CookieJar with per-container, per-origin
// cookie partitioning. Each (containerID, origin) pair gets its own isolated
// cookie store, enabling Firefox-style container tab cookie isolation.
// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies
type ContainerCookieJar struct {
	mu      sync.RWMutex
	cookies map[string][]*http.Cookie // key: containerID + "::" + originKey
}

// NewContainerCookieJar creates a new container-partitioned cookie jar.
func NewContainerCookieJar() *ContainerCookieJar {
	return &ContainerCookieJar{
		cookies: make(map[string][]*http.Cookie),
	}
}

// containerKey returns the jar key for a container ID and URL.
func containerKey(containerID string, u *url.URL) string {
	if u == nil {
		return ""
	}
	return containerID + "::" + originKey(u)
}

// SetCookies stores cookies for a container+origin in the partitioned cookie jar.
// Called automatically when processing HTTP responses.
// Filters cookies by validity and SameSite rules.
func (jar *ContainerCookieJar) SetCookies(containerID string, u *url.URL, cookies []*http.Cookie) {
	if u == nil {
		return
	}
	key := containerKey(containerID, u)
	if key == "" {
		return
	}
	jar.mu.Lock()
	defer jar.mu.Unlock()
	for _, cookie := range cookies {
		candidate, ok := prepareCookie(u, cookie)
		if !ok {
			continue
		}
		if candidate.MaxAge < 0 || (!candidate.Expires.IsZero() && !candidate.Expires.After(time.Now())) {
			jar.removeCookie(key, candidate)
			continue
		}
		if !isValidCookieForURL(u, candidate) {
			continue
		}
		jar.upsertCookie(key, candidate)
	}
}

// Cookies returns cookies for a container+origin request.
func (jar *ContainerCookieJar) Cookies(containerID string, u *url.URL) []*http.Cookie {
	return jar.CookiesForRequest(containerID, u, u, true, http.MethodGet)
}

// CookiesForRequest returns cookies for a container+origin request, applying SameSite rules.
// Parameters:
//
//	containerID - the container ID for cookie partitioning
//	u - request URL
//	initiator - initiating site URL (for SameSite comparison)
//	topLevel - whether this is a top-level navigation
//	method - HTTP method (GET, POST, etc.)
//
// Returns filtered cookies that should be sent with the request.
func (jar *ContainerCookieJar) CookiesForRequest(containerID string, u, initiator *url.URL, topLevel bool, method string) []*http.Cookie {
	jar.mu.RLock()
	defer jar.mu.RUnlock()
	if u == nil {
		return nil
	}
	key := containerKey(containerID, u)
	cookies := jar.cookies[key]
	if len(cookies) == 0 {
		return nil
	}
	result := make([]*http.Cookie, 0, len(cookies))
	for _, cookie := range cookies {
		if !isValidCookieForURL(u, cookie) || !sameSiteAllows(cookie, u, initiator, topLevel, method) {
			continue
		}
		result = append(result, cloneCookie(cookie))
	}
	return result
}

// removeCookie removes a cookie from the container+origin store.
func (jar *ContainerCookieJar) removeCookie(key string, cookie *http.Cookie) {
	kept := jar.cookies[key][:0]
	for _, existing := range jar.cookies[key] {
		if !cookieIdentity(existing, cookie) {
			kept = append(kept, existing)
		}
	}
	jar.cookies[key] = kept
}

// upsertCookie inserts or updates a cookie in the container+origin store.
func (jar *ContainerCookieJar) upsertCookie(key string, cookie *http.Cookie) {
	for i, existing := range jar.cookies[key] {
		if cookieIdentity(existing, cookie) {
			jar.cookies[key][i] = cookie
			return
		}
	}
	jar.cookies[key] = append(jar.cookies[key], cookie)
}

// ClearContainer removes all cookies for the given container.
func (jar *ContainerCookieJar) ClearContainer(containerID string) {
	jar.mu.Lock()
	defer jar.mu.Unlock()
	for key := range jar.cookies {
		if len(key) > len(containerID) && key[:len(containerID)] == containerID && key[len(containerID)] == ':' {
			delete(jar.cookies, key)
		}
	}
}

// ContainerCount returns the number of distinct containers with cookies.
func (jar *ContainerCookieJar) ContainerCount() int {
	jar.mu.RLock()
	defer jar.mu.RUnlock()
	containers := make(map[string]struct{})
	for key := range jar.cookies {
		// key format: containerID + "::" + originKey
		for i := 0; i < len(key); i++ {
			if key[i] == ':' && i+1 < len(key) && key[i+1] == ':' {
				containers[key[:i]] = struct{}{}
				break
			}
		}
	}
	return len(containers)
}

// defaultContainerCookieJar adapts a ContainerCookieJar to the http.CookieJar
// interface using the default (non-container) container ID.
type DefaultContainerCookieJar struct {
	jar *ContainerCookieJar
}

// NewDefaultContainerCookieJar wraps a ContainerCookieJar as an http.CookieJar
// using the default (non-container) container ID.
func NewDefaultContainerCookieJar(jar *ContainerCookieJar) *DefaultContainerCookieJar {
	return &DefaultContainerCookieJar{jar: jar}
}

// SetCookies implements http.CookieJar for the default container.
func (j *DefaultContainerCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.jar.SetCookies(DefaultContainerID, u, cookies)
}

// Cookies implements http.CookieJar for the default container.
func (j *DefaultContainerCookieJar) Cookies(u *url.URL) []*http.Cookie {
	return j.jar.Cookies(DefaultContainerID, u)
}
