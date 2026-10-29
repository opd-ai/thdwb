package debug

import (
	"sync"
	"time"
)

// NetworkRequest records a single HTTP request/response for debugging.
type NetworkRequest struct {
	Timestamp     time.Time
	Method        string
	URL           string
	StatusCode    int
	ResponseSize  int64
	Duration      time.Duration
	Origin        string
	ResourceType  string // "document", "script", "stylesheet", "image", "xhr", "other"
}

// NetworkMonitor captures network requests for debugging.
type NetworkMonitor struct {
	mu      sync.Mutex
	requests []*NetworkRequest
	max     int
}

// NewNetworkMonitor creates a NetworkMonitor with the given max entries.
func NewNetworkMonitor(max int) *NetworkMonitor {
	return &NetworkMonitor{max: max}
}

// Record adds a network request to the monitor.
func (nm *NetworkMonitor) Record(method, urlStr string, statusCode int, responseSize int64, duration time.Duration, origin, resourceType string) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	nm.requests = append(nm.requests, &NetworkRequest{
		Timestamp:    time.Now(),
		Method:       method,
		URL:          urlStr,
		StatusCode:   statusCode,
		ResponseSize: responseSize,
		Duration:     duration,
		Origin:       origin,
		ResourceType: resourceType,
	})

	if len(nm.requests) > nm.max {
		nm.requests = nm.requests[len(nm.requests)-nm.max:]
	}
}

// Requests returns a copy of all captured network requests.
func (nm *NetworkMonitor) Requests() []*NetworkRequest {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	out := make([]*NetworkRequest, len(nm.requests))
	copy(out, nm.requests)
	return out
}

// FailedRequests returns requests with status >= 400.
func (nm *NetworkMonitor) FailedRequests() []*NetworkRequest {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	var out []*NetworkRequest
	for _, r := range nm.requests {
		if r.StatusCode >= 400 {
			out = append(out, r)
		}
	}
	return out
}

// Clear removes all requests.
func (nm *NetworkMonitor) Clear() {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.requests = nil
}

// Count returns the number of captured requests.
func (nm *NetworkMonitor) Count() int {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	return len(nm.requests)
}
