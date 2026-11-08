package hotdog

import (
	"testing"
	"time"

	"github.com/danfragoso/thdwb/assets"
	"github.com/danfragoso/thdwb/profiler"
)

// TestAnimationFrameHostPanicContained (audit C-27) verifies a Go panic in a
// host-scheduled animation-frame callback does not kill the process. Page-JS
// rAF callbacks are wrapped by runJSListener, but host code -- fetch
// resolve/reject, XHR completion, postMessage handoff -- schedules raw
// closures executed directly on the time.AfterFunc goroutine. Pre-fix the
// panic unwound into runtime.goFunc and aborted the browser (probe:
// "panic: assignment to entry in nil map ... runAnimationFrame").
func TestAnimationFrameHostPanicContained(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	wc.SetActive(true)
	defer wc.Destroy()

	panicked := make(chan struct{})
	// A raw host closure that panics, exactly the C-27 shape.
	wc.ScheduleAnimationFrame(func() {
		var m map[string]string
		close(panicked)
		m["boom"] = "assignment to entry in nil map" // panics
	})

	// ...followed by a normal callback that must still run on a later frame.
	after := make(chan struct{})
	wc.ScheduleAnimationFrame(func() {
		close(after)
	})

	// The test process surviving to this point IS the assertion: an
	// unrecovered panic on the timer goroutine would abort the whole binary.
	select {
	case <-panicked:
	case <-time.After(2 * time.Second):
		t.Fatal("panicking host callback never executed")
	}
	select {
	case <-after:
	case <-time.After(2 * time.Second):
		t.Fatal("frame loop stopped after a callback panicked -- later callbacks must still run")
	}
}
