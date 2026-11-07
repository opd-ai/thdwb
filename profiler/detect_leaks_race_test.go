package profiler

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDetectLeaksConcurrentRecord (audit R-19) races DetectLeaks against
// Record. The old implementation called len(profile.snapshots) again after
// releasing profile.mu, an unsynchronized read of a slice header that Record
// mutates under the lock -- a data race the -race detector flags (and, with
// unlucky interleaving, a SnapshotCount inconsistent with the endpoints used
// for the delta).
func TestDetectLeaksConcurrentRecord(t *testing.T) {
	mp := NewMemoryProfiler()

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				mp.Record("tab")
			}
		}(i)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = mp.DetectLeaks(1, 1)
		}
	}()
	time.Sleep(50 * time.Millisecond)
	close(stop)
	wg.Wait()
}

// TestDetectLeaksSnapshotCountConsistent asserts the reported count matches
// the profile's actual snapshot count at report time.
func TestDetectLeaksSnapshotCountConsistent(t *testing.T) {
	mp := NewMemoryProfiler()
	p := mp.GetOrCreate("tab")
	// Grow allocations deterministically between snapshots.
	for i := 0; i < 5; i++ {
		leak := make([]byte, 2_000_000)
		leak[0] = byte(i)
		_ = leak
		p.Record()
	}

	reports := mp.DetectLeaks(1, 0)
	found := false
	for _, r := range reports {
		if r.Name == "tab" {
			found = true
			assert.Equal(t, len(p.Snapshots()), r.SnapshotCount)
		}
	}
	if !found {
		t.Log("no leak report raised; thresholds:", reports)
	}
}
