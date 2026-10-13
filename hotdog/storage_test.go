package hotdog

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	configDir, err := os.MkdirTemp("", "thdwb-test-config-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.Setenv("XDG_CONFIG_HOME", configDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.RemoveAll(configDir)
		os.Exit(1)
	}
	code := m.Run()
	if err := os.RemoveAll(configDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
	}
	os.Exit(code)
}

func TestOriginStorageManagerPersistsValuesPerOrigin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "thdwb", "local-storage.json")
	manager, err := NewOriginStorageManager(path)
	require.NoError(t, err)
	firstOrigin, err := url.Parse("https://example.com")
	require.NoError(t, err)
	secondOrigin, err := url.Parse("https://other.example")
	require.NoError(t, err)

	require.NoError(t, manager.SetItem(firstOrigin, "theme", "dark"))
	require.NoError(t, manager.SetItem(secondOrigin, "theme", "light"))
	require.NoError(t, manager.SetItem(firstOrigin, "language", "en"))

	reloaded, err := NewOriginStorageManager(path)
	require.NoError(t, err)
	require.Equal(t, 2, reloaded.length(firstOrigin))
	value, exists := reloaded.GetItem(firstOrigin, "theme")
	require.True(t, exists)
	require.Equal(t, "dark", value)
	value, exists = reloaded.GetItem(secondOrigin, "theme")
	require.True(t, exists)
	require.Equal(t, "light", value)

	require.NoError(t, reloaded.RemoveItem(firstOrigin, "language"))
	require.NoError(t, reloaded.ClearOrigin(secondOrigin))
	afterClear, err := NewOriginStorageManager(path)
	require.NoError(t, err)
	require.Equal(t, 1, afterClear.length(firstOrigin))
	require.Equal(t, 0, afterClear.length(secondOrigin))
}

func TestOriginStorageManagerWritesPrivateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "local-storage.json")
	manager, err := NewOriginStorageManager(path)
	require.NoError(t, err)
	origin, err := url.Parse("https://example.com")
	require.NoError(t, err)
	require.NoError(t, manager.SetItem(origin, "key", "value"))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestOriginStorageManagerRejectsCorruptPersistentData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "local-storage.json")
	require.NoError(t, os.WriteFile(path, []byte("{invalid"), 0o600))
	_, err := NewOriginStorageManager(path)
	require.ErrorContains(t, err, "decode localStorage")
}

func TestOriginStorageManagerSynchronizesConcurrentUpdates(t *testing.T) {
	manager, err := NewOriginStorageManager("")
	require.NoError(t, err)
	origin, err := url.Parse("https://example.com")
	require.NoError(t, err)
	var workers sync.WaitGroup
	for i := 0; i < 16; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			key := strconv.Itoa(i)
			for range 50 {
				require.NoError(t, manager.SetItem(origin, key, "value"))
				value, exists := manager.GetItem(origin, key)
				require.True(t, exists)
				require.Equal(t, "value", value)
			}
		}(i)
	}
	workers.Wait()
	require.Equal(t, 16, manager.length(origin))
}

func TestOriginStorageManagerInMemoryDoesNotPersist(t *testing.T) {
	manager, err := NewOriginStorageManager("")
	require.NoError(t, err)
	origin, err := url.Parse("https://example.com")
	require.NoError(t, err)
	require.NoError(t, manager.SetItem(origin, "key", "value"))
	value, exists := manager.GetItem(origin, "key")
	require.True(t, exists)
	require.Equal(t, "value", value)
}

func TestOriginStorageManagerRollsBackFailedPersistence(t *testing.T) {
	manager, err := NewOriginStorageManager("")
	require.NoError(t, err)
	origin, err := url.Parse("https://example.com")
	require.NoError(t, err)
	blockedPath := filepath.Join(t.TempDir(), "existing-directory")
	require.NoError(t, os.Mkdir(blockedPath, 0o700))
	manager.path = blockedPath

	require.Error(t, manager.SetItem(origin, "key", "value"))
	_, exists := manager.GetItem(origin, "key")
	require.False(t, exists)
}

func TestOriginKeyNormalizesSchemeAndHostname(t *testing.T) {
	uppercase, err := url.Parse("HTTPS://EXAMPLE.COM")
	require.NoError(t, err)
	lowercase, err := url.Parse("https://example.com:443")
	require.NoError(t, err)
	require.Equal(t, originKey(lowercase), originKey(uppercase))
}
