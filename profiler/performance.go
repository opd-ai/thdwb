package profiler

import (
	"sync"
	"time"
)

// PageLoadMetrics records timing for a single page load.
type PageLoadMetrics struct {
	URL             string
	NavigationStart time.Time
	DOMComplete     time.Time
	LoadComplete    time.Time
	FirstPaint      time.Time
	FirstContentful time.Time
}

// LoadDuration returns the time from navigation start to load complete.
func (m *PageLoadMetrics) LoadDuration() time.Duration {
	return m.LoadComplete.Sub(m.NavigationStart)
}

// DOMContentLoadedDuration returns the time from navigation start to DOM complete.
func (m *PageLoadMetrics) DOMContentLoadedDuration() time.Duration {
	return m.DOMComplete.Sub(m.NavigationStart)
}

// FirstPaintLatency returns the time from navigation start to first paint.
func (m *PageLoadMetrics) FirstPaintLatency() time.Duration {
	return m.FirstPaint.Sub(m.NavigationStart)
}

// FirstContentfulPaintLatency returns the time from navigation start to first contentful paint.
func (m *PageLoadMetrics) FirstContentfulPaintLatency() time.Duration {
	return m.FirstContentful.Sub(m.NavigationStart)
}

// PageLoadTracker records page load metrics for multiple URLs.
type PageLoadTracker struct {
	mu      sync.Mutex
	loads   map[string]*PageLoadMetrics
	order   []string
}

// NewPageLoadTracker creates a new PageLoadTracker.
func NewPageLoadTracker() *PageLoadTracker {
	return &PageLoadTracker{
		loads: make(map[string]*PageLoadMetrics),
	}
}

// StartLoad records the navigation start for a URL and returns a metrics object.
func (plt *PageLoadTracker) StartLoad(url string) *PageLoadMetrics {
	plt.mu.Lock()
	defer plt.mu.Unlock()

	m := &PageLoadMetrics{
		URL:             url,
		NavigationStart: time.Now(),
	}
	plt.loads[url] = m
	plt.order = append(plt.order, url)
	return m
}

// GetLoad returns the page load metrics for a URL, or nil if not found.
func (plt *PageLoadTracker) GetLoad(url string) *PageLoadMetrics {
	plt.mu.Lock()
	defer plt.mu.Unlock()
	return plt.loads[url]
}

// AllLoads returns a copy of all page load metrics in order.
func (plt *PageLoadTracker) AllLoads() []*PageLoadMetrics {
	plt.mu.Lock()
	defer plt.mu.Unlock()

	out := make([]*PageLoadMetrics, len(plt.order))
	for i, url := range plt.order {
		out[i] = plt.loads[url]
	}
	return out
}

// AverageLoadDuration returns the average load duration across all completed loads.
func (plt *PageLoadTracker) AverageLoadDuration() time.Duration {
	plt.mu.Lock()
	defer plt.mu.Unlock()

	var total time.Duration
	count := 0
	for _, url := range plt.order {
		m := plt.loads[url]
		if !m.LoadComplete.IsZero() {
			total += m.LoadDuration()
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / time.Duration(count)
}

// ExtensionStartupMetrics records timing for extension startup.
type ExtensionStartupMetrics struct {
	ExtensionID   string
	ManifestParse time.Duration
	VMInit        time.Duration
	APIBind       time.Duration
	Total         time.Duration
}

// ExtensionStartupTracker records extension startup metrics.
type ExtensionStartupTracker struct {
	mu      sync.Mutex
	startups map[string]*ExtensionStartupMetrics
}

// NewExtensionStartupTracker creates a new ExtensionStartupTracker.
func NewExtensionStartupTracker() *ExtensionStartupTracker {
	return &ExtensionStartupTracker{
		startups: make(map[string]*ExtensionStartupMetrics),
	}
}

// Record records extension startup metrics.
func (est *ExtensionStartupTracker) Record(extID string, m *ExtensionStartupMetrics) {
	est.mu.Lock()
	defer est.mu.Unlock()
	est.startups[extID] = m
}

// Get returns the startup metrics for an extension, or nil if not found.
func (est *ExtensionStartupTracker) Get(extID string) *ExtensionStartupMetrics {
	est.mu.Lock()
	defer est.mu.Unlock()
	return est.startups[extID]
}

// All returns all extension startup metrics.
func (est *ExtensionStartupTracker) All() map[string]*ExtensionStartupMetrics {
	est.mu.Lock()
	defer est.mu.Unlock()

	out := make(map[string]*ExtensionStartupMetrics, len(est.startups))
	for k, v := range est.startups {
		out[k] = v
	}
	return out
}

// AverageStartupTime returns the average total startup time across all extensions.
func (est *ExtensionStartupTracker) AverageStartupTime() time.Duration {
	est.mu.Lock()
	defer est.mu.Unlock()

	var total time.Duration
	for _, m := range est.startups {
		total += m.Total
	}
	if len(est.startups) == 0 {
		return 0
	}
	return total / time.Duration(len(est.startups))
}

// PerformanceMetrics aggregates all performance tracking.
type PerformanceMetrics struct {
	PageLoads   *PageLoadTracker
	Extensions  *ExtensionStartupTracker
}

// NewPerformanceMetrics creates a new PerformanceMetrics aggregator.
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		PageLoads:  NewPageLoadTracker(),
		Extensions: NewExtensionStartupTracker(),
	}
}
