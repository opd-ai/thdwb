package shell

import (
	"testing"

	extensions "github.com/danfragoso/thdwb/browser/extensions"
	"github.com/stretchr/testify/require"
)

func newTestShell() *BrowserShell {
	bs := NewBrowserShell(nil, nil)
	return bs
}

func TestExtensionOpsQueryTabs(t *testing.T) {
	bs := newTestShell()
	eo := newExtensionOps(bs)

	bs.tabManager.Add("Tab A", "https://example.com/a", "", "", "")
	bs.tabManager.Add("Tab B", "https://example.com/b", "c1", "Work", "#ff0000")

	tabs := eo.QueryTabs()
	require.Len(t, tabs, 2)
	require.Equal(t, "https://example.com/a", tabs[0].URL)
	require.Equal(t, "c1", tabs[1].ContainerID)
	require.Equal(t, "Work", tabs[1].ContainerName)
	require.Equal(t, "#ff0000", tabs[1].ContainerColor)
	require.True(t, tabs[1].IsActive)
}

func TestExtensionOpsGetTab(t *testing.T) {
	bs := newTestShell()
	eo := newExtensionOps(bs)

	bs.tabManager.Add("Tab A", "https://example.com/a", "", "", "")

	tab, err := eo.GetTab(1)
	require.NoError(t, err)
	require.NotNil(t, tab)
	require.Equal(t, "https://example.com/a", tab.URL)

	// Unknown tab returns nil, no error.
	tab, err = eo.GetTab(999)
	require.NoError(t, err)
	require.Nil(t, tab)
}

func TestExtensionOpsUpdateTab(t *testing.T) {
	bs := newTestShell()
	eo := newExtensionOps(bs)

	bs.tabManager.Add("Tab A", "https://example.com/a", "", "", "")

	active := true
	tab, err := eo.UpdateTab(1, extensions.TabUpdate{
		URL:    "https://example.com/b",
		Title:  "New Title",
		Active: &active,
	})
	require.NoError(t, err)
	require.NotNil(t, tab)
	require.Equal(t, "https://example.com/b", tab.URL)
	require.Equal(t, "New Title", tab.Title)
	require.True(t, tab.IsActive)

	// Unknown tab returns nil, no error.
	tab, err = eo.UpdateTab(999, extensions.TabUpdate{})
	require.NoError(t, err)
	require.Nil(t, tab)
}

func TestExtensionOpsCreateTab(t *testing.T) {
	bs := newTestShell()
	eo := newExtensionOps(bs)

	tab, err := eo.CreateTab(extensions.TabCreate{
		URL:            "https://example.com/new",
		Title:          "New Tab",
		ContainerID:    "c1",
		ContainerName:  "Work",
		ContainerColor: "#ff0000",
	})
	require.NoError(t, err)
	require.NotNil(t, tab)
	require.Equal(t, "https://example.com/new", tab.URL)
	require.Equal(t, "c1", tab.ContainerID)
	require.Equal(t, "Work", tab.ContainerName)
	require.Equal(t, "#ff0000", tab.ContainerColor)
	require.True(t, tab.IsActive)
}

func TestExtensionOpsRemoveTab(t *testing.T) {
	bs := newTestShell()
	eo := newExtensionOps(bs)

	bs.tabManager.Add("Tab A", "https://example.com/a", "", "", "")

	require.NoError(t, eo.RemoveTab(1))
	require.Nil(t, bs.tabManager.GetTab(1))

	// Removing unknown tab returns error.
	require.Error(t, eo.RemoveTab(999))
}

func TestExtensionOpsActivateTab(t *testing.T) {
	bs := newTestShell()
	eo := newExtensionOps(bs)

	bs.tabManager.Add("Tab A", "https://example.com/a", "", "", "")
	bs.tabManager.Add("Tab B", "https://example.com/b", "", "", "")

	tab, err := eo.ActivateTab(1)
	require.NoError(t, err)
	require.NotNil(t, tab)
	require.True(t, tab.IsActive)
	require.Equal(t, "https://example.com/a", tab.URL)

	// Unknown tab returns nil, no error.
	tab, err = eo.ActivateTab(999)
	require.NoError(t, err)
	require.Nil(t, tab)
}

func TestExtensionOpsWindowOperations(t *testing.T) {
	bs := newTestShell()
	eo := newExtensionOps(bs)

	win, err := eo.GetCurrent()
	require.NoError(t, err)
	require.NotNil(t, win)
	require.Equal(t, 1, win.ID)
	require.True(t, win.IsFocused)

	wins, err := eo.GetAll()
	require.NoError(t, err)
	require.Len(t, wins, 1)

	win, err = eo.Create(extensions.WindowCreate{
		Title:  "New Window",
		URL:    "https://example.com",
		Width:  800,
		Height: 600,
	})
	require.NoError(t, err)
	require.NotNil(t, win)
	require.Equal(t, 2, win.ID)
	require.Equal(t, "New Window", win.Title)

	// Cannot remove the only window.
	require.Error(t, eo.Remove(1))
	// Removing a non-existent window is a no-op.
	require.NoError(t, eo.Remove(999))
}

