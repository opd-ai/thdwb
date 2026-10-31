package shell

import (
	"testing"
)

func TestBookmarksSidebarCreateAndRebuild(t *testing.T) {
	window := newTestWindow(t)
	bm := NewBookmarkManager()
	bm.AddBookmark("Example", "https://example.com")
	bm.AddFolder("Work")

	sidebar := NewBookmarksSidebar(window, bm)
	if sidebar == nil {
		t.Fatal("NewBookmarksSidebar returned nil")
	}
	if sidebar.Frame == nil {
		t.Fatal("sidebar Frame should not be nil")
	}
	if sidebar.IsVisible() {
		t.Fatal("sidebar should be hidden by default")
	}
}

func TestBookmarksSidebarToggle(t *testing.T) {
	window := newTestWindow(t)
	bm := NewBookmarkManager()

	sidebar := NewBookmarksSidebar(window, bm)
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

func TestBookmarksSidebarSetVisible(t *testing.T) {
	window := newTestWindow(t)
	bm := NewBookmarkManager()

	sidebar := NewBookmarksSidebar(window, bm)
	sidebar.SetVisible(false)
	if sidebar.IsVisible() {
		t.Fatal("sidebar should be hidden after SetVisible(false)")
	}
	sidebar.SetVisible(true)
	if !sidebar.IsVisible() {
		t.Fatal("sidebar should be visible after SetVisible(true)")
	}
}

func TestBookmarksSidebarSelectCallback(t *testing.T) {
	window := newTestWindow(t)
	bm := NewBookmarkManager()
	b := bm.AddBookmark("Example", "https://example.com")

	sidebar := NewBookmarksSidebar(window, bm)

	var selected *Bookmark
	sidebar.SetSelectCallback(func(bookmark *Bookmark) {
		selected = bookmark
	})

	// Verify the sidebar's select callback is wired to the tree.
	if sidebar.onSelect == nil {
		t.Fatal("sidebar onSelect callback should be set")
	}

	// Directly invoke the sidebar's onSelect callback with the bookmark.
	sidebar.onSelect(b)
	if selected == nil || selected.ID != b.ID {
		t.Fatalf("selected = %v, want bookmark %d", selected, b.ID)
	}
}
