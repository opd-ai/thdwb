package sauce

import (
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/publicsuffix"

	"github.com/danfragoso/thdwb/assets"
	pages "github.com/danfragoso/thdwb/pages"
)

// OriginCookieJar implements http.CookieJar with per-origin cookie partitioning.
// Each origin (scheme://host:port) gets its own isolated cookie store.
// This prevents cross-site tracking and matches Firefox's behavior.
// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies
// See: https://searchfox.org/mozilla-central/source/netwerk/cookie/nsCookieService.cpp
type OriginCookieJar struct {
	mu      sync.RWMutex
	cookies map[string][]*http.Cookie // key: origin (scheme://host:port)
}

// hstsEntry represents a single HSTS policy for a hostname.
type hstsEntry struct {
	expires           time.Time
	includeSubdomains bool
}

// HSTSStore caches Strict-Transport-Security policies by hostname.
// When a site sends an HSTS header, the browser remembers to only use HTTPS for that host.
// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Strict-Transport-Security
// See: https://searchfox.org/mozilla-central/source/netwerk/protocol/http/HSTSService.cpp
type HSTSStore struct {
	mu      sync.Mutex
	entries map[string]hstsEntry
}

// NewHSTSStore creates an empty HSTS policy store.
func NewHSTSStore() *HSTSStore {
	return &HSTSStore{entries: make(map[string]hstsEntry)}
}

// Global HSTS store shared across requests (for default client).
var hstsStore = NewHSTSStore()

// NewOriginCookieJar creates a new origin-partitioned cookie jar.
// Each origin gets its own cookie storage, preventing cross-site tracking.
func NewOriginCookieJar() *OriginCookieJar {
	return &OriginCookieJar{
		cookies: make(map[string][]*http.Cookie),
	}
}

// ProcessHeader records a valid HSTS header received over a secure connection.
// Parses max-age and includeSubdomains directives.
// Called automatically when processing HTTPS responses.
func (store *HSTSStore) ProcessHeader(host, header string, secure bool) {
	if store == nil || !secure {
		return
	}
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if host == "" {
		return
	}
	if net.ParseIP(host) != nil {
		return
	}
	if !strings.EqualFold(host, "localhost") {
		if _, err := publicsuffix.EffectiveTLDPlusOne(host); err != nil {
			return
		}
	}
	var maxAge *int64
	includeSubdomains := false
	for _, directive := range strings.Split(header, ";") {
		name, value, hasValue := strings.Cut(strings.TrimSpace(directive), "=")
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "max-age":
			if !hasValue {
				return
			}
			seconds, err := strconv.ParseInt(strings.Trim(strings.TrimSpace(value), `"`), 10, 64)
			if err != nil || seconds < 0 {
				return
			}
			maxAge = &seconds
		case "includesubdomains":
			if !hasValue {
				includeSubdomains = true
			}
		}
	}
	if maxAge == nil {
		return
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if *maxAge == 0 {
		delete(store.entries, host)
		return
	}
	lifetime := time.Duration(*maxAge) * time.Second
	if *maxAge > int64((1<<63-1)/int64(time.Second)) {
		lifetime = time.Duration(1<<63 - 1)
	}
	store.entries[host] = hstsEntry{
		expires:           time.Now().Add(lifetime),
		includeSubdomains: includeSubdomains,
	}
}