func TestExtensionOpsCookieOperations(t *testing.T) {
	bs := newTestShell()
	eo := newExtensionOps(bs)

	// Set a cookie.
	err := eo.SetCookie("default", extensions.Cookie{
		Name:     "session",
		Value:    "abc123",
		Domain:   "https://example.com",
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
	})
	require.NoError(t, err)

	// Get the cookie.
	cookies, err := eo.GetCookies("default", "https://example.com/page")
	require.NoError(t, err)
	require.Len(t, cookies, 1)
	require.Equal(t, "session", cookies[0].Name)
	require.Equal(t, "abc123", cookies[0].Value)
	require.True(t, cookies[0].Secure)
	require.True(t, cookies[0].HttpOnly)

	// No cookies for a different origin.
	cookies, err = eo.GetCookies("default", "https://other.com/page")
	require.NoError(t, err)
	require.Empty(t, cookies)

	// Clear cookies.
	require.NoError(t, eo.ClearCookies("default"))
	cookies, err = eo.GetCookies("default", "https://example.com/page")
	require.NoError(t, err)
	require.Empty(t, cookies)
}

func TestExtensionOpsCookieStores(t *testing.T) {
	bs := newTestShell()
	eo := newExtensionOps(bs)

	stores, err := eo.GetAllCookieStores()
	require.NoError(t, err)
	require.NotEmpty(t, stores)

	// Should include the default store.
	foundDefault := false
	for _, s := range stores {
		if s.ID == "default" {
			foundDefault = true
		}
	}
	require.True(t, foundDefault)
}

func TestExtensionOpsScriptingOperations(t *testing.T) {
	bs := newTestShell()
	eo := newExtensionOps(bs)

	// Register a sandbox for the extension so ExecuteScript can find it.
	sandbox := bs.extManager.SandboxFor("ext1")
	bs.extManager.DynamicScripting().RegisterSandbox("ext1", sandbox)

	// Execute a simple script.
	result, err := eo.ExecuteScript("ext1", extensions.ISOLATEDWorld, "1 + 2")
	require.NoError(t, err)
	require.EqualValues(t, 3, result)

	// ExecuteScriptTab delegates to ExecuteScript.
	result, err = eo.ExecuteScriptTab("ext1", 42, extensions.ISOLATEDWorld, "40 + 2")
	require.NoError(t, err)
	require.EqualValues(t, 42, result)

	// Inject and remove CSS.
	require.NoError(t, eo.InjectCSS("ext1", "body { color: red; }"))
	require.NoError(t, eo.RemoveCSS("ext1"))
}

