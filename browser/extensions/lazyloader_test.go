package extensions

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLazyLoaderInitialState(t *testing.T) {
	ll := NewLazyLoader(func(extID string) error { return nil })
	require.Equal(t, 0, ll.Count())
	require.Equal(t, 0, ll.LoadedCount())
}

func TestLazyLoaderRegisterAndLoad(t *testing.T) {
	var mu sync.Mutex
	loaded := make(map[string]bool)
	ll := NewLazyLoader(func(extID string) error {
		mu.Lock()
		loaded[extID] = true
		mu.Unlock()
		return nil
	})

	ll.Register("ext1")
	require.False(t, ll.IsLoaded("ext1"))

	// Load if needed should trigger the loader.
	require.NoError(t, ll.LoadIfNeeded("ext1"))
	require.True(t, ll.IsLoaded("ext1"))

	mu.Lock()
	require.True(t, loaded["ext1"])
	mu.Unlock()

	// Calling again should not re-trigger the loader.
	require.NoError(t, ll.LoadIfNeeded("ext1"))
	mu.Lock()
	require.Len(t, loaded, 1) // still only loaded once
	mu.Unlock()
}

func TestLazyLoaderLoadError(t *testing.T) {
	loadErr := errors.New("load failed")
	ll := NewLazyLoader(func(extID string) error { return loadErr })

	ll.Register("ext1")
	require.Error(t, ll.LoadIfNeeded("ext1"))
	require.False(t, ll.IsLoaded("ext1"))

	// Should be retryable after failure.
	require.Error(t, ll.LoadIfNeeded("ext1"))
}

func TestLazyLoaderUnload(t *testing.T) {
	ll := NewLazyLoader(func(extID string) error { return nil })
	ll.Register("ext1")

	require.NoError(t, ll.LoadIfNeeded("ext1"))
	require.True(t, ll.IsLoaded("ext1"))

	ll.Unload("ext1")
	require.False(t, ll.IsLoaded("ext1"))

	// Can reload after unload.
	require.NoError(t, ll.LoadIfNeeded("ext1"))
	require.True(t, ll.IsLoaded("ext1"))
}

func TestLazyLoaderUnknownExtension(t *testing.T) {
	ll := NewLazyLoader(func(extID string) error { return nil })

	// Unknown extension should not error.
	require.NoError(t, ll.LoadIfNeeded("unknown"))
}

func TestManagerEnableLazy(t *testing.T) {
	m := NewManager("")

	// Create an extension directly.
	manifest := &Manifest{
		ManifestVersion: ManifestV3,
		Name:            "Lazy Ext",
		Version:         "1.0",
	}
	ext := NewExtension("lazy-ext", "/tmp/lazy-ext", manifest)

	// Register it directly in the manager's map.
	m.mu.Lock()
	m.extensions["lazy-ext"] = ext
	m.mu.Unlock()

	require.NoError(t, m.EnableLazy("lazy-ext"))
	require.True(t, m.IsEnabled("lazy-ext"))
	require.False(t, m.LazyLoader().IsLoaded("lazy-ext"))

	// Load on demand should start the background script.
	require.NoError(t, m.LazyLoader().LoadIfNeeded("lazy-ext"))
	require.True(t, m.LazyLoader().IsLoaded("lazy-ext"))
}
