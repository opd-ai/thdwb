package shell

import (
	mustard "github.com/danfragoso/thdwb/mustard"
)

// buildUI constructs the browser shell UI with tab bar, address bar, and navigation.
func (bs *BrowserShell) buildUI() {
	// Main vertical frame
	rootFrame := mustard.CreateFrame(mustard.VerticalFrame)

	// Tab bar
	tabBar := bs.buildTabBar()
	rootFrame.AttachWidget(tabBar)

	// Navigation bar (address bar + controls)
	navigationBar := bs.buildNavigationBar()
	rootFrame.AttachWidget(navigationBar)

	// Content area (will be filled with web content)
	contentArea := mustard.CreateFrame(mustard.VerticalFrame)
	contentArea.SetBackgroundColor("#ffffff")
	rootFrame.AttachWidget(contentArea)

	// Status bar
	statusBar := mustard.CreateLabelWidget("Ready")
	statusBar.SetFontSize(12)
	statusBar.SetFontColor("#666666")
	statusBar.SetHeight(20)
	rootFrame.AttachWidget(statusBar)

	bs.windowBase.SetRootFrame(rootFrame)
}

// buildTabBar creates the tab bar UI.
func (bs *BrowserShell) buildTabBar() *mustard.Frame {
	tabBar := mustard.CreateFrame(mustard.HorizontalFrame)
	tabBar.SetBackgroundColor("#e8e8e8")
	tabBar.SetHeight(32)

	bs.tabBarUI = BuildTabBar(bs.windowBase, bs.tabManager)
	tabBar.AttachWidget(bs.tabBarUI.Frame)

	return tabBar
}

// buildNavigationBar creates the navigation bar with address bar and controls.
func (bs *BrowserShell) buildNavigationBar() *mustard.Frame {
	navigationBar := mustard.CreateFrame(mustard.VerticalFrame)
	navigationBar.SetBackgroundColor("#f5f5f5")
	navigationBar.SetHeight(36)

	// Back button
	backButton := mustard.CreateButtonWidget("◀", nil)
	backButton.SetWidth(30)
	backButton.SetHeight(30)
	navigationBar.AttachWidget(backButton)

	// Forward button
	forwardButton := mustard.CreateButtonWidget("▶", nil)
	forwardButton.SetWidth(30)
	forwardButton.SetHeight(30)
	navigationBar.AttachWidget(forwardButton)

	// Reload button
	reloadButton := mustard.CreateButtonWidget("↻", nil)
	reloadButton.SetWidth(30)
	reloadButton.SetHeight(30)
	navigationBar.AttachWidget(reloadButton)

	// Home button
	homeButton := mustard.CreateButtonWidget("⌂", nil)
	homeButton.SetWidth(30)
	homeButton.SetHeight(30)
	navigationBar.AttachWidget(homeButton)

	// Address bar
	addressBar := mustard.CreateLabelWidget("thdwb://newtab/")
	addressBar.SetFontSize(14)
	addressBar.SetFontColor("#333333")
	navigationBar.AttachWidget(addressBar)

	// Bookmarks button
	bookmarkButton := mustard.CreateButtonWidget("★", nil)
	bookmarkButton.SetWidth(30)
	bookmarkButton.SetHeight(30)
	navigationBar.AttachWidget(bookmarkButton)

	// Menu button
	menuButton := mustard.CreateButtonWidget("≡", nil)
	menuButton.SetWidth(30)
	menuButton.SetHeight(30)
	navigationBar.AttachWidget(menuButton)

	return navigationBar
}