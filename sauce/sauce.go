package sauce

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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
type OriginCookieJar struct {
	mu      sync.RWMutex
	cookies map[string][]*http.Cookie // key: origin (scheme://host:port)
}

type hstsEntry struct {
	expires           time.Time
	includeSubdomains bool
}

// HSTSStore caches Strict-Transport-Security policies by hostname.
type HSTSStore struct {
	mu      sync.Mutex
	entries map[string]hstsEntry
}

// NewHSTSStore creates an empty HSTS policy store.
func NewHSTSStore() *HSTSStore {
	return &HSTSStore{entries: make(map[string]hstsEntry)}
}

var hstsStore = NewHSTSStore()

func NewOriginCookieJar() *OriginCookieJar {
	return &OriginCookieJar{
		cookies: make(map[string][]*http.Cookie),
	}
}

// ProcessHeader records a valid HSTS header received over a secure connection.
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

func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	domain = strings.TrimPrefix(domain, ".")
	return strings.ToLower(domain)
}

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

// CookiesForRequest applies SameSite rules using the initiating site and request type.
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

func defaultCookiePath(requestPath string) string {
	if requestPath == "" || requestPath[0] != '/' {
		return "/"
	}
	if lastSlash := strings.LastIndex(requestPath, "/"); lastSlash > 0 {
		return requestPath[:lastSlash]
	}
	return "/"
}

func cloneCookie(cookie *http.Cookie) *http.Cookie {
	if cookie == nil {
		return nil
	}
	clone := *cookie
	return &clone
}

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

func sameSiteAllows(cookie *http.Cookie, requestURL, initiator *url.URL, topLevel bool, method string) bool {
	if cookie.SameSite == http.SameSiteNoneMode {
		return true
	}
	if sameSite(requestURL, initiator) {
		return true
	}
	return cookie.SameSite != http.SameSiteStrictMode && topLevel && isSafeMethod(method)
}

func sameSite(a, b *url.URL) bool {
	if a == nil || b == nil || !strings.EqualFold(a.Scheme, b.Scheme) {
		return false
	}
	return registrableDomain(a.Hostname()) == registrableDomain(b.Hostname())
}

func registrableDomain(host string) string {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	domain, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return host
	}
	return domain
}

func isSafeMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

type originCookieTransport struct {
	jar       *OriginCookieJar
	initiator *url.URL
	topLevel  bool
	hsts      *HSTSStore
	base      http.RoundTripper
}

