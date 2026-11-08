package shell

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	assets "github.com/danfragoso/thdwb/assets"
	bun "github.com/danfragoso/thdwb/bun"
	extensions "github.com/danfragoso/thdwb/browser/extensions"
	hotdog "github.com/danfragoso/thdwb/hotdog"
	ketchup "github.com/danfragoso/thdwb/ketchup"
	mustard "github.com/danfragoso/thdwb/mustard"
	profiler "github.com/danfragoso/thdwb/profiler"
	debug "github.com/danfragoso/thdwb/browser/debug"
	errors "github.com/danfragoso/thdwb/browser/errors"
	render "github.com/danfragoso/thdwb/browser/render"
	telemetry "github.com/danfragoso/thdwb/browser/telemetry"
	crash "github.com/danfragoso/thdwb/browser/extensions/crash"
	optimize "github.com/danfragoso/thdwb/sauce/optimize"
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
	browserSettings  *BrowserSettings

	addressBar       *AddressBar
	navigation       *NavigationControls
	tabBarUI         *TabBarUI
	extensionToolbar *ExtensionToolbar
	bookmarksSidebar *BookmarksSidebar
	historySidebar   *HistorySidebar
	settingsPage     *SettingsPage
	newTabPage       *NewTabPage
	menuBar          *MenuBar

	webContent *mustard.CanvasWidget

	// tabWindowContexts maps tab IDs to their hotdog WindowContext for
	// per-tab rendering and JS isolation. Extension VM/alarm/observer
	// goroutines create and destroy contexts while the game goroutine reads
	// them (audit C-23), so all production access goes through the guarded
	// helpers below.
	tabWindowContexts map[int]*hotdog.WindowContext
	tabWCsMu          sync.Mutex

	sessionManager   *SessionManager
	memoryProfiler   *profiler.MemoryProfiler
	pageLoadTracker  *profiler.PageLoadTracker
	inspector        *debug.Inspector
	errorLogger      *errors.Logger
	consoleCollector *debug.ConsoleCollector
	networkMonitor   *debug.NetworkMonitor
	recoveryManager  *crash.RecoveryManager
	telemetry        *telemetry.Telemetry
	renderer         *render.Renderer
	networkOptimizer *optimize.NetworkOptimizer

	// injectedScripts tracks which content scripts have been injected into
	// which tab (tabID -> scriptID -> true) to prevent duplicate injection.
	injectedScripts map[int]map[string]bool
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

	sh := &BrowserShell{
		settings:         settings,
		buildInfo:        buildInfo,
		tabManager:       NewTabManager(),
		bookmarkManager:  NewBookmarkManager(),
		historyManager:   NewHistoryManager(),
		extensionManager: NewExtensionManager(),
		extManager:       extensions.NewManager(""),
		containerManager: extensions.NewContainerManager(),
		browserSettings:  NewBrowserSettings(),
		injectedScripts:  make(map[int]map[string]bool),
		tabWindowContexts: make(map[int]*hotdog.WindowContext),

		sessionManager:   NewSessionManager(""),
		memoryProfiler:   profiler.NewMemoryProfiler(),
		pageLoadTracker:  profiler.NewPageLoadTracker(),
		inspector:        debug.NewInspector(),
		errorLogger:      errors.NewLogger(1000),
		consoleCollector: debug.NewConsoleCollector(1000),
		networkMonitor:   debug.NewNetworkMonitor(1000),
		recoveryManager:  crash.NewRecoveryManager(100),
		telemetry:        telemetry.MustNew(telemetry.DefaultConfig(), ""),
		renderer:         render.NewRenderer(nil),
		networkOptimizer: optimize.NewNetworkOptimizer(),
	}

	// Destroy the tab's engine state on EVERY close path (Ctrl+W, session
	// restore, extension tabs.remove), not just the keyboard shortcut. The
	// WindowContext pins a goja VM, the full DOM, cookie jar and an rAF
	// timer, so a close that bypassed this destroyed nothing until the
	// process exited (audit R-5). Registered here rather than in the UI
	// setup so headless shells clean up too.
	sh.tabManager.OnClosed(func(tabID int) {
		sh.tabWCsMu.Lock()
		wc := sh.tabWindowContexts[tabID]
		delete(sh.tabWindowContexts, tabID)
		sh.tabWCsMu.Unlock()
		if wc != nil {
			wc.Destroy()
		}
	})

	return sh
}

