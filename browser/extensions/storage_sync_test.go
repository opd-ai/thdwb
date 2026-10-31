package extensions

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockSyncProvider is a SyncProvider for testing.
type mockSyncProvider struct {
	synced   bool
	folders  map[string]string // folderID -> folderPath
	onChange map[string]func() // folderID -> callback
	mu       sync.Mutex
}

func newMockSyncProvider() *mockSyncProvider {
	return &mockSyncProvider{
		synced:   true,
		folders:  make(map[string]string),
		onChange: make(map[string]func()),
	}
}

func (p *mockSyncProvider) SyncFolder(folderPath, folderID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.folders[folderID] = folderPath
	return nil
}

func (p *mockSyncProvider) StopSync(folderID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.folders, folderID)
	return nil
}

func (p *mockSyncProvider) IsSynced(folderID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.synced
}

func (p *mockSyncProvider) OnChange(folderID string, cb func()) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onChange[folderID] = cb
	return nil
}

func (p *mockSyncProvider) triggerChange(folderID string) {
	p.mu.Lock()
	cb := p.onChange[folderID]
	p.mu.Unlock()
	if cb != nil {
		cb()
	}
}

// TestStorageSyncManagerRegister verifies extension registration.
func TestStorageSyncManagerRegister(t *testing.T) {
	provider := newMockSyncProvider()
	dir := t.TempDir()
	m := NewStorageSyncManager(provider, dir)

	require.NoError(t, m.RegisterExtension("ext1"))
	require.True(t, m.IsSynced("ext1"))

	// Registering again should be a no-op.
	require.NoError(t, m.RegisterExtension("ext1"))
}

// TestStorageSyncManagerUnregister verifies extension unregistration.
func TestStorageSyncManagerUnregister(t *testing.T) {
	provider := newMockSyncProvider()
	dir := t.TempDir()
	m := NewStorageSyncManager(provider, dir)

	require.NoError(t, m.RegisterExtension("ext1"))
	require.NoError(t, m.UnregisterExtension("ext1"))
	require.False(t, m.IsSynced("ext1"))
}

// TestStorageSyncManagerWriteRead verifies sync data write/read.
func TestStorageSyncManagerWriteRead(t *testing.T) {
	provider := newMockSyncProvider()
	dir := t.TempDir()
	m := NewStorageSyncManager(provider, dir)

	require.NoError(t, m.RegisterExtension("ext1"))

	areas := map[string]string{"theme": "dark", "fontSize": "14"}
	require.NoError(t, m.WriteSyncData("ext1", areas))

	read, err := m.ReadSyncData("ext1")
	require.NoError(t, err)
	require.Equal(t, "dark", read["theme"])
	require.Equal(t, "14", read["fontSize"])
}

// TestStorageSyncManagerChangeCallback verifies change callbacks fire.
func TestStorageSyncManagerChangeCallback(t *testing.T) {
	provider := newMockSyncProvider()
	dir := t.TempDir()
	m := NewStorageSyncManager(provider, dir)

	require.NoError(t, m.RegisterExtension("ext1"))

	called := false
	require.NoError(t, m.SetChangeCallback("ext1", func() {
		called = true
	}))

	// Simulate a remote change.
	provider.triggerChange("thdwb-sync-ext1")
	require.True(t, called)
}

// TestStorageSyncManagerNotSynced verifies IsSynced reflects provider state.
func TestStorageSyncManagerNotSynced(t *testing.T) {
	provider := newMockSyncProvider()
	provider.synced = false
	dir := t.TempDir()
	m := NewStorageSyncManager(provider, dir)

	require.NoError(t, m.RegisterExtension("ext1"))
	require.False(t, m.IsSynced("ext1"))
}

// TestStorageSyncManagerStop verifies Stop unregisters all extensions.
func TestStorageSyncManagerStop(t *testing.T) {
	provider := newMockSyncProvider()
	dir := t.TempDir()
	m := NewStorageSyncManager(provider, dir)

	require.NoError(t, m.RegisterExtension("ext1"))
	require.NoError(t, m.RegisterExtension("ext2"))
	require.NoError(t, m.Stop())

	require.False(t, m.IsSynced("ext1"))
	require.False(t, m.IsSynced("ext2"))
}
