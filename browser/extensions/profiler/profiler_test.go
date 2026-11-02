package profiler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtensionProfileCreateAndMetrics(t *testing.T) {
	ep := NewExtensionProfile("ext1")
	require.NotNil(t, ep)

	m := ep.Metrics()
	assert.Equal(t, "ext1", m.ExtensionID)
	assert.Equal(t, int64(0), m.NetworkBytes)
	assert.Equal(t, 0, m.NetworkCount)
	assert.GreaterOrEqual(t, m.MemoryBytes, uint64(0))
	assert.GreaterOrEqual(t, m.Goroutines, 0)
}

func TestExtensionProfileCPUTime(t *testing.T) {
	ep := NewExtensionProfile("ext1")

	ep.RecordCPU()
	time.Sleep(10 * time.Millisecond)
	ep.RecordCPU()

	m := ep.Metrics()
	assert.Greater(t, m.CPUTime, time.Duration(0))
}

func TestExtensionProfileStopCPU(t *testing.T) {
	ep := NewExtensionProfile("ext1")

	ep.RecordCPU()
	time.Sleep(10 * time.Millisecond)
	ep.StopCPU()

	cpuBefore := ep.Metrics().CPUTime

	time.Sleep(10 * time.Millisecond)
	// CPU should not accumulate while stopped.
	cpuAfter := ep.Metrics().CPUTime
	assert.Equal(t, cpuBefore, cpuAfter)
}

func TestExtensionProfileNetwork(t *testing.T) {
	ep := NewExtensionProfile("ext1")

	ep.RecordNetwork(1024)
	ep.RecordNetwork(2048)

	m := ep.Metrics()
	assert.Equal(t, int64(3072), m.NetworkBytes)
	assert.Equal(t, 2, m.NetworkCount)
}

func TestExtensionProfilerGetOrCreate(t *testing.T) {
	mgr := NewExtensionProfiler()

	p1 := mgr.GetOrCreate("ext1")
	p2 := mgr.GetOrCreate("ext1")
	require.Same(t, p1, p2)

	p3 := mgr.GetOrCreate("ext2")
	require.NotSame(t, p1, p3)
	assert.Equal(t, 2, mgr.Count())
}

func TestExtensionProfilerRemove(t *testing.T) {
	mgr := NewExtensionProfiler()
	mgr.GetOrCreate("ext1")
	mgr.GetOrCreate("ext2")
	assert.Equal(t, 2, mgr.Count())

	mgr.Remove("ext1")
	assert.Equal(t, 1, mgr.Count())

	// AllMetrics should only contain ext2.
	metrics := mgr.AllMetrics()
	require.Len(t, metrics, 1)
	assert.Equal(t, "ext2", metrics[0].ExtensionID)
}

func TestExtensionProfilerAllMetrics(t *testing.T) {
	mgr := NewExtensionProfiler()
	mgr.GetOrCreate("ext1").RecordNetwork(512)
	mgr.GetOrCreate("ext2").RecordNetwork(1024)

	metrics := mgr.AllMetrics()
	require.Len(t, metrics, 2)

	ids := map[string]bool{}
	for _, m := range metrics {
		ids[m.ExtensionID] = true
	}
	assert.True(t, ids["ext1"])
	assert.True(t, ids["ext2"])
}
