package profiler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageLoadMetricsDurations(t *testing.T) {
	start := time.Now()
	dom := start.Add(100 * time.Millisecond)
	fp := start.Add(50 * time.Millisecond)
	fcp := start.Add(75 * time.Millisecond)
	load := start.Add(200 * time.Millisecond)

	m := &PageLoadMetrics{
		URL:             "https://example.com",
		NavigationStart: start,
		DOMComplete:     dom,
		LoadComplete:    load,
		FirstPaint:      fp,
		FirstContentful: fcp,
	}

	assert.Equal(t, 200*time.Millisecond, m.LoadDuration())
	assert.Equal(t, 100*time.Millisecond, m.DOMContentLoadedDuration())
	assert.Equal(t, 50*time.Millisecond, m.FirstPaintLatency())
	assert.Equal(t, 75*time.Millisecond, m.FirstContentfulPaintLatency())
}

func TestPageLoadTrackerStartAndGet(t *testing.T) {
	plt := NewPageLoadTracker()

	m := plt.StartLoad("https://example.com")
	require.NotNil(t, m)
	assert.Equal(t, "https://example.com", m.URL)
	assert.False(t, m.NavigationStart.IsZero())

	got := plt.GetLoad("https://example.com")
	require.NotNil(t, got)
	assert.Same(t, m, got)

	nilGot := plt.GetLoad("https://missing.com")
	assert.Nil(t, nilGot)
}

func TestPageLoadTrackerAllLoads(t *testing.T) {
	plt := NewPageLoadTracker()

	plt.StartLoad("https://a.com")
	plt.StartLoad("https://b.com")
	plt.StartLoad("https://c.com")

	all := plt.AllLoads()
	require.Len(t, all, 3)
	assert.Equal(t, "https://a.com", all[0].URL)
	assert.Equal(t, "https://b.com", all[1].URL)
	assert.Equal(t, "https://c.com", all[2].URL)
}

func TestPageLoadTrackerAverageLoadDuration(t *testing.T) {
	plt := NewPageLoadTracker()

	// No completed loads.
	assert.Equal(t, time.Duration(0), plt.AverageLoadDuration())

	m1 := plt.StartLoad("https://a.com")
	m1.LoadComplete = m1.NavigationStart.Add(100 * time.Millisecond)

	m2 := plt.StartLoad("https://b.com")
	m2.LoadComplete = m2.NavigationStart.Add(300 * time.Millisecond)

	// Third load not completed - should be excluded.
	plt.StartLoad("https://c.com")

	avg := plt.AverageLoadDuration()
	assert.Equal(t, 200*time.Millisecond, avg)
}

func TestExtensionStartupMetrics(t *testing.T) {
	m := &ExtensionStartupMetrics{
		ExtensionID:   "ext1",
		ManifestParse: 10 * time.Millisecond,
		VMInit:        20 * time.Millisecond,
		APIBind:       5 * time.Millisecond,
		Total:         35 * time.Millisecond,
	}

	assert.Equal(t, "ext1", m.ExtensionID)
	assert.Equal(t, 10*time.Millisecond, m.ManifestParse)
	assert.Equal(t, 20*time.Millisecond, m.VMInit)
	assert.Equal(t, 5*time.Millisecond, m.APIBind)
	assert.Equal(t, 35*time.Millisecond, m.Total)
}

func TestExtensionStartupTracker(t *testing.T) {
	est := NewExtensionStartupTracker()

	m1 := &ExtensionStartupMetrics{ExtensionID: "ext1", Total: 10 * time.Millisecond}
	m2 := &ExtensionStartupMetrics{ExtensionID: "ext2", Total: 30 * time.Millisecond}

	est.Record("ext1", m1)
	est.Record("ext2", m2)

	got := est.Get("ext1")
	require.NotNil(t, got)
	assert.Same(t, m1, got)

	assert.Nil(t, est.Get("ext3"))

	all := est.All()
	require.Len(t, all, 2)
	assert.Equal(t, 10*time.Millisecond, all["ext1"].Total)
	assert.Equal(t, 30*time.Millisecond, all["ext2"].Total)

	assert.Equal(t, 20*time.Millisecond, est.AverageStartupTime())
}

func TestExtensionStartupTrackerAverageEmpty(t *testing.T) {
	est := NewExtensionStartupTracker()
	assert.Equal(t, time.Duration(0), est.AverageStartupTime())
}

func TestPerformanceMetrics(t *testing.T) {
	pm := NewPerformanceMetrics()
	require.NotNil(t, pm.PageLoads)
	require.NotNil(t, pm.Extensions)

	pm.PageLoads.StartLoad("https://example.com")
	pm.Extensions.Record("ext1", &ExtensionStartupMetrics{Total: 5 * time.Millisecond})

	assert.Equal(t, 1, len(pm.PageLoads.AllLoads()))
	assert.Equal(t, 5*time.Millisecond, pm.Extensions.AverageStartupTime())
}