// NewBrowserShellWithConfigDir creates a BrowserShell that persists containers
// and sessions to the given config directory.
func NewBrowserShellWithConfigDir(settings *hotdog.Settings, buildInfo *assets.BuildInfo, configDir string) (*BrowserShell, error) {
	bs := NewBrowserShell(settings, buildInfo)
	if configDir != "" {
		containerPath := filepath.Join(configDir, "containers.json")
		cm, err := extensions.NewContainerManagerWithPath(containerPath)
		if err != nil {
			return nil, err
		}
		bs.containerManager = cm

		sessionPath := filepath.Join(configDir, "session.json")
		bs.sessionManager = NewSessionManager(sessionPath)
	}
	return bs, nil
}

// SessionManager returns the shell's session manager.
func (bs *BrowserShell) SessionManager() *SessionManager {
	return bs.sessionManager
}

// MemoryProfiler returns the shell's memory profiler.
func (bs *BrowserShell) MemoryProfiler() *profiler.MemoryProfiler {
	return bs.memoryProfiler
}

// PageLoadTracker returns the shell's page load tracker.
func (bs *BrowserShell) PageLoadTracker() *profiler.PageLoadTracker {
	return bs.pageLoadTracker
}

// Inspector returns the shell's extension inspector.
func (bs *BrowserShell) Inspector() *debug.Inspector {
	return bs.inspector
}

// ErrorLogger returns the shell's error logger.
func (bs *BrowserShell) ErrorLogger() *errors.Logger {
	return bs.errorLogger
}

// ConsoleCollector returns the shell's console collector.
func (bs *BrowserShell) ConsoleCollector() *debug.ConsoleCollector {
	return bs.consoleCollector
}

// NetworkMonitor returns the shell's network monitor.
func (bs *BrowserShell) NetworkMonitor() *debug.NetworkMonitor {
	return bs.networkMonitor
}

// RecoveryManager returns the shell's crash recovery manager.
func (bs *BrowserShell) RecoveryManager() *crash.RecoveryManager {
	return bs.recoveryManager
}

// Telemetry returns the shell's telemetry instance.
func (bs *BrowserShell) Telemetry() *telemetry.Telemetry {
	return bs.telemetry
}

// Renderer returns the shell's rendering optimizer (batch, dirty tracker,
// layer cache).
func (bs *BrowserShell) Renderer() *render.Renderer {
	return bs.renderer
}

// NetworkOptimizer returns the shell's network optimizer (DNS cache, TLS
// session reuse, connection pooling).
func (bs *BrowserShell) NetworkOptimizer() *optimize.NetworkOptimizer {
	return bs.networkOptimizer
}

// Init creates the mustard app and window and builds the full UI.
// If a session file exists, it restores tabs and window size from it.
func (bs *BrowserShell) Init() error {
	bs.app = mustard.CreateNewApp(nil, "thdwb")

	// Restore session before creating the window so window size is applied.
	if err := bs.sessionManager.Restore(bs); err != nil {
		return fmt.Errorf("restore session: %w", err)
	}

	window, err := mustard.CreateNewWindow(nil, "thdwb", bs.settings.WindowWidth, bs.settings.WindowHeight, bs.settings.HiDPI)
	if err != nil {
		return fmt.Errorf("create window: %w", err)
	}
	windowBase, ok := window.(*mustard.WindowBase)
	if !ok {
		return fmt.Errorf("window is not a *mustard.WindowBase")
	}
	bs.windowBase = windowBase

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
	bs.extManager.SetWebRequestOperations(eo)
	bs.extManager.SetNotificationOperations(eo)
	dlOps := newDownloadOps(bs)
	bs.extManager.SetDownloadOperations(dlOps)
	bs.extManager.SetBookmarkOperations(eo)
	bs.extManager.SetHistoryOperations(eo)
	bs.extManager.SetActionOperations(eo)

	// Auto-discover and load extensions from watch directories.
	if _, err := bs.extManager.Scan(); err != nil {
		return fmt.Errorf("scan extensions: %w", err)
	}

	// Enable all discovered extensions (fires onStartup for background scripts).
	for _, ext := range bs.extManager.GetEnabled() {
		if err := bs.extManager.Enable(ext.ID); err != nil {
			return fmt.Errorf("enable extension %s: %w", ext.ID, err)
		}
	}

	bs.buildUI()

	// Register the window with the app so widgets resolve their window and
	// the event loop knows which window to drive.
	bs.app.AddWindow(bs.windowBase)
	return nil
}

