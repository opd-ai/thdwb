package extensions

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// Discovery scans extension directories and loads manifests.
type Discovery struct {
	mu       sync.Mutex
	dirs     []string
	loaded   map[string]*Extension
	onFound  func(ext *Extension)
}

// NewDiscovery creates an extension discovery instance.
func NewDiscovery() *Discovery {
	return &Discovery{
		dirs:   make([]string, 0),
		loaded: make(map[string]*Extension),
	}
}

// AddWatchDir adds a directory to scan for extensions. Each immediate
// subdirectory is treated as a potential extension containing a manifest.json.
func (d *Discovery) AddWatchDir(dir string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.dirs = append(d.dirs, dir)
}

// SetOnFound registers a callback invoked when a new extension is discovered.
func (d *Discovery) SetOnFound(fn func(ext *Extension)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onFound = fn
}

// Scan scans all watch directories and returns discovered extensions.
// Extensions with invalid manifests are skipped.
func (d *Discovery) Scan() ([]*Extension, error) {
	d.mu.Lock()
	dirs := append([]string(nil), d.dirs...)
	onFound := d.onFound
	d.mu.Unlock()

	var found []*Extension
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read extension dir %s: %w", dir, err)
		}

		// Sort for deterministic ordering.
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			extDir := filepath.Join(dir, entry.Name())
			manifestPath := filepath.Join(extDir, "manifest.json")
			if _, err := os.Stat(manifestPath); err != nil {
				continue // not an extension directory
			}

			manifest, err := ParseManifest(manifestPath)
			if err != nil {
				continue // skip invalid manifests
			}

			id := ExtensionIDFromPath(extDir)
			ext := NewExtension(id, extDir, manifest)

			d.mu.Lock()
			if _, exists := d.loaded[id]; exists {
				d.mu.Unlock()
				continue // already loaded
			}
			d.loaded[id] = ext
			d.mu.Unlock()

			found = append(found, ext)
			if onFound != nil {
				onFound(ext)
			}
		}
	}
	return found, nil
}

// Loaded returns all discovered extensions.
func (d *Discovery) Loaded() []*Extension {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]*Extension, 0, len(d.loaded))
	for _, ext := range d.loaded {
		out = append(out, ext)
	}
	return out
}

// Get returns a discovered extension by ID, or nil if not found.
func (d *Discovery) Get(id string) *Extension {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.loaded[id]
}

// Count returns the number of discovered extensions.
func (d *Discovery) Count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.loaded)
}
