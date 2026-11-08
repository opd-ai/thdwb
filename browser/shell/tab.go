// Package shell implements the thdwb browser shell: the user-facing UI layer
// that sits on top of the hotdog engine. It provides tab management, the
// address bar, navigation controls, bookmarks, history, preferences,
// extension toolbar icons, and the new tab page.
//
// The shell is designed around a separation of concerns:
//
//   - Pure state managers/stores (TabManager, BookmarkStore, HistoryStore,
//     PreferencesStore, NavigationController) hold the browser's logical state
//     and are fully unit-testable without a display or network.
//   - UI builders (BuildTabBar, BuildToolbar, ...) construct mustard widgets,
//     wire their callbacks to the managers, and are testable headlessly by
//     constructing widgets and firing callbacks directly.
//
// All state managers are intended to be driven from the UI loop's goroutine.
package shell

import (
	"sync"
	"time"
)

// Tab is a single browser tab: its identity, display state, and container
// association. A tab is the unit of navigation in the shell.
type Tab struct {
	// ID uniquely identifies the tab within the shell.
	ID int

	// Title is the document title (or a placeholder) shown on the tab.
	Title string
	// URL is the current page URL. Empty for a fresh tab.
	URL string
	// Favicon is the URL (or asset key) of the page favicon, if any.
	Favicon string

	// ContainerID identifies the container (origin isolation group) this tab
	// belongs to. An empty value means the default (non-container) context.
	ContainerID string
	// ContainerName is the human-readable container name (e.g. "Work").
	ContainerName string
	// ContainerColor is the hex color used for the tab's container indicator.
	ContainerColor string

	// IsActive reports whether this is the currently active (visible) tab.
	IsActive bool

	// Suspended reports whether the tab has been suspended to save memory.
	// Suspended tabs have their page resources released but retain their URL,
	// title, and container association for restoration.
	Suspended bool

	// SuspendedAt records when the tab was suspended, if Suspended is true.
	SuspendedAt *time.Time
}

// IsContainerTab reports whether the tab is associated with a container.
func (t *Tab) IsContainerTab() bool {
	return t.ContainerID != ""
}

// ContainerIndicator returns the short label shown on the tab for its
// container, or an empty string for non-container tabs.
func (t *Tab) ContainerIndicator() string {
	if !t.IsContainerTab() {
		return ""
	}
	return t.ContainerName
}

// TabManager manages the set of open tabs, the active tab, and the
// recently-closed stack. It is the source of truth for tab state in the shell
// and notifies registered listeners after every structural change so UI
// components can stay in sync.
//
// TabManager is safe for concurrent use: extension APIs (chrome.tabs.*,
// chrome.windows.*) run on the background-VM, alarm-timer and webRequest
// observer goroutines and mutate tab state directly, while the Ebitengine
// game goroutine reads it every frame (audit C-23 -- probe: DATA RACE in
// TabManager.Add/Close vs concurrent readers, plus latent fatal
// concurrent-map writes on navs). Structural mutation happens under mu;
// the OnChanged/OnNavigated/OnClosed callbacks deliberately run *outside*
// the lock, because they may enter the render path (which takes a tab's
// jsRuntimeMu) and holding mu across them would create a lock-order
// inversion against page JS that reaches chrome.tabs.*.
type TabManager struct {
	mu sync.Mutex

	nextID      int
	tabs        []*Tab
	active      *Tab
	closedStack []*Tab // most-recently-closed is last
	maxClosed   int
	navs        map[int]*tabNav // per-tab back/forward history

	listeners   []func()
	onNavigated func(tab *Tab)
	onClosed    func(tabID int)
}

// tabNav holds the back/forward history for a single tab. The current URL is
// stored on the Tab itself; these stacks record where the tab has been (back)
// and where it can still go (forward).
type tabNav struct {
	back    []string
	forward []string
}

// NewTabManager returns an empty TabManager.
func NewTabManager() *TabManager {
	return &TabManager{nextID: 1, maxClosed: 25, navs: make(map[int]*tabNav)}
}

// OnChanged registers a listener invoked after any structural change (add,
// close, activate, move). Listeners are called in registration order from the
// UI goroutine, after the change has been applied.
func (m *TabManager) OnChanged(fn func()) {
	if fn == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = append(m.listeners, fn)
}

// OnNavigated registers a callback invoked when a tab performs a new
// navigation via Navigate. Unlike OnChanged, it does not fire for back/forward
// traversal or tab activation, making it suitable for recording history.
func (m *TabManager) OnNavigated(fn func(tab *Tab)) {
	if fn == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onNavigated = fn
}

// OnClosed registers a callback invoked with the tab ID after any close path
// (UI, session restore, extension tabs.remove) has removed the tab. The shell
// uses it to destroy the tab's WindowContext exactly once, so engine state
// (goja VM, DOM, HTTP client, rAF timer) cannot leak through a close route that
// bypasses the Ctrl+W handler (audit R-5).
func (m *TabManager) OnClosed(fn func(tabID int)) {
	if fn == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onClosed = fn
}

