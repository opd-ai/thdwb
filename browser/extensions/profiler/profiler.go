// Package profiler provides per-extension profiling: CPU time, memory usage,
// and network activity scoped to individual extensions.
package profiler

import (
	"sync"
	"time"

	"github.com/danfragoso/thdwb/profiler"
)

// ExtensionMetrics is a snapshot of profiling data for a single extension.
type ExtensionMetrics struct {
	ExtensionID   string    `json:"extension_id"`
	CPUTime       time.Duration `json:"cpu_time"`
	MemoryBytes   uint64    `json:"memory_bytes"`
	Goroutines    int       `json:"goroutines"`
	NetworkBytes  int64     `json:"network_bytes"`
	NetworkCount  int       `json:"network_count"`
	LastUpdated   time.Time `json:"last_updated"`
}

// ExtensionProfile tracks profiling data for a single extension.
type ExtensionProfile struct {
	mu          sync.Mutex
	extID       string
	cpuTime     time.Duration
	memoryProf  *profiler.MemoryProfile
	netBytes    int64
	netCount    int
	lastCPU     time.Time
	cpuRunning  bool
}

// NewExtensionProfile creates a profile for the given extension.
func NewExtensionProfile(extID string) *ExtensionProfile {
	return &ExtensionProfile{
		extID:      extID,
		memoryProf: profiler.NewMemoryProfile(extID),
	}
}

// Metrics returns a snapshot of current metrics.
func (ep *ExtensionProfile) Metrics() ExtensionMetrics {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	ep.memoryProf.Record()
	snaps := ep.memoryProf.Snapshots()
	var memBytes uint64
	var goroutines int
	if len(snaps) > 0 {
		last := snaps[len(snaps)-1]
		memBytes = last.AllocBytes
		goroutines = last.Goroutines
	}

	return ExtensionMetrics{
		ExtensionID:   ep.extID,
		CPUTime:       ep.cpuTime,
		MemoryBytes:   memBytes,
		Goroutines:    goroutines,
		NetworkBytes:  ep.netBytes,
		NetworkCount:  ep.netCount,
		LastUpdated:   time.Now(),
	}
}

// RecordCPU adds elapsed CPU time since the last CPU record call.
func (ep *ExtensionProfile) RecordCPU() {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	now := time.Now()
	if ep.cpuRunning {
		ep.cpuTime += now.Sub(ep.lastCPU)
	}
	ep.lastCPU = now
	ep.cpuRunning = true
}

// StopCPU stops CPU time accumulation.
func (ep *ExtensionProfile) StopCPU() {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if ep.cpuRunning {
		ep.cpuTime += time.Since(ep.lastCPU)
		ep.cpuRunning = false
	}
}

// RecordNetwork records network activity for the extension.
func (ep *ExtensionProfile) RecordNetwork(bytes int64) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	ep.netBytes += bytes
	ep.netCount++
}

// ExtensionProfiler manages profiling for multiple extensions.
type ExtensionProfiler struct {
	mu       sync.Mutex
	profiles map[string]*ExtensionProfile
}

// NewExtensionProfiler creates a new ExtensionProfiler.
func NewExtensionProfiler() *ExtensionProfiler {
	return &ExtensionProfiler{
		profiles: make(map[string]*ExtensionProfile),
	}
}

// GetOrCreate returns the profile for the given extension, creating it if needed.
func (ep *ExtensionProfiler) GetOrCreate(extID string) *ExtensionProfile {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if p, ok := ep.profiles[extID]; ok {
		return p
	}
	p := NewExtensionProfile(extID)
	ep.profiles[extID] = p
	return p
}

// Remove removes the profile for the given extension.
func (ep *ExtensionProfiler) Remove(extID string) {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	delete(ep.profiles, extID)
}

// AllMetrics returns metrics for all profiled extensions.
func (ep *ExtensionProfiler) AllMetrics() []ExtensionMetrics {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	out := make([]ExtensionMetrics, 0, len(ep.profiles))
	for _, p := range ep.profiles {
		out = append(out, p.Metrics())
	}
	return out
}

// Count returns the number of extensions being profiled.
func (ep *ExtensionProfiler) Count() int {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	return len(ep.profiles)
}
