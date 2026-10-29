package debug

import (
	"sync"
)

// ExtensionInfo is a summary of a loaded extension for the inspector.
type ExtensionInfo struct {
	ID          string
	Name        string
	Version     string
	Enabled     bool
	Permissions []string
	MemoryMB    float64
}

// Inspector provides read-only inspection of loaded extensions.
type Inspector struct {
	mu      sync.Mutex
	extensions map[string]*ExtensionInfo
}

// NewInspector creates a new Inspector.
func NewInspector() *Inspector {
	return &Inspector{extensions: make(map[string]*ExtensionInfo)}
}

// RegisterExtension adds or updates an extension in the inspector.
func (i *Inspector) RegisterExtension(info *ExtensionInfo) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.extensions[info.ID] = info
}

// UnregisterExtension removes an extension from the inspector.
func (i *Inspector) UnregisterExtension(id string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	delete(i.extensions, id)
}

// GetExtension returns the extension info for the given ID, or nil.
func (i *Inspector) GetExtension(id string) *ExtensionInfo {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.extensions[id]
}

// AllExtensions returns all registered extensions.
func (i *Inspector) AllExtensions() []*ExtensionInfo {
	i.mu.Lock()
	defer i.mu.Unlock()

	out := make([]*ExtensionInfo, 0, len(i.extensions))
	for _, info := range i.extensions {
		out = append(out, info)
	}
	return out
}

// EnabledExtensions returns only enabled extensions.
func (i *Inspector) EnabledExtensions() []*ExtensionInfo {
	i.mu.Lock()
	defer i.mu.Unlock()

	var out []*ExtensionInfo
	for _, info := range i.extensions {
		if info.Enabled {
			out = append(out, info)
		}
	}
	return out
}

// Count returns the number of registered extensions.
func (i *Inspector) Count() int {
	i.mu.Lock()
	defer i.mu.Unlock()
	return len(i.extensions)
}
