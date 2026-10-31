// Package extensions provides lazy-loading support for extensions: background
// scripts and service workers are not started until the extension is first
// used (e.g., a content script matches a page, an API is called, or a
// message is received).
package extensions

import (
	"sync"
	"time"
)

// LazyLoadState describes the lazy-loading state of an extension.
type LazyLoadState int

const (
	// LazyNotLoaded means the extension's background script has not been
	// started yet.
	LazyNotLoaded LazyLoadState = iota
	// LazyLoading means the extension is currently being loaded.
	LazyLoading
	// LazyLoaded means the extension's background script is running.
	LazyLoaded
)

// LazyLoader manages deferred startup of extensions. Extensions registered
// with the lazy loader start in a "not loaded" state and are only initialized
// when first accessed via LoadIfNeeded.
type LazyLoader struct {
	mu      sync.Mutex
	states  map[string]LazyLoadState
	loader  func(extID string) error // callback to actually load the extension
}

// NewLazyLoader creates a LazyLoader with the given load callback.
func NewLazyLoader(load func(extID string) error) *LazyLoader {
	return &LazyLoader{
		states: make(map[string]LazyLoadState),
		loader: load,
	}
}

// Register marks an extension as lazily loaded (not yet started).
func (ll *LazyLoader) Register(extID string) {
	ll.mu.Lock()
	defer ll.mu.Unlock()
	ll.states[extID] = LazyNotLoaded
}

// IsLoaded reports whether the extension's background script is running.
func (ll *LazyLoader) IsLoaded(extID string) bool {
	ll.mu.Lock()
	defer ll.mu.Unlock()
	return ll.states[extID] == LazyLoaded
}

// LoadIfNeeded loads the extension's background script if it has not been
// loaded yet. It is safe to call multiple times; only the first call
// triggers the actual load.
func (ll *LazyLoader) LoadIfNeeded(extID string) error {
	ll.mu.Lock()
	state, ok := ll.states[extID]
	if !ok {
		// Extension not registered with the lazy loader; treat as already
		// loaded (eager-started extensions).
		ll.mu.Unlock()
		return nil
	}
	if state == LazyLoaded {
		ll.mu.Unlock()
		return nil // already loaded
	}
	if state == LazyLoading {
		// Another goroutine is already loading this extension. Wait for it
		// by releasing the lock and polling briefly.
		ll.mu.Unlock()
		for i := 0; i < 100; i++ {
			time.Sleep(time.Millisecond)
			ll.mu.Lock()
			state = ll.states[extID]
			ll.mu.Unlock()
			if state == LazyLoaded || state == LazyNotLoaded {
				break
			}
		}
		return nil
	}

	// Mark as loading and release the lock while loading.
	ll.states[extID] = LazyLoading
	ll.mu.Unlock()

	if err := ll.loader(extID); err != nil {
		ll.mu.Lock()
		ll.states[extID] = LazyNotLoaded // reset so it can be retried
		ll.mu.Unlock()
		return err
	}

	ll.mu.Lock()
	ll.states[extID] = LazyLoaded
	ll.mu.Unlock()
	return nil
}

// Unload marks the extension as not loaded, allowing it to be reloaded on
// next access. This does not stop any running background script; callers
// must handle that separately.
func (ll *LazyLoader) Unload(extID string) {
	ll.mu.Lock()
	defer ll.mu.Unlock()
	ll.states[extID] = LazyNotLoaded
}

// Count returns the number of extensions tracked by the lazy loader.
func (ll *LazyLoader) Count() int {
	ll.mu.Lock()
	defer ll.mu.Unlock()
	return len(ll.states)
}

// LoadedCount returns the number of extensions whose background scripts are
// currently running.
func (ll *LazyLoader) LoadedCount() int {
	ll.mu.Lock()
	defer ll.mu.Unlock()
	count := 0
	for _, state := range ll.states {
		if state == LazyLoaded {
			count++
		}
	}
	return count
}
