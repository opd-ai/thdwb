package extensions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SyncProvider is the interface for peer-to-peer storage synchronization.
// Implementations integrate with Syncthing or another P2P sync tool.
type SyncProvider interface {
	// SyncFolder marks a folder for synchronization across peers.
	SyncFolder(folderPath, folderID string) error
	// StopSync stops synchronizing a folder.
	StopSync(folderID string) error
	// IsSynced reports whether the folder is currently in sync with peers.
	IsSynced(folderID string) bool
	// OnChange registers a callback invoked when remote changes are applied.
	OnChange(folderID string, cb func()) error
}

// NoopSyncProvider is a SyncProvider that does nothing. It is used when
// Syncthing is not available or sync is disabled.
type NoopSyncProvider struct{}

// SyncFolder is a no-op.
func (p *NoopSyncProvider) SyncFolder(folderPath, folderID string) error { return nil }

// StopSync is a no-op.
func (p *NoopSyncProvider) StopSync(folderID string) error { return nil }

// IsSynced always returns true for the no-op provider.
func (p *NoopSyncProvider) IsSynced(folderID string) bool { return true }

// OnChange is a no-op.
func (p *NoopSyncProvider) OnChange(folderID string, cb func()) error { return nil }

// StorageSyncManager manages chrome.storage.sync peer-to-peer synchronization
// via Syncthing. Each extension's sync storage is stored in a dedicated
// folder that Syncthing replicates across devices.
type StorageSyncManager struct {
	mu          sync.Mutex
	provider    SyncProvider
	syncDir     string            // base directory for all sync folders
	extensions  map[string]*syncExtensionState
	changeCB    map[string]func() // extID -> change callback
}

// syncExtensionState tracks sync state for a single extension.
type syncExtensionState struct {
	extID     string
	folderID  string
	folderPath string
	synced    bool
}

// NewStorageSyncManager creates a storage sync manager.
func NewStorageSyncManager(provider SyncProvider, syncDir string) *StorageSyncManager {
	return &StorageSyncManager{
		provider:   provider,
		syncDir:    syncDir,
		extensions: make(map[string]*syncExtensionState),
		changeCB:   make(map[string]func()),
	}
}

// RegisterExtension registers an extension's sync storage for P2P sync.
// It creates a dedicated folder under syncDir/<extID>/ and configures
// Syncthing to replicate it.
func (m *StorageSyncManager) RegisterExtension(extID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.extensions[extID]; exists {
		return nil // already registered
	}

	folderPath := filepath.Join(m.syncDir, extID)
	folderID := "thdwb-sync-" + extID

	if err := os.MkdirAll(folderPath, 0o750); err != nil {
		return fmt.Errorf("create sync folder: %w", err)
	}

	if err := m.provider.SyncFolder(folderPath, folderID); err != nil {
		return fmt.Errorf("sync folder: %w", err)
	}

	m.extensions[extID] = &syncExtensionState{
		extID:      extID,
		folderID:   folderID,
		folderPath: folderPath,
		synced:     m.provider.IsSynced(folderID),
	}
	return nil
}

// UnregisterExtension stops syncing an extension's storage and removes
// its registration.
func (m *StorageSyncManager) UnregisterExtension(extID string) error {
	m.mu.Lock()
	state, exists := m.extensions[extID]
	if exists {
		delete(m.extensions, extID)
		delete(m.changeCB, extID)
	}
	m.mu.Unlock()

	if !exists {
		return nil
	}

	if err := m.provider.StopSync(state.folderID); err != nil {
		return fmt.Errorf("stop sync: %w", err)
	}
	return nil
}

// IsSynced reports whether the extension's sync storage is in sync.
func (m *StorageSyncManager) IsSynced(extID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.extensions[extID]
	if !ok {
		return false
	}
	return state.synced && m.provider.IsSynced(state.folderID)
}

// SetChangeCallback registers a callback invoked when remote sync changes
// are applied for the extension.
func (m *StorageSyncManager) SetChangeCallback(extID string, cb func()) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.extensions[extID]
	if !ok {
		return fmt.Errorf("extension %s not registered for sync", extID)
	}
	m.changeCB[extID] = cb

	if err := m.provider.OnChange(state.folderID, func() {
		m.mu.Lock()
		state.synced = true
		cb := m.changeCB[extID]
		m.mu.Unlock()
		if cb != nil {
			cb()
		}
	}); err != nil {
		return fmt.Errorf("register change callback: %w", err)
	}
	return nil
}

// SyncPath returns the filesystem path where the extension's sync data is
// stored. Callers read/write this file directly; Syncthing replicates it.
func (m *StorageSyncManager) SyncPath(extID string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.extensions[extID]
	if !ok {
		return ""
	}
	return state.folderPath
}

// SyncFileName is the name of the sync storage file within the sync folder.
const SyncFileName = "storage.json"

// SyncSnapshot is the on-disk format for synced storage.
type SyncSnapshot struct {
	Schema  int                    `json:"schema"`
	Areas   map[string]string      `json:"areas"`
	Updated time.Time              `json:"updated"`
}

// WriteSyncData writes the sync storage data to the extension's sync folder.
func (m *StorageSyncManager) WriteSyncData(extID string, areas map[string]string) error {
	m.mu.Lock()
	state, ok := m.extensions[extID]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("extension %s not registered for sync", extID)
	}

	snap := SyncSnapshot{
		Schema:  StorageSchemaVersion,
		Areas:   areas,
		Updated: time.Now(),
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sync data: %w", err)
	}

	path := filepath.Join(state.folderPath, SyncFileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write sync data: %w", err)
	}
	return nil
}

// ReadSyncData reads the sync storage data from the extension's sync folder.
func (m *StorageSyncManager) ReadSyncData(extID string) (map[string]string, error) {
	m.mu.Lock()
	state, ok := m.extensions[extID]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("extension %s not registered for sync", extID)
	}

	path := filepath.Join(state.folderPath, SyncFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}

	var snap SyncSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("unmarshal sync data: %w", err)
	}
	return snap.Areas, nil
}

// Stop stops all sync activity and shuts down the manager.
func (m *StorageSyncManager) Stop() error {
	m.mu.Lock()
	extIDs := make([]string, 0, len(m.extensions))
	for extID := range m.extensions {
		extIDs = append(extIDs, extID)
	}
	m.mu.Unlock()

	for _, extID := range extIDs {
		if err := m.UnregisterExtension(extID); err != nil {
			return err
		}
	}
	return nil
}
