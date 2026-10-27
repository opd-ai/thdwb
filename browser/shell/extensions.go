package shell

import (
	"sync"
)

// Extension represents a browser extension.
type Extension struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Icon        string `json:"icon,omitempty"`
	Enabled     bool   `json:"enabled"`
}

// ExtensionManager manages browser extensions.
type ExtensionManager struct {
	mu         sync.Mutex
	extensions map[string]*Extension
}

// NewExtensionManager creates a new extension manager.
func NewExtensionManager() *ExtensionManager {
	return &ExtensionManager{
		extensions: make(map[string]*Extension),
	}
}

// AddExtension adds an extension.
func (em *ExtensionManager) AddExtension(ext *Extension) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.extensions[ext.ID] = ext
}

// RemoveExtension removes an extension by ID.
func (em *ExtensionManager) RemoveExtension(id string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	delete(em.extensions, id)
}

// GetExtension returns an extension by ID.
func (em *ExtensionManager) GetExtension(id string) *Extension {
	em.mu.Lock()
	defer em.mu.Unlock()
	return em.extensions[id]
}

// GetAllExtensions returns all extensions.
func (em *ExtensionManager) GetAllExtensions() []*Extension {
	em.mu.Lock()
	defer em.mu.Unlock()

	var result []*Extension
	for _, ext := range em.extensions {
		result = append(result, ext)
	}
	return result
}

// GetEnabledExtensions returns all enabled extensions.
func (em *ExtensionManager) GetEnabledExtensions() []*Extension {
	em.mu.Lock()
	defer em.mu.Unlock()

	var result []*Extension
	for _, ext := range em.extensions {
		if ext.Enabled {
			result = append(result, ext)
		}
	}
	return result
}

// EnableExtension enables an extension by ID.
func (em *ExtensionManager) EnableExtension(id string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	if ext, ok := em.extensions[id]; ok {
		ext.Enabled = true
	}
}

// DisableExtension disables an extension by ID.
func (em *ExtensionManager) DisableExtension(id string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	if ext, ok := em.extensions[id]; ok {
		ext.Enabled = false
	}
}

// ToggleExtension toggles an extension's enabled state.
func (em *ExtensionManager) ToggleExtension(id string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	if ext, ok := em.extensions[id]; ok {
		ext.Enabled = !ext.Enabled
	}
}

// ExtensionCount returns the number of extensions.
func (em *ExtensionManager) ExtensionCount() int {
	em.mu.Lock()
	defer em.mu.Unlock()
	return len(em.extensions)
}
