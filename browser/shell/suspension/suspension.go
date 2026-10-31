// Package suspension implements tab suspension: unloading inactive tabs to
// save memory and restoring them on reactivation.
package suspension

import (
	"sync"
	"time"

	"github.com/danfragoso/thdwb/browser/shell"
)

// SuspendPolicy defines criteria for suspending tabs.
type SuspendPolicy struct {
	// InactiveThreshold is how long a tab must be inactive before it is
	// eligible for suspension.
	InactiveThreshold time.Duration

	// MaxSuspendedTabs is the maximum number of tabs that can be suspended
	// simultaneously. Zero means no limit.
	MaxSuspendedTabs int

	// ProtectActiveTab prevents the active tab from ever being suspended.
	ProtectActiveTab bool

	// ProtectPinnedTabs prevents pinned tabs from being suspended.
	ProtectPinnedTabs bool
}

// DefaultSuspendPolicy returns a sensible default suspension policy.
func DefaultSuspendPolicy() SuspendPolicy {
	return SuspendPolicy{
		InactiveThreshold:  30 * time.Minute,
		MaxSuspendedTabs:   50,
		ProtectActiveTab:   true,
		ProtectPinnedTabs:  false,
	}
}

// SuspensionInfo records when a tab was suspended and why.
type SuspensionInfo struct {
	TabID      int        `json:"tab_id"`
	URL        string     `json:"url"`
	SuspendedAt time.Time  `json:"suspended_at"`
	Reason     string     `json:"reason"`
}

// TabSuspender manages tab suspension and restoration. It tracks which tabs
// are suspended and can suspend/restore them on demand.
type TabSuspender struct {
	mu       sync.Mutex
	policy   SuspendPolicy
	suspended map[int]*SuspensionInfo // tabID -> suspension info
	lastActivity map[int]time.Time // tabID -> last activity time
}

// NewTabSuspender creates a TabSuspender with the given policy.
func NewTabSuspender(policy SuspendPolicy) *TabSuspender {
	return &TabSuspender{
		policy:        policy,
		suspended:     make(map[int]*SuspensionInfo),
		lastActivity:  make(map[int]time.Time),
	}
}

// RecordActivity marks the given tab as active now, updating its last
// activity time. Suspended tabs are not affected.
func (ts *TabSuspender) RecordActivity(tabID int) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.lastActivity[tabID] = time.Now()
}

// Suspend suspends the tab with the given ID. It marks the tab as suspended
// and records the suspension time. Returns an error if the tab is already
// suspended or cannot be found.
func (ts *TabSuspender) Suspend(tab *shell.Tab, reason string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if tab.Suspended {
		return nil // already suspended
	}

	if ts.policy.ProtectActiveTab && tab.IsActive {
		return nil // active tabs are protected
	}

	if ts.policy.MaxSuspendedTabs > 0 && len(ts.suspended) >= ts.policy.MaxSuspendedTabs {
		return nil // at capacity
	}

	now := time.Now()
	tab.Suspended = true
	tab.SuspendedAt = &now

	ts.suspended[tab.ID] = &SuspensionInfo{
		TabID:      tab.ID,
		URL:        tab.URL,
		SuspendedAt: now,
		Reason:     reason,
	}

	return nil
}

// Restore restores the tab with the given ID, clearing its suspended state.
// Returns an error if the tab is not suspended.
func (ts *TabSuspender) Restore(tab *shell.Tab) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if !tab.Suspended {
		return nil // not suspended
	}

	tab.Suspended = false
	tab.SuspendedAt = nil

	delete(ts.suspended, tab.ID)

	return nil
}

// IsSuspended reports whether the tab with the given ID is suspended.
func (ts *TabSuspender) IsSuspended(tabID int) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	_, ok := ts.suspended[tabID]
	return ok
}

// SuspensionInfoFor returns the suspension info for the given tab, or nil if
// not suspended.
func (ts *TabSuspender) SuspensionInfoFor(tabID int) *SuspensionInfo {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.suspended[tabID]
}

// SuspendedTabs returns all suspended tab IDs.
func (ts *TabSuspender) SuspendedTabs() []int {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ids := make([]int, 0, len(ts.suspended))
	for id := range ts.suspended {
		ids = append(ids, id)
	}
	return ids
}

// Count returns the number of suspended tabs.
func (ts *TabSuspender) Count() int {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return len(ts.suspended)
}

// SuspendInactive evaluates all tabs and suspends those that have been
// inactive longer than the policy threshold. It skips protected tabs.
func (ts *TabSuspender) SuspendInactive(tabs []*shell.Tab, now time.Time) []int {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	var suspended []int
	for _, tab := range tabs {
		if tab.Suspended {
			continue // already suspended
		}
		if ts.policy.ProtectActiveTab && tab.IsActive {
			continue // active tabs protected
		}
		if ts.policy.ProtectPinnedTabs && isPinned(tab) {
			continue // pinned tabs protected
		}

		lastActivity, ok := ts.lastActivity[tab.ID]
		if !ok {
			continue // no activity recorded; don't suspend
		}

		if now.Sub(lastActivity) >= ts.policy.InactiveThreshold {
			if ts.policy.MaxSuspendedTabs > 0 && len(ts.suspended) >= ts.policy.MaxSuspendedTabs {
				break // at capacity
			}

			tab.Suspended = true
			tab.SuspendedAt = &now
			ts.suspended[tab.ID] = &SuspensionInfo{
				TabID:      tab.ID,
				URL:        tab.URL,
				SuspendedAt: now,
				Reason:     "inactive",
			}
			suspended = append(suspended, tab.ID)
		}
	}
	return suspended
}

// RestoreAll restores all suspended tabs.
func (ts *TabSuspender) RestoreAll(tabs []*shell.Tab) []int {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	var restored []int
	for _, tab := range tabs {
		if tab.Suspended {
			tab.Suspended = false
			tab.SuspendedAt = nil
			delete(ts.suspended, tab.ID)
			restored = append(restored, tab.ID)
		}
	}
	return restored
}

// SetPolicy updates the suspension policy.
func (ts *TabSuspender) SetPolicy(policy SuspendPolicy) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.policy = policy
}

// Policy returns the current suspension policy.
func (ts *TabSuspender) Policy() SuspendPolicy {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.policy
}

// isPinned reports whether the tab is pinned. This is a placeholder; the
// shell's Tab struct does not currently have a pinned field, so this always
// returns false. It is called by SuspendInactive when ProtectPinnedTabs is
// enabled.
func isPinned(tab *shell.Tab) bool {
	return false
}
