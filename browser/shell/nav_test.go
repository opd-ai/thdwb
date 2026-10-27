package shell

import "testing"

// These tests cover the per-tab navigation history (Navigate/Back/Forward)
// added to TabManager, and the shell's bookmark/history wiring.

func TestTabManagerNavigateRecordsBackHistory(t *testing.T) {
	m := NewTabManager()
	tab := m.Add("A", "https://a.example", "", "", "")
	m.Navigate(tab.ID, "https://b.example")
	m.Navigate(tab.ID, "https://c.example")

	if got := m.CurrentURL(tab.ID); got != "https://c.example" {
		t.Fatalf("CurrentURL = %q, want c", got)
	}
	if !m.CanGoBack(tab.ID) {
		t.Fatal("CanGoBack = false, want true")
	}
	if m.CanGoForward(tab.ID) {
		t.Fatal("CanGoForward = true, want false (no forward history yet)")
	}
}

func TestTabManagerNavigateToSameURLIsNoop(t *testing.T) {
	m := NewTabManager()
	tab := m.Add("A", "https://a.example", "", "", "")
	m.Navigate(tab.ID, "https://a.example")
	if m.CanGoBack(tab.ID) {
		t.Fatal("navigating to the same URL should not record history")
	}
}

func TestTabManagerNavigateClearsForward(t *testing.T) {
	m := NewTabManager()
	tab := m.Add("A", "https://a.example", "", "", "")
	m.Navigate(tab.ID, "https://b.example")
	m.Navigate(tab.ID, "https://c.example")
	m.Back(tab.ID) // now at b, forward = [c]
	if !m.CanGoForward(tab.ID) {
		t.Fatal("CanGoForward = false after Back, want true")
	}
	// A new navigation clears the forward stack.
	m.Navigate(tab.ID, "https://d.example")
	if m.CanGoForward(tab.ID) {
		t.Fatal("forward history should be cleared by a new navigation")
	}
}

func TestTabManagerBackAndForward(t *testing.T) {
	m := NewTabManager()
	tab := m.Add("A", "https://a.example", "", "", "")
	m.Navigate(tab.ID, "https://b.example")
	m.Navigate(tab.ID, "https://c.example")

	// Back: c -> b
	if !m.Back(tab.ID) {
		t.Fatal("Back returned false with available history")
	}
	if got := m.CurrentURL(tab.ID); got != "https://b.example" {
		t.Fatalf("after Back CurrentURL = %q, want b", got)
	}
	if !m.CanGoForward(tab.ID) {
		t.Fatal("CanGoForward = false after Back, want true")
	}

	// Forward: b -> c
	if !m.Forward(tab.ID) {
		t.Fatal("Forward returned false with available history")
	}
	if got := m.CurrentURL(tab.ID); got != "https://c.example" {
		t.Fatalf("after Forward CurrentURL = %q, want c", got)
	}

	// No more forward.
	if m.Forward(tab.ID) {
		t.Fatal("Forward returned true with empty forward stack")
	}
}

func TestTabManagerBackWithNoHistory(t *testing.T) {
	m := NewTabManager()
	tab := m.Add("A", "https://a.example", "", "", "")
	if m.Back(tab.ID) {
		t.Fatal("Back returned true with no back history")
	}
	if m.Forward(tab.ID) {
		t.Fatal("Forward returned true with no forward history")
	}
	if m.CanGoBack(tab.ID) || m.CanGoForward(tab.ID) {
		t.Fatal("CanGoBack/CanGoForward should be false with no history")
	}
}

func TestTabManagerNavUnknownTabIsNoop(t *testing.T) {
	m := NewTabManager()
	if m.Back(999) || m.Forward(999) {
		t.Fatal("Back/Forward on unknown tab should return false")
	}
	m.Navigate(999, "https://x.example") // no-op
	if m.CanGoBack(999) || m.CanGoForward(999) {
		t.Fatal("unknown tab should have no history")
	}
}

func TestTabManagerNavCleanedUpOnClose(t *testing.T) {
	m := NewTabManager()
	tab := m.Add("A", "https://a.example", "", "", "")
	m.Navigate(tab.ID, "https://b.example")
	m.Close(tab.ID)
	// After close, the nav state for that ID should be gone.
	if _, ok := m.navs[tab.ID]; ok {
		t.Fatal("nav state should be removed when a tab is closed")
	}
}

func TestTabManagerOnNavigatedFiresOnlyOnNavigate(t *testing.T) {
	m := NewTabManager()
	tab := m.Add("A", "https://a.example", "", "", "")

	navigations := 0
	m.OnNavigated(func(t *Tab) { navigations++ })

	m.Navigate(tab.ID, "https://b.example")
	if navigations != 1 {
		t.Fatalf("navigations after Navigate = %d, want 1", navigations)
	}
	m.Back(tab.ID)
	m.Forward(tab.ID)
	if navigations != 1 {
		t.Fatalf("navigations after Back/Forward = %d, want 1 (no extra fires)", navigations)
	}
}

func TestShellToggleBookmarkAddsAndRemoves(t *testing.T) {
	bs := NewBrowserShell(nil, nil)
	tab := &Tab{ID: 1, Title: "Example", URL: "https://example.com"}

	bs.toggleBookmark(tab)
	if got := len(bs.bookmarkManager.GetAllBookmarks()); got != 1 {
		t.Fatalf("bookmarks after toggle = %d, want 1", got)
	}
	// Toggling the same URL again removes it.
	bs.toggleBookmark(tab)
	if got := len(bs.bookmarkManager.GetAllBookmarks()); got != 0 {
		t.Fatalf("bookmarks after second toggle = %d, want 0", got)
	}
}

func TestShellRecordHistoryOnlyForWebPages(t *testing.T) {
	bs := NewBrowserShell(nil, nil)

	bs.recordHistory(&Tab{URL: "https://example.com"})
	bs.recordHistory(&Tab{URL: "http://example.com"})
	bs.recordHistory(&Tab{URL: "thdwb://newtab/"}) // internal, not recorded
	bs.recordHistory(&Tab{URL: ""})                // empty, not recorded
	bs.recordHistory(nil)                          // nil, not recorded

	if got := bs.historyManager.EntryCount(); got != 2 {
		t.Fatalf("history entries = %d, want 2 (only http/https)", got)
	}
}

func TestShellNavigationWiring(t *testing.T) {
	bs := NewBrowserShell(nil, nil)
	if err := bs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer bs.Close()

	tab := bs.TabManager().ActiveTab()
	if tab == nil {
		t.Fatal("no active tab after Init")
	}

	// Navigate via the manager (as the address bar does) and confirm history.
	tabID := tab.ID
	bs.TabManager().Navigate(tabID, "https://example.com")
	if !bs.TabManager().CanGoBack(tabID) {
		t.Fatal("expected back history after navigating")
	}
	// The web navigation should have been recorded in history.
	if bs.HistoryManager().EntryCount() != 1 {
		t.Fatalf("history entries = %d, want 1", bs.HistoryManager().EntryCount())
	}

	// Back via the navigation controls' manager.
	if !bs.TabManager().Back(tabID) {
		t.Fatal("Back failed")
	}
	if got := bs.TabManager().CurrentURL(tabID); got != tab.URL {
		t.Fatalf("after Back CurrentURL = %q, want %q", got, tab.URL)
	}
}