func TestExtensionOpsContextMenuOperations(t *testing.T) {
	bs := newTestShell()
	eo := newExtensionOps(bs)

	id, err := eo.CreateContextMenu(extensions.ContextMenuItem{
		Title:    "Test Item",
		Contexts: []string{"page", "selection"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, id)

	require.NoError(t, eo.UpdateContextMenu(id, extensions.ContextMenuItem{
		Title:    "Updated Item",
		Contexts: []string{"link"},
	}))
	require.NoError(t, eo.RemoveContextMenu(id))
	require.NoError(t, eo.RemoveAllContextMenus())
}

func TestExtensionOpsPermissionOperations(t *testing.T) {
	bs := newTestShell()
	eo := newExtensionOps(bs)

	granted, err := eo.RequestPermissions("ext1", []string{"tabs", "storage"})
	require.NoError(t, err)
	require.Contains(t, granted, "tabs")
	require.Contains(t, granted, "storage")

	contains, err := eo.ContainsPermissions("ext1", []string{"tabs", "unknown"})
	require.NoError(t, err)
	require.Contains(t, contains, "tabs")
	require.NotContains(t, contains, "unknown")

	all, err := eo.GetAllPermissions("ext1")
	require.NoError(t, err)
	require.Contains(t, all, "tabs")
	require.Contains(t, all, "storage")
}

func TestExtensionOpsAlarmOperations(t *testing.T) {
	bs := newTestShell()
	eo := newExtensionOps(bs)

	require.NoError(t, eo.CreateAlarm("ext1", "my-alarm", extensions.Alarm{
		Name:            "my-alarm",
		PeriodInMinutes: 5,
	}))

	alarm, err := eo.GetAlarm("ext1", "my-alarm")
	require.NoError(t, err)
	require.NotNil(t, alarm)
	require.Equal(t, "my-alarm", alarm.Name)
	require.Equal(t, float64(5), alarm.PeriodInMinutes)

	all, err := eo.GetAllAlarms("ext1")
	require.NoError(t, err)
	require.Len(t, all, 1)

	require.NoError(t, eo.ClearAlarm("ext1", "my-alarm"))
	alarm, err = eo.GetAlarm("ext1", "my-alarm")
	require.NoError(t, err)
	require.Nil(t, alarm)

	require.NoError(t, eo.ClearAllAlarms("ext1"))
	all, err = eo.GetAllAlarms("ext1")
	require.NoError(t, err)
	require.Empty(t, all)
}

func TestExtensionOpsI18nOperations(t *testing.T) {
	bs := newTestShell()
	eo := newExtensionOps(bs)

	// Unknown message returns the message name itself.
	msg, err := eo.GetMessage("ext1", "unknown.message")
	require.NoError(t, err)
	require.Equal(t, "unknown.message", msg)

	// GetAcceptLanguages returns defaults.
	langs, err := eo.GetAcceptLanguages("ext1")
	require.NoError(t, err)
	require.NotEmpty(t, langs)
}

func TestExtensionOpsWebRequestOperations(t *testing.T) {
	bs := newTestShell()
	eo := newExtensionOps(bs)

	// Add a listener for onBeforeRequest.
	err := eo.AddListener("ext1", "onBeforeRequest", extensions.WebRequestDetails{
		URLPatterns: []string{"<all_urls>"},
	})
	require.NoError(t, err)

	// Remove the listener.
	err = eo.RemoveListener("ext1", "onBeforeRequest")
	require.NoError(t, err)

	// Removing a non-existent listener should not error.
	err = eo.RemoveListener("ext2", "onCompleted")
	require.NoError(t, err)
}

func TestExtensionOpsNotificationOperations(t *testing.T) {
	bs := newTestShell()
	eo := newExtensionOps(bs)

	// Create a notification.
	id, err := eo.CreateNotification("ext1", "n1", extensions.NotificationOptions{
		Title:   "Test Notification",
		Message: "Hello, world!",
		IconURL: "https://example.com/icon.png",
		Priority: 1,
	})
	require.NoError(t, err)
	require.Equal(t, "n1", id)

	// Get all notifications.
	notifs, err := eo.GetAllNotifications("ext1")
	require.NoError(t, err)
	require.Len(t, notifs, 1)
	require.Equal(t, "Test Notification", notifs[0].Title)
	require.Equal(t, "Hello, world!", notifs[0].Message)

	// Clear the notification.
	cleared, err := eo.ClearNotification("ext1", "n1")
	require.NoError(t, err)
	require.True(t, cleared)

	// Verify it's gone.
	notifs, err = eo.GetAllNotifications("ext1")
	require.NoError(t, err)
	require.Len(t, notifs, 0)

	// Clearing a non-existent notification returns false.
	cleared, err = eo.ClearNotification("ext1", "nonexistent")
	require.NoError(t, err)
	require.False(t, cleared)
}

func TestExtensionOpsDownloadOperations(t *testing.T) {
	bs := newTestShell()
	dlOps := newDownloadOps(bs)

	// Start a download.
	id, err := dlOps.Download("ext1", extensions.DownloadOptions{
		URL:      "https://example.com/file.zip",
		Filename: "file.zip",
	})
	require.NoError(t, err)
	require.Equal(t, 1, id)

	// Get the download.
	item, err := dlOps.Get(id)
	require.NoError(t, err)
	require.NotNil(t, item)
	require.Equal(t, "https://example.com/file.zip", item.URL)
	require.Equal(t, "file.zip", item.Filename)
	require.Equal(t, "in_progress", item.State)

	// Get all downloads.
	all, err := dlOps.GetAll()
	require.NoError(t, err)
	require.Len(t, all, 1)

	// Pause and resume.
	require.NoError(t, dlOps.Pause(id))
	item, err = dlOps.Get(id)
	require.NoError(t, err)
	require.True(t, item.Paused)

	require.NoError(t, dlOps.Resume(id))
	item, err = dlOps.Get(id)
	require.NoError(t, err)
	require.False(t, item.Paused)

	// Cancel.
	require.NoError(t, dlOps.Cancel(id))
	item, err = dlOps.Get(id)
	require.NoError(t, err)
	require.Equal(t, "interrupted", item.State)
	require.Equal(t, "cancelled", item.Error)

	// Erase.
	require.NoError(t, dlOps.Erase(id))
	all, err = dlOps.GetAll()
	require.NoError(t, err)
	require.Len(t, all, 0)
}
