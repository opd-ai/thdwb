package extensions

import (
	"fmt"
	"path/filepath"
	"sync"
)

// Manager is the top-level extension manager. It combines discovery,
// lifecycle, storage, and content script registration into a single
// interface used by the browser shell.
type Manager struct {
	mu          sync.Mutex
	discovery   *Discovery
	lifecycle   *LifecycleManager
	extensions  map[string]*Extension
	storageDir  string
	contentReg  *ContentScriptRegistry
}

// NewManager creates a new extension manager.
func NewManager(storageDir string) *Manager {
	return &Manager{
		discovery:  NewDiscovery(),
		lifecycle:  NewLifecycleManager(),
		extensions: make(map[string]*Extension),
		storageDir: storageDir,
		contentReg: NewContentScriptRegistry(),
	}
}

// AddWatchDir adds a directory to scan for extensions.
func (m *Manager) AddWatchDir(dir string) {
	m.discovery.AddWatchDir(dir)
}

// Scan discovers and loads extensions from all watch directories.
func (m *Manager) Scan() ([]*Extension, error) {
	found, err := m.discovery.Scan()
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, ext := range found {
		if _, exists := m.extensions[ext.ID]; exists {
			continue
		}
		// Create per-extension storage partition.
		storagePath := ""
		if m.storageDir != "" {
			storagePath = filepath.Join(m.storageDir, ext.ID+".json")
		}
		ext.Storage = NewStoragePartition(ext.ID, storagePath)

		// Register content scripts from manifest.
		for _, cs := range ext.Manifest.ContentScripts {
			script := ContentScript{
				ID:             ext.ID + ":" + cs.JS[0],
				Matches:        cs.Matches,
				ExcludeMatches: cs.ExcludeMatches,
				JS:             cs.JS,
				CSS:            cs.CSS,
				RunAt:          cs.RunAt,
				AllFrames:      cs.AllFrames,
			}
			ext.ContentScripts = append(ext.ContentScripts, script)
			m.contentReg.Register(script)
		}

		// Initialize lifecycle state (extensions start enabled).
		m.lifecycle.SetInitialState(ext.ID, true)

		m.extensions[ext.ID] = ext
	}

	return found, nil
}

// Get returns an extension by ID, or nil if not loaded.
func (m *Manager) Get(id string) *Extension {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.extensions[id]
}

// GetAll returns all loaded extensions.
func (m *Manager) GetAll() []*Extension {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Extension, 0, len(m.extensions))
	for _, ext := range m.extensions {
		out = append(out, ext)
	}
	return out
}

// GetEnabled returns all enabled extensions.
func (m *Manager) GetEnabled() []*Extension {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*Extension
	for _, ext := range m.extensions {
		if ext.IsEnabled() {
			out = append(out, ext)
		}
	}
	return out
}

// Enable enables an extension by ID.
func (m *Manager) Enable(id string) error {
	m.mu.Lock()
	ext := m.extensions[id]
	m.mu.Unlock()
	if ext == nil {
		return fmt.Errorf("extension not found: %s", id)
	}
	return m.lifecycle.Enable(ext)
}

// Disable disables an extension by ID.
func (m *Manager) Disable(id string) error {
	m.mu.Lock()
	ext := m.extensions[id]
	m.mu.Unlock()
	if ext == nil {
		return fmt.Errorf("extension not found: %s", id)
	}
	return m.lifecycle.Disable(ext)
}

// Reload reloads an extension by ID.
func (m *Manager) Reload(id string) error {
	m.mu.Lock()
	ext := m.extensions[id]
	m.mu.Unlock()
	if ext == nil {
		return fmt.Errorf("extension not found: %s", id)
	}
	return m.lifecycle.Reload(ext)
}

// Uninstall removes an extension by ID.
func (m *Manager) Uninstall(id string) error {
	m.mu.Lock()
	ext := m.extensions[id]
	m.mu.Unlock()
	if ext == nil {
		return fmt.Errorf("extension not found: %s", id)
	}
	if err := m.lifecycle.Uninstall(ext); err != nil {
		return err
	}
	m.mu.Lock()
	delete(m.extensions, id)
	m.mu.Unlock()
	return nil
}

// IsEnabled reports whether an extension is enabled.
func (m *Manager) IsEnabled(id string) bool {
	return m.lifecycle.IsEnabled(id)
}

// Count returns the number of loaded extensions.
func (m *Manager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.extensions)
}

// EnabledCount returns the number of enabled extensions.
func (m *Manager) EnabledCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, ext := range m.extensions {
		if ext.IsEnabled() {
			count++
		}
	}
	return count
}

// StorageFor returns the storage partition for the given extension ID.
func (m *Manager) StorageFor(id string) *StoragePartition {
	m.mu.Lock()
	defer m.mu.Unlock()
	ext := m.extensions[id]
	if ext == nil {
		return nil
	}
	return ext.Storage
}

// ContentRegistry returns the content script registry.
func (m *Manager) ContentRegistry() *ContentScriptRegistry {
	return m.contentReg
}

// SetLifecycleHooks registers start/stop/reload hooks for background scripts.
func (m *Manager) SetLifecycleHooks(onStart, onStop, onReload func(ext *Extension) error) {
	m.lifecycle.SetHooks(onStart, onStop, onReload)
}
