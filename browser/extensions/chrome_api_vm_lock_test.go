package extensions

import (
	"sync"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

// TestFireAlarmConcurrentIsRaceFree (audit C-7) drives Fire* methods from
// several goroutines while the background script runs on another. goja
// runtimes are not goroutine-safe: before vmMu existed, probe P7 produced 98
// WARNING: DATA RACE blocks (goja runWrapped vs RunProgram). With the lock this
// must pass under -race.
func TestFireAlarmConcurrentIsRaceFree(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Ext", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetTabOperations(&mockTabOps{})
	api.SetCookieOperations(&mockCookieOps{cookies: map[string][]Cookie{}})
	api.SetDownloadOperations(&mockDownloadOps{})
	require.NoError(t, api.Bind())

	vm.Set("counter", 0)
	_, err := vm.RunString(`
		chrome.tabs.onUpdated.addListener(function(id, info) { counter = counter + 1; });
		chrome.cookies.onChanged.addListener(function(c) { counter = counter + 1; });
		chrome.downloads.onChanged.addListener(function(d) { counter = counter + 1; });
	`)
	require.NoError(t, err)

	var wg sync.WaitGroup

	// Production shape: alarm timers (time.AfterFunc), webRequest observers
	// (HTTP RoundTrip goroutines) and download paths all fire events at the
	// same background VM concurrently.
	fns := []func(){
		func() { api.FireAlarm("tick") },
		func() { api.FireDownloadEvent("onChanged", DownloadItem{ID: 1, State: "downloading"}) },
		func() { api.FireCookieChanged(CookieChange{Cookie: Cookie{Name: "n", Value: "v"}}) },
		func() { api.FireTabUpdated(1, map[string]interface{}{"status": "complete"}) },
	}
	for i := 0; i < 8; i++ {
		fn := fns[i%len(fns)]
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				fn()
			}
		}()
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("Fire* concurrency did not complete")
	}
}

// TestFireReentrantFromListener proves the VM lock must be reentrant: a
// listener invoked under the lock can synchronously call chrome.tabs.create,
// which fires onCreated on the same VM and same goroutine. A plain mutex would
// self-deadlock here.
func TestFireReentrantFromListener(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Ext", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetTabOperations(&mockTabOps{})
	require.NoError(t, api.Bind())

	vm.Set("fired", 0)
	_, err := vm.RunString(`
		chrome.tabs.onCreated.addListener(function(tab) { fired = fired + 1; });
		chrome.runtime.onStartup.addListener(function() {
			chrome.tabs.create({url: "https://example.com"});
		});
	`)
	require.NoError(t, err)

	finished := make(chan struct{})
	go func() {
		defer close(finished)
		// Runs on a foreign goroutine: onStartup -> tabs.create -> onCreated,
		// a nested Fire on the same goroutine.
		api.FireRuntimeStartup()
	}()

	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatal("nested Fire* deadlocked -- vmMu must be reentrant")
	}

	v := vm.Get("fired")
	require.EqualValues(t, 1, v.ToInteger(), "onCreated listener must have run once")
}

// TestReentrantMutexBasics covers the lock primitive itself.
func TestReentrantMutexBasics(t *testing.T) {
	mu := newReentrantMutex()
	mu.lock()
	mu.lock() // same goroutine: must not block
	mu.unlock()

	other := make(chan struct{})
	go func() {
		mu.lock() // still held by this goroutine -> blocks
		mu.unlock()
		close(other)
	}()

	select {
	case <-other:
		t.Fatal("reentrantMutex let a second goroutine in while still held")
	case <-time.After(100 * time.Millisecond):
	}

	mu.unlock() // release fully
	select {
	case <-other:
	case <-time.After(2 * time.Second):
		t.Fatal("reentrantMutex never released to the waiting goroutine")
	}
}
