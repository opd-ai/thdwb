package extensions

import (
	"sync"
)

// ContainerStoragePartition provides per-container, per-extension storage.
// Each container gets its own isolated StoragePartition for a given extension.
// This enables Firefox-style container storage isolation where cookies,
// localStorage, and sessionStorage are partitioned by container ID.
type ContainerStoragePartition struct {
	mu       sync.Mutex
	extID    string
	basePath string
	parts    map[string]*StoragePartition // key: containerID
}

// NewContainerStoragePartition creates a container-aware storage partition.
// If basePath is empty, storage is in-memory only.
func NewContainerStoragePartition(extID, basePath string) *ContainerStoragePartition {
	return &ContainerStoragePartition{
		extID:    extID,
		basePath: basePath,
		parts:    make(map[string]*StoragePartition),
	}
}

// partitionFor returns the StoragePartition for the given container ID,
// creating it if it doesn't exist.
func (csp *ContainerStoragePartition) partitionFor(containerID string) *StoragePartition {
	csp.mu.Lock()
	defer csp.mu.Unlock()
	sp, ok := csp.parts[containerID]
	if !ok {
		path := ""
		if csp.basePath != "" {
			path = csp.basePath + "_" + containerID + ".json"
		}
		sp = NewStoragePartition(csp.extID, path)
		csp.parts[containerID] = sp
	}
	return sp
}

// Get returns the value for key in the given storage area and container.
func (csp *ContainerStoragePartition) Get(containerID string, area StorageArea, key string) (string, bool) {
	return csp.partitionFor(containerID).Get(area, key)
}

// Set stores key=value in the given storage area and container.
func (csp *ContainerStoragePartition) Set(containerID string, area StorageArea, key, value string) error {
	return csp.partitionFor(containerID).Set(area, key, value)
}

// Remove deletes key from the given storage area and container.
func (csp *ContainerStoragePartition) Remove(containerID string, area StorageArea, key string) error {
	return csp.partitionFor(containerID).Remove(area, key)
}

// Clear removes all items from the given storage area and container.
func (csp *ContainerStoragePartition) Clear(containerID string, area StorageArea) error {
	return csp.partitionFor(containerID).Clear(area)
}

// Keys returns all keys in the given storage area and container.
func (csp *ContainerStoragePartition) Keys(containerID string, area StorageArea) []string {
	return csp.partitionFor(containerID).Keys(area)
}

// Length returns the number of items in the given storage area and container.
func (csp *ContainerStoragePartition) Length(containerID string, area StorageArea) int {
	return csp.partitionFor(containerID).Length(area)
}

// GetAll returns all key-value pairs in the given storage area and container.
func (csp *ContainerStoragePartition) GetAll(containerID string, area StorageArea) map[string]string {
	return csp.partitionFor(containerID).GetAll(area)
}

// SyncToDisk forces a persist of all dirty container partitions.
func (csp *ContainerStoragePartition) SyncToDisk() error {
	csp.mu.Lock()
	parts := make(map[string]*StoragePartition, len(csp.parts))
	for k, v := range csp.parts {
		parts[k] = v
	}
	csp.mu.Unlock()

	for _, sp := range parts {
		if err := sp.SyncToDisk(); err != nil {
			return err
		}
	}
	return nil
}

// ClearContainer removes all storage for the given container.
func (csp *ContainerStoragePartition) ClearContainer(containerID string) error {
	csp.mu.Lock()
	sp, ok := csp.parts[containerID]
	if !ok {
		csp.mu.Unlock()
		return nil
	}
	delete(csp.parts, containerID)
	csp.mu.Unlock()

	// Clear all storage areas for this container.
	for _, area := range []StorageArea{StorageLocal, StorageSync, StorageManaged} {
		if err := sp.Clear(area); err != nil {
			return err
		}
	}
	return nil
}

// ContainerCount returns the number of distinct containers with storage.
func (csp *ContainerStoragePartition) ContainerCount() int {
	csp.mu.Lock()
	defer csp.mu.Unlock()
	return len(csp.parts)
}

// ExtensionID returns the extension ID this partition belongs to.
func (csp *ContainerStoragePartition) ExtensionID() string {
	return csp.extID
}
