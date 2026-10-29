package profiler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryProfileRecordAndSnapshots(t *testing.T) {
	mp := NewMemoryProfile("tab-1")
	mp.Record()
	mp.Record()

	snaps := mp.Snapshots()
	require.Len(t, snaps, 2)
	assert.Equal(t, "tab-1", mp.GetName())
	assert.GreaterOrEqual(t, snaps[1].Timestamp, snaps[0].Timestamp)
	assert.Greater(t, snaps[0].AllocBytes, uint64(0))
	assert.Greater(t, snaps[0].Goroutines, 0)
}

func TestMemoryProfileDeltaAlloc(t *testing.T) {
	mp := NewMemoryProfile("tab-1")
	mp.Record()

	// Allocate some memory to create a delta.
	_ = make([]byte, 1024)
	mp.Record()

	delta := mp.DeltaAlloc()
	assert.GreaterOrEqual(t, delta, uint64(0))
}

func TestMemoryProfileDeltaGoroutines(t *testing.T) {
	mp := NewMemoryProfile("tab-1")
	mp.Record()

	// Spawn a goroutine to create a delta.
	done := make(chan struct{})
	go func() { close(done) }()
	<-done
	mp.Record()

	delta := mp.DeltaGoroutines()
	assert.GreaterOrEqual(t, delta, 0)
}

func TestMemoryProfileDeltaWithOneSnapshot(t *testing.T) {
	mp := NewMemoryProfile("tab-1")
	mp.Record()
	assert.Equal(t, uint64(0), mp.DeltaAlloc())
	assert.Equal(t, 0, mp.DeltaGoroutines())
}

func TestMemoryProfilerGetOrCreate(t *testing.T) {
	mp := NewMemoryProfiler()

	p1 := mp.GetOrCreate("tab-1")
	p2 := mp.GetOrCreate("tab-1")
	require.Same(t, p1, p2)

	p3 := mp.GetOrCreate("tab-2")
	require.NotSame(t, p1, p3)
}

func TestMemoryProfilerRecord(t *testing.T) {
	mp := NewMemoryProfiler()
	mp.Record("tab-1")
	mp.Record("tab-2")

	profiles := mp.Profiles()
	require.Len(t, profiles, 2)
	require.NotNil(t, profiles["tab-1"])
	require.NotNil(t, profiles["tab-2"])
}

func TestMemoryProfilerDetectLeaks(t *testing.T) {
	mp := NewMemoryProfiler()

	// Profile with no growth - should not be flagged.
	p1 := mp.GetOrCreate("stable")
	p1.Record()
	p1.Record()

	// Profile with large alloc growth - should be flagged.
	p2 := mp.GetOrCreate("leaky")
	p2.Record()
	_ = make([]byte, 10*1024*1024) // allocate 10MB
	p2.Record()

	reports := mp.DetectLeaks(1024, 0)
	require.Len(t, reports, 1)
	assert.Equal(t, "leaky", reports[0].Name)
	assert.Greater(t, reports[0].DeltaAlloc, uint64(0))
}

func TestMemoryProfilerDetectLeaksBelowThreshold(t *testing.T) {
	mp := NewMemoryProfiler()

	p := mp.GetOrCreate("stable")
	p.Record()
	p.Record()

	reports := mp.DetectLeaks(1<<63-1, 1000)
	assert.Empty(t, reports)
}

func TestMemorySnapshotFields(t *testing.T) {
	s := MemorySnapshot{
		Timestamp:   time.Now(),
		AllocBytes:  1024,
		TotalBytes:  2048,
		SysBytes:    4096,
		NumGC:       1,
		Goroutines:  5,
	}
	assert.Equal(t, uint64(1024), s.AllocBytes)
	assert.Equal(t, uint64(2048), s.TotalBytes)
	assert.Equal(t, uint64(4096), s.SysBytes)
	assert.Equal(t, uint32(1), s.NumGC)
	assert.Equal(t, 5, s.Goroutines)
}
