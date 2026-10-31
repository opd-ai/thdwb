package shell

import (
	extensions "github.com/danfragoso/thdwb/browser/extensions"
	mustard "github.com/danfragoso/thdwb/mustard"
)

// NewTabPage is the browser's new tab page. It displays shortcuts to
// frequently visited sites, a search bar, recently closed tabs, bookmarks
// quick access, and a container selector for opening new tabs in a specific
// container.
type NewTabPage struct {
	Frame   *mustard.Frame
	window  *mustard.WindowBase
	manager *TabManager
	cont    *extensions.ContainerManager

	searchInput *mustard.InputWidget
	visible     bool
	onNewTab    func(url, containerID string) // callback when a new tab should be opened
}

// NewNewTabPage creates a new tab page bound to the tab manager and
// container manager.
func NewNewTabPage(window *mustard.WindowBase, manager *TabManager, cont *extensions.ContainerManager) *NewTabPage {
	ntp := &NewTabPage{
		window:   window,
		manager:  manager,
		cont:     cont,
		visible:  false,
	}

	ntp.Frame = mustard.CreateFrame(mustard.VerticalFrame)
	ntp.Frame.SetBackgroundColor("#ffffff")

	// Header.
	header := mustard.CreateLabelWidget("New Tab")
	header.SetFontSize(24)
	header.SetFontColor("#333333")
	ntp.Frame.AttachWidget(header)

	// Search bar.
	searchRow := mustard.CreateFrame(mustard.HorizontalFrame)
	searchRow.SetBackgroundColor("#fafafa")

	ntp.searchInput = mustard.CreateInputWidget()
	ntp.searchInput.SetWidth(400)
	ntp.searchInput.SetReturnCallback(func() {
		ntp.navigate(ntp.searchInput.GetValue(), "")
	})
	searchRow.AttachWidget(ntp.searchInput)

	searchBtn := mustard.CreateButtonWidget("Go", nil)
	searchBtn.SetWidth(50)
	searchBtn.SetHeight(30)
	ntp.window.RegisterButton(searchBtn, func() {
		ntp.navigate(ntp.searchInput.GetValue(), "")
	})
	searchRow.AttachWidget(searchBtn)

	ntp.Frame.AttachWidget(searchRow)

	// Container selector.
	containerRow := mustard.CreateFrame(mustard.HorizontalFrame)
	containerRow.SetBackgroundColor("#fafafa")

	containerLabel := mustard.CreateLabelWidget("Open in container:")
	containerLabel.SetFontSize(12)
	containerLabel.SetFontColor("#666666")
	containerRow.AttachWidget(containerLabel)

	// Default container button.
	defaultBtn := mustard.CreateButtonWidget("Default", nil)
	defaultBtn.SetWidth(80)
	defaultBtn.SetHeight(24)
	ntp.window.RegisterButton(defaultBtn, func() {
		ntp.navigate(ntp.searchInput.GetValue(), "")
	})
	containerRow.AttachWidget(defaultBtn)

	// Container buttons.
	for _, c := range cont.List() {
		c := c
		btn := mustard.CreateButtonWidget(c.Name, nil)
		btn.SetWidth(80)
		btn.SetHeight(24)
		if c.Color != "" {
			btn.SetBackgroundColor(c.Color)
		}
		btnID := c.ID
		ntp.window.RegisterButton(btn, func() {
			ntp.navigate(ntp.searchInput.GetValue(), btnID)
		})
		containerRow.AttachWidget(btn)
	}

	ntp.Frame.AttachWidget(containerRow)

	// Shortcuts section.
	shortcutsLabel := mustard.CreateLabelWidget("Shortcuts")
	shortcutsLabel.SetFontSize(16)
	shortcutsLabel.SetFontColor("#333333")
	ntp.Frame.AttachWidget(shortcutsLabel)

	shortcutsRow := mustard.CreateFrame(mustard.HorizontalFrame)
	shortcutsRow.SetBackgroundColor("#ffffff")

	// Add some default shortcuts.
	for _, shortcut := range []struct {
		title, url string
	}{
		{"Google", "https://google.com"},
		{"GitHub", "https://github.com"},
		{"DuckDuckGo", "https://duckduckgo.com"},
	} {
		shortcut := shortcut
		btn := mustard.CreateButtonWidget(shortcut.title, nil)
		btn.SetWidth(100)
		btn.SetHeight(40)
		btn.SetFontSize(12)
		ntp.window.RegisterButton(btn, func() {
			ntp.navigate(shortcut.url, "")
		})
		shortcutsRow.AttachWidget(btn)
	}

	ntp.Frame.AttachWidget(shortcutsRow)

	return ntp
}

// navigate opens a new tab with the given URL and container.
func (ntp *NewTabPage) navigate(url, containerID string) {
	if url == "" {
		url = "https://duckduckgo.com/"
	}
	if ntp.onNewTab != nil {
		ntp.onNewTab(url, containerID)
	} else {
		ntp.manager.Add("New Tab", url, containerID, "", "")
	}
	ntp.searchInput.SetValue("")
}

// SetNewTabCallback sets the function invoked when a new tab should be
// opened from the new tab page.
func (ntp *NewTabPage) SetNewTabCallback(fn func(url, containerID string)) {
	ntp.onNewTab = fn
}

// Toggle shows or hides the new tab page.
func (ntp *NewTabPage) Toggle() {
	ntp.visible = !ntp.visible
	ntp.Frame.SetVisible(ntp.visible)
}

// IsVisible reports whether the new tab page is currently visible.
func (ntp *NewTabPage) IsVisible() bool {
	return ntp.visible
}

// SetVisible sets the new tab page visibility.
func (ntp *NewTabPage) SetVisible(visible bool) {
	ntp.visible = visible
	ntp.Frame.SetVisible(visible)
}
