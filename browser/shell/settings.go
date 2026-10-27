package shell

import (
	"encoding/json"
	"os"
	"sync"
)

// BrowserSettings holds browser configuration.
type BrowserSettings struct {
	mu             sync.Mutex
	Homepage       string                 `json:"homepage"`
	DefaultSearch  string                 `json:"default_search"`
	ShowBookmarks  bool                   `json:"show_bookmarks"`
	ShowHistory    bool                   `json:"show_history"`
	ShowExtensions bool                   `json:"show_extensions"`
	Extensions     []string               `json:"extensions"`
	Containers     []string               `json:"containers"`
	Privacy        map[string]interface{} `json:"privacy"`
	Security       map[string]interface{} `json:"security"`
}

// NewBrowserSettings creates new browser settings with defaults.
func NewBrowserSettings() *BrowserSettings {
	return &BrowserSettings{
		Homepage:       "thdwb://newtab/",
		DefaultSearch:  "https://duckduckgo.com/?q=",
		ShowBookmarks:  true,
		ShowHistory:    true,
		ShowExtensions: true,
		Extensions:     make([]string, 0),
		Containers:     make([]string, 0),
		Privacy:        make(map[string]interface{}),
		Security:       make(map[string]interface{}),
	}
}

// LoadSettings loads settings from a file.
func (bs *BrowserSettings) LoadSettings(path string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, bs)
}

// SaveSettings saves settings to a file.
func (bs *BrowserSettings) SaveSettings(path string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	data, err := json.MarshalIndent(bs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// GetHomepage returns the homepage URL.
func (bs *BrowserSettings) GetHomepage() string {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	return bs.Homepage
}

// SetHomepage sets the homepage URL.
func (bs *BrowserSettings) SetHomepage(url string) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.Homepage = url
}

// GetDefaultSearch returns the default search engine URL.
func (bs *BrowserSettings) GetDefaultSearch() string {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	return bs.DefaultSearch
}

// SetDefaultSearch sets the default search engine URL.
func (bs *BrowserSettings) SetDefaultSearch(url string) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.DefaultSearch = url
}

// IsShowBookmarks returns whether bookmarks bar is visible.
func (bs *BrowserSettings) IsShowBookmarks() bool {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	return bs.ShowBookmarks
}

// SetShowBookmarks sets whether bookmarks bar is visible.
func (bs *BrowserSettings) SetShowBookmarks(show bool) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.ShowBookmarks = show
}

// IsShowHistory returns whether history is visible.
func (bs *BrowserSettings) IsShowHistory() bool {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	return bs.ShowHistory
}

// SetShowHistory sets whether history is visible.
func (bs *BrowserSettings) SetShowHistory(show bool) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.ShowHistory = show
}

// IsShowExtensions returns whether extensions toolbar is visible.
func (bs *BrowserSettings) IsShowExtensions() bool {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	return bs.ShowExtensions
}

// SetShowExtensions sets whether extensions toolbar is visible.
func (bs *BrowserSettings) SetShowExtensions(show bool) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.ShowExtensions = show
}

// GetExtensions returns the list of enabled extensions.
func (bs *BrowserSettings) GetExtensions() []string {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	return append([]string(nil), bs.Extensions...)
}

// SetExtensions sets the list of enabled extensions.
func (bs *BrowserSettings) SetExtensions(extensions []string) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.Extensions = append([]string(nil), extensions...)
}

// GetContainers returns the list of configured containers.
func (bs *BrowserSettings) GetContainers() []string {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	return append([]string(nil), bs.Containers...)
}

// SetContainers sets the list of configured containers.
func (bs *BrowserSettings) SetContainers(containers []string) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.Containers = append([]string(nil), containers...)
}
