package hotdog

import (
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/danfragoso/thdwb/assets"
	"github.com/danfragoso/thdwb/profiler"
)

// TestLoadHTMLAgainstRAFLoopNoRace (audit C-25) reproduces the race the
// round-1 fixes left in the navigation path: a self-arming
// requestAnimationFrame loop from the outgoing document runs on the
// time.AfterFunc goroutine *under* jsRuntimeMu and reads wc.Origin /
// wc.ActiveDocument, while LoadHTML assigned those fields with no lock held
// before it stopped the timer. Pre-fix the audit probe reported 11 data
// races (SetOrigin struct write vs console.log/GetOrigin reads inside goja).
// Run with -race.
func TestLoadHTMLAgainstRAFLoopNoRace(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()

	if err := wc.SetOrigin("https://a.example"); err != nil {
		t.Fatalf("SetOrigin: %v", err)
	}
	if err := wc.LoadHTML(`<html><body>first</body></html>`, "https://a.example/1"); err != nil {
		t.Fatalf("LoadHTML: %v", err)
	}

	// Page JS that keeps the frame loop alive and touches window state every
	// frame -- the exact reader shape from the probe (console.log -> GetOrigin,
	// eval -> ActiveDocument).
	if _, err := wc.RunJS(`
		var ticks = 0;
		function loop() {
			console.log("tick " + ticks);
			void document.title;
			ticks++;
			window.requestAnimationFrame(loop);
		}
		window.requestAnimationFrame(loop);
		"ok"`); err != nil {
		t.Fatalf("start rAF loop: %v", err)
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			if i > 40 {
				return
			}
			// Alternate origins so every navigation writes wc.Origin while the
			// frame loop reads it.
			page := "https://nav.example/p" + strconv.Itoa(i)
			if err := wc.LoadHTML(`<html><body>navigated</body></html>`, page); err != nil {
				if strings.Contains(err.Error(), "destroyed") {
					return
				}
				t.Errorf("concurrent LoadHTML: %v", err)
				return
			}
			// Re-arm the reader loop after each rebinding: LoadHTML clears the
			// previous document's frames, so without this the race window only
			// exists for the first navigation.
			if _, err := wc.RunJS(`
				window.requestAnimationFrame(function loop() {
					console.log("tick");
					void document.title;
					window.requestAnimationFrame(loop);
				});
				"ok"`); err != nil {
				return
			}
		}
	}()

	time.Sleep(300 * time.Millisecond)
	close(stop)
	wg.Wait()

	// Final navigation must have taken effect cleanly.
	if err := wc.LoadHTML(`<html><body>final</body></html>`, "https://c.example/final"); err != nil {
		t.Fatalf("final LoadHTML: %v", err)
	}
	if got := wc.GetOrigin().Host; !strings.HasSuffix(got, "c.example") {
		t.Fatalf("origin after final navigation = %q, want c.example", got)
	}
}

// TestFindWindowsByOriginSeesConsistentSnapshot verifies the registry scan
// reads origins through the guarded accessor: with targetOrigin != "*" it
// calls GetOrigin on foreign windows, which must not take their jsRuntimeMu
// (that would recreate the C-2 cross-window deadlock class).
func TestFindWindowsByOriginUsesOriginSnapshot(t *testing.T) {
	wc1 := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc1.Destroy()
	wc2 := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc2.Destroy()

	if err := wc1.SetOrigin("https://match.example"); err != nil {
		t.Fatal(err)
	}
	if err := wc2.SetOrigin("https://other.example"); err != nil {
		t.Fatal(err)
	}

	// Hold wc1's VM lock with a foreign goroutine: a registry scan matching
	// origins must still complete (leaf originMu, no jsRuntimeMu needed).
	held := make(chan struct{})
	unhold := make(chan struct{})
	go func() {
		wc1.WithJSRLock(func() {
			close(held)
			<-unhold
		})
	}()
	<-held

	done := make(chan []*WindowContext, 1)
	go func() {
		done <- FindWindowsByOrigin("https://match.example")
	}()
	select {
	case got := <-done:
		if len(got) != 1 || got[0] != wc1 {
			t.Fatalf("FindWindowsByOrigin = %d windows, want exactly wc1", len(got))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("FindWindowsByOrigin blocked on a foreign window's jsRuntimeMu")
	}
	close(unhold)
}
