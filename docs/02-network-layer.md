# Chapter 2: Network Layer — The sauce Package

> **In this chapter**: How thdwb talks to servers — HTTP requests, cookies, caching, certificates, and security. This is the "postal service" of the browser.

---

## 2.1 What Is the Network Layer?

When you type a URL and press Enter, the browser needs to **fetch** the HTML, CSS, JavaScript, images, and other resources from servers. The network layer handles all of this.

**Analogy**: The network layer is the **postal service**. You give it an address (URL), it delivers your request, picks up the response, and hands it to the rest of the browser.

```
┌─────────────────────────────────────────────────────────────────┐
│                      NETWORK LAYER (sauce)                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌──────────────┐    ┌──────────────┐    ┌──────────────┐     │
│   │   HTTP       │    │   COOKIE     │    │   CACHE      │     │
│   │   CLIENT     │◄───│    JAR       │    │   (Resources,│     │
│   │   (Fetch)    │    │ (Partitioned)│    │   Images)    │     │
│   └──────┬───────┘    └──────┬───────┘    └──────┬───────┘     │
│          │                   │                   │              │
│          ▼                   ▼                   ▼              │
│   ┌──────────────────────────────────────────────────────┐     │
│   │              SECURITY LAYER                           │     │
│   │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │     │
│   │  │   TLS    │  │   HSTS   │  │   CORS   │           │     │
│   │  │ (Certs)  │  │ (HTTPS    │  │ (Cross-  │           │     │
│   │  │          │  │  Only)   │  │  Origin) │           │     │
│   │  └──────────┘  └──────────┘  └──────────┘           │     │
│   └──────────────────────────────────────────────────────┘     │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2.2 Key Terms (Defined in [Glossary](../glossary.md))

- **HTTP/HTTPS** — The protocol for web communication
- **Origin** — Scheme + host + port (e.g., `https://example.com:443`)
- **Cookie** — Small data stored by server, sent back with requests
- **Cookie Jar** — Browser's cookie storage, partitioned by origin
- **Cache** — Temporary storage to avoid re-downloading
- **HSTS** — Forces HTTPS for a domain
- **CORS** — Controls cross-origin requests
- **Certificate** — Digital ID proving a site's identity
- **TOFU** — Trust On First Use (for self-signed certs)

---

## 2.3 The sauce Package: File Overview

```
sauce/
├── sauce.go                    # Main HTTP client, cache, cookies, HSTS
├── tofu.go                     # Trust On First Use certificate store
├── tofu_workflow.go            # Certificate inspection workflow
├── certificate_validation.go   # System trust store integration
├── sauce_test.go               # Tests
├── tofu_test.go
├── tofu_workflow_test.go
└── certificate_validation_test.go
```

---

## 2.4 HTTP Client: Fetching Resources

### The Main Entry Point: `GetResource`

```go
// GetResource makes an HTTP request and returns a Resource struct.
// It handles:
// - Automatic redirect following
// - Cookie jar (per-origin partitioning)
// - Response caching
// - Error page generation on failure
func GetResource(urlString string, options ...ResourceOption) (*Resource, error)
```

**Analogy**: You give the postal service an address, it delivers the letter and brings back the reply.

