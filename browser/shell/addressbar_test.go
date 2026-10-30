package shell

import (
	"testing"
)

func TestAddressBarReflectsActiveTabURL(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	ab := NewAddressBar(window, m)

	a := m.Add("A", "https://example.com", "", "", "")
	if got := ab.URL(); got != "https://example.com" {
		t.Fatalf("URL = %q, want https://example.com", got)
	}
	if ab.SecurityState() != "secure" {
		t.Fatalf("SecurityState = %q, want secure", ab.SecurityState())
	}

	m.Add("B", "http://insecure.example", "", "", "")
	m.Activate(a.ID)
	if got := ab.URL(); got != "https://example.com" {
		t.Fatalf("URL after reactivate = %q, want https://example.com", got)
	}
}

func TestAddressBarSecurityIndicator(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	ab := NewAddressBar(window, m)

	m.Add("HTTPS", "https://secure.site", "", "", "")
	if ab.SecurityState() != "secure" {
		t.Fatalf("https SecurityState = %q, want secure", ab.SecurityState())
	}

	m.Add("HTTP", "http://insecure.site", "", "", "")
	m.Activate(m.ActiveTab().ID)
	if ab.SecurityState() != "insecure" {
		t.Fatalf("http SecurityState = %q, want insecure", ab.SecurityState())
	}

	m.Add("THDWB", "thdwb://newtab/", "", "", "")
	m.Activate(m.ActiveTab().ID)
	if ab.SecurityState() != "secure" {
		t.Fatalf("thdwb SecurityState = %q, want secure", ab.SecurityState())
	}
}

func TestAddressBarContainerIndicator(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	ab := NewAddressBar(window, m)

	m.Add("Default", "https://example.com", "", "", "")
	if ab.ContainerID() != "" {
		t.Fatalf("default ContainerID = %q, want empty", ab.ContainerID())
	}

	ctr := m.Add("Work", "https://work.example", "c1", "Work", "#ff0000")
	m.Activate(ctr.ID)
	if ab.ContainerID() != "c1" {
		t.Fatalf("container ContainerID = %q, want c1", ab.ContainerID())
	}
}

func TestAddressBarNavigateUpdatesTab(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	ab := NewAddressBar(window, m)

	a := m.Add("A", "https://example.com", "", "", "")
	ab.navigate("https://newsite.com")
	if got := a.URL; got != "https://newsite.com" {
		t.Fatalf("tab URL = %q, want https://newsite.com", got)
	}
	if got := ab.URL(); got != "https://newsite.com" {
		t.Fatalf("address bar URL = %q, want https://newsite.com", got)
	}
}

func TestAddressBarNavigateAddsHTTPScheme(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	ab := NewAddressBar(window, m)

	m.Add("A", "https://example.com", "", "", "")
	ab.navigate("example.com")
	if got := m.ActiveTab().URL; got != "https://example.com" {
		t.Fatalf("tab URL = %q, want https://example.com", got)
	}
}

func TestAddressBarBuildSuggestionsFromHistory(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	ab := NewAddressBar(window, m)

	ab.SetHistoryProvider(func() []*HistoryEntry {
		return []*HistoryEntry{
			{ID: 1, Title: "Example Site", URL: "https://example.com"},
			{ID: 2, Title: "Other Site", URL: "https://other.com"},
		}
	})

	ab.buildSuggestions("example")
	if len(ab.suggestions) == 0 {
		t.Fatal("expected suggestions for 'example'")
	}
	found := false
	for _, s := range ab.suggestions {
		if s.URL == "https://example.com" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected example.com in suggestions, got %+v", ab.suggestions)
	}
}

func TestAddressBarBuildSuggestionsFromBookmarks(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	ab := NewAddressBar(window, m)

	ab.SetBookmarkProvider(func() []*Bookmark {
		return []*Bookmark{
			{ID: 1, Title: "Bookmark A", URL: "https://bookmark-a.com"},
		}
	})

	ab.buildSuggestions("bookmark")
	if len(ab.suggestions) == 0 {
		t.Fatal("expected suggestions for 'bookmark'")
	}
	found := false
	for _, s := range ab.suggestions {
		if s.URL == "https://bookmark-a.com" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected bookmark-a.com in suggestions, got %+v", ab.suggestions)
	}
}

func TestAddressBarSelectSuggestionNavigates(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	ab := NewAddressBar(window, m)

	a := m.Add("A", "https://example.com", "", "", "")
	ab.selectSuggestion(suggestion{Title: "Test", URL: "https://suggested.com"})
	if got := a.URL; got != "https://suggested.com" {
		t.Fatalf("tab URL after suggestion select = %q, want https://suggested.com", got)
	}
}

func TestAddressBarSelectNextAndPreviousSuggestion(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	ab := NewAddressBar(window, m)

	ab.suggestions = []suggestion{
		{Title: "One", URL: "https://one.com"},
		{Title: "Two", URL: "https://two.com"},
		{Title: "Three", URL: "https://three.com"},
	}

	if ab.selectedSugg != 0 {
		t.Fatalf("initial selectedSugg = %d, want 0", ab.selectedSugg)
	}

	ab.selectNextSuggestion()
	if ab.selectedSugg != 1 {
		t.Fatalf("after next selectedSugg = %d, want 1", ab.selectedSugg)
	}

	ab.selectNextSuggestion()
	if ab.selectedSugg != 2 {
		t.Fatalf("after second next selectedSugg = %d, want 2", ab.selectedSugg)
	}

	// Wrap around.
	ab.selectNextSuggestion()
	if ab.selectedSugg != 0 {
		t.Fatalf("after wrap next selectedSugg = %d, want 0", ab.selectedSugg)
	}

	ab.selectPreviousSuggestion()
	if ab.selectedSugg != 2 {
		t.Fatalf("after previous selectedSugg = %d, want 2", ab.selectedSugg)
	}
}

func TestAddressBarHideSuggestions(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	ab := NewAddressBar(window, m)

	ab.suggestions = []suggestion{{Title: "One", URL: "https://one.com"}}
	ab.suggestionsVisible = true
	ab.hideSuggestions()

	if ab.suggestionsVisible {
		t.Fatal("suggestions should be hidden")
	}
	if len(ab.suggestions) != 0 {
		t.Fatal("suggestions should be cleared")
	}
}
