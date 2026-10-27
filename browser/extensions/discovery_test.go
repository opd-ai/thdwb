package extensions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeExtDir(t *testing.T, root, name, manifest string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))
	return dir
}

func TestDiscoveryScan(t *testing.T) {
	root := t.TempDir()

	writeExtDir(t, root, "ext-a", `{
		"manifest_version": 3,
		"name": "Extension A",
		"version": "1.0"
	}`)
	writeExtDir(t, root, "ext-b", `{
		"manifest_version": 3,
		"name": "Extension B",
		"version": "2.0"
	}`)
	// Non-extension directory (no manifest.json).
	require.NoError(t, os.MkdirAll(filepath.Join(root, "not-ext"), 0o755))

	d := NewDiscovery()
	d.AddWatchDir(root)

	found, err := d.Scan()
	require.NoError(t, err)
	require.Len(t, found, 2)

	ids := []string{found[0].ID, found[1].ID}
	require.Contains(t, ids, "ext-a")
	require.Contains(t, ids, "ext-b")
}

func TestDiscoveryScanInvalidManifestSkipped(t *testing.T) {
	root := t.TempDir()

	writeExtDir(t, root, "valid", `{
		"manifest_version": 3,
		"name": "Valid",
		"version": "1.0"
	}`)
	writeExtDir(t, root, "invalid", `{
		"manifest_version": 99,
		"name": "Invalid",
		"version": "1.0"
	}`)

	d := NewDiscovery()
	d.AddWatchDir(root)

	found, err := d.Scan()
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, "valid", found[0].ID)
}

func TestDiscoveryScanNonexistentDir(t *testing.T) {
	d := NewDiscovery()
	d.AddWatchDir("/nonexistent/path/that/does/not/exist")

	found, err := d.Scan()
	require.NoError(t, err)
	require.Empty(t, found)
}

func TestDiscoveryOnFound(t *testing.T) {
	root := t.TempDir()
	writeExtDir(t, root, "ext-c", `{
		"manifest_version": 3,
		"name": "Extension C",
		"version": "1.0"
	}`)

	var foundIDs []string
	d := NewDiscovery()
	d.AddWatchDir(root)
	d.SetOnFound(func(ext *Extension) {
		foundIDs = append(foundIDs, ext.ID)
	})

	found, err := d.Scan()
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, []string{"ext-c"}, foundIDs)
}

func TestDiscoveryLoadedAndGet(t *testing.T) {
	root := t.TempDir()
	writeExtDir(t, root, "ext-d", `{
		"manifest_version": 3,
		"name": "Extension D",
		"version": "1.0"
	}`)

	d := NewDiscovery()
	d.AddWatchDir(root)
	_, err := d.Scan()
	require.NoError(t, err)

	require.Equal(t, 1, d.Count())
	ext := d.Get("ext-d")
	require.NotNil(t, ext)
	require.Equal(t, "Extension D", ext.Name)
	require.Nil(t, d.Get("nonexistent"))
}

func TestDiscoveryScanIdempotent(t *testing.T) {
	root := t.TempDir()
	writeExtDir(t, root, "ext-e", `{
		"manifest_version": 3,
		"name": "Extension E",
		"version": "1.0"
	}`)

	d := NewDiscovery()
	d.AddWatchDir(root)

	_, err := d.Scan()
	require.NoError(t, err)
	_, err = d.Scan() // second scan should not duplicate
	require.NoError(t, err)

	require.Equal(t, 1, d.Count())
}
