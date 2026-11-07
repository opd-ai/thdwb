package hotdog

import (
	"sync"
	"testing"
	"time"

	"github.com/danfragoso/thdwb/assets"
	"github.com/danfragoso/thdwb/profiler"
)

// newABBAWindow builds a registered, JS-initialized window on origin with a
// message counter listener.
func newABBAWindow(t *testing.T, origin string) *WindowContext {
	t.Helper()
	settings := &Settings{}
	wc := NewWindowContext(settings, &assets.BuildInfo{}, profiler.CreateProfiler())
	if err := wc.SetOrigin(origin); err != nil {
		t.Fatalf("SetOrigin: %v", err)
	}
	if err := wc.InitJSRuntime(); err != nil {
		t.Fatalf("InitJSRuntime: %v", err)
	}
	if _, err := wc.RunJS(`window.__n=0; window.addEventListener("message", function(e){ window.__n++; }); "ok"`); err != nil {
		t.Fatalf("install listener: %v", err)
	}
	return wc
}

// TestPostMessageConcurrentNoDeadlock guards against the ABBA lock cycle that
// occurred when two windows posted to each other at the same time: the sender
// held its own jsRuntimeMu while blocking on the target's, and Go map
// iteration order made the acquisition order differ per goroutine. Delivery
// now uses try-lock semantics, so a contended target is served by its own
// animation-frame pump instead of blocking the sender.
func TestPostMessageConcurrentNoDeadlock(t *testing.T) {
	origin := "https://abba.example"
	a := newABBAWindow(t, origin)
	b := newABBAWindow(t, origin)
	defer a.Destroy()
	defer b.Destroy()

	done := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = a.RunJS(`window.postMessage("x", "` + origin + `")`)
		}()
		go func() {
			defer wg.Done()
			_, _ = b.RunJS(`window.postMessage("y", "` + origin + `")`)
		}()
	}
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("concurrent cross-window postMessage deadlocked")
	}
}

// TestPostMessageDeliversToTarget checks that delivery still lands (either
// synchronously when uncontended or via the target's pump under contention).
func TestPostMessageDeliversToTarget(t *testing.T) {
	sender := newABBAWindow(t, "https://sender.example")
	target := newABBAWindow(t, "https://target.example")
	defer sender.Destroy()
	defer target.Destroy()

	if _, err := sender.RunJS(`window.postMessage("payload", "https://target.example")`); err != nil {
		t.Fatalf("postMessage: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		v, err := target.RunJS(`window.__n`)
		if err == nil && v.String() != "0" {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("target window never received the posted message")
}
