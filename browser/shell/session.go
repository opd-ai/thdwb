// Package shell implements the thdwb browser shell: the user-facing UI layer
// that sits on top of the hotdog engine. It provides tab management, the
// address bar, navigation controls, bookmarks, history, preferences,
// extension toolbar icons, and the new tab page.
//
// The shell is designed around a separation of concerns:
//
//   - Pure state managers/stores (TabManager, BookmarkStore, HistoryStore,
//     PreferencesStore, NavigationController) hold the browser's logical state
//     and are fully unit-testable without a display or network.
//   - UI builders (BuildTabBar, BuildToolbar, ...) construct mustard widgets,
//     wire their callbacks to the managers, and are testable headlessly by
//     constructing widgets and firing callbacks directly.
//
// All state managers are intended to be driven from the UI loop's goroutine.
package shell

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// SessionData represents a persistable browser session: open tabs, their
// container associations, and the active tab.
type SessionData struct {
	// Version is the session file format version.
	Version int `json:"version"`

	// SavedAt is when the session was saved.
	SavedAt time.Time `json:"saved_at"`

	// Tabs is the list of open tabs in display order.
	Tabs []*Tab `json:"tabs"`

	// ActiveTabID is the ID of the active tab at save time.
	ActiveTabID int `json:"active_tab_id"`

	// WindowWidth and WindowHeight record the last window size.
	WindowWidth  int `json:"window_width"`
	WindowHeight int `json:"window_height"`
}

// SessionManager saves and restores browser sessions to disk.
type SessionManager struct {
	path string
}

// NewSessionManager creates a SessionManager that persists sessions to the
// given file path. If path is empty, the manager operates in-memory only
// (Save/Restore are no-ops).
func NewSessionManager(path string) *SessionManager {
	return &SessionManager{path: path}
}

// Save writes the current browser session (tabs, active tab, window size) to
// disk. It is a no-op if no path was configured.
func (sm *SessionManager) Save(bs *BrowserShell) error {
	if sm.path == "" {
		return nil
	}

	tabs := bs.tabManager.Tabs()
	var activeID int
	if bs.tabManager.ActiveTab() != nil {
		activeID = bs.tabManager.ActiveTab().ID
	}

	width, height := bs.settings.WindowWidth, bs.settings.WindowHeight

	session := &SessionData{
		Version:      1,
		SavedAt:      time.Now(),
		Tabs:         tabs,
		ActiveTabID:  activeID,
		WindowWidth:  width,
		WindowHeight: height,
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	return writeSessionFile(sm.path, data)
}

// Restore reads a saved session from disk and applies it to the browser shell.
// It returns an error if the session file does not exist or is invalid.
// Existing tabs are replaced by the restored tabs.
func (sm *SessionManager) Restore(bs *BrowserShell) error {
	if sm.path == "" {
		return nil
	}

	data, err := os.ReadFile(sm.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil // no session to restore; not an error
	}
	if err != nil {
		return err
	}

	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return err
	}

	if session.Version != 1 {
		return errors.New("unsupported session version")
	}

	// Clear existing tabs.
	for _, t := range bs.tabManager.Tabs() {
		bs.tabManager.Close(t.ID)
	}

	// Restore tabs in order.
	for _, t := range session.Tabs {
		bs.tabManager.Add(t.Title, t.URL, t.ContainerID, t.ContainerName, t.ContainerColor)
	}

	// Restore active tab.
	if session.ActiveTabID != 0 {
		if t := bs.tabManager.GetTab(session.ActiveTabID); t != nil {
			bs.tabManager.Activate(session.ActiveTabID)
		}
	}

	// Restore window size.
	bs.settings.WindowWidth = session.WindowWidth
	bs.settings.WindowHeight = session.WindowHeight

	return nil
}

// writeSessionFile atomically writes session data to path.
func writeSessionFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
