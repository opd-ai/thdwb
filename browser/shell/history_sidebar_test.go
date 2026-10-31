package shell

import (
	"testing"

	mustard "github.com/danfragoso/thdwb/mustard"
)

func TestHistorySidebarCreateAndRebuild(t *testing.T) {
	window := newTestWindow(t)
	hm := NewHistoryManager()
	hm.AddEntry("Example", "https://example.com")
	hm.AddEntry("Other", "https://other.com")

	sidebar := NewHistorySidebar(window, hm)
	if sidebar == nil {
		t.Fatal("NewHistorySidebar returned nil")
	}
	if sidebar.Frame == nil {
		t.Fatal("sidebar Frame should not be nil")
	}
	if sidebar.IsVisible() {
		t.Fatal("sidebar should be hidden by default")
	}
}

func TestHistorySidebarToggle(t *testing.T) {
	window := newTestWindow(t)
	hm := NewHistoryManager()

	sidebar := NewHistorySidebar(window, hm)
	if sidebar.IsVisible() {
		t.Fatal("sidebar should start hidden")
	}
	sidebar.Toggle()
	if !sidebar.IsVisible() {
		t.Fatal("sidebar should be visible after Toggle")
	}
	sidebar.Toggle()
	if sidebar.IsVisible() {
		t.Fatal("sidebar should be hidden after second Toggle")
	}
}

func TestHistorySidebarSearch(t *testing.T) {
	window := newTestWindow(t)
	hm := NewHistoryManager()
	hm.AddEntry("Example Site", "https://example.com")
	hm.AddEntry("Other Site", "https://other.com")

	sidebar := NewHistorySidebar(window, hm)

	// Search for "example" should match one entry.
	sidebar.Search("example")
	visible := 0
	for _, w := range sidebar.list.Widgets() {
		if frame, ok := w.(*mustard.Frame); ok && frame.GetVisible() {
			visible++
		}
	}
	if visible != 1 {
		t.Fatalf("visible entries after search 'example' = %d, want 1", visible)
	}

	// Empty search shows all.
	sidebar.Search("")
	visible = 0
	for _, w := range sidebar.list.Widgets() {
		if frame, ok := w.(*mustard.Frame); ok && frame.GetVisible() {
			visible++
		}
	}
	if visible != 2 {
		t.Fatalf("visible entries after empty search = %d, want 2", visible)
	}
}

func TestHistorySidebarSetVisible(t *testing.T) {
	window := newTestWindow(t)
	hm := NewHistoryManager()

	sidebar := NewHistorySidebar(window, hm)
	sidebar.SetVisible(false)
	if sidebar.IsVisible() {
		t.Fatal("sidebar should be hidden after SetVisible(false)")
	}
	sidebar.SetVisible(true)
	if !sidebar.IsVisible() {
		t.Fatal("sidebar should be visible after SetVisible(true)")
	}
}
