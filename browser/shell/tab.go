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
// TabManager is not safe for concurrent use; drive it from the UI loop.
type TabManager struct {
	nextID      int
	tabs        []*Tab
	active      *Tab
	closedStack []*Tab // most-recently-closed is last
	maxClosed   int
	navs        map[int]*tabNav // per-tab back/forward history

	listeners   []func()
	onNavigated func(tab *Tab)
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
	m.listeners = append(m.listeners, fn)
}

// OnNavigated registers a callback invoked when a tab performs a new
// navigation via Navigate. Unlike OnChanged, it does not fire for back/forward
// traversal or tab activation, making it suitable for recording history.
func (m *TabManager) OnNavigated(fn func(tab *Tab)) {
	if fn == nil {
		return
	}
	m.onNavigated = fn
}

// Add creates a new tab, makes it active, and returns it. The container
// arguments may be empty for a default-context tab.
func (m *TabManager) Add(title, url, containerID, containerName, containerColor string) *Tab {
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
	m.notify()
	return t
}

// Close removes the tab with the given ID, pushes it onto the recently-closed
// stack, and activates a neighboring tab. It returns the closed tab (useful for
// persistence) or nil if the ID is unknown.
func (m *TabManager) Close(tabID int) *Tab {
	t := m.GetTab(tabID)
	if t == nil {
		return nil
	}
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
	m.notify()
	return t
}

// CloseActive closes the active tab, if any.
func (m *TabManager) CloseActive() *Tab {
	if m.active == nil {
		return nil
	}
	return m.Close(m.active.ID)
}

// Activate makes the tab with the given ID the active tab. It returns the
// activated tab, or nil if the ID is unknown.
func (m *TabManager) Activate(tabID int) *Tab {
	t := m.GetTab(tabID)
	if t == nil {
		return nil
	}
	if m.active == t {
		return t
	}
	if m.active != nil {
		m.active.IsActive = false
	}
	m.setActive(t)
	m.notify()
	return t
}

// MoveTab reorders the tab at index from to index to. Out-of-range indices are
// ignored.
func (m *TabManager) MoveTab(from, to int) {
	if from < 0 || from >= len(m.tabs) || to < 0 || to >= len(m.tabs) || from == to {
		return
	}
	t := m.tabs[from]
	m.tabs = append(m.tabs[:from], m.tabs[from+1:]...)
	m.tabs = append(m.tabs[:to], append([]*Tab{t}, m.tabs[to:]...)...)
	m.notify()
}

// MoveTabToContainer moves the tab with the given ID to a different
// container. It updates the tab's container fields and returns the tab, or
// nil if the tab is unknown. Storage transfer (cookies, localStorage,
// sessionStorage) between containers is the caller's responsibility.
func (m *TabManager) MoveTabToContainer(tabID int, containerID, containerName, containerColor string) *Tab {
	t := m.GetTab(tabID)
	if t == nil {
		return nil
	}
	t.ContainerID = containerID
	t.ContainerName = containerName
	t.ContainerColor = containerColor
	m.notify()
	return t
}

// GetTab returns the tab with the given ID, or nil.
func (m *TabManager) GetTab(tabID int) *Tab {
	for _, t := range m.tabs {
		if t.ID == tabID {
			return t
		}
	}
	return nil
}

// ActiveTab returns the currently active tab, or nil if there are no tabs.
func (m *TabManager) ActiveTab() *Tab {
	return m.active
}

// Tabs returns a snapshot slice of the open tabs in display order.
func (m *TabManager) Tabs() []*Tab {
	out := make([]*Tab, len(m.tabs))
	copy(out, m.tabs)
	return out
}

// Count returns the number of open tabs.
func (m *TabManager) Count() int {
	return len(m.tabs)
}

// SetTitle updates the title of the tab with the given ID.
func (m *TabManager) SetTitle(tabID int, title string) {
	if t := m.GetTab(tabID); t != nil {
		t.Title = title
		m.notify()
	}
}

// SetURL updates the current URL of the tab with the given ID.
func (m *TabManager) SetURL(tabID int, u string) {
	if t := m.GetTab(tabID); t != nil {
		t.URL = u
		m.notify()
	}
}

// Navigate records a user-initiated navigation of the tab to u: the current
// URL is pushed onto the back stack, the forward stack is cleared, and the
// tab's URL is updated. Navigating to the same URL is a no-op.
func (m *TabManager) Navigate(tabID int, u string) {
	t := m.GetTab(tabID)
	if t == nil || u == "" || t.URL == u {
		return
	}
	nav := m.navFor(tabID)
	if t.URL != "" {
		nav.back = append(nav.back, t.URL)
	}
	nav.forward = nil
	t.URL = u
	if m.onNavigated != nil {
		m.onNavigated(t)
	}
	m.notify()
}

// Back navigates the tab back one history entry. It reports whether a
// navigation occurred.
func (m *TabManager) Back(tabID int) bool {
	t := m.GetTab(tabID)
	if t == nil {
		return false
	}
	nav := m.navFor(tabID)
	if len(nav.back) == 0 {
		return false
	}
	prev := nav.back[len(nav.back)-1]
	nav.back = nav.back[:len(nav.back)-1]
	nav.forward = append(nav.forward, t.URL)
	t.URL = prev
	m.notify()
	return true
}

// Forward navigates the tab forward one history entry. It reports whether a
// navigation occurred.
func (m *TabManager) Forward(tabID int) bool {
	t := m.GetTab(tabID)
	if t == nil {
		return false
	}
	nav := m.navFor(tabID)
	if len(nav.forward) == 0 {
		return false
	}
	next := nav.forward[len(nav.forward)-1]
	nav.forward = nav.forward[:len(nav.forward)-1]
	nav.back = append(nav.back, t.URL)
	t.URL = next
	m.notify()
	return true
}

// CanGoBack reports whether the tab has back-history.
func (m *TabManager) CanGoBack(tabID int) bool {
	return m.GetTab(tabID) != nil && len(m.navFor(tabID).back) > 0
}

// CanGoForward reports whether the tab has forward-history.
func (m *TabManager) CanGoForward(tabID int) bool {
	return m.GetTab(tabID) != nil && len(m.navFor(tabID).forward) > 0
}

// CurrentURL returns the tab's current URL, or empty if the tab is unknown.
func (m *TabManager) CurrentURL(tabID int) string {
	if t := m.GetTab(tabID); t != nil {
		return t.URL
	}
	return ""
}

func (m *TabManager) navFor(tabID int) *tabNav {
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
	out := make([]*Tab, len(m.closedStack))
	copy(out, m.closedStack)
	return out
}

// RestoreClosed re-opens the most recently closed tab (if any), removing it
// from the closed stack, and makes it active. It returns the restored tab.
func (m *TabManager) RestoreClosed() *Tab {
	if len(m.closedStack) == 0 {
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
	m.notify()
	return t
}

func (m *TabManager) setActive(t *Tab) {
	t.IsActive = true
	m.active = t
}

func (m *TabManager) notify() {
	for _, fn := range append([]func(){}, m.listeners...) {
		fn()
	}
}
