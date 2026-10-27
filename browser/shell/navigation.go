package shell

import (
	mustard "github.com/danfragoso/thdwb/mustard"
)

// NavigationControls holds the browser's core navigation buttons: back,
// forward, reload, home, bookmark, and menu. It is bound to a TabManager so
// button availability reflects the active tab's navigation state, and it
// drives real back/forward/reload/home navigation through the manager.
type NavigationControls struct {
	Frame   *mustard.Frame
	window  *mustard.WindowBase
	manager *TabManager

	backBtn     *mustard.ButtonWidget
	forwardBtn  *mustard.ButtonWidget
	reloadBtn   *mustard.ButtonWidget
	homeBtn     *mustard.ButtonWidget
	bookmarkBtn *mustard.ButtonWidget
	menuBtn     *mustard.ButtonWidget

	homeURL      string
	bookmarkFunc func(tab *Tab)
}

// NewNavigationControls constructs a horizontal navigation bar bound to
// manager. Buttons that have no effect in the current state (e.g. back when
// there is no history) are visually disabled.
func NewNavigationControls(window *mustard.WindowBase, manager *TabManager) *NavigationControls {
	nc := &NavigationControls{
		window:  window,
		manager: manager,
		homeURL: "thdwb://newtab/",
	}

	nc.Frame = mustard.CreateFrame(mustard.HorizontalFrame)
	nc.Frame.SetHeight(36)
	nc.Frame.SetBackgroundColor("#f5f5f5")

	nc.backBtn = mustard.CreateButtonWidget("◀", nil)
	nc.backBtn.SetWidth(30)
	nc.backBtn.SetHeight(30)
	nc.backBtn.SetFontSize(14)
	nc.Frame.AttachWidget(nc.backBtn)

	nc.forwardBtn = mustard.CreateButtonWidget("▶", nil)
	nc.forwardBtn.SetWidth(30)
	nc.forwardBtn.SetHeight(30)
	nc.forwardBtn.SetFontSize(14)
	nc.Frame.AttachWidget(nc.forwardBtn)

	nc.reloadBtn = mustard.CreateButtonWidget("↻", nil)
	nc.reloadBtn.SetWidth(30)
	nc.reloadBtn.SetHeight(30)
	nc.reloadBtn.SetFontSize(14)
	nc.Frame.AttachWidget(nc.reloadBtn)

	nc.homeBtn = mustard.CreateButtonWidget("⌂", nil)
	nc.homeBtn.SetWidth(30)
	nc.homeBtn.SetHeight(30)
	nc.homeBtn.SetFontSize(14)
	nc.Frame.AttachWidget(nc.homeBtn)

	nc.bookmarkBtn = mustard.CreateButtonWidget("★", nil)
	nc.bookmarkBtn.SetWidth(30)
	nc.bookmarkBtn.SetHeight(30)
	nc.bookmarkBtn.SetFontSize(14)
	nc.Frame.AttachWidget(nc.bookmarkBtn)

	nc.menuBtn = mustard.CreateButtonWidget("≡", nil)
	nc.menuBtn.SetWidth(30)
	nc.menuBtn.SetHeight(30)
	nc.menuBtn.SetFontSize(14)
	nc.Frame.AttachWidget(nc.menuBtn)

	window.RegisterButton(nc.backBtn, func() { nc.goBack() })
	window.RegisterButton(nc.forwardBtn, func() { nc.goForward() })
	window.RegisterButton(nc.reloadBtn, func() { nc.reload() })
	window.RegisterButton(nc.homeBtn, func() { nc.goHome() })
	window.RegisterButton(nc.bookmarkBtn, func() { nc.toggleBookmark() })
	window.RegisterButton(nc.menuBtn, func() { nc.openMenu() })

	manager.OnChanged(func() { nc.updateStates() })
	nc.updateStates()

	return nc
}

// SetHomeURL sets the URL the home button navigates to.
func (nc *NavigationControls) SetHomeURL(u string) {
	if u != "" {
		nc.homeURL = u
	}
}

// SetBookmarkCallback sets the function invoked when the bookmark button is
// pressed, passed the active tab. The shell wires this to its BookmarkManager.
func (nc *NavigationControls) SetBookmarkCallback(fn func(tab *Tab)) {
	nc.bookmarkFunc = fn
}

// goBack navigates the active tab back one history entry.
func (nc *NavigationControls) goBack() {
	tab := nc.manager.ActiveTab()
	if tab == nil {
		return
	}
	nc.manager.Back(tab.ID)
}

// goForward navigates the active tab forward one history entry.
func (nc *NavigationControls) goForward() {
	tab := nc.manager.ActiveTab()
	if tab == nil {
		return
	}
	nc.manager.Forward(tab.ID)
}

// reload re-issues the active tab's current URL.
func (nc *NavigationControls) reload() {
	tab := nc.manager.ActiveTab()
	if tab == nil || tab.URL == "" {
		return
	}
	nc.manager.SetURL(tab.ID, tab.URL)
}

// goHome navigates the active tab to the configured home URL.
func (nc *NavigationControls) goHome() {
	tab := nc.manager.ActiveTab()
	if tab == nil {
		return
	}
	nc.manager.Navigate(tab.ID, nc.homeURL)
}

// toggleBookmark bookmarks or unbookmarks the active tab's URL via the
// configured callback.
func (nc *NavigationControls) toggleBookmark() {
	tab := nc.manager.ActiveTab()
	if tab == nil || tab.URL == "" || nc.bookmarkFunc == nil {
		return
	}
	nc.bookmarkFunc(tab)
}

// openMenu opens the browser menu (placeholder for the settings/history/
// bookmarks menu).
func (nc *NavigationControls) openMenu() {
	// Placeholder for a future menu.
}

// updateStates refreshes button enabled/disabled styling based on the active
// tab and its navigation history.
func (nc *NavigationControls) updateStates() {
	tab := nc.manager.ActiveTab()
	active := tab != nil
	canBack := active && nc.manager.CanGoBack(tab.ID)
	canForward := active && nc.manager.CanGoForward(tab.ID)
	hasURL := active && tab.URL != ""

	setEnabled := func(btn *mustard.ButtonWidget, enabled bool) {
		if btn == nil {
			return
		}
		if enabled {
			btn.SetBackgroundColor("#ffffff")
			btn.SetFontColor("#000000")
		} else {
			btn.SetBackgroundColor("#cccccc")
			btn.SetFontColor("#666666")
		}
	}

	setEnabled(nc.backBtn, canBack)
	setEnabled(nc.forwardBtn, canForward)
	setEnabled(nc.reloadBtn, hasURL)
	setEnabled(nc.homeBtn, active)
	setEnabled(nc.bookmarkBtn, hasURL)
	setEnabled(nc.menuBtn, true)
}