// buildUI constructs the complete browser shell UI.
func (bs *BrowserShell) buildUI() {
	rootFrame := mustard.CreateFrame(mustard.VerticalFrame)

	// Menu bar.
	bs.menuBar = NewMenuBar(bs.windowBase)
	bs.menuBar.SetNewTabCallback(func() {
		bs.tabManager.Add("New Tab", bs.settings.Homepage, "", "", "")
	})
	bs.menuBar.SetShowHistoryCallback(func() {
		bs.historySidebar.Toggle()
	})
	bs.menuBar.SetShowBookmarksCallback(func() {
		bs.bookmarksSidebar.Toggle()
	})
	bs.menuBar.SetShowSettingsCallback(func() {
		bs.settingsPage.Toggle()
	})
	bs.menuBar.SetShowExtensionsCallback(func() {
		bs.extensionToolbar.Frame.SetVisible(!bs.extensionToolbar.Frame.GetVisible())
	})
	rootFrame.AttachWidget(bs.menuBar.Frame)

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
	bs.addressBar.SetHistoryProvider(bs.historyManager.GetAllEntries)
	bs.addressBar.SetBookmarkProvider(bs.bookmarkManager.GetAllBookmarks)
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

	// Bookmarks sidebar (hidden by default; toggled via Ctrl+B).
	bs.bookmarksSidebar = NewBookmarksSidebar(bs.windowBase, bs.bookmarkManager)
	bs.bookmarksSidebar.SetSelectCallback(func(b *Bookmark) {
		tab := bs.tabManager.ActiveTab()
		if tab != nil {
			bs.tabManager.Navigate(tab.ID, b.URL)
		}
	})
	bs.bookmarksSidebar.SetVisible(false)
	rootFrame.AttachWidget(bs.bookmarksSidebar.Frame)

	// History sidebar (hidden by default; toggled via Ctrl+H).
	bs.historySidebar = NewHistorySidebar(bs.windowBase, bs.historyManager)
	bs.historySidebar.SetSelectCallback(func(entry *HistoryEntry) {
		tab := bs.tabManager.ActiveTab()
		if tab != nil {
			bs.tabManager.Navigate(tab.ID, entry.URL)
		}
	})
	bs.historySidebar.SetVisible(false)
	rootFrame.AttachWidget(bs.historySidebar.Frame)

	// Settings page (hidden by default; toggled via Ctrl+,).
	bs.settingsPage = NewSettingsPage(bs.windowBase, bs.browserSettings)
	bs.settingsPage.SetVisible(false)
	rootFrame.AttachWidget(bs.settingsPage.Frame)

	// New tab page (shown on new tabs; toggled via Ctrl+T when no URL).
	bs.newTabPage = NewNewTabPage(bs.windowBase, bs.tabManager, bs.containerManager)
	bs.newTabPage.SetNewTabCallback(func(url, containerID string) {
		bs.tabManager.Add("New Tab", url, containerID, "", "")
	})
	bs.newTabPage.SetVisible(false)
	rootFrame.AttachWidget(bs.newTabPage.Frame)

	// Content area (web content rendered via bun layout engine)
	bs.webContent = mustard.CreateCanvasWidget(func(cw *mustard.CanvasWidget) {
		// Rendering is driven by renderWebContent; this is a no-op placeholder.
	})
	bs.webContent.SetBackgroundColor("#ffffff")
	rootFrame.AttachWidget(bs.webContent)

	// Status bar
	statusBar := mustard.CreateLabelWidget("Ready")
	statusBar.SetFontSize(12)
	statusBar.SetFontColor("#666666")
	statusBar.SetHeight(20)
	rootFrame.AttachWidget(statusBar)

	bs.windowBase.SetRootFrame(rootFrame)

	// Record web (http/https) navigations in the history manager.
	bs.tabManager.OnNavigated(func(tab *Tab) {
		bs.recordHistory(tab)

		// Record memory snapshots on navigation for per-tab profiling.
		if tab != nil {
			bs.memoryProfiler.Record("tab:" + tab.URL)
		}

		// Record page load metrics for web navigations.
		if tab != nil && (strings.HasPrefix(tab.URL, "http://") || strings.HasPrefix(tab.URL, "https://")) {
			m := bs.pageLoadTracker.StartLoad(tab.URL)
			_ = m // Mark as started; completed when page finishes loading.
		}

		// Inject content scripts for all enabled extensions that match this URL.
		if tab != nil {
			if err := bs.injectContentScripts(tab); err != nil {
				// Content script injection failures should not block navigation.
				bs.errorLogger.Log(errors.SeverityError, "shell", "content_script", fmt.Sprintf("tab %d", tab.ID), err.Error())
			}
		}

		// Load the tab's URL into a per-tab WindowContext and render it.
		if tab != nil {
			if err := bs.loadTabContent(tab); err != nil {
				bs.errorLogger.Log(errors.SeverityError, "shell", "load", fmt.Sprintf("tab %d", tab.ID), err.Error())
			}
		}
	})

	// Clean up injected script tracking when tabs are closed.
	bs.tabManager.OnChanged(func() {
		activeIDs := make(map[int]bool)
		for _, t := range bs.tabManager.Tabs() {
			activeIDs[t.ID] = true
		}
		for id := range bs.injectedScripts {
			if !activeIDs[id] {
				delete(bs.injectedScripts, id)
			}
		}
	})

	// Open a default tab only if no tabs were restored from session.
	if bs.tabManager.Count() == 0 {
		bs.tabManager.Add("New Tab", bs.settings.Homepage, "", "", "")
	}

	// Wire global keyboard shortcuts.
	bs.windowBase.AttachKeyboardInputEventListener(bs.handleKeyboardInput)
}

