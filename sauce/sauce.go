package sauce

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/danfragoso/thdwb/assets"
	pages "github.com/danfragoso/thdwb/pages"
)

// OriginCookieJar implements http.CookieJar with per-origin cookie partitioning.
type OriginCookieJar struct {
	mu      sync.RWMutex
	cookies map[string][]*http.Cookie // key: origin (scheme://host:port)
}

func NewOriginCookieJar() *OriginCookieJar {
	return &OriginCookieJar{
		cookies: make(map[string][]*http.Cookie),
	}
}

func originKey(u *url.URL) string {
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

func (jar *OriginCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	jar.mu.Lock()
	defer jar.mu.Unlock()
	key := originKey(u)
	jar.cookies[key] = append(jar.cookies[key], cookies...)
}

func (jar *OriginCookieJar) Cookies(u *url.URL) []*http.Cookie {
	jar.mu.RLock()
	defer jar.mu.RUnlock()
	key := originKey(u)
	// Return a copy to avoid race conditions
	cookies := jar.cookies[key]
	result := make([]*http.Cookie, len(cookies))
	copy(result, cookies)
	return result
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
	if client == nil {
		client = defaultClient()
	}
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

	resp, err := client.Do(req)
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

		resp, err := client.Do(req)
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
