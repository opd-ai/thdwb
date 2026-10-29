package shell

import (
	"os"
	"path/filepath"
	"testing"

	extensions "github.com/danfragoso/thdwb/browser/extensions"
	"github.com/stretchr/testify/require"
)

func TestExtensionSettingsPageRebuild(t *testing.T) {
	settings := NewBrowserSettings()
	extManager := NewExtensionManager()
	extMgr := extensions.NewManager("")

	// Add some extensions.
	require.NoError(t, extMgr.AddExtension(&extensions.Extension{
		ID:          "ext1",
		Name:        "Extension One",
		Version:     "1.0",
		Description: "First extension",
		Enabled:     true,
	}))
	require.NoError(t, extMgr.AddExtension(&extensions.Extension{
		ID:          "ext2",
		Name:        "Extension Two",
		Version:     "2.0",
		Enabled:     false,
	}))

	page := NewExtensionSettingsPage(nil, extManager, extMgr, settings)
	require.NotNil(t, page.Frame)
	require.Len(t, page.rows, 2)
	require.Contains(t, page.rows, "ext1")
	require.Contains(t, page.rows, "ext2")
}

func TestExtensionSettingsPageNoExtensions(t *testing.T) {
	settings := NewBrowserSettings()
	extManager := NewExtensionManager()
	extMgr := extensions.NewManager("")

	page := NewExtensionSettingsPage(nil, extManager, extMgr, settings)
	require.NotNil(t, page.Frame)
	require.Empty(t, page.rows)
}

func TestExtensionSettingsPageEnableDisable(t *testing.T) {
	settings := NewBrowserSettings()
	extManager := NewExtensionManager()
	extMgr := extensions.NewManager("")

	require.NoError(t, extMgr.AddExtension(&extensions.Extension{
		ID:     "ext1",
		Name:   "Extension One",
		Version: "1.0",
		Enabled: true,
	}))

	_ = NewExtensionSettingsPage(nil, extManager, extMgr, settings)
	require.True(t, extMgr.Get("ext1").IsEnabled())

	// Toggle enabled state directly on the extension (Manager.Enable triggers
	// lifecycle hooks that require a full shell setup).
	ext := extMgr.Get("ext1")
	ext.SetEnabled(false)
	require.False(t, ext.IsEnabled())

	ext.SetEnabled(true)
	require.True(t, ext.IsEnabled())
}

func TestExtensionSettingsPageUninstall(t *testing.T) {
	settings := NewBrowserSettings()
	extManager := NewExtensionManager()
	extMgr := extensions.NewManager("")

	require.NoError(t, extMgr.AddExtension(&extensions.Extension{
		ID:     "ext1",
		Name:   "Extension One",
		Version: "1.0",
		Enabled: true,
	}))

	page := NewExtensionSettingsPage(nil, extManager, extMgr, settings)
	require.Contains(t, page.rows, "ext1")

	// Uninstall via manager directly (button click requires a window).
	require.NoError(t, extMgr.Uninstall("ext1"))
	page.Rebuild()

	require.NotContains(t, page.rows, "ext1")
	require.Nil(t, extMgr.Get("ext1"))
}

func TestGetExtensionInfo(t *testing.T) {
	root := t.TempDir()
	extDir := filepath.Join(root, "extensions")
	storageDir := filepath.Join(root, "storage")
	require.NoError(t, os.MkdirAll(extDir, 0o755))
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	// Write an extension directory and scan it into the manager.
	writeExtDirShell(t, extDir, "ext1", `{
		"manifest_version": 3,
		"name": "Test Ext",
		"version": "1.0",
		"description": "A test extension",
		"permissions": ["tabs", "storage"]
	}`)

	extMgr := extensions.NewManager(storageDir)
	extMgr.AddWatchDir(extDir)
	_, err := extMgr.Scan()
	require.NoError(t, err)

	info, err := GetExtensionInfo(extMgr, "ext1")
	require.NoError(t, err)
	require.Equal(t, "ext1", info.ID)
	require.Equal(t, "Test Ext", info.Name)
	require.Equal(t, "1.0", info.Version)
	require.Equal(t, "A test extension", info.Description)
	require.True(t, info.Enabled)
	require.Contains(t, info.Permissions, "tabs")
	require.Contains(t, info.Permissions, "storage")
	require.Equal(t, filepath.Join(extDir, "ext1"), info.InstallPath)
}

func writeExtDirShell(t *testing.T, root, name, manifest string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))
	return dir
}

func TestGetExtensionInfoUnknown(t *testing.T) {
	extMgr := extensions.NewManager("")

	_, err := GetExtensionInfo(extMgr, "nonexistent")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestExtensionSettingsPageSetUninstallCallback(t *testing.T) {
	settings := NewBrowserSettings()
	extManager := NewExtensionManager()
	extMgr := extensions.NewManager("")

	require.NoError(t, extMgr.AddExtension(&extensions.Extension{
		ID:     "ext1",
		Name:   "Extension One",
		Version: "1.0",
		Enabled: true,
	}))

	var uninstalledID string
	page := NewExtensionSettingsPage(nil, extManager, extMgr, settings)
	page.SetUninstallCallback(func(extID string) {
		uninstalledID = extID
	})

	// Verify callback is registered (button click requires a window).
	require.NotNil(t, page.onUninstall)
	page.onUninstall("ext1")
	require.Equal(t, "ext1", uninstalledID)
}
