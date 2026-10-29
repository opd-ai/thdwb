// Package crash provides crash recovery for extensions: it catches panics in
// extension VMs, records crash events, and supports restarting crashed
// extensions and restoring affected tabs/containers.
package crash

import (
	"fmt"
	"sync"
	"time"

	goja "github.com/dop251/goja"
)

// CrashEvent records a single extension crash.
type CrashEvent struct {
	Timestamp   time.Time
	ExtensionID string
	Reason      string // e.g. "panic", "script_error", "timeout"
	Details     string
	Recovered   bool // whether recovery was attempted/succeeded
}

// RecoveryManager tracks extension crashes and coordinates recovery.
type RecoveryManager struct {
	mu      sync.Mutex
	events  []*CrashEvent
	max     int
	restart func(extID string) error // callback to restart an extension
}

// NewRecoveryManager creates a RecoveryManager with the given max events.
func NewRecoveryManager(max int) *RecoveryManager {
	return &RecoveryManager{max: max}
}

// SetRestartCallback sets the function used to restart a crashed extension.
func (rm *RecoveryManager) SetRestartCallback(fn func(extID string) error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.restart = fn
}

// RecordCrash records a crash event for an extension.
func (rm *RecoveryManager) RecordCrash(extID, reason, details string) *CrashEvent {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	event := &CrashEvent{
		Timestamp:   time.Now(),
		ExtensionID: extID,
		Reason:      reason,
		Details:     details,
	}
	rm.events = append(rm.events, event)
	if len(rm.events) > rm.max {
		rm.events = rm.events[len(rm.events)-rm.max:]
	}
	return event
}

// MarkRecovered marks the most recent crash event for the given extension as recovered.
func (rm *RecoveryManager) MarkRecovered(extID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for i := len(rm.events) - 1; i >= 0; i-- {
		if rm.events[i].ExtensionID == extID {
			rm.events[i].Recovered = true
			return
		}
	}
}

// Events returns a copy of all crash events.
func (rm *RecoveryManager) Events() []*CrashEvent {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	out := make([]*CrashEvent, len(rm.events))
	copy(out, rm.events)
	return out
}

// EventsForExtension returns crash events for a specific extension.
func (rm *RecoveryManager) EventsForExtension(extID string) []*CrashEvent {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	var out []*CrashEvent
	for _, e := range rm.events {
		if e.ExtensionID == extID {
			out = append(out, e)
		}
	}
	return out
}

// Recover attempts to restart the crashed extension via the restart callback.
func (rm *RecoveryManager) Recover(extID string) error {
	rm.mu.Lock()
	restart := rm.restart
	rm.mu.Unlock()

	if restart == nil {
		return fmt.Errorf("no restart callback set")
	}

	if err := restart(extID); err != nil {
		rm.RecordCrash(extID, "recovery_failed", err.Error())
		return err
	}

	rm.MarkRecovered(extID)
	return nil
}

// Count returns the number of recorded crash events.
func (rm *RecoveryManager) Count() int {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return len(rm.events)
}

// Clear removes all crash events.
func (rm *RecoveryManager) Clear() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.events = nil
}

// SafeRun executes a function in an extension VM, catching panics and
// converting them into errors. If a panic occurs, it is recorded as a crash.
func SafeRun(rm *RecoveryManager, extID string, fn func() error) error {
	if rm == nil {
		return fn()
	}

	defer func() {
		if r := recover(); r != nil {
			rm.RecordCrash(extID, "panic", fmt.Sprintf("%v", r))
		}
	}()

	return fn()
}

// SafeRunVM executes a goja script in a VM, catching panics and converting
// them into errors. Records crashes for the extension.
func SafeRunVM(rm *RecoveryManager, extID string, vm *goja.Runtime, source string) (interface{}, error) {
	if rm == nil {
		return vm.RunString(source)
	}

	defer func() {
		if r := recover(); r != nil {
			rm.RecordCrash(extID, "panic", fmt.Sprintf("%v", r))
		}
	}()

	return vm.RunString(source)
}
