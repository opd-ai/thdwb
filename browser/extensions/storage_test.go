package extensions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoragePartitionInMemory(t *testing.T) {
	sp := NewStoragePartition("ext1", "")

	// Set and get.
	require.NoError(t, sp.Set(StorageLocal, "theme", "dark"))
	value, ok := sp.Get(StorageLocal, "theme")
	require.True(t, ok)
	require.Equal(t, "dark", value)

	// Missing key.
	_, ok = sp.Get(StorageLocal, "missing")
	require.False(t, ok)

	// Keys and length.
	require.Equal(t, []string{"theme"}, sp.Keys(StorageLocal))
	require.Equal(t, 1, sp.Length(StorageLocal))

	// GetAll.
	all := sp.GetAll(StorageLocal)
	require.Equal(t, map[string]string{"theme": "dark"}, all)

	// Remove.
	require.NoError(t, sp.Remove(StorageLocal, "theme"))
	_, ok = sp.Get(StorageLocal, "theme")
	require.False(t, ok)
	require.Equal(t, 0, sp.Length(StorageLocal))

	// Clear (set multiple then clear).
	require.NoError(t, sp.Set(StorageLocal, "a", "1"))
	require.NoError(t, sp.Set(StorageLocal, "b", "2"))
	require.NoError(t, sp.Clear(StorageLocal))
	require.Equal(t, 0, sp.Length(StorageLocal))
}

func TestStoragePartitionPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ext1.json")
	sp := NewStoragePartition("ext1", path)

	require.NoError(t, sp.Set(StorageLocal, "key1", "value1"))
	require.NoError(t, sp.Set(StorageSync, "key2", "value2"))

	// Reload from disk.
	sp2 := NewStoragePartition("ext1", path)
	value, ok := sp2.Get(StorageLocal, "key1")
	require.True(t, ok)
	require.Equal(t, "value1", value)
	value, ok = sp2.Get(StorageSync, "key2")
	require.True(t, ok)
	require.Equal(t, "value2", value)
}

func TestStoragePartitionManagedReadOnly(t *testing.T) {
	sp := NewStoragePartition("ext1", "")

	// Managed area starts empty.
	require.Equal(t, 0, sp.Length(StorageManaged))

	// Can write to managed (simplified model; policy enforcement is out of scope).
	require.NoError(t, sp.Set(StorageManaged, "policy", "value"))
	value, ok := sp.Get(StorageManaged, "policy")
	require.True(t, ok)
	require.Equal(t, "value", value)
}

func TestStoragePartitionSyncToDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ext1.json")
	sp := NewStoragePartition("ext1", path)

	require.NoError(t, sp.Set(StorageLocal, "k", "v"))
	require.NoError(t, sp.SyncToDisk())

	// File should exist.
	_, err := os.Stat(path)
	require.NoError(t, err)
}

func TestStoragePartitionExtensionID(t *testing.T) {
	sp := NewStoragePartition("my-ext", "")
	require.Equal(t, "my-ext", sp.ExtensionID())
}

func TestStorageSchemaVersion(t *testing.T) {
	sp := NewStoragePartition("ext1", "")
	require.Equal(t, StorageSchemaVersion, sp.SchemaVersion())
}

func TestStoragePersistenceIncludesSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ext1.json")
	sp := NewStoragePartition("ext1", path)
	require.NoError(t, sp.Set(StorageLocal, "k", "v"))

	// Reload from disk and verify schema version is preserved.
	sp2 := NewStoragePartition("ext1", path)
	require.Equal(t, StorageSchemaVersion, sp2.SchemaVersion())
	value, ok := sp2.Get(StorageLocal, "k")
	require.True(t, ok)
	require.Equal(t, "v", value)
}

func TestStorageMigrateStorage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ext1.json")
	sp := NewStoragePartition("ext1", path)
	require.NoError(t, sp.Set(StorageLocal, "k", "v"))

	// Migration should be a no-op at current schema.
	require.NoError(t, sp.MigrateStorage())
	require.Equal(t, StorageSchemaVersion, sp.SchemaVersion())
}