// handleKeyboardInput processes global keyboard shortcuts:
//   - Ctrl+L: focus the address bar
//   - Ctrl+T: open a new tab
//   - Ctrl+W: close the active tab
//   - Ctrl+B: toggle the bookmarks sidebar
func (bs *BrowserShell) handleKeyboardInput(input mustard.KeyboardInput) {
	if input.Action != mustard.KeyboardPress {
		return
	}

	if input.Control {
		switch input.Key {
		case "l":
			// Focus address bar (select all text).
			ab := bs.addressBar
			if ab != nil && ab.input != nil {
				ab.input.SetFocused(true)
			}
		case "t":
			bs.tabManager.Add("New Tab", bs.settings.Homepage, "", "", "")
		case "w":
			bs.closeTab(bs.tabManager.ActiveTab())
		case "b":
			bs.bookmarksSidebar.Toggle()
		case "h":
			bs.historySidebar.Toggle()
		case "s":
			bs.settingsPage.Toggle()
		}
	}
}

// renderWebContent renders the active tab's document into the web content
// canvas using the bun layout engine. It is called after a successful page
// load so the shell actually displays web content instead of a blank placeholder.
func (bs *BrowserShell) renderWebContent(tab *Tab) {
	if bs.webContent == nil || tab == nil {
		return
	}

	wc := bs.tabWindowContext(tab)
	if wc == nil || wc.ActiveDocument == nil {
		return
	}

	ctx := bs.webContent.GetContext()
	if ctx == nil {
		return
	}

	_, _, width, height := bs.webContent.ComputedBox().GetCoords()
	if width <= 0 || height <= 0 {
		return
	}

	ctx.SetRGB(1, 1, 1)
	ctx.Clear()

	// Serialize the DOM walk against page JS: animation-frame and event
	// callbacks mutate Children/RenderBox/Style from other goroutines while
	// holding jsRuntimeMu, so the render must take the same lock (audit R-7).
	renderErr := error(nil)
	wc.WithJSRLock(func() {
		renderErr = bun.RenderDocument(ctx, wc.ActiveDocument, false)
	})
	if renderErr != nil {
		bs.errorLogger.Log(errors.SeverityError, "shell", "render", fmt.Sprintf("tab %d", tab.ID), renderErr.Error())
		return
	}

	bs.webContent.Repaint()
}

// tabWindowContext returns the hotdog WindowContext for the given tab, or nil.
// Each tab owns a WindowContext created when the tab navigates.
func (bs *BrowserShell) tabWindowContext(tab *Tab) *hotdog.WindowContext {
	if tab == nil {
		return nil
	}
	bs.tabWCsMu.Lock()
	defer bs.tabWCsMu.Unlock()
	return bs.tabWindowContexts[tab.ID]
}

