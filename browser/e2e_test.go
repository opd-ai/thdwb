package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/danfragoso/thdwb/browser/shell"
)

// TestE2ETabLifecycle exercises the full tab lifecycle: add, navigate,
// back/forward, close, restore, and activate.
func TestE2ETabLifecycle(t *testing.T) {
	bs := shell.NewBrowserShell(nil, nil)
	require.NoError(t, bs.Init())
	defer bs.Close()

	tm := bs.TabManager()

	// Add two tabs (Init already added one default tab).
	a := tm.Add("Tab A", "https://example.com/a", "", "", "")
	b := tm.Add("Tab B", "https://example.com/b", "", "", "")
	require.Equal(t, 3, tm.Count())
	require.True(t, b.IsActive)
	require.False(t, a.IsActive)

	// Navigate tab A.
	tm.Navigate(a.ID, "https://example.com/a/page2")
	require.True(t, tm.CanGoBack(a.ID))
	require.False(t, tm.CanGoForward(a.ID))
	require.Equal(t, "https://example.com/a/page2", tm.CurrentURL(a.ID))

	// Back and forward.
	require.True(t, tm.Back(a.ID))
	require.Equal(t, "https://example.com/a", tm.CurrentURL(a.ID))
	require.True(t, tm.CanGoForward(a.ID))
	require.True(t, tm.Forward(a.ID))
	require.Equal(t, "https://example.com/a/page2", tm.CurrentURL(a.ID))

	// Close tab A; tab B should become active.
	closed := tm.Close(a.ID)
	require.NotNil(t, closed)
	require.Equal(t, a.ID, closed.ID)
	require.Equal(t, 2, tm.Count())
	require.True(t, b.IsActive)

	// Restore closed tab A.
	restored := tm.RestoreClosed()
	require.NotNil(t, restored)
	require.Equal(t, a.ID, restored.ID)
	require.True(t, restored.IsActive)
	require.Equal(t, 3, tm.Count())
}

// TestE2EBookmarkAndHistory exercises bookmark add/remove and history
// recording for web navigations only.
func TestE2EBookmarkAndHistory(t *testing.T) {
	bs := shell.NewBrowserShell(nil, nil)
	require.NoError(t, bs.Init())
	defer bs.Close()

	bm := bs.BookmarkManager()
	hm := bs.HistoryManager()

	// Add bookmarks.
	b1 := bm.AddBookmark("Example", "https://example.com")
	bm.AddBookmark("Example 2", "https://example.org")
	require.Equal(t, 2, len(bm.GetAllBookmarks()))

	// Remove one.
	bm.RemoveBookmark(b1.ID)
	require.Nil(t, bm.GetBookmark(b1.ID))
	require.Equal(t, 1, len(bm.GetAllBookmarks()))

	// Record history for web pages only via the exported history manager.
	hm.AddEntry("Example", "https://example.com")
	hm.AddEntry("Org", "http://example.org")
	// Internal (thdwb://) and empty URLs are not recorded by the shell's
	// recordHistory, so we only add http/https here to match that behavior.

	require.Equal(t, 2, hm.EntryCount())
	require.Equal(t, "https://example.com", hm.GetEntry(1).URL)
	require.Equal(t, "http://example.org", hm.GetEntry(2).URL)
}

// TestE2EExtensionManager exercises extension enable/disable/toggle.
func TestE2EExtensionManager(t *testing.T) {
	bs := shell.NewBrowserShell(nil, nil)
	require.NoError(t, bs.Init())
	defer bs.Close()

	em := bs.ExtensionManager()

	em.AddExtension(&shell.Extension{ID: "ext1", Name: "Ext One", Version: "1.0", Enabled: true})
	em.AddExtension(&shell.Extension{ID: "ext2", Name: "Ext Two", Version: "2.0", Enabled: false})

	require.Equal(t, 2, em.ExtensionCount())
	require.Equal(t, 1, len(em.GetEnabledExtensions()))

	em.DisableExtension("ext1")
	require.Equal(t, 0, len(em.GetEnabledExtensions()))

	em.EnableExtension("ext2")
	require.Equal(t, 1, len(em.GetEnabledExtensions()))

	em.ToggleExtension("ext1")
	require.True(t, em.GetExtension("ext1").Enabled)

	em.RemoveExtension("ext2")
	require.Nil(t, em.GetExtension("ext2"))
	require.Equal(t, 1, em.ExtensionCount())
}

// TestE2ESettingsSaveLoad exercises settings persistence round-trip.
func TestE2ESettingsSaveLoad(t *testing.T) {
	settings := shell.NewBrowserSettings()
	require.NotNil(t, settings)
	require.Equal(t, "thdwb://newtab/", settings.Homepage)

	// Modify and save to temp file.
	settings.Homepage = "https://example.com/"
	tmp := t.TempDir() + "/settings.json"
	require.NoError(t, settings.SaveSettings(tmp))

	loaded := shell.NewBrowserSettings()
	require.NoError(t, loaded.LoadSettings(tmp))
	require.Equal(t, "https://example.com/", loaded.Homepage)
}

// TestE2ENavigationBackForward exercises tab back/forward history.
func TestE2ENavigationBackForward(t *testing.T) {
	bs := shell.NewBrowserShell(nil, nil)
	require.NoError(t, bs.Init())
	defer bs.Close()

	tm := bs.TabManager()
	tab := tm.Add("A", "https://a.example", "", "", "")

	tm.Navigate(tab.ID, "https://b.example")
	tm.Navigate(tab.ID, "https://c.example")

	require.Equal(t, "https://c.example", tm.CurrentURL(tab.ID))
	require.True(t, tm.CanGoBack(tab.ID))
	require.False(t, tm.CanGoForward(tab.ID))

	require.True(t, tm.Back(tab.ID))
	require.Equal(t, "https://b.example", tm.CurrentURL(tab.ID))
	require.True(t, tm.CanGoForward(tab.ID))

	require.True(t, tm.Forward(tab.ID))
	require.Equal(t, "https://c.example", tm.CurrentURL(tab.ID))
	require.False(t, tm.CanGoForward(tab.ID))
}

// TestE2ETabContainerAssociation verifies container tabs are tracked correctly.
func TestE2ETabContainerAssociation(t *testing.T) {
	bs := shell.NewBrowserShell(nil, nil)
	require.NoError(t, bs.Init())
	defer bs.Close()

	tm := bs.TabManager()

	// Default tab has no container.
	def := tm.Add("Default", "https://example.com", "", "", "")
	require.False(t, def.IsContainerTab())
	require.Equal(t, "", def.ContainerIndicator())

	// Container tab.
	ctr := tm.Add("Work", "https://work.example", "c1", "Work", "#ff0000")
	require.True(t, ctr.IsContainerTab())
	require.Equal(t, "Work", ctr.ContainerIndicator())
}

// TestE2EShellCloseIsSafe verifies Close is safe to call multiple times.
func TestE2EShellCloseIsSafe(t *testing.T) {
	bs := shell.NewBrowserShell(nil, nil)
	require.NoError(t, bs.Init())
	bs.Close()
	// Second close should not panic.
	bs.Close()
}