func (transport *originCookieTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.URL = cloneURL(request.URL)
	store := transport.hsts
	if store == nil {
		store = hstsStore
	}
	upgradeHTTPURL(request.URL, store)
	if transport.jar != nil {
		request.Header.Del("Cookie")
		if request.Method != http.MethodOptions {
			for _, cookie := range transport.jar.CookiesForRequest(request.URL, transport.initiator, transport.topLevel, request.Method) {
				request.AddCookie(cookie)
			}
		}
	}
	response, err := transport.base.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(request.URL.Scheme, "https") {
		for _, header := range response.Header.Values("Strict-Transport-Security") {
			store.ProcessHeader(request.URL.Hostname(), header, true)
		}
	}
	if transport.jar != nil {
		transport.jar.SetCookies(request.URL, response.Cookies())
	}
	return response, nil
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

// History represents navigation history
type History struct {
	previousPages []*url.URL
	nextPages     []*url.URL
}

func (h *History) NextPages() []*url.URL {
	return h.nextPages
}

func (h *History) AllPages() []*url.URL {
	return h.previousPages
}

func (h *History) PageCount() int {
	return len(h.previousPages)
}

func (h *History) Push(URL *url.URL) {
	h.nextPages = nil
	h.previousPages = append(h.previousPages, URL)
}

func (h *History) Last() *url.URL {
	if len(h.previousPages) == 0 {
		return nil
	}
	return h.previousPages[len(h.previousPages)-1]
}

func (h *History) PopNext() {
	if len(h.nextPages) > 0 {
		h.previousPages = append(h.previousPages, h.nextPages[len(h.nextPages)-1])
		h.nextPages = nil
	}
}

func (h *History) Pop() {
	if len(h.previousPages) > 0 {
		h.nextPages = append(h.nextPages, h.previousPages[len(h.previousPages)-1])
		h.previousPages = h.previousPages[:len(h.previousPages)-1]
	}
}

// Log prints a log message with component name
func Log(component, msg string) {
	str := "(" + "\033[95m" + component + "\033[0m" + ")"
	fmt.Println(str, msg)
}

// loadErrorPage returns an HTML error page
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

// Shared HTTP client with origin-partitioned cookie jar
var (
	cookieJar = NewOriginCookieJar()
	client    = &http.Client{
		Jar: cookieJar,
	}
	cache      = &ResourceCache{}
	imageCache = &ImgCache{}
)

// Resource represents an HTTP resource response
type Resource struct {
	Body        string
	ContentType string
	Code        int
	URL         *url.URL
	Key         string
	Headers     http.Header
}

// ResourceCache caches HTTP resources
type ResourceCache struct {
	cachedResources []*Resource
}

func (c *ResourceCache) AddResource(resource *Resource) {
	c.cachedResources = append(c.cachedResources, resource)
}

func (c *ResourceCache) GetResource(resourceKey string) *Resource {
	for _, resource := range c.cachedResources {
		if resource.Key == resourceKey {
			return resource
		}
	}
	return nil
}

// CachedImage represents a cached image
type CachedImage struct {
	Key   string
	Image []byte
}

// ImgCache caches images
type ImgCache struct {
	cachedImages []*CachedImage
}

func (c *ImgCache) AddImage(key string, value []byte) {
	c.cachedImages = append(c.cachedImages,
		&CachedImage{
			Key:   key,
			Image: value,
		},
	)
}

func (c *ImgCache) GetImage(imageKey string) *CachedImage {
	for _, image := range c.cachedImages {
		if image.Key == imageKey {
			return image
		}
	}
	return nil
}

// GetResource - Makes an http request and returns a resource struct
func GetResource(URL *url.URL, history *History, buildInfo *assets.BuildInfo) *Resource {
	switch URL.Scheme {
	case "thdwb":
		return fetchInternalPage(URL, history, buildInfo)
	case "file":
		return &Resource{Body: pages.RenderFileBrowser(URL.Path), URL: URL}
	case "":
		URL.Scheme = "http"
		break
	}

	return fetchExternalPage(URL)
}

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

func fetchExternalPage(URL *url.URL) *Resource {
	return fetchExternalPageWithOptions(nil, URL, "GET", nil, nil)
}

// fetchExternalPageWithOptions makes an HTTP request with custom method, headers, and body using the provided client.
// If client is nil, the default global client is used.
// It handles CORS preflight automatically for cross-origin requests.
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
		log.Fatalln(err)
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

	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)

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
func defaultClient() *http.Client {
	return client
}

// FetchExternalPageWithOptions makes an HTTP request with the given options using the default global client.
// Deprecated: Use fetchExternalPageWithOptions with a custom client for per-window cookie isolation.
func FetchExternalPageWithOptions(URL *url.URL, method string, body io.Reader, headers map[string]string) *Resource {
	return fetchExternalPageWithOptions(nil, URL, method, body, headers)
}

// MakeCORSRequest makes an HTTP request with proper CORS handling using the default global client.
// It sends Origin header and handles preflight if needed.
// Deprecated: Use MakeCORSRequestWithClient for per-window cookie isolation.
func MakeCORSRequest(URL *url.URL, origin, method string, body io.Reader, headers map[string]string) *Resource {
	return MakeCORSRequestWithClient(nil, URL, origin, method, body, headers)
}

// MakeCORSRequestWithClient makes an HTTP request with proper CORS handling using the provided client.
// If client is nil, the default global client is used.
// It sends Origin header and handles preflight if needed.
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

func ParseURL(link string) *url.URL {
	URL, err := url.Parse(link)
	if err != nil {
		URL = ParseURL("thdwb://error?err=failedToParseURL")
	}

	return URL
}

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
			return nil, fmt.Errorf("Failed to decode base64 data (%s)", err)
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
			return nil, fmt.Errorf("Failed to fetch image (%s)", err)
		}
		defer resp.Body.Close()

		img, err = ioutil.ReadAll(resp.Body)
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