// loadTabContent loads the tab's URL into a per-tab WindowContext and renders
// it into the web content canvas.
func (bs *BrowserShell) loadTabContent(tab *Tab) error {
	if tab == nil {
		return fmt.Errorf("tab is nil")
	}

	bs.tabWCsMu.Lock()
	wc, ok := bs.tabWindowContexts[tab.ID]
	if !ok || wc == nil {
		wc = hotdog.NewWindowContext(bs.settings, bs.buildInfo, nil)
		bs.tabWindowContexts[tab.ID] = wc
	}
	bs.tabWCsMu.Unlock()

	if err := wc.LoadURL(tab.URL); err != nil {
		return fmt.Errorf("load URL %s: %w", tab.URL, err)
	}

	// Route the loaded HTML through ketchup so HTMLRoot and StyleSheets are
	// populated, enabling CSS styling and querySelector/querySelectorAll.
	if wc.ActiveDocument != nil && wc.ActiveDocument.HTMLRoot == nil {
		if err := bs.applyKetchupParsing(wc, tab.URL); err != nil {
			bs.errorLogger.Log(errors.SeverityError, "shell", "ketchup", fmt.Sprintf("tab %d", tab.ID), err.Error())
		}
	}

	bs.renderWebContent(tab)
	return nil
}

// closeTab removes the tab from the tab manager, which fires the OnClosed
// callback that destroys the tab's WindowContext and drops it from the shell's
// tab context map, releasing all engine state (goja VM, DOM, cookie jar, HTTP
// client, rAF timer). Routing through TabManager.Close keeps the tab strip and
// the engine state consistent: the old version destroyed the context but left
// the tab alive in the manager (audit R-5).
func (bs *BrowserShell) closeTab(tab *Tab) {
	if tab == nil {
		return
	}
	bs.tabManager.Close(tab.ID)
}

// applyKetchupParsing re-parses the active document's HTML through ketchup to
// populate HTMLRoot and StyleSheets, enabling CSS and selector queries.
func (bs *BrowserShell) applyKetchupParsing(wc *hotdog.WindowContext, baseURL string) error {
	if wc.ActiveDocument == nil {
		return fmt.Errorf("no active document")
	}

	// LoadHTML stores the fetched source in RawDocument, so the ketchup
	// re-parse sees the original markup (including <style> blocks). ketchup
	// handles an empty string safely, so no fallback serialization is needed.
	html := wc.ActiveDocument.RawDocument

	ketchupDoc := ketchup.ParseHTMLDocument(html, wc)
	if ketchupDoc == nil {
		return fmt.Errorf("ketchup parsing returned nil document")
	}

	// Preserve the loaded URL and title on the ketchup document. Its DOM nodes
	// carry a Document back-pointer to ketchupDoc, so they need correct origin
	// and title metadata for querySelector and document.title.
	ketchupDoc.URL = wc.ActiveDocument.URL
	ketchupDoc.Title = wc.ActiveDocument.Title

	// Swap the fully-populated ketchup DOM into the active document in place.
	// Keeping the *Document pointer stable preserves the JS runtime's document
	// binding and the wc.Documents history, while replacing the x/net fragment
	// DOM (which lacks RenderBox/Style) with one bun can lay out and paint.
	wc.ActiveDocument.DOM = ketchupDoc.DOM
	wc.ActiveDocument.HTMLRoot = ketchupDoc.HTMLRoot
	wc.ActiveDocument.StyleSheets = ketchupDoc.StyleSheets
	wc.ActiveDocument.RawDocument = ketchupDoc.RawDocument

	return nil
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
	bs.app.Run(bs.frameTick)
}

// frameTick runs once per animation frame on the game goroutine. JS DOM
// mutations call RequestReflow, which queues invalidated roots, but nothing
// drained that queue or repainted, so a page changed by script stayed stale
// until the next navigation (audit C-16, probe P13). When the active tab has
// pending reflows, re-render it. The callback runs on the same goroutine as
// the mustard render pass, so it touches shell UI state directly; the DOM walk
// inside renderWebContent still takes jsRuntimeMu to serialize against rAF
// callbacks running on timer goroutines.
func (bs *BrowserShell) frameTick() {
	tab := bs.tabManager.ActiveTab()
	if tab == nil {
		return
	}
	wc := bs.tabWindowContext(tab)
	if wc == nil {
		return
	}
	if roots := wc.DrainReflowQueue(); len(roots) > 0 {
		bs.renderWebContent(tab)
	}
}

