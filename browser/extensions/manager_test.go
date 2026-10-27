package extensions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManagerScanAndGet(t *testing.T) {
	root := t.TempDir()
	storageDir := filepath.Join(root, "storage")
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	writeExtDir(t, root, "ext-m1", `{
		"manifest_version": 3,
		"name": "Manager Ext 1",
		"version": "1.0",
		"permissions": ["tabs"]
	}`)
	writeExtDir(t, root, "ext-m2", `{
		"manifest_version": 3,
		"name": "Manager Ext 2",
		"version": "2.0"
	}`)

	m := NewManager(storageDir)
	m.AddWatchDir(root)

	found, err := m.Scan()
	require.NoError(t, err)
	require.Len(t, found, 2)

	require.Equal(t, 2, m.Count())
	require.NotNil(t, m.Get("ext-m1"))
	require.NotNil(t, m.Get("ext-m2"))
	require.Nil(t, m.Get("nonexistent"))

	ext := m.Get("ext-m1")
	require.Equal(t, "Manager Ext 1", ext.Name)
	require.Contains(t, ext.Permissions, "tabs")
	require.NotNil(t, ext.Storage)
}

func TestManagerEnableDisable(t *testing.T) {
	root := t.TempDir()
	storageDir := filepath.Join(root, "storage")
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	writeExtDir(t, root, "ext-enable", `{
		"manifest_version": 3,
		"name": "Enable Test",
		"version": "1.0"
	}`)

	m := NewManager(storageDir)
	m.AddWatchDir(root)
	_, err := m.Scan()
	require.NoError(t, err)

	// Extensions start enabled by default.
	require.True(t, m.IsEnabled("ext-enable"))

	require.NoError(t, m.Disable("ext-enable"))
	require.False(t, m.IsEnabled("ext-enable"))

	require.NoError(t, m.Enable("ext-enable"))
	require.True(t, m.IsEnabled("ext-enable"))
}

func TestManagerEnableUnknown(t *testing.T) {
	root := t.TempDir()
	storageDir := filepath.Join(root, "storage")
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	m := NewManager(storageDir)
	err := m.Enable("unknown")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestManagerGetAllAndEnabled(t *testing.T) {
	root := t.TempDir()
	storageDir := filepath.Join(root, "storage")
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	writeExtDir(t, root, "ext-a1", `{
		"manifest_version": 3,
		"name": "A",
		"version": "1.0"
	}`)
	writeExtDir(t, root, "ext-b1", `{
		"manifest_version": 3,
		"name": "B",
		"version": "1.0"
	}`)
	writeExtDir(t, root, "ext-c1", `{
		"manifest_version": 3,
		"name": "C",
		"version": "1.0"
	}`)

	m := NewManager(storageDir)
	m.AddWatchDir(root)
	_, err := m.Scan()
	require.NoError(t, err)

	// Extensions start enabled by default.
	require.NoError(t, m.Disable("ext-b1"))

	all := m.GetAll()
	require.Len(t, all, 3)

	enabled := m.GetEnabled()
	require.Len(t, enabled, 2)
	enabledIDs := []string{enabled[0].ID, enabled[1].ID}
	require.Contains(t, enabledIDs, "ext-a1")
	require.Contains(t, enabledIDs, "ext-c1")
}

func TestManagerUninstall(t *testing.T) {
	root := t.TempDir()
	storageDir := filepath.Join(root, "storage")
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	writeExtDir(t, root, "ext-uninstall", `{
		"manifest_version": 3,
		"name": "Uninstall Test",
		"version": "1.0"
	}`)

	m := NewManager(storageDir)
	m.AddWatchDir(root)
	_, err := m.Scan()
	require.NoError(t, err)

	require.NoError(t, m.Enable("ext-uninstall"))
	require.NoError(t, m.Uninstall("ext-uninstall"))

	require.Equal(t, 0, m.Count())
	require.Nil(t, m.Get("ext-uninstall"))
	require.False(t, m.IsEnabled("ext-uninstall"))
}

func TestManagerUninstallUnknown(t *testing.T) {
	root := t.TempDir()
	storageDir := filepath.Join(root, "storage")
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	m := NewManager(storageDir)
	err := m.Uninstall("unknown")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestManagerStorageFor(t *testing.T) {
	root := t.TempDir()
	storageDir := filepath.Join(root, "storage")
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	writeExtDir(t, root, "ext-storage", `{
		"manifest_version": 3,
		"name": "Storage Test",
		"version": "1.0"
	}`)

	m := NewManager(storageDir)
	m.AddWatchDir(root)
	_, err := m.Scan()
	require.NoError(t, err)

	sp := m.StorageFor("ext-storage")
	require.NotNil(t, sp)
	require.Equal(t, "ext-storage", sp.ExtensionID())

	require.NoError(t, sp.Set(StorageLocal, "k", "v"))
	value, ok := sp.Get(StorageLocal, "k")
	require.True(t, ok)
	require.Equal(t, "v", value)

	require.Nil(t, m.StorageFor("nonexistent"))
}

func TestManagerContentRegistry(t *testing.T) {
	root := t.TempDir()
	storageDir := filepath.Join(root, "storage")
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	writeExtDir(t, root, "ext-content", `{
		"manifest_version": 3,
		"name": "Content Test",
		"version": "1.0",
		"content_scripts": [
			{
				"matches": ["*://example.com/*"],
				"js": ["content.js"],
				"run_at": "document_idle"
			}
		]
	}`)

	m := NewManager(storageDir)
	m.AddWatchDir(root)
	_, err := m.Scan()
	require.NoError(t, err)

	reg := m.ContentRegistry()
	require.NotNil(t, reg)
	require.Equal(t, 1, reg.Count())

	matches := reg.Matching("https://example.com/page")
	require.Len(t, matches, 1)
	require.Equal(t, "document_idle", matches[0].RunAt)

	noMatch := reg.Matching("https://other.com/page")
	require.Empty(t, noMatch)
}
