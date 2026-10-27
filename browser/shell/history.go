package shell

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// HistoryEntry represents a single page visit in the history.
type HistoryEntry struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	VisitedAt time.Time `json:"visited_at"`
}

// HistoryManager manages browsing history.
type HistoryManager struct {
	mu      sync.Mutex
	entries []*HistoryEntry
	nextID  int
}

// NewHistoryManager creates a new history manager.
func NewHistoryManager() *HistoryManager {
	return &HistoryManager{
		entries: make([]*HistoryEntry, 0),
		nextID:  1,
	}
}

// AddEntry adds a new history entry.
func (hm *HistoryManager) AddEntry(title, url string) *HistoryEntry {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	entry := &HistoryEntry{
		ID:        hm.nextID,
		Title:     title,
		URL:       url,
		VisitedAt: time.Now(),
	}
	hm.nextID++
	hm.entries = append(hm.entries, entry)
	return entry
}

// GetEntry returns a history entry by ID.
func (hm *HistoryManager) GetEntry(id int) *HistoryEntry {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	for _, entry := range hm.entries {
		if entry.ID == id {
			return entry
		}
	}
	return nil
}

// GetEntriesByURL returns all history entries for a URL.
func (hm *HistoryManager) GetEntriesByURL(url string) []*HistoryEntry {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	var result []*HistoryEntry
	for _, entry := range hm.entries {
		if entry.URL == url {
			result = append(result, entry)
		}
	}
	return result
}

// GetAllEntries returns all history entries.
func (hm *HistoryManager) GetAllEntries() []*HistoryEntry {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	return append([]*HistoryEntry(nil), hm.entries...)
}

// SearchEntries searches history by title or URL (case-insensitive substring).
func (hm *HistoryManager) SearchEntries(query string) []*HistoryEntry {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	q := foldCase(query)
	var result []*HistoryEntry
	for _, entry := range hm.entries {
		if containsFold(entry.Title, q) || containsFold(entry.URL, q) {
			result = append(result, entry)
		}
	}
	return result
}

// Clear clears all history entries.
func (hm *HistoryManager) Clear() {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.entries = make([]*HistoryEntry, 0)
}

// EntryCount returns the number of history entries.
func (hm *HistoryManager) EntryCount() int {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	return len(hm.entries)
}

// SaveHistory saves history to a JSON file.
func (hm *HistoryManager) SaveHistory(path string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	type serializable struct {
		Entries []*HistoryEntry `json:"entries"`
		NextID  int             `json:"next_id"`
	}
	data, err := json.MarshalIndent(serializable{
		Entries: hm.entries,
		NextID:  hm.nextID,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadHistory loads history from a JSON file.
func (hm *HistoryManager) LoadHistory(path string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	type serializable struct {
		Entries []*HistoryEntry `json:"entries"`
		NextID  int             `json:"next_id"`
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var s serializable
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	hm.entries = s.Entries
	hm.nextID = s.NextID
	return nil
}

// foldCase lowercases ASCII for case-insensitive comparison.
func foldCase(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b = append(b, c)
	}
	return string(b)
}

// containsFold reports whether substr is contained in s, case-insensitively.
func containsFold(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	sf := foldCase(s)
	for i := 0; i <= len(sf)-len(substr); i++ {
		if sf[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}