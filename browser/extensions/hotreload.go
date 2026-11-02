// Package extensions provides hot reload support for extension development.
// When enabled, the hot reloader watches extension source files and reloads
// the extension when changes are detected.
package extensions

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HotReloader watches extension directories for file changes and triggers
// reloads when source files are modified. Intended for development use only.
type HotReloader struct {
	mu          sync.Mutex
	watching    map[string]*watchState
	reloadFunc  func(extID string) error
	pollInterval time.Duration
}

// watchState tracks the last modification time of a watched extension.
type watchState struct {
	extID      string
	path       string
	lastMod    time.Time
	manifest   string
	background string
}

// NewHotReloader creates a HotReloader with the given reload callback and
// poll interval.
func NewHotReloader(reload func(extID string) error, pollInterval time.Duration) *HotReloader {
	return &HotReloader{
		watching:     make(map[string]*watchState),
		reloadFunc:   reload,
		pollInterval: pollInterval,
	}
}

// Watch starts watching an extension directory for changes.
func (hr *HotReloader) Watch(extID, path string) error {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	if _, ok := hr.watching[extID]; ok {
		return fmt.Errorf("extension %s already being watched", extID)
	}

	ws := &watchState{
		extID:   extID,
		path:    path,
		manifest: filepath.Join(path, "manifest.json"),
		background: filepath.Join(path, "background.js"),
	}

	modTime, err := latestModTime(path)
	if err != nil {
		return fmt.Errorf("stat extension path: %w", err)
	}
	ws.lastMod = modTime

	hr.watching[extID] = ws
	return nil
}

// Unwatch stops watching an extension directory.
func (hr *HotReloader) Unwatch(extID string) {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	delete(hr.watching, extID)
}

// Check checks all watched extensions for changes and reloads any that have
// been modified. Returns the IDs of extensions that were reloaded.
func (hr *HotReloader) Check() ([]string, error) {
	hr.mu.Lock()
	states := make(map[string]*watchState, len(hr.watching))
	for k, v := range hr.watching {
		states[k] = v
	}
	hr.mu.Unlock()

	var reloaded []string
	for extID, ws := range states {
		modTime, err := latestModTime(ws.path)
		if err != nil {
			continue // skip extensions that can't be stat'd
		}

		if modTime.After(ws.lastMod) {
			// File changed; reload the extension.
			if err := hr.reloadFunc(extID); err != nil {
				return reloaded, fmt.Errorf("reload extension %s: %w", extID, err)
			}

			hr.mu.Lock()
			if cur, ok := hr.watching[extID]; ok {
				cur.lastMod = modTime
			}
			hr.mu.Unlock()

			reloaded = append(reloaded, extID)
		}
	}
	return reloaded, nil
}

// latestModTime returns the most recent modification time of any file in
// the directory tree rooted at path. An empty directory is valid (returns
// the zero time); a non-existent path returns an error.
func latestModTime(path string) (time.Time, error) {
	if _, err := os.Stat(path); err != nil {
		return time.Time{}, err
	}

	var latest time.Time
	found := false
	walkErr := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err // propagate walk errors
		}
		if info.IsDir() {
			return nil
		}
		if !found || info.ModTime().After(latest) {
			latest = info.ModTime()
			found = true
		}
		return nil
	})
	if walkErr != nil {
		return time.Time{}, walkErr
	}
	// Empty directory is valid; return zero time so the first check
	// establishes a baseline once files appear.
	if !found {
		return time.Time{}, nil
	}
	return latest, nil
}

// IsWatching reports whether the given extension is being watched.
func (hr *HotReloader) IsWatching(extID string) bool {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	_, ok := hr.watching[extID]
	return ok
}

// Count returns the number of extensions being watched.
func (hr *HotReloader) Count() int {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	return len(hr.watching)
}
