package hotdog

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/danfragoso/thdwb/assets"
	"github.com/danfragoso/thdwb/profiler"
)

// TestXHRReopenDuringInFlightNoRace (audit C-24) guards the race the C-6 fix
// left behind: the network goroutine read the live closure variables
// (requestURL, requestMethod, requestHeaders, requestBody, withCredentials)
// while open() on the JS thread rebuilt them, and two in-flight sends wrote
// the same response vars. Pre-fix this probe reported six data races under
// -race, including a concurrent map read/write that can escalate to a fatal
// "concurrent map writes".
func TestXHRReopenDuringInFlightNoRace(t *testing.T) {
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

	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	if err := wc.SetOrigin(srv.URL); err != nil {
		t.Fatalf("SetOrigin: %v", err)
	}
	if err := wc.InitJSRuntime(); err != nil {
		t.Fatalf("InitJSRuntime: %v", err)
	}
	defer wc.Destroy()

	// Same XHR object: send, then re-open + setRequestHeader + send while the
	// first request is still in flight. Exactly the audit probe shape.
	script := `
		var x = new XMLHttpRequest();
		x.open("GET", "` + srv.URL + `/slow");
		x.setRequestHeader("X-A", "one");
		x.send("body-one");
		x.open("POST", "` + srv.URL + `/slow2");
		x.setRequestHeader("X-B", "two");
		x.send("body-two");
		"ok"`
	if _, err := wc.RunJS(script); err != nil {
		t.Fatalf("run XHR script: %v", err)
	}

	// Two windows, one shared object shape: also fire two sends on separate
	// objects rapidly to keep goroutines overlapping.
	if _, err := wc.RunJS(`
		for (var i = 0; i < 10; i++) {
			var y = new XMLHttpRequest();
			y.open("GET", "` + srv.URL + `/slow?i=" + i);
			y.send();
		}`); err != nil {
		t.Fatalf("multi-send: %v", err)
	}

	unblock()
	// Let the frame queue settle so callbacks execute against the generation
	// guard rather than being cancelled mid-test.
	time.Sleep(200 * time.Millisecond)
}

// TestXHRStaleResponseDroppedAfterReopen verifies the generation guard: a
// response from a superseded send must not overwrite the state of the current
// request.
func TestXHRStaleResponseDroppedAfterReopen(t *testing.T) {
	var mu sync.Mutex
	delays := map[string]time.Duration{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		d := delays[strings.TrimPrefix(r.URL.Path, "/")]
		mu.Unlock()
		time.Sleep(d)
		_, _ = w.Write([]byte("body:" + r.URL.Path))
	}))
	defer srv.Close()

	mu.Lock()
	delays["slow"] = 300 * time.Millisecond
	delays["fast"] = 10 * time.Millisecond
	mu.Unlock()

	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	if err := wc.SetOrigin(srv.URL); err != nil {
		t.Fatalf("SetOrigin: %v", err)
	}
	if err := wc.InitJSRuntime(); err != nil {
		t.Fatalf("InitJSRuntime: %v", err)
	}
	defer wc.Destroy()

	// First send to /slow is superseded by the second send to /fast before it
	// completes. The stale /slow response must be dropped.
	if _, err := wc.RunJS(`
		var x = new XMLHttpRequest();
		x.open("GET", "` + srv.URL + `/slow");
		x.send();
		x.open("GET", "` + srv.URL + `/fast");
		x.send();
		var seen = [];
		x.onreadystatechange = function() { seen.push(x.responseText); };
		"ok"`); err != nil {
		t.Fatalf("run script: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		v, err := wc.RunJS(`seen.length > 0 ? String(seen[0]) : ""`)
		if err == nil && v.String() != "" {
			got := v.String()
			if strings.Contains(got, "/slow") {
				t.Fatalf("stale response from superseded send applied: %q", got)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("no readystatechange observed within deadline")
}