// Close destroys the browser shell's window and app, saving the session
// first if a session path is configured.
func (bs *BrowserShell) Close() {
	_ = bs.sessionManager.Save(bs)

	if bs.extManager != nil {
		bs.extManager.Stop()
	}

	if bs.windowBase != nil {
		bs.windowBase.Close()
	}

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

// injectContentScripts injects matching content scripts into the given tab
// for all enabled extensions. Scripts are executed in isolated sandboxes.
// Each script is injected only once per tab to prevent duplicate execution.
func (bs *BrowserShell) injectContentScripts(tab *Tab) error {
	if bs.injectedScripts[tab.ID] == nil {
		bs.injectedScripts[tab.ID] = make(map[string]bool)
	}

	exts := bs.extManager.GetEnabled()
	for _, ext := range exts {
		scripts := bs.extManager.ContentLoader().MatchingForExtension(ext.ID, tab.URL)
		for _, script := range scripts {
			// Skip if this script was already injected into this tab.
			if bs.injectedScripts[tab.ID][script.ID] {
				continue
			}

			// Check CSP if the manifest defines a content security policy.
			if ext.Manifest.ContentSecurityPolicy != "" {
				csp := extensions.NewContentScriptCSP(ext.Manifest.ContentSecurityPolicy)
				for _, js := range script.JS {
					jsURL := ext.ResolvePath(js)
					if err := csp.CheckScript(jsURL); err != nil {
						bs.errorLogger.Log(errors.SeverityError, "shell", "csp", fmt.Sprintf("script %s", js), err.Error())
						continue
					}
				}
			}

			sandbox := bs.extManager.SandboxForTab(ext.ID, tab.ID)
			// Load and execute each JS file in the sandbox.
			for _, js := range script.JS {
				if err := sandbox.LoadScript(ext.Path, js); err != nil {
					return fmt.Errorf("load content script %s: %w", js, err)
				}
			}

			bs.injectedScripts[tab.ID][script.ID] = true
		}
	}
	return nil
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
	info := &extensions.ContainerInfo{
		ID:    c.ID,
		Name:  c.Name,
		Color: c.Color,
		Icon:  c.Icon,
	}
	// Fire onCreated event to all extensions.
	for _, ext := range bs.extManager.GetAll() {
		if api, ok := bs.extManager.ChromeAPIFor(ext.ID); ok {
			api.FireContainerCreated(info)
		}
	}
	return info, nil
}

// UpdateContainer implements ContainerOperations for chrome.contextualIdentities.
func (bs *BrowserShell) UpdateContainer(id, name, color string) (*extensions.ContainerInfo, error) {
	c, err := bs.containerManager.Update(id, name, color)
	if err != nil {
		return nil, err
	}
	info := &extensions.ContainerInfo{
		ID:    c.ID,
		Name:  c.Name,
		Color: c.Color,
		Icon:  c.Icon,
	}
	// Fire onUpdated event to all extensions.
	for _, ext := range bs.extManager.GetAll() {
		if api, ok := bs.extManager.ChromeAPIFor(ext.ID); ok {
			api.FireContainerUpdated(info)
		}
	}
	return info, nil
}

// RemoveContainer implements ContainerOperations for chrome.contextualIdentities.
func (bs *BrowserShell) RemoveContainer(id string) error {
	if !bs.containerManager.Remove(id) {
		return fmt.Errorf("container %q not found", id)
	}
	// Fire onRemoved event to all extensions.
	for _, ext := range bs.extManager.GetAll() {
		if api, ok := bs.extManager.ChromeAPIFor(ext.ID); ok {
			api.FireContainerRemoved(id)
		}
	}
	return nil
}

// Settings returns the shell's settings.
func (bs *BrowserShell) Settings() *hotdog.Settings {
	return bs.settings
}

// BrowserSettings returns the shell's browser preferences settings.
func (bs *BrowserShell) BrowserSettings() *BrowserSettings {
	return bs.browserSettings
}

// Window returns the shell's mustard window base.
func (bs *BrowserShell) Window() *mustard.WindowBase {
	return bs.windowBase
}