// Add creates a new tab, makes it active, and returns it. The container
// arguments may be empty for a default-context tab.
func (m *TabManager) Add(title, url, containerID, containerName, containerColor string) *Tab {
	m.mu.Lock()
	if m.active != nil {
		m.active.IsActive = false
	}
	t := &Tab{
		ID:             m.nextID,
		Title:          title,
		URL:            url,
		ContainerID:    containerID,
		ContainerName:  containerName,
		ContainerColor: containerColor,
		IsActive:       true,
	}
	m.nextID++
	m.tabs = append(m.tabs, t)
	m.active = t
	m.mu.Unlock()
	m.notify()
	return t
}

// Close removes the tab with the given ID, pushes it onto the recently-closed
// stack, and activates a neighboring tab. It returns the closed tab (useful for
// persistence) or nil if the ID is unknown.
func (m *TabManager) Close(tabID int) *Tab {
	// Snapshot the callbacks and run them after releasing mu: onClosed
	// destroys the tab's WindowContext (jsRuntimeMu), and a page-JS path
	// holding that lock can reach chrome.tabs.* -- taking mu there would
	// invert the order (audit C-23).
	m.mu.Lock()
	t := m.tabLocked(tabID)
	if t == nil {
		m.mu.Unlock()
		return nil
	}
	onClosed := m.onClosed
	idx := -1
	for i, candidate := range m.tabs {
		if candidate.ID == tabID {
			idx = i
			break
		}
	}
	m.tabs = append(m.tabs[:idx], m.tabs[idx+1:]...)
	delete(m.navs, tabID)

	m.closedStack = append(m.closedStack, t)
	if len(m.closedStack) > m.maxClosed {
		m.closedStack = m.closedStack[len(m.closedStack)-m.maxClosed:]
	}

	if m.active == t {
		m.active = nil
		if len(m.tabs) == 0 {
			// Closing the last tab leaves no active tab; the shell decides
			// whether to open a fresh tab or quit.
		} else if idx >= len(m.tabs) {
			m.setActive(m.tabs[len(m.tabs)-1])
		} else {
			m.setActive(m.tabs[idx])
		}
	}
	m.mu.Unlock()

	if onClosed != nil {
		onClosed(tabID)
	}
	m.notify()
	return t
}

// CloseActive closes the active tab, if any.
func (m *TabManager) CloseActive() *Tab {
	m.mu.Lock()
	active := m.active
	m.mu.Unlock()
	if active == nil {
		return nil
	}
	return m.Close(active.ID)
}

// Activate makes the tab with the given ID the active tab. It returns the
// activated tab, or nil if the ID is unknown.
func (m *TabManager) Activate(tabID int) *Tab {
	m.mu.Lock()
	if m.active != nil && m.active.ID == tabID {
		t := m.active
		m.mu.Unlock()
		return t
	}
	t := m.tabLocked(tabID)
	if t == nil {
		m.mu.Unlock()
		return nil
	}
	if m.active != nil {
		m.active.IsActive = false
	}
	m.setActive(t)
	m.mu.Unlock()
	m.notify()
	return t
}

// MoveTab reorders the tab at index from to index to. Out-of-range indices are
// ignored.
func (m *TabManager) MoveTab(from, to int) {
	m.mu.Lock()
	if from < 0 || from >= len(m.tabs) || to < 0 || to >= len(m.tabs) || from == to {
		m.mu.Unlock()
		return
	}
	t := m.tabs[from]
	m.tabs = append(m.tabs[:from], m.tabs[from+1:]...)
	m.tabs = append(m.tabs[:to], append([]*Tab{t}, m.tabs[to:]...)...)
	m.mu.Unlock()
	m.notify()
}

// MoveTabToContainer moves the tab with the given ID to a different
// container. It updates the tab's container fields and returns the tab, or
// nil if the tab is unknown. Storage transfer (cookies, localStorage,
// sessionStorage) between containers is the caller's responsibility.
func (m *TabManager) MoveTabToContainer(tabID int, containerID, containerName, containerColor string) *Tab {
	m.mu.Lock()
	t := m.tabLocked(tabID)
	if t == nil {
		m.mu.Unlock()
		return nil
	}
	t.ContainerID = containerID
	t.ContainerName = containerName
	t.ContainerColor = containerColor
	m.mu.Unlock()
	m.notify()
	return t
}

// GetTab returns the tab with the given ID, or nil.
func (m *TabManager) GetTab(tabID int) *Tab {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tabLocked(tabID)
}

// tabLocked is the lock-held lookup used by all methods that already hold mu.
func (m *TabManager) tabLocked(tabID int) *Tab {
	for _, t := range m.tabs {
		if t.ID == tabID {
			return t
		}
	}
	return nil
}

// ActiveTab returns the currently active tab, or nil if there are no tabs.
// The pointer aliases live state: callers must not mutate its fields from a
// goroutine other than the one performing structural changes.
func (m *TabManager) ActiveTab() *Tab {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.active
}

// Tabs returns a snapshot slice of the open tabs in display order.
func (m *TabManager) Tabs() []*Tab {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Tab, len(m.tabs))
	copy(out, m.tabs)
	return out
}

// Count returns the number of open tabs.
func (m *TabManager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.tabs)
}

