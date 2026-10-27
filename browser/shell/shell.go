package shell

import (
	"fmt"

	assets "github.com/danfragoso/thdwb/assets"
	hotdog "github.com/danfragoso/thdwb/hotdog"
	mustard "github.com/danfragoso/thdwb/mustard"
)

// BrowserShell is the top-level browser UI. It owns the mustard window and
// all browser state managers (tabs, bookmarks, history, settings, extensions)
// and builds the user-facing UI (tab bar, address bar, navigation controls,
// extension toolbar).
type BrowserShell struct {
	windowBase *mustard.WindowBase
	settings   *hotdog.Settings
	buildInfo  *assets.BuildInfo

	tabManager       *TabManager
	bookmarkManager  *BookmarkManager
	historyManager   *HistoryManager
	extensionManager *ExtensionManager

	addressBar       *AddressBar
	navigation       *NavigationControls
	tabBarUI         *TabBarUI
	extensionToolbar *ExtensionToolbar
}

// NewBrowserShell creates a browser shell with the given settings and build
// info. It does not open a window until Init is called.
func NewBrowserShell(settings *hotdog.Settings, buildInfo *assets.BuildInfo) *BrowserShell {
	if settings == nil {
		settings = &hotdog.Settings{
			Homepage:     "thdwb://newtab/",
			WindowWidth:  1024,
			WindowHeight: 768,
			HiDPI:        true,
		}
	}
	if buildInfo == nil {
		buildInfo = &assets.BuildInfo{
			GitRevision: "dev",
			GitBranch:   "dev",
			HostInfo:    "unknown",
			BuildTime:   "unknown",
		}
	}

	return &BrowserShell{
		settings:         settings,
		buildInfo:        buildInfo,
		tabManager:       NewTabManager(),
		bookmarkManager:  NewBookmarkManager(),
		historyManager:   NewHistoryManager(),
		extensionManager: NewExtensionManager(),
	}
}

// Init creates the mustard window and builds the full UI.
func (bs *BrowserShell) Init() error {
	window, err := mustard.CreateNewWindow(nil, "thdwb", bs.settings.WindowWidth, bs.settings.WindowHeight, bs.settings.HiDPI)
	if err != nil {
		return fmt.Errorf("create window: %w", err)
	}
	bs.windowBase = window.(*mustard.WindowBase)

	bs.buildUI()
	return nil
}

// buildUI constructs the complete browser shell UI.
func (bs *BrowserShell) buildUI() {
	rootFrame := mustard.CreateFrame(mustard.VerticalFrame)

	// Tab bar
	bs.tabBarUI = BuildTabBar(bs.windowBase, bs.tabManager)
	tabBarFrame := mustard.CreateFrame(mustard.HorizontalFrame)
	tabBarFrame.SetBackgroundColor("#e8e8e8")
	tabBarFrame.SetHeight(32)
	tabBarFrame.AttachWidget(bs.tabBarUI.Frame)
	rootFrame.AttachWidget(tabBarFrame)

	// Navigation bar (address bar + controls)
	navFrame := mustard.CreateFrame(mustard.HorizontalFrame)
	navFrame.SetBackgroundColor("#f5f5f5")
	navFrame.SetHeight(36)

	bs.addressBar = NewAddressBar(bs.windowBase, bs.tabManager)
	navFrame.AttachWidget(bs.addressBar.Frame)

	bs.navigation = NewNavigationControls(bs.windowBase, bs.tabManager)
	navFrame.AttachWidget(bs.navigation.Frame)

	rootFrame.AttachWidget(navFrame)

	// Extension toolbar
	bs.extensionToolbar = NewExtensionToolbar(bs.windowBase, bs.extensionManager)
	extFrame := mustard.CreateFrame(mustard.HorizontalFrame)
	extFrame.SetBackgroundColor("#f0f0f0")
	extFrame.SetHeight(32)
	extFrame.AttachWidget(bs.extensionToolbar.Frame)
	rootFrame.AttachWidget(extFrame)

	// Content area (placeholder for web content)
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

	// Open a default tab.
	bs.tabManager.Add("New Tab", bs.settings.Homepage, "", "", "")
}

// Show makes the window visible.
func (bs *BrowserShell) Show() {
	bs.windowBase.Show()
}

// Run starts the browser event loop. This blocks until the window closes.
func (bs *BrowserShell) Run() {
	// The mustard driver's event loop is driven by the platform backend.
	// For now, Run is a no-op placeholder; the window is shown and the
	// application lifecycle is managed by the embedder.
	_ = bs.buildInfo
}

// TabManager returns the shell's tab manager.
func (bs *BrowserShell) TabManager() *TabManager {
	return bs.tabManager
}

// BookmarkManager returns the shell's bookmark manager.
func (bs *BrowserShell) BookmarkManager() *BookmarkManager {
	return bs.bookmarkManager
}

// HistoryManager returns the shell's history manager.
func (bs *BrowserShell) HistoryManager() *HistoryManager {
	return bs.historyManager
}

// ExtensionManager returns the shell's extension manager.
func (bs *BrowserShell) ExtensionManager() *ExtensionManager {
	return bs.extensionManager
}

// Settings returns the shell's settings.
func (bs *BrowserShell) Settings() *hotdog.Settings {
	return bs.settings
}
