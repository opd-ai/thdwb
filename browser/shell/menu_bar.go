package shell

import (
	mustard "github.com/danfragoso/thdwb/mustard"
)

// MenuBar is the browser's top menu bar with standard items: File, Edit,
// View, History, Bookmarks, Tools, Help. Each menu item is a button that can
// trigger actions via callbacks.
type MenuBar struct {
	Frame   *mustard.Frame
	window  *mustard.WindowBase
	visible bool

	fileBtn     *mustard.ButtonWidget
	editBtn     *mustard.ButtonWidget
	viewBtn     *mustard.ButtonWidget
	historyBtn  *mustard.ButtonWidget
	bookmarksBtn *mustard.ButtonWidget
	toolsBtn    *mustard.ButtonWidget
	helpBtn     *mustard.ButtonWidget

	onNewTab       func()
	onOpenFile     func()
	onSavePage     func()
	onPrint        func()
	onFind         func()
	onZoomIn       func()
	onZoomOut      func()
	onZoomReset    func()
	onShowHistory  func()
	onShowBookmarks func()
	onShowSettings func()
	onShowExtensions func()
	onAbout        func()
}

// NewMenuBar constructs a menu bar with standard browser menu items.
func NewMenuBar(window *mustard.WindowBase) *MenuBar {
	mb := &MenuBar{
		window:  window,
		visible: true,
	}

	mb.Frame = mustard.CreateFrame(mustard.HorizontalFrame)
	mb.Frame.SetBackgroundColor("#e8e8e8")
	mb.Frame.SetHeight(24)

	mb.fileBtn = mustard.CreateButtonWidget("File", nil)
	mb.fileBtn.SetHeight(24)
	mb.fileBtn.SetFontSize(12)
	mb.window.RegisterButton(mb.fileBtn, func() { mb.openFileMenu() })
	mb.Frame.AttachWidget(mb.fileBtn)

	mb.editBtn = mustard.CreateButtonWidget("Edit", nil)
	mb.editBtn.SetHeight(24)
	mb.editBtn.SetFontSize(12)
	mb.window.RegisterButton(mb.editBtn, func() { mb.openEditMenu() })
	mb.Frame.AttachWidget(mb.editBtn)

	mb.viewBtn = mustard.CreateButtonWidget("View", nil)
	mb.viewBtn.SetHeight(24)
	mb.viewBtn.SetFontSize(12)
	mb.window.RegisterButton(mb.viewBtn, func() { mb.openViewMenu() })
	mb.Frame.AttachWidget(mb.viewBtn)

	mb.historyBtn = mustard.CreateButtonWidget("History", nil)
	mb.historyBtn.SetHeight(24)
	mb.historyBtn.SetFontSize(12)
	mb.window.RegisterButton(mb.historyBtn, func() {
		if mb.onShowHistory != nil {
			mb.onShowHistory()
		}
	})
	mb.Frame.AttachWidget(mb.historyBtn)

	mb.bookmarksBtn = mustard.CreateButtonWidget("Bookmarks", nil)
	mb.bookmarksBtn.SetHeight(24)
	mb.bookmarksBtn.SetFontSize(12)
	mb.window.RegisterButton(mb.bookmarksBtn, func() {
		if mb.onShowBookmarks != nil {
			mb.onShowBookmarks()
		}
	})
	mb.Frame.AttachWidget(mb.bookmarksBtn)

	mb.toolsBtn = mustard.CreateButtonWidget("Tools", nil)
	mb.toolsBtn.SetHeight(24)
	mb.toolsBtn.SetFontSize(12)
	mb.window.RegisterButton(mb.toolsBtn, func() { mb.openToolsMenu() })
	mb.Frame.AttachWidget(mb.toolsBtn)

	mb.helpBtn = mustard.CreateButtonWidget("Help", nil)
	mb.helpBtn.SetHeight(24)
	mb.helpBtn.SetFontSize(12)
	mb.window.RegisterButton(mb.helpBtn, func() { mb.openHelpMenu() })
	mb.Frame.AttachWidget(mb.helpBtn)

	return mb
}

func (mb *MenuBar) openFileMenu() {
	// File menu actions are handled via callbacks.
	if mb.onNewTab != nil {
		mb.onNewTab()
	}
}

func (mb *MenuBar) openEditMenu() {
	if mb.onFind != nil {
		mb.onFind()
	}
}

func (mb *MenuBar) openViewMenu() {
	if mb.onZoomIn != nil {
		mb.onZoomIn()
	}
}

func (mb *MenuBar) openToolsMenu() {
	if mb.onShowSettings != nil {
		mb.onShowSettings()
	}
}

func (mb *MenuBar) openHelpMenu() {
	if mb.onAbout != nil {
		mb.onAbout()
	}
}

// SetNewTabCallback sets the callback for the New Tab action.
func (mb *MenuBar) SetNewTabCallback(fn func()) {
	mb.onNewTab = fn
}

// SetOpenFileCallback sets the callback for the Open File action.
func (mb *MenuBar) SetOpenFileCallback(fn func()) {
	mb.onOpenFile = fn
}

// SetSavePageCallback sets the callback for the Save Page action.
func (mb *MenuBar) SetSavePageCallback(fn func()) {
	mb.onSavePage = fn
}

// SetPrintCallback sets the callback for the Print action.
func (mb *MenuBar) SetPrintCallback(fn func()) {
	mb.onPrint = fn
}

// SetFindCallback sets the callback for the Find action.
func (mb *MenuBar) SetFindCallback(fn func()) {
	mb.onFind = fn
}

// SetZoomInCallback sets the callback for the Zoom In action.
func (mb *MenuBar) SetZoomInCallback(fn func()) {
	mb.onZoomIn = fn
}

// SetZoomOutCallback sets the callback for the Zoom Out action.
func (mb *MenuBar) SetZoomOutCallback(fn func()) {
	mb.onZoomOut = fn
}

// SetZoomResetCallback sets the callback for the Zoom Reset action.
func (mb *MenuBar) SetZoomResetCallback(fn func()) {
	mb.onZoomReset = fn
}

// SetShowHistoryCallback sets the callback for showing history.
func (mb *MenuBar) SetShowHistoryCallback(fn func()) {
	mb.onShowHistory = fn
}

// SetShowBookmarksCallback sets the callback for showing bookmarks.
func (mb *MenuBar) SetShowBookmarksCallback(fn func()) {
	mb.onShowBookmarks = fn
}

// SetShowSettingsCallback sets the callback for showing settings.
func (mb *MenuBar) SetShowSettingsCallback(fn func()) {
	mb.onShowSettings = fn
}

// SetShowExtensionsCallback sets the callback for showing extensions.
func (mb *MenuBar) SetShowExtensionsCallback(fn func()) {
	mb.onShowExtensions = fn
}

// SetAboutCallback sets the callback for the About action.
func (mb *MenuBar) SetAboutCallback(fn func()) {
	mb.onAbout = fn
}

// Toggle shows or hides the menu bar.
func (mb *MenuBar) Toggle() {
	mb.visible = !mb.visible
	mb.Frame.SetVisible(mb.visible)
}

// IsVisible reports whether the menu bar is currently visible.
func (mb *MenuBar) IsVisible() bool {
	return mb.visible
}

// SetVisible sets the menu bar visibility.
func (mb *MenuBar) SetVisible(visible bool) {
	mb.visible = visible
	mb.Frame.SetVisible(visible)
}
