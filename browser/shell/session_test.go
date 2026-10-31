package shell

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSessionSaveAndRestore verifies that saving and restoring a session
// preserves tabs, active tab, and window size.
func TestSessionSaveAndRestore(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "session.json")
	sm := NewSessionManager(path)

	bs := NewBrowserShell(nil, nil)
	require.NoError(t, bs.Init())

	// Close the default tab added by Init.
	bs.tabManager.Close(bs.tabManager.ActiveTab().ID)

	// Add some tabs.
	bs.tabManager.Add("Tab A", "https://example.com/a", "", "", "")
	bs.tabManager.Add("Tab B", "https://example.com/b", "c1", "Work", "#ff0000")
	bs.tabManager.Add("Tab C", "https://example.com/c", "c2", "Personal", "#00ff00")

	// Activate Tab B (ID 3).
	bs.tabManager.Activate(3)

	// Set window size.
	bs.settings.WindowWidth = 1200
	bs.settings.WindowHeight = 800

	// Save session.
	require.NoError(t, sm.Save(bs))

	// Create a new shell and restore.
	bs2 := NewBrowserShell(nil, nil)
	require.NoError(t, bs2.Init())
	require.NoError(t, sm.Restore(bs2))

	// Verify tabs were restored.
	tabs := bs2.tabManager.Tabs()
	require.Len(t, tabs, 3)
	require.Equal(t, "Tab A", tabs[0].Title)
	require.Equal(t, "https://example.com/a", tabs[0].URL)
	require.Equal(t, "", tabs[0].ContainerID)

	require.Equal(t, "Tab B", tabs[1].Title)
	require.Equal(t, "https://example.com/b", tabs[1].URL)
	require.Equal(t, "c1", tabs[1].ContainerID)
	require.Equal(t, "Work", tabs[1].ContainerName)
	require.Equal(t, "#ff0000", tabs[1].ContainerColor)

	require.Equal(t, "Tab C", tabs[2].Title)
	require.Equal(t, "https://example.com/c", tabs[2].URL)
	require.Equal(t, "c2", tabs[2].ContainerID)

	// Verify active tab.
	require.Equal(t, 3, bs2.tabManager.ActiveTab().ID)
	require.Equal(t, "Tab B", bs2.tabManager.ActiveTab().Title)

	// Verify window size.
	require.Equal(t, 1200, bs2.settings.WindowWidth)
	require.Equal(t, 800, bs2.settings.WindowHeight)
}

// TestSessionRestoreNoFile verifies that restoring when no session file
// exists is a no-op and does not error.
func TestSessionRestoreNoFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nonexistent.json")
	sm := NewSessionManager(path)

	bs := NewBrowserShell(nil, nil)
	require.NoError(t, bs.Init())

	// Should not error when no session file exists.
	require.NoError(t, sm.Restore(bs))
}

// TestSessionRestoreInvalidJSON verifies that restoring from an invalid JSON
// file returns an error.
func TestSessionRestoreInvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "session.json")
	sm := NewSessionManager(path)

	// Write invalid JSON.
	require.NoError(t, os.WriteFile(path, []byte("not valid json"), 0o644))

	bs := NewBrowserShell(nil, nil)
	require.NoError(t, bs.Init())
	require.Error(t, sm.Restore(bs))
}

// TestSessionSaveNoPath verifies that saving with no path is a no-op.
func TestSessionSaveNoPath(t *testing.T) {
	sm := NewSessionManager("")
	bs := NewBrowserShell(nil, nil)
	require.NoError(t, bs.Init())

	// Should not error with no path configured.
	require.NoError(t, sm.Save(bs))
}

// TestSessionRestoreEmptyTabs verifies restoring a session with no tabs.
func TestSessionRestoreEmptyTabs(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "session.json")
	sm := NewSessionManager(path)

	bs := NewBrowserShell(nil, nil)
	require.NoError(t, bs.Init())

	// Close the default tab added by Init.
	bs.tabManager.Close(bs.tabManager.ActiveTab().ID)

	// Save an empty session.
	require.NoError(t, sm.Save(bs))

	// Restore into a new shell.
	bs2 := NewBrowserShell(nil, nil)
	require.NoError(t, bs2.Init())

	// Close the default tab added by Init before restoring.
	bs2.tabManager.Close(bs2.tabManager.ActiveTab().ID)

	require.NoError(t, sm.Restore(bs2))

	require.Equal(t, 0, bs2.tabManager.Count())
}

// TestSessionRestoreUnsupportedVersion verifies that an unsupported session
// version returns an error.
func TestSessionRestoreUnsupportedVersion(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "session.json")
	sm := NewSessionManager(path)

	// Write a session with an unsupported version.
	session := &SessionData{Version: 99}
	data, _ := json.Marshal(session)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	bs := NewBrowserShell(nil, nil)
	require.NoError(t, bs.Init())
	require.Error(t, sm.Restore(bs))
}
