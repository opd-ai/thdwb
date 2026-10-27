package extensions

import (
	"fmt"
	"sync"
)

// LifecycleState describes the execution state of an extension.
type LifecycleState int

const (
	// LifecycleDisabled means the extension is disabled and not running.
	LifecycleDisabled LifecycleState = iota
	// LifecycleEnabled means the extension is enabled and its background
	// script (if any) is running.
	LifecycleEnabled
	// LifecycleStopped means the extension was stopped (e.g. on reload).
	LifecycleStopped
)

// LifecycleManager manages the enable/disable/reload/uninstall lifecycle of
// extensions. It coordinates background script startup/shutdown and tracks
// per-extension state.
type LifecycleManager struct {
	mu       sync.Mutex
	states   map[string]LifecycleState
	enabled  map[string]bool
	onStart  func(ext *Extension) error
	onStop   func(ext *Extension) error
	onReload func(ext *Extension) error
}

// NewLifecycleManager creates a lifecycle manager with no-op hooks.
func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{
		states:  make(map[string]LifecycleState),
		enabled: make(map[string]bool),
	}
}

// SetHooks registers callbacks for extension start, stop, and reload events.
func (lm *LifecycleManager) SetHooks(onStart, onStop, onReload func(ext *Extension) error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.onStart = onStart
	lm.onStop = onStop
	lm.onReload = onReload
}

// SetInitialState sets the initial lifecycle state for an extension without
// invoking hooks. Used when registering extensions discovered at startup.
func (lm *LifecycleManager) SetInitialState(extID string, enabled bool) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.enabled[extID] = enabled
	if enabled {
		lm.states[extID] = LifecycleEnabled
	} else {
		lm.states[extID] = LifecycleDisabled
	}
}

// Enable enables the extension and starts its background script.
func (lm *LifecycleManager) Enable(ext *Extension) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.enabled[ext.ID] {
		return nil // already enabled
	}

	if lm.onStart != nil {
		if err := lm.onStart(ext); err != nil {
			return fmt.Errorf("start extension %s: %w", ext.ID, err)
		}
	}
	lm.enabled[ext.ID] = true
	lm.states[ext.ID] = LifecycleEnabled
	ext.SetEnabled(true)
	return nil
}

// Disable disables the extension and stops its background script.
func (lm *LifecycleManager) Disable(ext *Extension) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if !lm.enabled[ext.ID] {
		return nil // already disabled
	}

	if lm.onStop != nil {
		if err := lm.onStop(ext); err != nil {
			return fmt.Errorf("stop extension %s: %w", ext.ID, err)
		}
	}
	lm.enabled[ext.ID] = false
	lm.states[ext.ID] = LifecycleDisabled
	ext.SetEnabled(false)
	return nil
}

// Reload stops and restarts the extension.
func (lm *LifecycleManager) Reload(ext *Extension) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.onReload != nil {
		if err := lm.onReload(ext); err != nil {
			return fmt.Errorf("reload extension %s: %w", ext.ID, err)
		}
	}
	lm.states[ext.ID] = LifecycleEnabled
	return nil
}

// Uninstall removes the extension entirely.
func (lm *LifecycleManager) Uninstall(ext *Extension) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.enabled[ext.ID] {
		if lm.onStop != nil {
			if err := lm.onStop(ext); err != nil {
				return fmt.Errorf("stop extension %s: %w", ext.ID, err)
			}
		}
	}
	delete(lm.enabled, ext.ID)
	delete(lm.states, ext.ID)
	ext.SetEnabled(false)
	return nil
}

// IsEnabled reports whether the extension is currently enabled.
func (lm *LifecycleManager) IsEnabled(extID string) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	return lm.enabled[extID]
}

// State returns the current lifecycle state of the extension.
func (lm *LifecycleManager) State(extID string) LifecycleState {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	return lm.states[extID]
}

// EnabledIDs returns the IDs of all enabled extensions.
func (lm *LifecycleManager) EnabledIDs() []string {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	var ids []string
	for id, enabled := range lm.enabled {
		if enabled {
			ids = append(ids, id)
		}
	}
	return ids
}
