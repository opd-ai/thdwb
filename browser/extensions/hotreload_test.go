package extensions

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHotReloaderWatchAndUnwatch(t *testing.T) {
	dir := t.TempDir()
	hr := NewHotReloader(func(extID string) error { return nil }, 10*time.Millisecond)

	require.False(t, hr.IsWatching("ext1"))
	require.NoError(t, hr.Watch("ext1", dir))
	require.True(t, hr.IsWatching("ext1"))
	require.Equal(t, 1, hr.Count())

	hr.Unwatch("ext1")
	require.False(t, hr.IsWatching("ext1"))
	require.Equal(t, 0, hr.Count())
}

func TestHotReloaderDetectsChange(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	require.NoError(t, os.WriteFile(manifestPath, []byte(`{"name":"test","version":"1.0"}`), 0644))

	var mu sync.Mutex
	reloaded := make(map[string]bool)
	hr := NewHotReloader(func(extID string) error {
		mu.Lock()
		reloaded[extID] = true
		mu.Unlock()
		return nil
	}, 10*time.Millisecond)

	require.NoError(t, hr.Watch("ext1", dir))

	// Wait for initial scan.
	time.Sleep(50 * time.Millisecond)

	// Modify a file.
	require.NoError(t, os.WriteFile(manifestPath, []byte(`{"name":"test","version":"1.1"}`), 0644))

	// Poll for change detection.
	reloadedCount := 0
	for i := 0; i < 100; i++ {
		time.Sleep(20 * time.Millisecond)
		ids, err := hr.Check()
		require.NoError(t, err)
		if len(ids) > 0 {
			reloadedCount = len(ids)
			break
		}
	}

	mu.Lock()
	require.True(t, reloaded["ext1"], "extension should have been reloaded")
	mu.Unlock()
	require.Greater(t, reloadedCount, 0)
}

func TestHotReloaderNoChangeNoReload(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	require.NoError(t, os.WriteFile(manifestPath, []byte(`{"name":"test","version":"1.0"}`), 0644))

	var mu sync.Mutex
	reloaded := make(map[string]bool)
	hr := NewHotReloader(func(extID string) error {
		mu.Lock()
		reloaded[extID] = true
		mu.Unlock()
		return nil
	}, 10*time.Millisecond)

	require.NoError(t, hr.Watch("ext1", dir))

	// Wait for initial scan.
	time.Sleep(50 * time.Millisecond)

	// No changes; Check should not trigger reload.
	ids, err := hr.Check()
	require.NoError(t, err)
	require.Empty(t, ids)

	mu.Lock()
	require.False(t, reloaded["ext1"])
	mu.Unlock()
}

func TestHotReloaderDoubleWatch(t *testing.T) {
	dir := t.TempDir()
	hr := NewHotReloader(func(extID string) error { return nil }, 10*time.Millisecond)

	require.NoError(t, hr.Watch("ext1", dir))
	require.Error(t, hr.Watch("ext1", dir)) // already watching
}

func TestManagerWatchExtension(t *testing.T) {
	m := NewManager("")

	manifest := &Manifest{
		ManifestVersion: ManifestV3,
		Name:            "Watch Ext",
		Version:         "1.0",
	}
	ext := NewExtension("watch-ext", t.TempDir(), manifest)
	m.mu.Lock()
	m.extensions["watch-ext"] = ext
	m.mu.Unlock()

	// Watching a non-existent path should error.
	ext.Path = "/nonexistent/path"
	require.Error(t, m.WatchExtension("watch-ext"))

	// Watching a valid path should succeed.
	ext.Path = t.TempDir()
	require.NoError(t, m.WatchExtension("watch-ext"))
	require.True(t, m.HotReloader().IsWatching("watch-ext"))

	m.UnwatchExtension("watch-ext")
	require.False(t, m.HotReloader().IsWatching("watch-ext"))
}