**Reference**: [MDN - Fetch API](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API) | [Firefox nsHttpChannel](https://searchfox.org/mozilla-central/source/netwerk/protocol/http/nsHttpChannel.cpp)

### Resource Struct: The Response Container

```go
// Resource represents an HTTP response with metadata.
type Resource struct {
    URL         string            // Final URL after redirects
    StatusCode  int               // HTTP status (200, 404, etc.)
    Headers     http.Header       // Response headers
    Body        []byte            // Response body (HTML, CSS, JS, image data)
    ContentType string            // MIME type (text/html, image/png, etc.)
    FromCache   bool              // True if served from cache
}
```

**Key terms:**
- **Status Code**: 200=OK, 301/302=Redirect, 404=Not Found, 500=Server Error
- **MIME Type**: Tells the browser what kind of data it is
- **Headers**: Metadata like `Content-Type`, `Set-Cookie`, `Cache-Control`

---

## 2.5 Cookie Jar: Origin-Partitioned Storage

### Why Partition Cookies?

**Problem**: Without partitioning, a cookie set by `siteA.com` could be sent to `siteB.com` if both embed the same tracker. This enables cross-site tracking.

**Solution**: **Partition cookies by origin**. Each origin gets its own cookie jar.

```
┌────────────────────────────────────────────────────────────┐
│                    COOKIE JAR PARTITIONING                  │
├────────────────────────────────────────────────────────────┤
│                                                             │
│   Traditional (Shared Jar):          Partitioned (thdwb):  │
│   ─────────────────────────          ─────────────────────  │
│   Jar: [A.com, B.com, Tracker.com]   Jar A.com: [A.com]    │
│   │                                 Jar B.com: [B.com]     │
│   ▼                                 Jar Tracker: [Tracker] │
│   Tracker sees ALL cookies           Tracker sees ONLY     │
│   from all sites                     its own cookies       │
│                                                             │
└────────────────────────────────────────────────────────────┘
```

### OriginCookieJar Implementation

```go
// OriginCookieJar implements http.CookieJar with per-origin cookie partitioning.
// Each origin (scheme://host:port) gets its own isolated cookie store.
// This prevents cross-site tracking and matches Firefox's behavior.
type OriginCookieJar struct {
    mu      sync.RWMutex
    cookies map[string][]*http.Cookie  // key: origin (scheme://host:port)
}
```

**Methods:**
- `SetCookies(u *url.URL, cookies []*http.Cookie)` — Store cookies for an origin
- `Cookies(u *url.URL) []*http.Cookie` — Retrieve cookies for an origin

**Reference**: [MDN - Cookies](https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies) | [Firefox nsCookieService](https://searchfox.org/mozilla-central/source/netwerk/cookie/nsCookieService.cpp)

### SameSite Cookie Attribute

Cookies can have a `SameSite` attribute controlling when they're sent:

| Value | When Sent |
|-------|-----------|
| `Strict` | Only same-site requests (no cross-site) |
| `Lax` | Same-site + top-level navigations (links) |
| `None` | All requests (requires `Secure` flag) |

```go
// CookiesForRequest applies SameSite rules using the initiating site
// and request type (navigation, subresource, etc.).
func (jar *OriginCookieJar) CookiesForRequest(requestURL, siteForCookies *url.URL, requestType RequestType) []*http.Cookie
```

**Reference**: [MDN - SameSite](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite) | [Firefox CookieBehavior](https://searchfox.org/mozilla-central/source/netwerk/cookie/CookieBehavior.cpp)

---

## 2.6 Caching: Avoiding Re-downloads

### ResourceCache: HTTP Response Cache

```go
// ResourceCache caches HTTP responses with ETag/Last-Modified validation.
// Implements RFC 7234 HTTP caching semantics.
type ResourceCache struct {
    mu    sync.RWMutex
    items map[string]*CachedResource
}
```

**How it works:**
1. First request → store response with `ETag` and/or `Last-Modified`
2. Subsequent request → send `If-None-Match` (ETag) or `If-Modified-Since`
3. Server responds `304 Not Modified` → use cached copy
4. Server responds `200 OK` → update cache with new response

**Analogy**: Keeping a photocopy of a document. Next time you need it, ask "has the original changed?" If not, use your copy.

**Reference**: [MDN - HTTP Caching](https://developer.mozilla.org/en-US/docs/Web/HTTP/Caching) | [Firefox nsHttpCache](https://searchfox.org/mozilla-central/source/netwerk/protocol/http/nsHttpCache.cpp)

### ImgCache: Image-Specific Cache

```go
// ImgCache caches decoded images (image.Image) to avoid re-decoding.
// Keyed by URL. Uses LRU eviction when memory pressure is high.
type ImgCache struct {
    mu    sync.Mutex
    cache map[string]*CachedImage
    lru   *list.List
}
```

Images are expensive to decode (JPEG → pixels). Caching decoded pixels saves CPU.

---

## 2.7 Security: TLS, Certificates, HSTS, CORS

### Certificate Validation

thdwb validates TLS certificates against the **system trust store** (OS-provided root CAs).

```go
// DefaultTOFUStore returns the persistent certificate store used by default requests.
// It validates certificates against the system trust store first.
// If validation fails (e.g., self-signed), TOFU allows explicit user trust.
func DefaultTOFUStore() (*TOFUStore, error)
```

**Flow:**
```
Server presents certificate
        │
        ▼
┌───────────────────┐
│ System Trust Store │──Trusted?──Yes──► Allow connection
│ (OS Root CAs)      │
└───────────────────┘
        │ No
        ▼
┌───────────────────┐
│   TOFU Store      │──Trusted?──Yes──► Allow connection
│ (User-approved    │
│  self-signed)     │
└───────────────────┘
        │ No
        ▼
   BLOCK connection
```

**Reference**: [MDN - Certificate Transparency](https://developer.mozilla.org/en-US/docs/Web/Security/Certificate_Transparency) | [Firefox CertVerifier](https://searchfox.org/mozilla-central/source/security/manager/ssl/src/CertVerifier.cpp)

### TOFU: Trust On First Use

For self-signed certificates (common in development), thdwb uses **TOFU**:

```go
// InspectTOFUCertificate performs a TLS handshake WITHOUT sending application data.
// Returns the certificate and its SHA-256 fingerprint for user verification.
// User MUST verify fingerprint through a separate trusted channel before trusting.
func InspectTOFUCertificate(ctx context.Context, host string) (*x509.Certificate, string, error)

// TrustCertificate validates and records cert as the trusted certificate for host.
// Stores in persistent JSON file with owner-only permissions.
func (store *TOFUStore) TrustCertificate(host string, cert *x509.Certificate) error
```

**Workflow:**
1. `InspectTOFUCertificate("localhost:8080")` → gets cert + fingerprint
2. User verifies fingerprint via separate channel (e.g., SSH to server, check cert)
3. `TrustCertificate("localhost:8080", cert)` → saves to `~/.config/thdwb/tofu-certificates.json`
4. Future connections to `localhost:8080` are allowed

**Reference**: [MDN - TOFU](https://developer.mozilla.org/en-US/docs/Glossary/TOFU) | [Firefox TOFU](https://searchfox.org/mozilla-central/source/security/manager/ssl/src/TOFU.cpp)

### HSTS: HTTP Strict Transport Security

```go
// HSTSStore caches Strict-Transport-Security policies by hostname.
// When a site sends HSTS header, browser remembers: "always use HTTPS for this host."
type HSTSStore struct {
    mu      sync.Mutex
    entries map[string]hstsEntry
}

type hstsEntry struct {
    expires           time.Time
    includeSubdomains bool
}
```

**How it works:**
1. Server sends `Strict-Transport-Security: max-age=31536000; includeSubDomains`
2. Browser stores: "For `example.com` and subdomains, ONLY use HTTPS for 1 year"
3. Any HTTP request to that host is automatically upgraded to HTTPS

**Reference**: [MDN - HSTS](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Strict-Transport-Security) | [Firefox HSTSService](https://searchfox.org/mozilla-central/source/netwerk/protocol/http/HSTSService.cpp)

### CORS: Cross-Origin Resource Sharing

```go
// MakeCORSRequestWithClient makes an HTTP request with proper CORS handling.
// - Sends Origin header
// - Handles preflight (OPTIONS) for non-simple requests
// - Validates response headers (Access-Control-Allow-Origin, etc.)
func MakeCORSRequestWithClient(client *http.Client, req *http.Request) (*http.Response, error)
```

**Simple Request** (no preflight): GET/POST with simple headers
**Preflight Request** (OPTIONS first): Other methods, custom headers, etc.

```
Simple Request:           Preflight Request:
───────────────           ──────────────────
GET /api/data             OPTIONS /api/data
Origin: a.com             Origin: a.com
                          Access-Control-Request-Method: POST
                          Access-Control-Request-Headers: X-Custom
                          ────────────────────────────────
                          200 OK
                          Access-Control-Allow-Origin: a.com
                          Access-Control-Allow-Methods: POST
                          ────────────────────────────────
                          POST /api/data (actual request)
```

**Reference**: [MDN - CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS) | [Firefox nsCORSListenerProxy](https://searchfox.org/mozilla-central/source/netwerk/protocol/http/nsCORSListenerProxy.cpp)

---

## 2.8 Per-Window Isolation: Client Per WindowContext

In thdwb, each browser window/tab has its own `WindowContext` with isolated network state:

```go
// WindowContext holds all state for a single browser window/tab instance.
// This enables multiple independent windows to run without memory interference.
// Each window has its own:
// - HTTP client (with isolated cookie jar)
// - Resource cache
// - Image cache
// - HSTS store
// - TOFU store
type WindowContext struct {
    HTTPClient    *http.Client
    ResourceCache *ResourceCache
    ImgCache      *ImgCache
    HSTSStore     *HSTSStore
    TOFUStore     *TOFUStore
    // ... other window-specific state
}
```

**Why isolate?**
- Tab A logs into `site.com` → cookies in Tab A's jar
- Tab B visits `site.com` → NO cookies (different jar)
- This is **Container Tabs** behavior (Firefox feature)

**Reference**: [Firefox Container Tabs](https://searchfox.org/mozilla-central/source/browser/components/tabbrowser) | [MDN - Origin](https://developer.mozilla.org/en-US/docs/Glossary/Origin)

---

## 2.9 Code Walkthrough: sauce.go Key Functions

### `fetchExternalPageWithOptions` — Core Fetch Logic

```go
// fetchExternalPageWithOptions makes an HTTP request with custom method,
// headers, and body using the provided client.
// Handles:
// - CORS preflight automatically for cross-origin requests
// - Redirect following (up to 10)
// - Cookie jar attachment
// - Response caching
func fetchExternalPageWithOptions(
    client *http.Client,
    method, urlString string,
    headers map[string]string,
    body io.Reader,
) (*Resource, error)
```

**Flow:**
```
1. Parse URL, determine origin
2. Check HSTS store → upgrade to HTTPS if needed
3. Build HTTP request with headers
4. Attach cookies from origin's jar
5. If cross-origin & not simple → send OPTIONS preflight
6. Send actual request
7. Process response:
   - Store cookies in origin's jar
   - Update HSTS from headers
   - Cache if cacheable
8. Return Resource
```

### `loadErrorPage` — User-Friendly Errors

```go
// loadErrorPage returns an HTML error page for display when fetch fails.
// Includes: error code, message, suggestion to retry.
func loadErrorPage(urlString string, err error) *Resource
```

Returns a `Resource` with `text/html` body so the HTML parser can render it.

---

## 2.10 Cross-References

| Concept | Chapter |
|---------|---------|
| How HTML parser uses fetched HTML | [Chapter 3: HTML Parsing](../03-html-parsing-dom.md) |
| How CSS parser uses fetched CSS | [Chapter 4: CSS Styling](../04-css-styling.md) |
| How JS Fetch API uses network | [Chapter 7: JavaScript Runtime](../07-javascript-runtime.md) |
| Cookie partitioning & origins | [Glossary: Origin](../glossary.md#origin), [Glossary: Cookie](../glossary.md#cookie) |
| Certificate validation details | [Glossary: HTTPS](../glossary.md#https) |

---

## 2.11 Example Test: Fetching a Resource

```go
// Example_GetResource demonstrates fetching a resource with the sauce package.
func Example_GetResource() {
    // Create a window context (provides isolated HTTP client, caches, etc.)
    ctx := hotdog.NewWindowContext()
    defer ctx.Close()

    // Fetch a page
    resource, err := sauce.GetResource("https://example.com")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Status: %d\n", resource.StatusCode)
    fmt.Printf("Content-Type: %s\n", resource.ContentType)
    fmt.Printf("Body length: %d bytes\n", len(resource.Body))
    fmt.Printf("From cache: %v\n", resource.FromCache)

    // Output:
    // Status: 200
    // Content-Type: text/html; charset=UTF-8
    // Body length: 1256 bytes
    // From cache: false
}
```

Run with: `go test -run Example_GetResource ./sauce`

---

## 2.12 Summary

- **sauce** is thdwb's network layer — the "postal service"
- **OriginCookieJar** partitions cookies by origin (prevents tracking)
- **ResourceCache** and **ImgCache** avoid re-downloading/re-decoding
- **TOFU** lets users explicitly trust self-signed certificates
- **HSTS** forces HTTPS after first secure visit
- **CORS** controls cross-origin requests with preflight
- Each **WindowContext** has isolated network state (container tabs)

---

## 2.13 Next Chapter

👉 **[Chapter 3: HTML Parsing & DOM — ketchup + hotdog](../03-html-parsing-dom.md)**

Learn how raw HTML text becomes a structured DOM tree that the browser can work with.

---

*Key terms from this chapter: [Cache](../glossary.md#cache), [CORS](../glossary.md#cors), [Cookie](../glossary.md#cookie), [Cookie Jar](../glossary.md#cookie-jar), [Certificate](../glossary.md#certificate), [HSTS](../glossary.md#hsts), [HTTP](../glossary.md#http), [HTTPS](../glossary.md#https), [Origin](../glossary.md#origin), [TOFU](../glossary.md#tofu)*