// SetTitle updates the title of the tab with the given ID.
func (m *TabManager) SetTitle(tabID int, title string) {
	m.mu.Lock()
	t := m.tabLocked(tabID)
	if t != nil {
		t.Title = title
	}
	m.mu.Unlock()
	if t != nil {
		m.notify()
	}
}

// SetURL updates the current URL of the tab with the given ID.
func (m *TabManager) SetURL(tabID int, u string) {
	m.mu.Lock()
	t := m.tabLocked(tabID)
	if t != nil {
		t.URL = u
	}
	m.mu.Unlock()
	if t != nil {
		m.notify()
	}
}

// Navigate records a user-initiated navigation of the tab to u: the current
// URL is pushed onto the back stack, the forward stack is cleared, and the
// tab's URL is updated. Navigating to the same URL is a no-op.
func (m *TabManager) Navigate(tabID int, u string) {
	m.mu.Lock()
	t := m.tabLocked(tabID)
	if t == nil || u == "" || t.URL == u {
		m.mu.Unlock()
		return
	}
	nav := m.navForLocked(tabID)
	if t.URL != "" {
		nav.back = append(nav.back, t.URL)
	}
	nav.forward = nil
	t.URL = u
	onNavigated := m.onNavigated
	m.mu.Unlock()

	// Callbacks run outside mu (see Close): they reach shell UI and the
	// render path, which must never be nested inside the tab lock.
	if onNavigated != nil {
		onNavigated(t)
	}
	m.notify()
}

// Back navigates the tab back one history entry. It reports whether a
// navigation occurred.
func (m *TabManager) Back(tabID int) bool {
	m.mu.Lock()
	t := m.tabLocked(tabID)
	if t == nil {
		m.mu.Unlock()
		return false
	}
	nav := m.navForLocked(tabID)
	if len(nav.back) == 0 {
		m.mu.Unlock()
		return false
	}
	prev := nav.back[len(nav.back)-1]
	nav.back = nav.back[:len(nav.back)-1]
	nav.forward = append(nav.forward, t.URL)
	t.URL = prev
	m.mu.Unlock()
	m.notify()
	return true
}

// Forward navigates the tab forward one history entry. It reports whether a
// navigation occurred.
func (m *TabManager) Forward(tabID int) bool {
	m.mu.Lock()
	t := m.tabLocked(tabID)
	if t == nil {
		m.mu.Unlock()
		return false
	}
	nav := m.navForLocked(tabID)
	if len(nav.forward) == 0 {
		m.mu.Unlock()
		return false
	}
	next := nav.forward[len(nav.forward)-1]
	nav.forward = nav.forward[:len(nav.forward)-1]
	nav.back = append(nav.back, t.URL)
	t.URL = next
	m.mu.Unlock()
	m.notify()
	return true
}

// CanGoBack reports whether the tab has back-history.
func (m *TabManager) CanGoBack(tabID int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tabLocked(tabID) != nil && len(m.navForLocked(tabID).back) > 0
}

// CanGoForward reports whether the tab has forward-history.
func (m *TabManager) CanGoForward(tabID int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tabLocked(tabID) != nil && len(m.navForLocked(tabID).forward) > 0
}

// CurrentURL returns the tab's current URL, or empty if the tab is unknown.
func (m *TabManager) CurrentURL(tabID int) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t := m.tabLocked(tabID); t != nil {
		return t.URL
	}
	return ""
}

// navForLocked returns (creating if needed) the nav stack for a tab. mu must
// be held: navs is a plain map and concurrent map access is fatal.
func (m *TabManager) navForLocked(tabID int) *tabNav {
	nav, ok := m.navs[tabID]
	if !ok {
		nav = &tabNav{}
		m.navs[tabID] = nav
	}
	return nav
}

// RecentlyClosed returns a snapshot of the recently-closed stack, oldest
// first.
func (m *TabManager) RecentlyClosed() []*Tab {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Tab, len(m.closedStack))
	copy(out, m.closedStack)
	return out
}

// RestoreClosed re-opens the most recently closed tab (if any), removing it
// from the closed stack, and makes it active. It returns the restored tab.
func (m *TabManager) RestoreClosed() *Tab {
	m.mu.Lock()
	if len(m.closedStack) == 0 {
		m.mu.Unlock()
		return nil
	}
	t := m.closedStack[len(m.closedStack)-1]
	m.closedStack = m.closedStack[:len(m.closedStack)-1]
	t.IsActive = true
	if m.active != nil {
		m.active.IsActive = false
	}
	m.tabs = append(m.tabs, t)
	m.active = t
	m.mu.Unlock()
	m.notify()
	return t
}

func (m *TabManager) setActive(t *Tab) {
	t.IsActive = true
	m.active = t
}

// notify fires OnChanged listeners outside m.mu (callers release the lock
// first). The listener snapshot itself is taken under mu so a concurrent
// OnChanged registration cannot race the slice iteration.
func (m *TabManager) notify() {
	m.mu.Lock()
	listeners := append([]func(){}, m.listeners...)
	m.mu.Unlock()
	for _, fn := range listeners {
		fn()
	}
}
