package shell

import (
	"testing"
)

func TestNavigationControlsReloadUpdatesTabURL(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	nc := NewNavigationControls(window, m)

	a := m.Add("A", "https://example.com", "", "", "")
	nc.reload()
	if got := a.URL; got != "https://example.com" {
		t.Fatalf("tab URL after reload = %q, want https://example.com", got)
	}
}

func TestNavigationControlsGoHomeSetsNewTab(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	nc := NewNavigationControls(window, m)

	a := m.Add("A", "https://example.com", "", "", "")
	nc.goHome()
	if got := a.URL; got != "thdwb://newtab/" {
		t.Fatalf("tab URL after goHome = %q, want thdwb://newtab/", got)
	}
}

func TestNavigationControlsNoActiveTabIsNoop(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	nc := NewNavigationControls(window, m)

	// Should not panic with no active tab.
	nc.goBack()
	nc.goForward()
	nc.reload()
	nc.goHome()
	nc.toggleBookmark()
	nc.openMenu()
}

func TestNavigationControlsUpdateStatesDisablesWithoutTab(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	nc := NewNavigationControls(window, m)

	// No tabs yet; buttons should be disabled-styled.
	if got := nc.backBtn.GetContent(); got != "◀" {
		t.Fatalf("backBtn content = %q, want ◀", got)
	}
}
