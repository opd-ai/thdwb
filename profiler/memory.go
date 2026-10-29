package profiler

import (
	"runtime"
	"sync"
	"time"
)

// MemorySnapshot records a point-in-time memory measurement.
type MemorySnapshot struct {
	Timestamp     time.Time
	AllocBytes    uint64 // bytes allocated and not yet freed
	TotalBytes    uint64 // bytes allocated (total, including freed)
	SysBytes      uint64 // bytes obtained from OS
	NumGC         uint32 // number of completed GC cycles
	Goroutines    int    // number of goroutines
}

// MemoryProfile tracks memory usage for a named entity (tab, window, extension).
type MemoryProfile struct {
	name      string
	snapshots []MemorySnapshot
	mu        sync.Mutex
}

// NewMemoryProfile creates a memory profile for the given entity name.
func NewMemoryProfile(name string) *MemoryProfile {
	return &MemoryProfile{name: name}
}

// Record takes a memory snapshot and appends it to the profile.
func (mp *MemoryProfile) Record() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	mp.mu.Lock()
	defer mp.mu.Unlock()

	mp.snapshots = append(mp.snapshots, MemorySnapshot{
		Timestamp:   time.Now(),
		AllocBytes:  m.Alloc,
		TotalBytes:  m.TotalAlloc,
		SysBytes:    m.Sys,
		NumGC:       m.NumGC,
		Goroutines:  runtime.NumGoroutine(),
	})
}

// Snapshots returns a copy of all recorded snapshots.
func (mp *MemoryProfile) Snapshots() []MemorySnapshot {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	out := make([]MemorySnapshot, len(mp.snapshots))
	copy(out, mp.snapshots)
	return out
}

// GetName returns the profile name.
func (mp *MemoryProfile) GetName() string {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return mp.name
}

// DeltaAlloc returns the difference in allocated bytes between the last and
// first snapshot, or 0 if fewer than two snapshots exist.
func (mp *MemoryProfile) DeltaAlloc() uint64 {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if len(mp.snapshots) < 2 {
		return 0
	}
	first := mp.snapshots[0]
	last := mp.snapshots[len(mp.snapshots)-1]
	if last.AllocBytes >= first.AllocBytes {
		return last.AllocBytes - first.AllocBytes
	}
	return 0
}

// DeltaGoroutines returns the difference in goroutine count between the last
// and first snapshot, or 0 if fewer than two snapshots exist.
func (mp *MemoryProfile) DeltaGoroutines() int {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if len(mp.snapshots) < 2 {
		return 0
	}
	return mp.snapshots[len(mp.snapshots)-1].Goroutines - mp.snapshots[0].Goroutines
}

// MemoryProfiler manages memory profiles for multiple named entities.
type MemoryProfiler struct {
	profiles map[string]*MemoryProfile
	mu       sync.Mutex
}

// NewMemoryProfiler creates a new MemoryProfiler.
func NewMemoryProfiler() *MemoryProfiler {
	return &MemoryProfiler{profiles: make(map[string]*MemoryProfile)}
}

// GetOrCreate returns the memory profile for name, creating it if needed.
func (mp *MemoryProfiler) GetOrCreate(name string) *MemoryProfile {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if p, ok := mp.profiles[name]; ok {
		return p
	}
	p := NewMemoryProfile(name)
	mp.profiles[name] = p
	return p
}

// Record takes a snapshot for the named entity.
func (mp *MemoryProfiler) Record(name string) {
	mp.GetOrCreate(name).Record()
}

// Profiles returns all memory profiles.
func (mp *MemoryProfiler) Profiles() map[string]*MemoryProfile {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	out := make(map[string]*MemoryProfile, len(mp.profiles))
	for k, v := range mp.profiles {
		out[k] = v
	}
	return out
}

// LeakReport describes a suspected memory leak for a named entity.
type LeakReport struct {
	Name           string
	DeltaAlloc     uint64
	DeltaGoroutines int
	SnapshotCount  int
}

// DetectLeaks checks all profiles for potential leaks: allocated bytes or
// goroutines that grew significantly between first and last snapshot.
func (mp *MemoryProfiler) DetectLeaks(allocThreshold uint64, goroutineThreshold int) []LeakReport {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	var reports []LeakReport
	for name, profile := range mp.profiles {
		profile.mu.Lock()
		if len(profile.snapshots) < 2 {
			profile.mu.Unlock()
			continue
		}
		first := profile.snapshots[0]
		last := profile.snapshots[len(profile.snapshots)-1]
		profile.mu.Unlock()

		deltaAlloc := uint64(0)
		if last.AllocBytes >= first.AllocBytes {
			deltaAlloc = last.AllocBytes - first.AllocBytes
		}
		deltaGoroutines := last.Goroutines - first.Goroutines

		if deltaAlloc > allocThreshold || deltaGoroutines > goroutineThreshold {
			reports = append(reports, LeakReport{
				Name:            name,
				DeltaAlloc:      deltaAlloc,
				DeltaGoroutines: deltaGoroutines,
				SnapshotCount:   len(profile.snapshots),
			})
		}
	}
	return reports
}
