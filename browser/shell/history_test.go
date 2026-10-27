package shell

import (
	"path/filepath"
	"testing"
)

func TestHistoryManagerAddAndGet(t *testing.T) {
	hm := NewHistoryManager()

	e := hm.AddEntry("Example", "https://example.com")
	if e.ID != 1 {
		t.Fatalf("ID = %d, want 1", e.ID)
	}
	if e.Title != "Example" || e.URL != "https://example.com" {
		t.Fatalf("entry = %+v, want title Example url https://example.com", e)
	}
	if e.VisitedAt.IsZero() {
		t.Fatal("VisitedAt should not be zero")
	}

	got := hm.GetEntry(1)
	if got == nil || got.ID != 1 {
		t.Fatalf("GetEntry(1) = %v, want entry with ID 1", got)
	}
	if hm.GetEntry(999) != nil {
		t.Fatal("GetEntry(999) should return nil")
	}
}

func TestHistoryManagerGetAll(t *testing.T) {
	hm := NewHistoryManager()
	hm.AddEntry("A", "https://a.com")
	hm.AddEntry("B", "https://b.com")

	all := hm.GetAllEntries()
	if len(all) != 2 {
		t.Fatalf("GetAllEntries = %d, want 2", len(all))
	}
	if all[0].Title != "A" || all[1].Title != "B" {
		t.Fatalf("unexpected order: %+v", all)
	}
}

func TestHistoryManagerEntriesByURL(t *testing.T) {
	hm := NewHistoryManager()
	hm.AddEntry("Visit 1", "https://example.com")
	hm.AddEntry("Visit 2", "https://example.com")
	hm.AddEntry("Other", "https://other.com")

	byURL := hm.GetEntriesByURL("https://example.com")
	if len(byURL) != 2 {
		t.Fatalf("GetEntriesByURL = %d, want 2", len(byURL))
	}
}

func TestHistoryManagerSearch(t *testing.T) {
	hm := NewHistoryManager()
	hm.AddEntry("Example Site", "https://example.com")
	hm.AddEntry("Other Site", "https://other.com")

	results := hm.SearchEntries("example")
	if len(results) != 1 || results[0].Title != "Example Site" {
		t.Fatalf("SearchEntries(example) = %+v, want [Example Site]", results)
	}

	// Case-insensitive.
	results = hm.SearchEntries("EXAMPLE")
	if len(results) != 1 {
		t.Fatalf("SearchEntries(EXAMPLE) = %d, want 1", len(results))
	}

	// Search by URL.
	results = hm.SearchEntries("other.com")
	if len(results) != 1 || results[0].Title != "Other Site" {
		t.Fatalf("SearchEntries(other.com) = %+v, want [Other Site]", results)
	}

	// Empty query matches all.
	results = hm.SearchEntries("")
	if len(results) != 2 {
		t.Fatalf("SearchEntries(\"\") = %d, want 2", len(results))
	}
}

func TestHistoryManagerClear(t *testing.T) {
	hm := NewHistoryManager()
	hm.AddEntry("A", "https://a.com")
	hm.AddEntry("B", "https://b.com")

	hm.Clear()
	if got := hm.EntryCount(); got != 0 {
		t.Fatalf("EntryCount after clear = %d, want 0", got)
	}
	if len(hm.GetAllEntries()) != 0 {
		t.Fatal("GetAllEntries should be empty after clear")
	}
}

func TestHistoryManagerSaveAndLoad(t *testing.T) {
	hm := NewHistoryManager()
	hm.AddEntry("Example", "https://example.com")

	tmp := filepath.Join(t.TempDir(), "history.json")
	if err := hm.SaveHistory(tmp); err != nil {
		t.Fatalf("SaveHistory: %v", err)
	}

	hm2 := NewHistoryManager()
	if err := hm2.LoadHistory(tmp); err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}

	all := hm2.GetAllEntries()
	if len(all) != 1 || all[0].Title != "Example" {
		t.Fatalf("loaded entries = %+v, want [Example]", all)
	}
}
