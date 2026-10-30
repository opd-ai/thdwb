package shell

import (
	"fmt"
	"path/filepath"
	"strings"

	assets "github.com/danfragoso/thdwb/assets"
	extensions "github.com/danfragoso/thdwb/browser/extensions"
	hotdog "github.com/danfragoso/thdwb/hotdog"
	mustard "github.com/danfragoso/thdwb/mustard"
)

// BrowserShell is the top-level browser UI. It owns the mustard window and
// app and all browser state managers (tabs, bookmarks, history, settings,
// extensions) and builds the user-facing UI (tab bar, address bar, navigation
// controls, extension toolbar).
type BrowserShell struct {
	app        *mustard.App
	windowBase *mustard.WindowBase
	settings   *hotdog.Settings
	buildInfo  *assets.BuildInfo

	tabManager       *TabManager
	bookmarkManager  *BookmarkManager
	historyManager   *HistoryManager
	extensionManager *ExtensionManager
	extManager       *extensions.Manager
	containerManager *extensions.ContainerManager

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
		extManager:       extensions.NewManager(""),
		containerManager: extensions.NewContainerManager(),
	}
}

// NewBrowserShellWithConfigDir creates a BrowserShell that persists containers
// to the given config directory.
func NewBrowserShellWithConfigDir(settings *hotdog.Settings, buildInfo *assets.BuildInfo, configDir string) (*BrowserShell, error) {
	bs := NewBrowserShell(settings, buildInfo)
	if configDir != "" {
		containerPath := filepath.Join(configDir, "containers.json")
		cm, err := extensions.NewContainerManagerWithPath(containerPath)
		if err != nil {
			return nil, err
		}
		bs.containerManager = cm
	}
	return bs, nil
}

// Init creates the mustard app and window and builds the full UI.
func (bs *BrowserShell) Init() error {
	bs.app = mustard.CreateNewApp(nil, "thdwb")

	window, err := mustard.CreateNewWindow(nil, "thdwb", bs.settings.WindowWidth, bs.settings.WindowHeight, bs.settings.HiDPI)
	if err != nil {
		return fmt.Errorf("create window: %w", err)
	}
	bs.windowBase = window.(*mustard.WindowBase)

	// Wire container operations to the extension manager for chrome.contextualIdentities.
	bs.extManager.SetContainerOperations(bs)

	// Wire all WebExtensions operations interfaces to the shell.
	eo := newExtensionOps(bs)
	bs.extManager.SetTabOperations(eo)
	bs.extManager.SetWindowOperations(eo)
	bs.extManager.SetCookieOperations(eo)
	bs.extManager.SetScriptingOperations(eo)
	bs.extManager.SetContextMenuOperations(eo)
	bs.extManager.SetPermissionOperations(eo)
	bs.extManager.SetAlarmOperations(eo)
	bs.extManager.SetI18nOperations(eo)

	bs.buildUI()

	// Register the window with the app so widgets resolve their window and
	// the event loop knows which window to drive.
	bs.app.AddWindow(bs.windowBase)
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
	bs.navigation.SetHomeURL(bs.settings.Homepage)
	bs.navigation.SetBookmarkCallback(bs.toggleBookmark)
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

	// Record web (http/https) navigations in the history manager.
	bs.tabManager.OnNavigated(func(tab *Tab) { bs.recordHistory(tab) })

	// Open a default tab.
	bs.tabManager.Add("New Tab", bs.settings.Homepage, "", "", "")
}

// Show makes the window visible.
func (bs *BrowserShell) Show() {
	if bs.windowBase != nil {
		bs.windowBase.Show()
	}
}

// Run starts the browser event loop, driving the window via the platform
// backend. It blocks until the window is closed.
func (bs *BrowserShell) Run() {
	if bs.app == nil {
		return
	}
	bs.app.Run(func() {})
}

// Close destroys the browser shell's window and app.
func (bs *BrowserShell) Close() {
	if bs.app != nil && bs.windowBase != nil {
		bs.app.DestroyWindow(bs.windowBase)
	}
	bs.windowBase = nil
	bs.app = nil
}

// toggleBookmark adds a bookmark for the tab's URL if one is not already
// present, otherwise removes the existing bookmark.
func (bs *BrowserShell) toggleBookmark(tab *Tab) {
	if tab == nil || tab.URL == "" {
		return
	}
	for _, b := range bs.bookmarkManager.GetAllBookmarks() {
		if b.URL == tab.URL {
			bs.bookmarkManager.RemoveBookmark(b.ID)
			return
		}
	}
	bs.bookmarkManager.AddBookmark(tab.Title, tab.URL)
}

// recordHistory records a visited web page (http/https) in the history
// manager. Internal thdwb:// pages are not recorded.
func (bs *BrowserShell) recordHistory(tab *Tab) {
	if tab == nil || tab.URL == "" {
		return
	}
	if !strings.HasPrefix(tab.URL, "http://") && !strings.HasPrefix(tab.URL, "https://") {
		return
	}
	bs.historyManager.AddEntry(tab.Title, tab.URL)
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

// ContainerManager returns the shell's container manager.
func (bs *BrowserShell) ContainerManager() *extensions.ContainerManager {
	return bs.containerManager
}

// QueryContainers implements ContainerOperations for chrome.contextualIdentities.
func (bs *BrowserShell) QueryContainers() []extensions.ContainerInfo {
	containers := bs.containerManager.List()
	result := make([]extensions.ContainerInfo, 0, len(containers))
	for _, c := range containers {
		result = append(result, extensions.ContainerInfo{
			ID:    c.ID,
			Name:  c.Name,
			Color: c.Color,
			Icon:  c.Icon,
		})
	}
	return result
}

// GetContainer implements ContainerOperations for chrome.contextualIdentities.
func (bs *BrowserShell) GetContainer(id string) (*extensions.ContainerInfo, error) {
	c := bs.containerManager.Get(id)
	if c == nil {
		return nil, fmt.Errorf("container %q not found", id)
	}
	return &extensions.ContainerInfo{
		ID:    c.ID,
		Name:  c.Name,
		Color: c.Color,
		Icon:  c.Icon,
	}, nil
}

// CreateContainer implements ContainerOperations for chrome.contextualIdentities.
func (bs *BrowserShell) CreateContainer(name, color string) (*extensions.ContainerInfo, error) {
	c, err := bs.containerManager.Create(name, color)
	if err != nil {
		return nil, err
	}
	return &extensions.ContainerInfo{
		ID:    c.ID,
		Name:  c.Name,
		Color: c.Color,
		Icon:  c.Icon,
	}, nil
}

// UpdateContainer implements ContainerOperations for chrome.contextualIdentities.
func (bs *BrowserShell) UpdateContainer(id, name, color string) (*extensions.ContainerInfo, error) {
	c, err := bs.containerManager.Update(id, name, color)
	if err != nil {
		return nil, err
	}
	return &extensions.ContainerInfo{
		ID:    c.ID,
		Name:  c.Name,
		Color: c.Color,
		Icon:  c.Icon,
	}, nil
}

// RemoveContainer implements ContainerOperations for chrome.contextualIdentities.
func (bs *BrowserShell) RemoveContainer(id string) error {
	if !bs.containerManager.Remove(id) {
		return fmt.Errorf("container %q not found", id)
	}
	return nil
}

// Settings returns the shell's settings.
func (bs *BrowserShell) Settings() *hotdog.Settings {
	return bs.settings
}

// Window returns the shell's mustard window base.
func (bs *BrowserShell) Window() *mustard.WindowBase {
	return bs.windowBase
}
