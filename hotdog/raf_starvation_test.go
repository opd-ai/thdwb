package hotdog

import (
	"testing"
	"time"

	"github.com/danfragoso/thdwb/assets"
	"github.com/danfragoso/thdwb/profiler"
)

// newRAFWindow builds a registered, JS-initialized window for rAF tests.
func newRAFWindow(t *testing.T) *WindowContext {
	t.Helper()
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	if err := wc.SetOrigin("https://raf.example"); err != nil {
		t.Fatalf("SetOrigin: %v", err)
	}
	if err := wc.InitJSRuntime(); err != nil {
		t.Fatalf("InitJSRuntime: %v", err)
	}
	return wc
}

// TestAnimationFrameSelfReArmDoesNotStarve guards the starvation bug where the
// drain loop ran until animationFrames was empty while holding jsRuntimeMu. A
// callback that re-arms requestAnimationFrame (the canonical canvas animation
// idiom) kept the map non-empty forever, so the loop never exited, never
// released the mutex, and permanently blocked RunJS and Destroy.
func TestAnimationFrameSelfReArmDoesNotStarve(t *testing.T) {
	wc := newRAFWindow(t)

	if _, err := wc.RunJS(`
		window.__frames = 0;
		function loop() { window.__frames++; requestAnimationFrame(loop); }
		requestAnimationFrame(loop);
		"ok"`); err != nil {
		t.Fatalf("start rAF loop: %v", err)
	}

	// Give the timer pump several ticks to run.
	time.Sleep(150 * time.Millisecond)

	// A concurrent RunJS must acquire jsRuntimeMu. Before the fix this blocked
	// forever because the drain loop held it continuously.
	runDone := make(chan error, 1)
	go func() {
		_, err := wc.RunJS(`1+1`)
		runDone <- err
	}()

	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("RunJS while rAF loop active: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("RunJS starved by self-re-arming requestAnimationFrame loop")
	}

	// Frames should actually have advanced, proving the loop still animates.
	v, err := wc.RunJS(`window.__frames`)
	if err != nil {
		t.Fatalf("read frame counter: %v", err)
	}
	if v.Export() == nil || v.ToInteger() == 0 {
		t.Fatal("self-re-arming rAF loop made no progress")
	}

	// Destroy must also complete: it acquires jsRuntimeMu first.
	destroyDone := make(chan struct{})
	go func() {
		wc.Destroy()
		close(destroyDone)
	}()
	select {
	case <-destroyDone:
	case <-time.After(10 * time.Second):
		t.Fatal("Destroy starved by self-re-arming requestAnimationFrame loop")
	}
}

// TestAnimationFrameCallbackOrdering checks that callbacks queued within one
// frame run in schedule order, and that a callback scheduled by another
// callback is deferred to a later frame rather than run in the same tick.
func TestAnimationFrameCallbackOrdering(t *testing.T) {
	wc := newRAFWindow(t)
	defer wc.Destroy()

	if _, err := wc.RunJS(`
		window.__log = [];
		requestAnimationFrame(function(){ window.__log.push("a");
			requestAnimationFrame(function(){ window.__log.push("nested"); }); });
		requestAnimationFrame(function(){ window.__log.push("b"); });
		"ok"`); err != nil {
		t.Fatalf("schedule callbacks: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		v, err := wc.RunJS(`window.__log.join(",")`)
		if err == nil && v.String() == "a,b,nested" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	v, _ := wc.RunJS(`window.__log.join(",")`)
	t.Fatalf("unexpected rAF ordering/deferral: got %q want %q", v.String(), "a,b,nested")
}
