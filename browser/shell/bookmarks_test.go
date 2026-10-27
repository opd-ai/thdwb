package shell

import (
	"path/filepath"
	"testing"
)

func TestBookmarkManagerAddAndGet(t *testing.T) {
	bm := NewBookmarkManager()

	b := bm.AddBookmark("Example", "https://example.com")
	if b.ID != 1 {
		t.Fatalf("ID = %d, want 1", b.ID)
	}
	if b.Title != "Example" || b.URL != "https://example.com" {
		t.Fatalf("bookmark = %+v, want title Example url https://example.com", b)
	}
	if b.CreatedAt.IsZero() {
		t.Fatal("CreatedAt should not be zero")
	}

	got := bm.GetBookmark(1)
	if got == nil || got.ID != 1 {
		t.Fatalf("GetBookmark(1) = %v, want bookmark with ID 1", got)
	}
	if bm.GetBookmark(999) != nil {
		t.Fatal("GetBookmark(999) should return nil")
	}
}

func TestBookmarkManagerGetAll(t *testing.T) {
	bm := NewBookmarkManager()
	bm.AddBookmark("A", "https://a.com")
	bm.AddBookmark("B", "https://b.com")

	all := bm.GetAllBookmarks()
	if len(all) != 2 {
		t.Fatalf("GetAllBookmarks = %d, want 2", len(all))
	}
	if all[0].Title != "A" || all[1].Title != "B" {
		t.Fatalf("unexpected order: %+v", all)
	}
}

func TestBookmarkManagerRemove(t *testing.T) {
	bm := NewBookmarkManager()
	a := bm.AddBookmark("A", "https://a.com")
	b := bm.AddBookmark("B", "https://b.com")

	bm.RemoveBookmark(a.ID)
	if bm.GetBookmark(a.ID) != nil {
		t.Fatal("bookmark A should be removed")
	}
	if bm.GetBookmark(b.ID) == nil {
		t.Fatal("bookmark B should still exist")
	}
	if got := len(bm.GetAllBookmarks()); got != 1 {
		t.Fatalf("GetAllBookmarks len = %d, want 1", got)
	}
	bm.RemoveBookmark(999) // unknown ID is a no-op
}

func TestBookmarkManagerFolders(t *testing.T) {
	bm := NewBookmarkManager()
	f := bm.AddFolder("Work")
	if f.ID != 1 {
		t.Fatalf("folder ID = %d, want 1", f.ID)
	}
	if f.Title != "Work" {
		t.Fatalf("folder title = %q, want Work", f.Title)
	}

	got := bm.GetFolder(f.ID)
	if got == nil || got.Title != "Work" {
		t.Fatalf("GetFolder(%d) = %v, want Work folder", f.ID, got)
	}

	all := bm.GetAllFolders()
	if len(all) != 1 {
		t.Fatalf("GetAllFolders len = %d, want 1", len(all))
	}
}

func TestBookmarkManagerBookmarksByFolder(t *testing.T) {
	bm := NewBookmarkManager()
	f := bm.AddFolder("Personal")
	b := bm.AddBookmark("Personal Site", "https://personal.com")
	b.FolderID = f.ID

	// Re-add to folder properly by directly setting folder ID and adding to items
	f.Items = append(f.Items, b.ID)

	byFolder := bm.GetBookmarksByFolder(f.ID)
	if len(byFolder) != 1 || byFolder[0].ID != b.ID {
		t.Fatalf("GetBookmarksByFolder = %+v, want [b]", byFolder)
	}
}

func TestBookmarkManagerSaveAndLoad(t *testing.T) {
	bm := NewBookmarkManager()
	bm.AddBookmark("Example", "https://example.com")
	bm.AddFolder("Work")

	tmp := filepath.Join(t.TempDir(), "bookmarks.json")
	if err := bm.SaveBookmarks(tmp); err != nil {
		t.Fatalf("SaveBookmarks: %v", err)
	}

	bm2 := NewBookmarkManager()
	if err := bm2.LoadBookmarks(tmp); err != nil {
		t.Fatalf("LoadBookmarks: %v", err)
	}

	all := bm2.GetAllBookmarks()
	if len(all) != 1 || all[0].Title != "Example" {
		t.Fatalf("loaded bookmarks = %+v, want [Example]", all)
	}
	folders := bm2.GetAllFolders()
	if len(folders) != 2 {
		t.Fatalf("loaded folders = %+v, want 2 (Bookmarks + Work)", folders)
	}
	var hasWork bool
	for _, f := range folders {
		if f.Title == "Work" {
			hasWork = true
		}
	}
	if !hasWork {
		t.Fatalf("loaded folders = %+v, want one titled Work", folders)
	}
}
