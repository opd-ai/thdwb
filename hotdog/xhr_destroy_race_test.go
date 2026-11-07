package hotdog

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/danfragoso/thdwb/assets"
	"github.com/danfragoso/thdwb/profiler"
)

// TestXHRSendAgainstDestroyNoRace guards the race where the XHR network
// goroutine read window-scoped fields (Origin, httpClient) with no lock held
// while Destroy() nil-ed them on tab close. Those reads are now snapshotted on
// the JS thread before the goroutine is spawned, which serializes them against
// Destroy via jsRuntimeMu. Run with -race.
//
// The server stalls so the request goroutine stays alive across iterations,
// overlapping a concurrent Destroy.
func TestXHRSendAgainstDestroyNoRace(t *testing.T) {
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-time.After(30 * time.Second):
		}
		_, _ = w.Write([]byte("done"))
	}))
	defer srv.Close()

	for i := 0; i < 25; i++ {
		wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
		if err := wc.SetOrigin(srv.URL); err != nil {
			t.Fatalf("SetOrigin: %v", err)
		}
		if err := wc.InitJSRuntime(); err != nil {
			t.Fatalf("InitJSRuntime: %v", err)
		}

		if _, err := wc.RunJS(`
			var x = new XMLHttpRequest();
			x.open("GET", "` + srv.URL + `/slow");
			x.send();
			"ok"`); err != nil {
			t.Fatalf("start XHR: %v", err)
		}

		// Destroy immediately, racing the goroutine's window-state reads.
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			wc.Destroy()
		}()

		done := make(chan struct{})
		go func() { wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(20 * time.Second):
			t.Fatal("Destroy blocked while an XHR was in flight")
		}
	}
	unblock()
}
