package shell

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTabManagerConcurrentUse (audit C-23) reproduces the extension-API shape:
// chrome.tabs.create/remove/activate run on VM/alarm/observer goroutines while
// the Ebitengine game goroutine reads tab state every frame. Before the fix
// TabManager was explicitly "drive it from the UI loop" and the audit probe
// produced WARNING: DATA RACE on Add/Close from exactly this pattern; under
// load it escalates to fatal concurrent map writes on navs. Run with -race.
func TestTabManagerConcurrentUse(t *testing.T) {
	m := NewTabManager()

	var readers, writers sync.WaitGroup
	done := make(chan struct{})

	// UI-loop readers, running until the writers finish.
	for i := 0; i < 3; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-done:
					return
				default:
				}
				_ = m.ActiveTab()
				_ = m.Tabs()
				_ = m.Count()
				_ = m.CurrentURL(1)
			}
		}()
	}

	// Extension-goroutine writers.
	for w := 0; w < 4; w++ {
		writers.Add(1)
		go func() {
			defer writers.Done()
			for i := 0; i < 60; i++ {
				tab := m.Add("t", "https://example.com", "", "", "")
				m.SetTitle(tab.ID, "updated")
				m.Navigate(tab.ID, "https://example.com/next")
				m.Activate(tab.ID)
				_ = m.CanGoBack(tab.ID)
				m.Close(tab.ID)
			}
		}()
	}

	writers.Wait()
	close(done)
	readers.Wait()

	assert.NotNil(t, m)
}

// TestTabManagerCallbacksRunOutsideLock guards the deadlock the fix must not
// introduce: OnClosed destroys the tab's WindowContext (jsRuntimeMu), and a
// page-JS path holding that lock reaches chrome.tabs.* -> TabManager.mu. If a
// structural method held mu across onClosed, that re-entrant call would
// self-deadlock. This test fires callbacks that re-enter the manager.
func TestTabManagerCallbacksRunOutsideLock(t *testing.T) {
	m := NewTabManager()

	var reentered *Tab
	m.OnClosed(func(tabID int) {
		// Simulates a close path that re-enters the manager while the
		// structural change is being finalized: must not deadlock.
		reentered = m.Add("restore", "https://example.com", "", "", "")
	})
	m.OnChanged(func() {
		_ = m.Count() // listeners touching the manager must not deadlock either
	})

	tab := m.Add("x", "u", "", "", "")
	closed := m.Close(tab.ID)
	assert.NotNil(t, closed)
	if reentered == nil {
		t.Fatal("OnClosed callback did not run")
	}
	assert.Equal(t, "restore", reentered.Title)
}