// ShouldUpgrade reports whether HSTS applies to host.
// Returns true if the host has an active HSTS policy (not expired).
// Automatically cleans up expired entries.
func (store *HSTSStore) ShouldUpgrade(host string) bool {
	if store == nil {
		return false
	}
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	store.mu.Lock()
	defer store.mu.Unlock()
	now := time.Now()
	for domain, entry := range store.entries {
		if !entry.expires.After(now) {
			delete(store.entries, domain)
		}
	}
	for domain, entry := range store.entries {
		if host == domain || entry.includeSubdomains && strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

// upgradeHTTPURL upgrades HTTP URLs to HTTPS when HSTS policy applies.
// Modifies the URL in place if HSTS requires HTTPS for the host.
// Skips localhost and loopback addresses unless HSTS explicitly applies.
func upgradeHTTPURL(requestURL *url.URL, store *HSTSStore) {
	if requestURL == nil || !strings.EqualFold(requestURL.Scheme, "http") {
		return
	}
	host := requestURL.Hostname()
	ip := net.ParseIP(host)
	localHost := strings.EqualFold(host, "localhost") || ip != nil && ip.IsLoopback()
	if !localHost || store != nil && store.ShouldUpgrade(host) {
		requestURL.Scheme = "https"
		if requestURL.Port() == "80" {
			requestURL.Host = net.JoinHostPort(host, "443")
		} else if port := requestURL.Port(); port != "" {
			requestURL.Host = net.JoinHostPort(host, port)
		}
	}
}

// originKey returns the origin key for a URL (scheme://host:port).
// Used as the key in OriginCookieJar for cookie partitioning.
// Returns empty string for nil URL.
func originKey(u *url.URL) string {
	if u == nil {
		return ""
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return u.Scheme + "://" + u.Hostname() + ":" + port
}

// normalizeDomain normalizes a domain for comparison.
// Trims whitespace, leading dots, and converts to lowercase.
func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	domain = strings.TrimPrefix(domain, ".")
	return strings.ToLower(domain)
}

// domainMatches checks if a host matches a cookie domain.
// Implements RFC 6265 domain matching rules.
// Returns true if host == domain or host is a subdomain of domain.
func domainMatches(host, domain string) bool {
	host = normalizeDomain(host)
	domain = normalizeDomain(domain)
	if domain == "" {
		return host != ""
	}
	if host == domain {
		return true
	}
	return strings.HasSuffix(host, "."+domain)
}

// pathMatches checks if a request path matches a cookie path.
// Implements RFC 6265 path matching rules.
// Cookie path "/" matches all paths.
func pathMatches(requestURL *url.URL, cookiePath string) bool {
	if requestURL == nil {
		return false
	}
	requestPath := requestURL.EscapedPath()
	if requestPath == "" {
		requestPath = "/"
	}
	if cookiePath == "" {
		cookiePath = "/"
	}
	if cookiePath == "/" {
		return true
	}
	if !strings.HasPrefix(requestPath, cookiePath) {
		return false
	}
	if cookiePath[len(cookiePath)-1] == '/' {
		return true
	}
	if requestPath == cookiePath {
		return true
	}
	return strings.HasPrefix(requestPath, cookiePath+"/")
}

// isValidCookieForURL validates a cookie against a request URL.
// Checks: Secure flag, SameSite=None requires Secure, domain match, path match, expiration.
// Returns true if cookie should be sent with request to URL.
func isValidCookieForURL(u *url.URL, cookie *http.Cookie) bool {
	if u == nil || cookie == nil {
		return false
	}
	if cookie.Name == "" {
		return false
	}
	if cookie.Secure && !strings.EqualFold(u.Scheme, "https") {
		return false
	}
	if cookie.SameSite == http.SameSiteNoneMode && !cookie.Secure {
		return false
	}
	if !domainMatches(u.Hostname(), cookie.Domain) {
		return false
	}
	if !pathMatches(u, cookie.Path) {
		return false
	}
	if !cookie.Expires.IsZero() && cookie.Expires.Before(time.Now()) {
		return false
	}
	if cookie.MaxAge < 0 {
		return false
	}
	return true
}

// SetCookies stores cookies for an origin in the partitioned cookie jar.
// Called automatically when processing HTTP responses.
// Filters cookies by validity and SameSite rules.
func (jar *OriginCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	if u == nil {
		return
	}
	key := originKey(u)
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

func (jar *OriginCookieJar) Cookies(u *url.URL) []*http.Cookie {
	return jar.CookiesForRequest(u, u, true, http.MethodGet)
}

// CookiesForRequest returns cookies for a request, applying SameSite rules.
// Parameters:
//
//	u - request URL
//	initiator - initiating site URL (for SameSite comparison)
//	topLevel - whether this is a top-level navigation
//	method - HTTP method (GET, POST, etc.)
//
// Returns filtered cookies that should be sent with the request.
// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite
func (jar *OriginCookieJar) CookiesForRequest(u, initiator *url.URL, topLevel bool, method string) []*http.Cookie {
	jar.mu.RLock()
	defer jar.mu.RUnlock()
	if u == nil {
		return nil
	}
	key := originKey(u)
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

// prepareCookie normalizes and validates a cookie for storage.
// Sets default path if missing, normalizes domain, validates SameSite=None requires Secure.
// Returns normalized cookie and true if valid for storage.
func prepareCookie(u *url.URL, cookie *http.Cookie) (*http.Cookie, bool) {
	if u == nil || cookie == nil || cookie.Name == "" {
		return nil, false
	}
	candidate := cloneCookie(cookie)
	candidate.Domain = normalizeDomain(candidate.Domain)
	if candidate.Path == "" || !strings.HasPrefix(candidate.Path, "/") {
		candidate.Path = defaultCookiePath(u.EscapedPath())
	}
	if candidate.SameSite == http.SameSiteNoneMode && !candidate.Secure {
		return nil, false
	}
	return candidate, validCookieDomain(u.Hostname(), candidate.Domain)
}

// validCookieDomain checks if a cookie domain is valid for a host.
// Implements RFC 6265: domain must be a suffix of host, and must be a registrable domain
// (or exact match for host-only cookies).
func validCookieDomain(host, domain string) bool {
	if !domainMatches(host, domain) {
		return false
	}
	if domain == "" || normalizeDomain(host) == domain {
		return true
	}
	_, err := publicsuffix.EffectiveTLDPlusOne(domain)
	return err == nil
}

// defaultCookiePath computes the default cookie path from a request path.
// Returns the directory portion of the path (e.g., "/foo/bar" -> "/foo").
func defaultCookiePath(requestPath string) string {
	if requestPath == "" || requestPath[0] != '/' {
		return "/"
	}
	if lastSlash := strings.LastIndex(requestPath, "/"); lastSlash > 0 {
		return requestPath[:lastSlash]
	}
	return "/"
}

// cloneCookie creates a deep copy of an http.Cookie.
func cloneCookie(cookie *http.Cookie) *http.Cookie {
	if cookie == nil {
		return nil
	}
	clone := *cookie
	return &clone
}

// cookieIdentity checks if two cookies have the same identity (name, domain, path).
// Used for deduplication in the cookie jar.
func cookieIdentity(a, b *http.Cookie) bool {
	return a != nil && b != nil && a.Name == b.Name &&
		a.Domain == b.Domain && a.Path == b.Path
}

func (jar *OriginCookieJar) removeCookie(key string, cookie *http.Cookie) {
	kept := jar.cookies[key][:0]
	for _, existing := range jar.cookies[key] {
		if !cookieIdentity(existing, cookie) {
			kept = append(kept, existing)
		}
	}
	jar.cookies[key] = kept
}

func (jar *OriginCookieJar) upsertCookie(key string, cookie *http.Cookie) {
	for i, existing := range jar.cookies[key] {
		if cookieIdentity(existing, cookie) {
			jar.cookies[key][i] = cookie
			return
		}
	}
	jar.cookies[key] = append(jar.cookies[key], cookie)
}

// sameSiteAllows checks if a cookie's SameSite policy allows it for a request.
// SameSite=None: always allowed (requires Secure)
// SameSite=Lax: allowed for same-site OR top-level safe-method navigations
// SameSite=Strict: only allowed for same-site requests
func sameSiteAllows(cookie *http.Cookie, requestURL, initiator *url.URL, topLevel bool, method string) bool {
	if cookie.SameSite == http.SameSiteNoneMode {
		return true
	}
	if sameSite(requestURL, initiator) {
		return true
	}
	return cookie.SameSite != http.SameSiteStrictMode && topLevel && isSafeMethod(method)
}

// sameSite checks if two URLs are same-site (same registrable domain).
// Uses public suffix list to determine registrable domain.
func sameSite(a, b *url.URL) bool {
	if a == nil || b == nil || !strings.EqualFold(a.Scheme, b.Scheme) {
		return false
	}
	return registrableDomain(a.Hostname()) == registrableDomain(b.Hostname())
}

// registrableDomain returns the registrable domain (eTLD+1) of a host.
// Uses the public suffix list. Falls back to host if not found.
// Example: "www.example.co.uk" -> "example.co.uk"
func registrableDomain(host string) string {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	domain, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return host
	}
	return domain
}

// isSafeMethod returns true for HTTP methods considered "safe" for SameSite=Lax.
// Safe methods: GET, HEAD, OPTIONS, TRACE (do not modify server state).
func isSafeMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

// originCookieTransport is an http.RoundTripper that adds cookie handling, HSTS upgrades,
// and certificate verification to HTTP requests.
// Each transport is bound to a specific origin (initiator) for cookie partitioning.
type originCookieTransport struct {
	jar             *OriginCookieJar
	initiator       *url.URL
	topLevel        bool
	hsts            *HSTSStore
	base            http.RoundTripper
	tofu            *TOFUStore
	httpsMu         sync.Mutex
	httpsTransports map[string]http.RoundTripper
}

// RoundTrip executes an HTTP request with cookie handling, HSTS, and TLS verification.
// 1. Clones request and upgrades HTTP->HTTPS if HSTS applies
// 2. Adds cookies from origin-partitioned jar
// 3. Uses verified HTTPS transport for HTTPS requests
// 4. Processes response: updates HSTS, stores cookies
func (transport *originCookieTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.URL = cloneURL(request.URL)
	store := transport.hsts
	if store == nil {
		store = hstsStore
	}
	upgradeHTTPURL(request.URL, store)
	transport.addRequestCookies(request)
	base := transport.base
	if strings.EqualFold(request.URL.Scheme, "https") {
		var err error
		base, err = transport.httpsTransport(request.URL.Hostname())
		if err != nil {
			return nil, fmt.Errorf("configure HTTPS transport: %w", err)
		}
	} else if base == nil {
		base = http.DefaultTransport
	}
	response, err := base.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	transport.processResponse(request, response, store)
	return response, nil
}

// addRequestCookies adds cookies from the jar to the request.
// Skips cookies for OPTIONS requests (preflight).
// Uses CookiesForRequest with the transport's initiator and topLevel settings.
func (transport *originCookieTransport) addRequestCookies(request *http.Request) {
	if transport.jar == nil {
		return
	}
	request.Header.Del("Cookie")
	if request.Method == http.MethodOptions {
		return
	}
	for _, cookie := range transport.jar.CookiesForRequest(request.URL, transport.initiator, transport.topLevel, request.Method) {
		request.AddCookie(cookie)
	}
}

// processResponse processes an HTTP response: updates HSTS store and stores cookies.
// Called after a successful response.
// For HTTPS responses, extracts and processes Strict-Transport-Security headers.
// Stores response cookies in the origin-partitioned jar.
func (transport *originCookieTransport) processResponse(request *http.Request, response *http.Response, store *HSTSStore) {
	if strings.EqualFold(request.URL.Scheme, "https") {
		for _, header := range response.Header.Values("Strict-Transport-Security") {
			store.ProcessHeader(request.URL.Hostname(), header, true)
		}
	}
	if transport.jar != nil {
		transport.jar.SetCookies(request.URL, response.Cookies())
	}
}

// httpsTransport returns a verified HTTPS transport for a hostname.
// Caches transports per hostname. Creates a new verified transport using TOFU store
// for certificate validation (supports self-signed certs via TOFU).
func (transport *originCookieTransport) httpsTransport(hostname string) (http.RoundTripper, error) {
	transport.httpsMu.Lock()
	defer transport.httpsMu.Unlock()
	if cached := transport.httpsTransports[hostname]; cached != nil {
		return cached, nil
	}
	store, err := transport.certificateStore()
	if err != nil {
		return nil, err
	}
	base := transport.base
	if base == nil {
		base = http.DefaultTransport
	}
	verified, err := newVerifiedTransport(base, store, hostname)
	if err != nil {
		return nil, err
	}
	if transport.httpsTransports == nil {
		transport.httpsTransports = make(map[string]http.RoundTripper)
	}
	transport.httpsTransports[hostname] = verified
	return verified, nil
}

// certificateStore returns the TOFU store for certificate validation.
// Uses the transport's TOFU store if set, otherwise falls back to the default store.
func (transport *originCookieTransport) certificateStore() (*TOFUStore, error) {
	if transport.tofu != nil {
		return transport.tofu, nil
	}
	return DefaultTOFUStore()
}

func cloneURL(requestURL *url.URL) *url.URL {
	if requestURL == nil {
		return nil
	}
	clone := *requestURL
	return &clone
}

func requestClient(client *http.Client, requestURL *url.URL, origin string) *http.Client {
	if client == nil || requestURL == nil {
		return client
	}
	jar, ok := client.Jar.(*OriginCookieJar)
	if !ok {
		jar = nil
	}
	clone := *client
	if jar != nil {
		clone.Jar = nil
	}
	initiator, err := url.Parse(origin)
	topLevel := origin == ""
	if err != nil {
		initiator = nil
	}
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	clone.Transport = &originCookieTransport{
		jar: jar, initiator: initiator, topLevel: topLevel, hsts: hstsStore, base: base,
	}
	return &clone
}

// History represents navigation history for a browser tab.
// Maintains a list of previous pages and next pages (for back/forward).
type History struct {
	previousPages []*url.URL
	nextPages     []*url.URL
}

// NextPages returns the forward history (pages after current).
func (h *History) NextPages() []*url.URL {
	return h.nextPages
}

// AllPages returns the complete history (previous pages).
func (h *History) AllPages() []*url.URL {
	return h.previousPages
}

// PageCount returns the number of pages in history.
func (h *History) PageCount() int {
	return len(h.previousPages)
}

// Push adds a new page to history, clearing forward history.
func (h *History) Push(URL *url.URL) {
	h.nextPages = nil
	h.previousPages = append(h.previousPages, URL)
}

// Last returns the most recent page in history.
func (h *History) Last() *url.URL {
	if len(h.previousPages) == 0 {
		return nil
	}
	return h.previousPages[len(h.previousPages)-1]
}

// PopNext moves forward in history (not fully implemented).
// PopNext moves forward in history.
// Moves the last nextPage to previousPages.
func (h *History) PopNext() {
	if len(h.nextPages) > 0 {
		h.previousPages = append(h.previousPages, h.nextPages[len(h.nextPages)-1])
		h.nextPages = nil
	}
}

// Pop moves back in history.
// Moves the last previousPage to nextPages.
func (h *History) Pop() {
	if len(h.previousPages) > 0 {
		h.nextPages = append(h.nextPages, h.previousPages[len(h.previousPages)-1])
		h.previousPages = h.previousPages[:len(h.previousPages)-1]
	}
}

// Log prints a log message with component name in magenta.
// Used for debugging network activity.
func Log(component, msg string) {
	str := "(" + "\033[95m" + component + "\033[0m" + ")"
	fmt.Println(str, msg)
}

// loadErrorPage returns a simple HTML error page for display when fetch fails.
// Used as fallback content when network requests fail.
func loadErrorPage(err string) string {
	return `
	<html>
		<head>
			<title>
				Error!
			</title>
		</head>
		<body>
			<h1>` + err + `</h1>
		</body>
	</html>`
}

// Shared HTTP client with origin-partitioned cookie jar.
// This is the default global client; per-window clients should use their own.
var (
	cookieJar = NewOriginCookieJar()
	client    = &http.Client{
		Jar: cookieJar,
	}
	cache      = &ResourceCache{}
	imageCache = &ImgCache{}
)

// Resource represents an HTTP resource response.
// Contains the response body, metadata, and caching key.
type Resource struct {
	Body        string      // Response body as string
	ContentType string      // MIME type from Content-Type header
	Code        int         // HTTP status code
	URL         *url.URL    // Final URL after redirects
	Key         string      // Cache key (usually URL string)
	Headers     http.Header // Response headers
}

// ResourceCache caches HTTP responses in memory.
// Simple linear search cache; not optimized for large scale.
// Implements basic HTTP caching semantics (ETag/Last-Modified not yet implemented).
type ResourceCache struct {
	cachedResources []*Resource
}

// AddResource adds a resource to the cache.
func (c *ResourceCache) AddResource(resource *Resource) {
	c.cachedResources = append(c.cachedResources, resource)
}

// GetResource retrieves a resource from cache by key.
// Returns nil if not found.
func (c *ResourceCache) GetResource(resourceKey string) *Resource {
	for _, resource := range c.cachedResources {
		if resource.Key == resourceKey {
			return resource
		}
	}
	return nil
}

// CachedImage represents a cached image (raw bytes).
type CachedImage struct {
	Key   string // Cache key (URL)
	Image []byte // Raw image data
}

// ImgCache caches decoded images in memory.
// Avoids re-decoding images on subsequent loads.
// Uses simple linear search; LRU eviction not yet implemented.
type ImgCache struct {
	cachedImages []*CachedImage
}

// AddImage adds an image to the cache.
func (c *ImgCache) AddImage(key string, value []byte) {
	c.cachedImages = append(c.cachedImages,
		&CachedImage{
			Key:   key,
			Image: value,
		},
	)
}

// GetImage retrieves an image from cache by key.
// Returns nil if not found.
func (c *ImgCache) GetImage(imageKey string) *CachedImage {
	for _, image := range c.cachedImages {
		if image.Key == imageKey {
			return image
		}
	}
	return nil
}

// GetResource fetches a resource for the given URL.
// Handles internal schemes (thdwb://, file://) and external HTTP/HTTPS.
// For external URLs, delegates to fetchExternalPage.
// Parameters:
//
//	URL - the URL to fetch
//	history - navigation history (for internal pages)
//	buildInfo - build information (for about page)
//
// Returns a Resource with the response body and metadata.
func GetResource(URL *url.URL, history *History, buildInfo *assets.BuildInfo) *Resource {
	switch URL.Scheme {
	case "thdwb":
		return fetchInternalPage(URL, history, buildInfo)
	case "file":
		return &Resource{Body: pages.RenderFileBrowser(URL.Path), URL: URL}
	case "":
		URL.Scheme = "http"
	}

	return fetchExternalPage(URL)
}

// fetchInternalPage handles thdwb:// internal URLs.
// Supported hosts: homepage, history, about.
// Returns a Resource with generated HTML content.
func fetchInternalPage(URL *url.URL, history *History, buildInfo *assets.BuildInfo) *Resource {
	switch URL.Host {
	case "homepage":
		return &Resource{
			Body: string(assets.HomePage()),
			URL:  URL,
		}

	case "history":
		return &Resource{
			Body: buildHistoryPage(history),
			URL:  URL,
		}
	case "about":
		return &Resource{
			Body: pages.RenderAboutPage(buildInfo),
			URL:  URL,
		}
	default:
		return &Resource{
			Body: string(assets.DefaultPage()),
			URL:  URL,
		}
	}
}

// fetchExternalPage fetches an external HTTP/HTTPS resource.
// Uses the default global client with shared cookie jar.
// For per-window isolation, use fetchExternalPageWithOptions with a custom client.
func fetchExternalPage(URL *url.URL) *Resource {
	return fetchExternalPageWithOptions(nil, URL, "GET", nil, nil)
}

// fetchExternalPageWithOptions makes an HTTP request with custom method, headers, and body.
// If client is nil, the default global client is used.
// Handles CORS by sending Origin header (caller responsible for preflight).
// Parameters:
//
//	client - HTTP client (nil = default global client)
//	URL - target URL
//	method - HTTP method (GET, POST, etc.)
//	body - request body reader
//	headers - custom headers (including Origin for CORS)
//
// Returns a Resource with response data.
func fetchExternalPageWithOptions(client *http.Client, URL *url.URL, method string, body io.Reader, headers map[string]string) *Resource {
	URL = cloneURL(URL)
	upgradeHTTPURL(URL, hstsStore)
	if client == nil {
		client = defaultClient()
	}
	origin := ""
	if headers != nil {
		origin = headers["Origin"]
	}
	client = requestClient(client, URL, origin)
	url := URL.String()
	go Log("sauce", "Downloading page "+url)

	cachedResource := cache.GetResource(url)
	if cachedResource != nil && method == "GET" && body == nil {
		return cachedResource
	}

	resource := &Resource{Key: url, URL: URL}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		resource.Body = loadErrorPage(err.Error())
		return resource
	}

	req.Header.Set("User-Agent", "THDWB (The HotDog Web Browser);")

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// For cross-origin requests, we may need to handle CORS preflight
	// The actual preflight is handled by the caller (JS layer) which sends OPTIONS first
	// This function just makes the actual request

	resp, err := requestClient(client, URL, "").Do(req)
	if err != nil {
		resource.Body = loadErrorPage(err.Error())
		return resource
	}

	defer func() { _ = resp.Body.Close() }()
	responseBody, _ := io.ReadAll(resp.Body)

	resource.ContentType = resp.Header.Get("Content-Type")
	resource.URL = resp.Request.URL
	resource.Code = resp.StatusCode
	resource.Headers = resp.Header

	resource.Body = string(responseBody)

	// Only cache GET requests without body
	if method == "GET" && body == nil {
		cache.AddResource(resource)
	}
	return resource
}

// defaultClient returns the default global HTTP client.
// Uses shared cookie jar and transport.
// For per-window isolation, create a new client with its own OriginCookieJar.
func defaultClient() *http.Client {
	return client
}

// FetchExternalPageWithOptions makes an HTTP request using the default global client.
// Deprecated: Use fetchExternalPageWithOptions with a custom client for per-window cookie isolation.
// The default client shares cookies across all windows, which breaks container tab isolation.
func FetchExternalPageWithOptions(URL *url.URL, method string, body io.Reader, headers map[string]string) *Resource {
	return fetchExternalPageWithOptions(nil, URL, method, body, headers)
}

// MakeCORSRequest makes an HTTP request with CORS handling using the default global client.
// Sends Origin header. Does NOT automatically send preflight (OPTIONS) - caller must handle.
// Deprecated: Use MakeCORSRequestWithClient for per-window cookie isolation.
func MakeCORSRequest(URL *url.URL, origin, method string, body io.Reader, headers map[string]string) *Resource {
	return MakeCORSRequestWithClient(nil, URL, origin, method, body, headers)
}

// MakeCORSRequestWithClient makes an HTTP request with CORS handling using the provided client.
// If client is nil, the default global client is used.
// Sends Origin header. Does NOT automatically send preflight (OPTIONS) - caller must handle.
// For methods requiring preflight (non-simple methods), caller should send OPTIONS first.
// Parameters:
//
//	client - HTTP client (nil = default global client)
//	URL - target URL
//	origin - Origin header value
//	method - HTTP method
//	body - request body
//	headers - additional headers
//
// Returns a Resource with response data.
// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS
func MakeCORSRequestWithClient(client *http.Client, URL *url.URL, origin, method string, body io.Reader, headers map[string]string) *Resource {
	// Add Origin header to all CORS requests
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Origin"] = origin

	// For methods that require preflight (not simple methods), we would need to send OPTIONS first
	// Simple methods: GET, HEAD, POST with simple headers
	// For now, we just make the request with Origin header
	// The server should respond with Access-Control-Allow-Origin

	return fetchExternalPageWithOptions(client, URL, method, body, headers)
}

// ParseURL parses a URL string, returning a *url.URL.
// On parse error, returns a thdwb://error URL with the error message.
func ParseURL(link string) *url.URL {
	URL, err := url.Parse(link)
	if err != nil {
		URL = ParseURL("thdwb://error?err=failedToParseURL")
	}

	return URL
}

// GetImage fetches and decodes an image from a URL.
// Supports HTTP/HTTPS URLs and data: URLs (base64 encoded PNG).
// Caches images in memory to avoid re-downloading and re-decoding.
// Returns raw image bytes (not decoded image.Image).
// Parameters:
//
//	URL - image URL (http, https, or data: scheme)
//
// Returns image bytes and error.
func GetImage(URL *url.URL) ([]byte, error) {
	URL = cloneURL(URL)
	upgradeHTTPURL(URL, hstsStore)
	imgUrl := URL.String()

	cachedImage := imageCache.GetImage(imgUrl)

	if cachedImage != nil {
		return cachedImage.Image, nil
	}

	var img []byte
	if len(imgUrl) >= 22 && imgUrl[:22] == "data:image/png;base64," {
		imgData := imgUrl[strings.IndexByte(imgUrl, ',')+1:]

		decodedData, err := base64.RawStdEncoding.DecodeString(imgData)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 data (%s)", err)
		}

		img = decodedData
	} else {
		req, err := http.NewRequest("GET", imgUrl, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("User-Agent", "THDWB (The HotDog Web Browser);")

		resp, err := requestClient(client, URL, "").Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch image (%s)", err)
		}
		defer func() { _ = resp.Body.Close() }()

		img, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	}

	imageCache.AddImage(imgUrl, img)
	return img, nil
}

func buildHistoryPage(history *History) string {
	d := `
	<html>
		<head>
			<title>History</title>
		</head>
		<body>
		<h1>History</h1>
		<ul>
	`
	for _, page := range history.AllPages() {
		d += `<li><a href="` + page.String() + `">` + page.String() + `</a></li>`
	}

	d += `
		</ul>
		</body>
	</html>
	`
	return d
}
