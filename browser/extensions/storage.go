package extensions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// StorageArea is a named storage area within an extension's partition.
type StorageArea string

const (
	// StorageLocal is local storage (persisted per-machine).
	StorageLocal StorageArea = "local"
	// StorageSync is synced storage (synced across devices).
	StorageSync StorageArea = "sync"
	// StorageManaged is managed storage (policy-controlled, read-only).
	StorageManaged StorageArea = "managed"
)

// StorageItem is a single key-value pair in storage.
type StorageItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// StoragePartition provides per-extension, per-area key-value storage.
// Each extension gets its own isolated storage namespace.
type StoragePartition struct {
	mu      sync.Mutex
	extID   string
	path    string
	areas   map[StorageArea]map[string]string
	dirty   bool
}

// NewStoragePartition creates a storage partition for the given extension.
// If path is empty, storage is in-memory only.
func NewStoragePartition(extID, path string) *StoragePartition {
	sp := &StoragePartition{
		extID: extID,
		path:  path,
		areas: make(map[StorageArea]map[string]string),
	}
	sp.areas[StorageLocal] = make(map[string]string)
	sp.areas[StorageSync] = make(map[string]string)
	sp.areas[StorageManaged] = make(map[string]string)
	if path != "" {
		_ = sp.load()
	}
	return sp
}

// load reads persisted storage from disk.
func (sp *StoragePartition) load() error {
	data, err := os.ReadFile(sp.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var raw map[StorageArea]map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for area, items := range raw {
		if sp.areas[area] == nil {
			sp.areas[area] = make(map[string]string)
		}
		for k, v := range items {
			sp.areas[area][k] = v
		}
	}
	return nil
}

// persist writes the current storage state to disk.
func (sp *StoragePartition) persist() error {
	if sp.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(sp.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sp.areas, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sp.path, data, 0o644)
}

// Get returns the value for key in the given storage area.
func (sp *StoragePartition) Get(area StorageArea, key string) (string, bool) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	items, ok := sp.areas[area]
	if !ok {
		return "", false
	}
	v, ok := items[key]
	return v, ok
}

// Set stores key=value in the given storage area and persists.
func (sp *StoragePartition) Set(area StorageArea, key, value string) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	items, ok := sp.areas[area]
	if !ok {
		items = make(map[string]string)
		sp.areas[area] = items
	}
	items[key] = value
	sp.dirty = true
	return sp.persist()
}

// Remove deletes key from the given storage area and persists.
func (sp *StoragePartition) Remove(area StorageArea, key string) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	items, ok := sp.areas[area]
	if !ok {
		return nil
	}
	delete(items, key)
	sp.dirty = true
	return sp.persist()
}

// Clear removes all items from the given storage area and persists.
func (sp *StoragePartition) Clear(area StorageArea) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.areas[area] = make(map[string]string)
	sp.dirty = true
	return sp.persist()
}

// Keys returns all keys in the given storage area.
func (sp *StoragePartition) Keys(area StorageArea) []string {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	items, ok := sp.areas[area]
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}
	return keys
}

// Length returns the number of items in the given storage area.
func (sp *StoragePartition) Length(area StorageArea) int {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	items, ok := sp.areas[area]
	if !ok {
		return 0
	}
	return len(items)
}

// GetAll returns all key-value pairs in the given storage area.
func (sp *StoragePartition) GetAll(area StorageArea) map[string]string {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	items, ok := sp.areas[area]
	if !ok {
		return nil
	}
	out := make(map[string]string, len(items))
	for k, v := range items {
		out[k] = v
	}
	return out
}

// SyncToDisk forces a persist of dirty storage.
func (sp *StoragePartition) SyncToDisk() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if !sp.dirty {
		return nil
	}
	sp.dirty = false
	return sp.persist()
}

// ExtensionID returns the extension ID this partition belongs to.
func (sp *StoragePartition) ExtensionID() string {
	return sp.extID
}

// Format: storage area -> key -> value
type storageSnapshot struct {
	Areas map[StorageArea]map[string]string `json:"areas"`
}
