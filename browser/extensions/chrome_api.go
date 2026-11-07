package extensions

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"

	goja "github.com/dop251/goja"
)

// TabInfo describes a browser tab as exposed to extension APIs.
type TabInfo struct {
	ID             int    `json:"id"`
	Title          string `json:"title"`
	URL            string `json:"url"`
	Favicon        string `json:"favicon,omitempty"`
	ContainerID    string `json:"containerId,omitempty"`
	ContainerName  string `json:"containerName,omitempty"`
	ContainerColor string `json:"containerColor,omitempty"`
	IsActive       bool   `json:"active"`
	WindowID       int    `json:"windowId"`
}

// TabUpdate describes fields that can be updated on a tab.
type TabUpdate struct {
	URL            string `json:"url,omitempty"`
	Title          string `json:"title,omitempty"`
	Active         *bool  `json:"active,omitempty"`
	Pinned         *bool  `json:"pinned,omitempty"`
	Muted          *bool  `json:"muted,omitempty"`
	ContainerID    string `json:"containerId,omitempty"`
	ContainerName  string `json:"containerName,omitempty"`
	ContainerColor string `json:"containerColor,omitempty"`
}

// TabQuery describes filter criteria for chrome.tabs.query.
type TabQuery struct {
	Active       *bool   `json:"active,omitempty"`
	CurrentWindow *bool  `json:"currentWindow,omitempty"`
	URL          string  `json:"url,omitempty"`
	Title        string  `json:"title,omitempty"`
	Status       string  `json:"status,omitempty"`
	WindowID     *int    `json:"windowId,omitempty"`
}

// TabCreate describes fields for creating a new tab.
type TabCreate struct {
	URL            string `json:"url,omitempty"`
	Title          string `json:"title,omitempty"`
	Active         *bool  `json:"active,omitempty"`
	ContainerID    string `json:"containerId,omitempty"`
	ContainerName  string `json:"containerName,omitempty"`
	ContainerColor string `json:"containerColor,omitempty"`
}

// WindowInfo describes a browser window as exposed to extension APIs.
type WindowInfo struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	IsFocused bool `json:"focused"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

// TabOperations is implemented by the browser shell to provide tab
// operations to extension APIs without creating an import cycle.
type TabOperations interface {
	QueryTabs() []TabInfo
	QueryTabsFiltered(q TabQuery) []TabInfo
	GetTab(tabID int) (*TabInfo, error)
	UpdateTab(tabID int, updates TabUpdate) (*TabInfo, error)
	CreateTab(tab TabCreate) (*TabInfo, error)
	RemoveTab(tabID int) error
	ActivateTab(tabID int) (*TabInfo, error)
}

// WindowOperations is implemented by the browser shell to provide window
// operations to extension APIs without creating an import cycle.
type WindowOperations interface {
	GetCurrent() (*WindowInfo, error)
	GetAll() ([]WindowInfo, error)
	Create(win WindowCreate) (*WindowInfo, error)
	Remove(windowID int) error
}

// WindowCreate describes fields for creating a new window.
type WindowCreate struct {
	Title   string `json:"title,omitempty"`
	URL     string `json:"url,omitempty"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
}

// ContainerInfo describes a container as exposed to extension APIs.
type ContainerInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Icon  string `json:"icon,omitempty"`
}

// ContainerOperations is implemented by the browser shell to provide
// container management operations to extension APIs without creating an
// import cycle.
type ContainerOperations interface {
	QueryContainers() []ContainerInfo
	GetContainer(id string) (*ContainerInfo, error)
	CreateContainer(name, color string) (*ContainerInfo, error)
	UpdateContainer(id, name, color string) (*ContainerInfo, error)
	RemoveContainer(id string) error
}

// Cookie describes a browser cookie as exposed to extension APIs.
type Cookie struct {
	Name       string `json:"name"`
	Value      string `json:"value"`
	Domain     string `json:"domain"`
	Path       string `json:"path"`
	Secure     bool   `json:"secure"`
	HttpOnly   bool   `json:"httpOnly"`
	SameSite   string `json:"sameSite"` // "no_restriction", "lax", "strict"
	Session    bool   `json:"session"`
	Expiration string `json:"expiration,omitempty"` // ISO 8601, omitted for session cookies
	StoreID    string `json:"storeId,omitempty"`    // container ID
}

// CookieOperations is implemented by the browser shell to provide cookie
// operations to extension APIs without creating an import cycle.
type CookieOperations interface {
	GetCookies(storeID string, url string) ([]Cookie, error)
	SetCookie(storeID string, cookie Cookie) error
	RemoveCookie(storeID string, cookie Cookie) error
	GetAllCookies(storeID string) ([]Cookie, error)
	ClearCookies(storeID string) error
	GetAllCookieStores() ([]CookieStore, error)
}

// CookieStore describes a cookie store as exposed to extension APIs.
type CookieStore struct {
	ID       string `json:"id"`
	Incognito bool  `json:"incognito"`
}

// ScriptingOperations is implemented by the browser shell to provide
// dynamic script injection operations to extension APIs.
type ScriptingOperations interface {
	ExecuteScript(extID string, world ScriptingWorld, script string) (interface{}, error)
	ExecuteScriptFile(extID, extPath, worldStr, jsPath string) (interface{}, error)
	InjectCSS(extID, css string) error
	RemoveCSS(extID string) error
	ExecuteScriptTab(extID string, tabID int, world ScriptingWorld, script string) (interface{}, error)
	RegisterContentScripts(extID string, scripts []ContentScript) error
	UnregisterContentScripts(extID string, scriptIDs []string) error
}

// ContextMenuItem describes a context menu item as exposed to extension APIs.
type ContextMenuItem struct {
	ID         string `json:"id"`
	Title      string `json:"title,omitempty"`
	Contexts   []string `json:"contexts,omitempty"`
	Checked    bool   `json:"checked,omitempty"`
	Enabled    bool   `json:"enabled,omitempty"`
	Visible    bool   `json:"visible,omitempty"`
	ParentID   string `json:"parentId,omitempty"`
	DocumentURLPatterns []string `json:"documentUrlPatterns,omitempty"`
	TargetURLPatterns   []string `json:"targetUrlPatterns,omitempty"`
}

// ContextMenuOperations is implemented by the browser shell to provide
// context menu operations to extension APIs.
type ContextMenuOperations interface {
	CreateContextMenu(item ContextMenuItem) (string, error)
	UpdateContextMenu(id string, item ContextMenuItem) error
	RemoveContextMenu(id string) error
	RemoveAllContextMenus() error
}

// Permission describes a permission as exposed to extension APIs.
type Permission struct {
	Name string `json:"name"`
}

// PermissionOperations is implemented by the browser shell to provide
// permission request and grant operations to extension APIs.
type PermissionOperations interface {
	RequestPermissions(extID string, permissions []string) ([]string, error)
	ContainsPermissions(extID string, permissions []string) ([]string, error)
	GetAllPermissions(extID string) ([]string, error)
	RemovePermissions(extID string, permissions []string) ([]string, error)
}

// Alarm describes an alarm as exposed to extension APIs.
type Alarm struct {
	Name      string `json:"name"`
	Scheduled float64 `json:"scheduled,omitempty"` // epoch milliseconds
	PeriodInMinutes float64 `json:"periodInMinutes,omitempty"`
	When      float64 `json:"when,omitempty"`
}

// AlarmOperations is implemented by the browser shell to provide
// alarm scheduling operations to extension APIs.
type AlarmOperations interface {
	CreateAlarm(extID string, name string, alarm Alarm) error
	GetAlarm(extID string, name string) (*Alarm, error)
	GetAllAlarms(extID string) ([]Alarm, error)
	ClearAlarm(extID string, name string) error
	ClearAllAlarms(extID string) error
}

// I18nOperations is implemented by the browser shell to provide
// internationalization operations to extension APIs.
type I18nOperations interface {
	GetMessage(extID string, messageName string, substitutions ...string) (string, error)
	GetAcceptLanguages(extID string) ([]string, error)
}

// WebRequestDetails describes an HTTP request/response observed by the
// WebRequest API.
type WebRequestDetails struct {
	RequestID      string            `json:"requestId"`
	URL            string            `json:"url"`
	Method         string            `json:"method"`
	Headers        map[string]string `json:"headers"`
	StatusCode     int               `json:"statusCode,omitempty"`
	StatusLine     string            `json:"statusLine,omitempty"`
	ResponseHeaders map[string]string `json:"responseHeaders,omitempty"`
	Initiator      string            `json:"initiator,omitempty"`
	TabID          int               `json:"tabId,omitempty"`
	Timestamp      float64           `json:"timestamp"`
	URLPatterns    []string          `json:"urlPatterns,omitempty"`
}

// WebRequestEvent is a filterable event subscription for WebRequest API.
type WebRequestEvent struct {
	URLPatterns []string `json:"urlPatterns,omitempty"`
	Types       []string `json:"types,omitempty"`
}

// WebRequestOperations is implemented by the browser shell to provide
// network request interception operations to extension APIs.
type WebRequestOperations interface {
	AddListener(extID string, event string, details WebRequestDetails) error
	RemoveListener(extID string, event string) error
}

// Notification describes a desktop notification as exposed to extension APIs.
type Notification struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	IconURL   string `json:"iconUrl,omitempty"`
	Context   string `json:"context,omitempty"`
	Priority  int    `json:"priority,omitempty"`
	Timestamp float64 `json:"timestamp,omitempty"`
}

// NotificationOptions describes options for creating a notification.
type NotificationOptions struct {
	Title     string `json:"title,omitempty"`
	Message   string `json:"message,omitempty"`
	IconURL   string `json:"iconUrl,omitempty"`
	Context   string `json:"context,omitempty"`
	Priority  int    `json:"priority,omitempty"`
	Timestamp float64 `json:"timestamp,omitempty"`
}

// NotificationOperations is implemented by the browser shell to provide
// desktop notification operations to extension APIs.
type NotificationOperations interface {
	CreateNotification(extID string, id string, options NotificationOptions) (string, error)
	ClearNotification(extID string, id string) (bool, error)
	GetAllNotifications(extID string) ([]Notification, error)
}

// Bookmark describes a bookmark as exposed to extension APIs.
type Bookmark struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	ParentID string `json:"parentId,omitempty"`
	Index   int    `json:"index,omitempty"`
}

// BookmarkOperations is implemented by the browser shell to provide
// bookmark management operations to extension APIs.
type BookmarkOperations interface {
	CreateBookmark(bookmark Bookmark) (*Bookmark, error)
	GetBookmark(id string) (*Bookmark, error)
	GetChildren(parentID string) ([]Bookmark, error)
	MoveBookmark(id, parentID string, index int) (*Bookmark, error)
	RemoveBookmark(id string) error
	UpdateBookmark(id string, updates Bookmark) (*Bookmark, error)
}

// HistoryEntry describes a history entry as exposed to extension APIs.
type HistoryEntry struct {
	ID        int     `json:"id"`
	URL       string  `json:"url"`
	Title     string  `json:"title"`
	VisitTime float64 `json:"visitTime"`
	Typed     bool    `json:"typed"`
}

// HistoryOperations is implemented by the browser shell to provide
// history operations to extension APIs.
type HistoryOperations interface {
	AddUrl(url, title string) error
	DeleteUrl(url string) error
	Search(query string, maxResults int) ([]HistoryEntry, error)
	GetAllHistory() ([]HistoryEntry, error)
}

// ActionBadge describes badge text for chrome.action.
type ActionBadge struct {
	Text    string `json:"text,omitempty"`
	Color   string `json:"color,omitempty"`
	TabID   int    `json:"tabId,omitempty"`
}

// ActionOperations is implemented by the browser shell to provide
// chrome.action operations to extension APIs.
type ActionOperations interface {
	SetBadgeText(extID string, text string, tabID int) error
	SetBadgeColor(extID string, color string, tabID int) error
	SetIcon(extID string, path string, tabID int) error
	SetPopup(extID string, popup string, tabID int) error
	GetPopup(extID string, tabID int) (string, error)
}

// DownloadItem describes a download as exposed to extension APIs.
type DownloadItem struct {
	ID           int     `json:"id"`
	URL          string  `json:"url"`
	Filename     string  `json:"filename,omitempty"`
	State        string  `json:"state"` // "in_progress", "complete", "interrupted"
	TotalBytes   int64   `json:"totalBytes,omitempty"`
	BytesReceived int64  `json:"bytesReceived,omitempty"`
	Mime         string  `json:"mime,omitempty"`
	StartTime    float64 `json:"startTime,omitempty"`
	EndTime      float64 `json:"endTime,omitempty"`
	Paused       bool    `json:"paused,omitempty"`
	Error        string  `json:"error,omitempty"`
}

// DownloadOptions describes options for starting a download.
type DownloadOptions struct {
	URL      string `json:"url"`
	Filename string `json:"filename,omitempty"`
	SaveAs   bool   `json:"saveAs,omitempty"`
}

// DownloadOperations is implemented by the browser shell to provide
// download management operations to extension APIs.
type DownloadOperations interface {
	Download(extID string, options DownloadOptions) (int, error)
	Cancel(downloadID int) error
	Pause(downloadID int) error
	Resume(downloadID int) error
	Get(downloadID int) (*DownloadItem, error)
	GetAll() ([]DownloadItem, error)
	Erase(downloadID int) error
	Search(query DownloadQuery) ([]DownloadItem, error)
}

// DownloadQuery describes filter criteria for chrome.downloads.search.
type DownloadQuery struct {
	ID        *int    `json:"id,omitempty"`
	URL       string  `json:"url,omitempty"`
	Filename  string  `json:"filename,omitempty"`
	State     string  `json:"state,omitempty"`
	Paused    *bool   `json:"paused,omitempty"`
	Error     string  `json:"error,omitempty"`
}

// PlatformInfo describes the browser platform as reported by chrome.runtime.
type PlatformInfo struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	Platform     string `json:"platform"`
	BrowserVersion string `json:"browser_version"`
}

// ChromeAPI binds WebExtensions APIs (chrome.runtime, chrome.storage,
// chrome.tabs, chrome.windows, chrome.contextualIdentities, chrome.scripting,
// chrome.contextMenus, chrome.permissions, chrome.alarms, chrome.i18n) to an
// extension's background Goja VM.
type ChromeAPI struct {
	mu            sync.Mutex
	ext           *Extension
	vm            *goja.Runtime
	// vmMu serializes every access to the background goja runtime. goja is not
	// goroutine-safe, and Fire* methods run on timer/HTTP goroutines while the
	// extension's own code runs elsewhere. It is reentrant because a listener
	// invoked under this lock can call chrome.tabs.create (and similar), which
	// fires another event on the same VM from the same goroutine.
	vmMu          *reentrantMutex
	router        *MessageRouter
	tabOps        TabOperations
	winOps        WindowOperations
	containerOps  ContainerOperations
	cookieOps     CookieOperations
	scriptingOps  ScriptingOperations
	contextMenuOps ContextMenuOperations
	permissionOps PermissionOperations
	alarmOps      AlarmOperations
	i18nOps       I18nOperations
	webRequestOps WebRequestOperations
	notificationOps NotificationOperations
	downloadOps DownloadOperations
	bookmarkOps BookmarkOperations
	historyOps HistoryOperations
	actionOps ActionOperations
	platform      PlatformInfo

	// tabEventListeners holds listeners for tab events keyed by event name.
	tabEventListeners map[string][]goja.Callable
	// runtimeEventListeners holds listeners for runtime events keyed by event name.
	runtimeEventListeners map[string][]goja.Callable
	// webRequestEventListeners holds listeners for webRequest events keyed by event name.
	webRequestEventListeners map[string][]goja.Callable
	// downloadEventListeners holds listeners for download events keyed by event name.
	downloadEventListeners map[string][]goja.Callable
	// cookieEventListeners holds listeners for cookies.onChanged events.
	cookieEventListeners []goja.Callable
	// alarmEventListeners holds listeners for alarms.onAlarm events keyed by alarm name.
	alarmEventListeners map[string][]goja.Callable
	// containerEventListeners holds listeners for contextualIdentities events keyed by event name.
	containerEventListeners map[string][]goja.Callable
	// contextMenuEventListeners holds listeners for contextMenus.onClicked events.
	contextMenuEventListeners []goja.Callable
	// permissionEventListeners holds listeners for permissions.onAdded/onRemoved events.
	permissionAddedListeners   []goja.Callable
	permissionRemovedListeners []goja.Callable
	// notificationEventListeners holds listeners for notifications.onClicked/onClosed events.
	notificationClickedListeners   []goja.Callable
	notificationClosedListeners    []goja.Callable
}

// NewChromeAPI creates a ChromeAPI binding for the given extension and VM.
func NewChromeAPI(ext *Extension, vm *goja.Runtime, router *MessageRouter) *ChromeAPI {
	return &ChromeAPI{
		ext:    ext,
		vm:     vm,
		vmMu:   newReentrantMutex(),
		router: router,
		platform: PlatformInfo{
			OS:           runtime.GOOS,
			Arch:           runtime.GOARCH,
			Platform:     runtime.GOOS + "/" + runtime.GOARCH,
			BrowserVersion: "thdwb dev",
		},
		tabEventListeners:    make(map[string][]goja.Callable),
		runtimeEventListeners: make(map[string][]goja.Callable),
		webRequestEventListeners: make(map[string][]goja.Callable),
		downloadEventListeners: make(map[string][]goja.Callable),
		cookieEventListeners:   make([]goja.Callable, 0),
		alarmEventListeners:    make(map[string][]goja.Callable),
		containerEventListeners: make(map[string][]goja.Callable),
		contextMenuEventListeners: make([]goja.Callable, 0),
		permissionAddedListeners:   make([]goja.Callable, 0),
		permissionRemovedListeners: make([]goja.Callable, 0),
		notificationClickedListeners:   make([]goja.Callable, 0),
		notificationClosedListeners:    make([]goja.Callable, 0),
	}
}

// SetTabOperations sets the tab operations provider (called by the shell).
func (c *ChromeAPI) SetTabOperations(ops TabOperations) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tabOps = ops
}

// SetWindowOperations sets the window operations provider (called by the shell).
func (c *ChromeAPI) SetWindowOperations(ops WindowOperations) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.winOps = ops
}

// SetContainerOperations sets the container operations provider (called by the shell).
func (c *ChromeAPI) SetContainerOperations(ops ContainerOperations) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.containerOps = ops
}

// SetCookieOperations sets the cookie operations provider (called by the shell).
func (c *ChromeAPI) SetCookieOperations(ops CookieOperations) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cookieOps = ops
}

// SetScriptingOperations sets the scripting operations provider (called by the shell).
func (c *ChromeAPI) SetScriptingOperations(ops ScriptingOperations) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scriptingOps = ops
}

// SetContextMenuOperations sets the context menu operations provider (called by the shell).
func (c *ChromeAPI) SetContextMenuOperations(ops ContextMenuOperations) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.contextMenuOps = ops
}

// SetPermissionOperations sets the permission operations provider (called by the shell).
func (c *ChromeAPI) SetPermissionOperations(ops PermissionOperations) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.permissionOps = ops
}

// SetAlarmOperations sets the alarm operations provider (called by the shell).
func (c *ChromeAPI) SetAlarmOperations(ops AlarmOperations) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.alarmOps = ops
}

// SetI18nOperations sets the i18n operations provider (called by the shell).
func (c *ChromeAPI) SetI18nOperations(ops I18nOperations) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.i18nOps = ops
}

// SetWebRequestOperations sets the web request operations provider (called by the shell).
func (c *ChromeAPI) SetWebRequestOperations(ops WebRequestOperations) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.webRequestOps = ops
}

// SetNotificationOperations sets the notification operations provider (called by the shell).
func (c *ChromeAPI) SetNotificationOperations(ops NotificationOperations) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.notificationOps = ops
}

// SetDownloadOperations sets the download operations provider (called by the shell).
func (c *ChromeAPI) SetDownloadOperations(ops DownloadOperations) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.downloadOps = ops
}

// SetBookmarkOperations sets the bookmark operations provider (called by the shell).
func (c *ChromeAPI) SetBookmarkOperations(ops BookmarkOperations) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bookmarkOps = ops
}

// SetHistoryOperations sets the history operations provider (called by the shell).
func (c *ChromeAPI) SetHistoryOperations(ops HistoryOperations) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.historyOps = ops
}

// SetActionOperations sets the action operations provider (called by the shell).
func (c *ChromeAPI) SetActionOperations(ops ActionOperations) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.actionOps = ops
}

// Bind installs all chrome.* APIs into the extension's background VM.
func (c *ChromeAPI) Bind() error {
	c.mu.Lock()
	ext := c.ext
	vm := c.vm
	router := c.router
	tabOps := c.tabOps
	winOps := c.winOps
	containerOps := c.containerOps
	cookieOps := c.cookieOps
	scriptingOps := c.scriptingOps
	contextMenuOps := c.contextMenuOps
	permissionOps := c.permissionOps
	alarmOps := c.alarmOps
	i18nOps := c.i18nOps
	webRequestOps := c.webRequestOps
	notificationOps := c.notificationOps
	downloadOps := c.downloadOps
	bookmarkOps := c.bookmarkOps
	historyOps := c.historyOps
	actionOps := c.actionOps
	c.mu.Unlock()

	if vm == nil {
		return fmt.Errorf("no VM bound to ChromeAPI")
	}

	// Bind chrome.runtime.
	if err := c.bindRuntime(vm, ext, router); err != nil {
		return fmt.Errorf("bind chrome.runtime: %w", err)
	}

	// Bind chrome.storage.
	if err := c.bindStorage(vm, ext); err != nil {
		return fmt.Errorf("bind chrome.storage: %w", err)
	}

	// Bind chrome.tabs (requires tabOps).
	if tabOps != nil {
		if err := c.bindTabs(vm, ext, router, tabOps); err != nil {
			return fmt.Errorf("bind chrome.tabs: %w", err)
		}
	}

	// Bind chrome.windows (requires winOps).
	if winOps != nil {
		if err := c.bindWindows(vm, ext, winOps); err != nil {
			return fmt.Errorf("bind chrome.windows: %w", err)
		}
	}

	// Bind chrome.contextualIdentities (requires containerOps).
	if containerOps != nil {
		if err := c.bindContextualIdentities(vm, ext, containerOps); err != nil {
			return fmt.Errorf("bind chrome.contextualIdentities: %w", err)
		}
	}

	// Bind chrome.cookies (requires cookieOps).
	if cookieOps != nil {
		if err := c.bindCookies(vm, ext, cookieOps); err != nil {
			return fmt.Errorf("bind chrome.cookies: %w", err)
		}
	}

	// Bind chrome.scripting (requires scriptingOps).
	if scriptingOps != nil {
		if err := c.bindScripting(vm, ext, scriptingOps); err != nil {
			return fmt.Errorf("bind chrome.scripting: %w", err)
		}
	}

	// Bind chrome.contextMenus (requires contextMenuOps).
	if contextMenuOps != nil {
		if err := c.bindContextMenus(vm, ext, contextMenuOps); err != nil {
			return fmt.Errorf("bind chrome.contextMenus: %w", err)
		}
	}

	// Bind chrome.permissions (requires permissionOps).
	if permissionOps != nil {
		if err := c.bindPermissions(vm, ext, permissionOps); err != nil {
			return fmt.Errorf("bind chrome.permissions: %w", err)
		}
	}

	// Bind chrome.alarms (requires alarmOps).
	if alarmOps != nil {
		if err := c.bindAlarms(vm, ext, alarmOps); err != nil {
			return fmt.Errorf("bind chrome.alarms: %w", err)
		}
	}

	// Bind chrome.i18n (requires i18nOps).
	if i18nOps != nil {
		if err := c.bindI18n(vm, ext, i18nOps); err != nil {
			return fmt.Errorf("bind chrome.i18n: %w", err)
		}
	}

	// Bind chrome.webRequest (requires webRequestOps).
	if webRequestOps != nil {
		if err := c.bindWebRequest(vm, ext, webRequestOps); err != nil {
			return fmt.Errorf("bind chrome.webRequest: %w", err)
		}
	}

	// Bind chrome.notifications (requires notificationOps).
	if notificationOps != nil {
		if err := c.bindNotifications(vm, ext, notificationOps); err != nil {
			return fmt.Errorf("bind chrome.notifications: %w", err)
		}
	}

	// Bind chrome.downloads (requires downloadOps).
	if downloadOps != nil {
		if err := c.bindDownloads(vm, ext, downloadOps); err != nil {
			return fmt.Errorf("bind chrome.downloads: %w", err)
		}
	}

	// Bind chrome.bookmarks (requires bookmarkOps).
	if bookmarkOps != nil {
		if err := c.bindBookmarks(vm, ext, bookmarkOps); err != nil {
			return fmt.Errorf("bind chrome.bookmarks: %w", err)
		}
	}

	// Bind chrome.history (requires historyOps).
	if historyOps != nil {
		if err := c.bindHistory(vm, ext, historyOps); err != nil {
			return fmt.Errorf("bind chrome.history: %w", err)
		}
	}

	// Bind chrome.action (requires actionOps).
	if actionOps != nil {
		if err := c.bindAction(vm, ext, actionOps); err != nil {
			return fmt.Errorf("bind chrome.action: %w", err)
		}
	}

	return nil
}

// bindRuntime installs chrome.runtime into the VM.
func (c *ChromeAPI) bindRuntime(vm *goja.Runtime, ext *Extension, router *MessageRouter) error {
	chrome := vm.Get("chrome")
	var chromeObj *goja.Object
	if chrome == nil || chrome.ToObject(vm) == nil {
		chromeObj = vm.NewObject()
		_ = vm.Set("chrome", chromeObj)
	} else {
		chromeObj = chrome.ToObject(vm)
	}

	runtimeObj := vm.NewObject()

	// chrome.runtime.id
	_ = runtimeObj.Set("id", ext.ID)

	// chrome.runtime.getManifest
	_ = runtimeObj.Set("getManifest", func(call goja.FunctionCall) goja.Value {
		manifest := map[string]interface{}{
			"manifest_version": ext.Manifest.ManifestVersion,
			"name":             ext.Manifest.Name,
			"version":          ext.Manifest.Version,
		}
		if ext.Manifest.Description != "" {
			manifest["description"] = ext.Manifest.Description
		}
		if ext.Manifest.Permissions != nil {
			manifest["permissions"] = ext.Manifest.Permissions
		}
		if ext.Manifest.Icons != nil {
			manifest["icons"] = ext.Manifest.Icons
		}
		return vm.ToValue(manifest)
	})

	// chrome.runtime.getURL
	_ = runtimeObj.Set("getURL", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return vm.ToValue("")
		}
		path := call.Argument(0).String()
		return vm.ToValue(ext.ResolvePath(path))
	})

	// chrome.runtime.getPlatformInfo
	_ = runtimeObj.Set("getPlatformInfo", func(call goja.FunctionCall) goja.Value {
		c.mu.Lock()
		platform := c.platform
		c.mu.Unlock()
		return vm.ToValue(map[string]interface{}{
			"os":             platform.OS,
			"arch":           platform.Arch,
			"platform":       platform.Platform,
			"browser_version": platform.BrowserVersion,
		})
	})

	// chrome.runtime.sendMessage
	_ = runtimeObj.Set("sendMessage", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("sendMessage requires a message argument")))
		}
		msg := Message{
			Sender:  ext.ID,
			Type:    "background",
			Payload: call.Argument(0).Export(),
		}
		if err := router.Route(msg); err != nil {
			panic(vm.NewGoError(fmt.Errorf("sendMessage: %w", err)))
		}
		// Deliver to content script sandboxes for this extension.
		if err := router.DeliverToSandbox(ext.ID, msg); err != nil {
			// No sandbox registered; message is still routed to background handlers.
		}
		return goja.Undefined()
	})

	// chrome.runtime.onMessage (event registration)
	onMessageObj := vm.NewObject()
	_ = onMessageObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("addListener requires a callback")))
		}
		cb, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			panic(vm.NewGoError(fmt.Errorf("addListener callback must be a function")))
		}
		router.RegisterHandler(ext.ID, func(msg Message) error {
			_, err := cb(vm.ToValue(msg.Payload))
			return err
		})
		return goja.Undefined()
	})
	_ = runtimeObj.Set("onMessage", onMessageObj)

	// chrome.runtime.onInstalled event
	onInstalledObj := vm.NewObject()
	_ = onInstalledObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.runtimeEventListeners["onInstalled"] = append(c.runtimeEventListeners["onInstalled"], cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = runtimeObj.Set("onInstalled", onInstalledObj)

	// chrome.runtime.onStartup event
	onStartupObj := vm.NewObject()
	_ = onStartupObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.runtimeEventListeners["onStartup"] = append(c.runtimeEventListeners["onStartup"], cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = runtimeObj.Set("onStartup", onStartupObj)

	_ = chromeObj.Set("runtime", runtimeObj)
	return nil
}

// FireRuntimeInstalled fires the chrome.runtime.onInstalled event to all
// registered listeners for this extension.
func (c *ChromeAPI) FireRuntimeInstalled(reason string) {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.runtimeEventListeners["onInstalled"]))
	copy(listeners, c.runtimeEventListeners["onInstalled"])
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		_, _ = cb(goja.Undefined(), c.vm.ToValue(map[string]interface{}{
			"reason": reason,
		}))
	}
}

// FireRuntimeStartup fires the chrome.runtime.onStartup event to all
// registered listeners for this extension.
func (c *ChromeAPI) FireRuntimeStartup() {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.runtimeEventListeners["onStartup"]))
	copy(listeners, c.runtimeEventListeners["onStartup"])
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		_, _ = cb(goja.Undefined(), c.vm.ToValue(map[string]interface{}{
			"reason": "startup",
		}))
	}
}

// bindStorage installs chrome.storage into the VM.
func (c *ChromeAPI) bindStorage(vm *goja.Runtime, ext *Extension) error {
	chrome := vm.Get("chrome")
	var chromeObj *goja.Object
	if chrome == nil || chrome.ToObject(vm) == nil {
		chromeObj = vm.NewObject()
		_ = vm.Set("chrome", chromeObj)
	} else {
		chromeObj = chrome.ToObject(vm)
	}

	storageObj := vm.NewObject()

	// Helper to create a storage area object.
	newStorageArea := func(area StorageArea) *goja.Object {
		areaObj := vm.NewObject()
		sp := ext.Storage
		if sp == nil {
			return areaObj
		}

		_ = areaObj.Set("get", func(call goja.FunctionCall) goja.Value {
			var keys interface{}
			var callback goja.Callable

			switch len(call.Arguments) {
			case 0:
				// get() - return all.
			case 1:
				// Could be get(keys) or get(callback).
				if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
					callback = cb
				} else {
					keys = call.Argument(0).Export()
				}
			default:
				// get(keys, callback).
				keys = call.Argument(0).Export()
				callback, _ = goja.AssertFunction(call.Argument(1))
			}

			result := make(map[string]interface{})
			switch k := keys.(type) {
			case string:
				v, ok := sp.Get(area, k)
				if ok {
					result[k] = v
				}
			case []interface{}:
				for _, key := range k {
					keyStr, ok := key.(string)
					if !ok {
						continue
					}
					if v, ok := sp.Get(area, keyStr); ok {
						result[keyStr] = v
					}
				}
			case nil:
				// Get all.
				for k, v := range sp.GetAll(area) {
					result[k] = v
				}
			}

			if callback != nil {
				_, _ = callback(vm.ToValue(result))
				return goja.Undefined()
			}
			// No callback: return result synchronously for ease of use.
			return vm.ToValue(result)
		})

		_ = areaObj.Set("set", func(call goja.FunctionCall) goja.Value {
			items := call.Argument(0).Export()
			itemsMap, ok := items.(map[string]interface{})
			if !ok {
				return goja.Undefined()
			}
			for k, v := range itemsMap {
				// Store the raw value (string, number, bool) directly.
				switch val := v.(type) {
				case string:
					_ = sp.Set(area, k, val)
				default:
					data, _ := json.Marshal(val)
					_ = sp.Set(area, k, string(data))
				}
			}
			return goja.Undefined()
		})

		_ = areaObj.Set("remove", func(call goja.FunctionCall) goja.Value {
			keys := call.Argument(0).Export()
			switch k := keys.(type) {
			case string:
				_ = sp.Remove(area, k)
			case []interface{}:
				for _, key := range k {
					keyStr, ok := key.(string)
					if !ok {
						continue
					}
					_ = sp.Remove(area, keyStr)
				}
			}
			return goja.Undefined()
		})

		_ = areaObj.Set("clear", func(call goja.FunctionCall) goja.Value {
			_ = sp.Clear(area)
			return goja.Undefined()
		})

		_ = areaObj.Set("getBytesInUse", func(call goja.FunctionCall) goja.Value {
			var keys interface{}
			var callback goja.Callable

			switch len(call.Arguments) {
			case 0:
				// getBytesInUse() - return total for all keys.
			case 1:
				if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
					callback = cb
				} else {
					keys = call.Argument(0).Export()
				}
			default:
				keys = call.Argument(0).Export()
				callback, _ = goja.AssertFunction(call.Argument(1))
			}

			total := 0
			switch k := keys.(type) {
			case string:
				if v, ok := sp.Get(area, k); ok {
					total += len(k) + len(v)
				}
			case []interface{}:
				for _, key := range k {
					keyStr, ok := key.(string)
					if !ok {
						continue
					}
					if v, ok := sp.Get(area, keyStr); ok {
						total += len(keyStr) + len(v)
					}
				}
			case nil:
				for k, v := range sp.GetAll(area) {
					total += len(k) + len(v)
				}
			}

			if callback != nil {
				_, _ = callback(vm.ToValue(float64(total)))
				return goja.Undefined()
			}
			return vm.ToValue(float64(total))
		})

		return areaObj
	}

	_ = storageObj.Set("local", newStorageArea(StorageLocal))
	_ = storageObj.Set("sync", newStorageArea(StorageSync))
	_ = storageObj.Set("managed", newStorageArea(StorageManaged))

	_ = chromeObj.Set("storage", storageObj)
	return nil
}

// bindTabs installs chrome.tabs into the VM.
func (c *ChromeAPI) bindTabs(vm *goja.Runtime, ext *Extension, router *MessageRouter, tabOps TabOperations) error {
	chrome := vm.Get("chrome")
	var chromeObj *goja.Object
	if chrome == nil || chrome.ToObject(vm) == nil {
		chromeObj = vm.NewObject()
		_ = vm.Set("chrome", chromeObj)
	} else {
		chromeObj = chrome.ToObject(vm)
	}

	tabsObj := vm.NewObject()

	// chrome.tabs.query
	_ = tabsObj.Set("query", func(call goja.FunctionCall) goja.Value {
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		q := TabQuery{}
		if len(call.Arguments) > 0 {
			qObj := call.Argument(0).ToObject(vm)
			if qObj != nil {
				if v := qObj.Get("active"); v != nil {
					b := v.ToBoolean()
					q.Active = &b
				}
				if v := qObj.Get("currentWindow"); v != nil {
					b := v.ToBoolean()
					q.CurrentWindow = &b
				}
				if v := qObj.Get("url"); v != nil {
					q.URL = v.String()
				}
				if v := qObj.Get("title"); v != nil {
					q.Title = v.String()
				}
				if v := qObj.Get("status"); v != nil {
					q.Status = v.String()
				}
				if v := qObj.Get("windowId"); v != nil {
					w := int(v.ToInteger())
					q.WindowID = &w
				}
			}
		}
		tabs := tabOps.QueryTabsFiltered(q)
		result := make([]interface{}, 0, len(tabs))
		for _, t := range tabs {
			result = append(result, map[string]interface{}{
				"id":             t.ID,
				"title":          t.Title,
				"url":            t.URL,
				"favicon":        t.Favicon,
				"containerId":    t.ContainerID,
				"containerName":  t.ContainerName,
				"containerColor": t.ContainerColor,
				"active":         t.IsActive,
				"windowId":       t.WindowID,
			})
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(result))
		}
		return goja.Undefined()
	})

	// chrome.tabs.get
	_ = tabsObj.Set("get", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		tabID := int(call.Argument(0).ToInteger())
		callback, _ := goja.AssertFunction(call.Argument(1))
		t, err := tabOps.GetTab(tabID)
		if err != nil || t == nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(map[string]interface{}{
				"id":             t.ID,
				"title":          t.Title,
				"url":            t.URL,
				"favicon":        t.Favicon,
				"containerId":    t.ContainerID,
				"containerName":  t.ContainerName,
				"containerColor": t.ContainerColor,
				"active":         t.IsActive,
				"windowId":       t.WindowID,
			}))
		}
		return goja.Undefined()
	})

	// chrome.tabs.update
	_ = tabsObj.Set("update", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		tabID := int(call.Argument(0).ToInteger())
		updates := TabUpdate{}
		if len(call.Arguments) > 1 {
			updateObj := call.Argument(1).ToObject(vm)
			if updateObj != nil {
				if urlVal := updateObj.Get("url"); urlVal != nil {
					updates.URL = urlVal.String()
				}
				if titleVal := updateObj.Get("title"); titleVal != nil {
					updates.Title = titleVal.String()
				}
				if activeVal := updateObj.Get("active"); activeVal != nil {
					b := activeVal.ToBoolean()
					updates.Active = &b
				}
				if containerIDVal := updateObj.Get("containerId"); containerIDVal != nil {
					updates.ContainerID = containerIDVal.String()
				}
				if containerNameVal := updateObj.Get("containerName"); containerNameVal != nil {
					updates.ContainerName = containerNameVal.String()
				}
				if containerColorVal := updateObj.Get("containerColor"); containerColorVal != nil {
					updates.ContainerColor = containerColorVal.String()
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		t, err := tabOps.UpdateTab(tabID, updates)
		if err != nil || t == nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		// Fire onActivated if the tab was activated.
		if updates.Active != nil && *updates.Active {
			c.FireTabActivated(tabID)
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(map[string]interface{}{
				"id":             t.ID,
				"title":          t.Title,
				"url":            t.URL,
				"active":         t.IsActive,
				"windowId":       t.WindowID,
			}))
		}
		return goja.Undefined()
		})

	// chrome.tabs.create
	_ = tabsObj.Set("create", func(call goja.FunctionCall) goja.Value {
		create := TabCreate{}
		if len(call.Arguments) > 0 {
			createObj := call.Argument(0).ToObject(vm)
			if createObj != nil {
				if urlVal := createObj.Get("url"); urlVal != nil {
					create.URL = urlVal.String()
				}
				if titleVal := createObj.Get("title"); titleVal != nil {
					create.Title = titleVal.String()
				}
				if activeVal := createObj.Get("active"); activeVal != nil {
					b := activeVal.ToBoolean()
					create.Active = &b
				}
				if containerIDVal := createObj.Get("containerId"); containerIDVal != nil {
					create.ContainerID = containerIDVal.String()
				}
				if containerNameVal := createObj.Get("containerName"); containerNameVal != nil {
					create.ContainerName = containerNameVal.String()
				}
				if containerColorVal := createObj.Get("containerColor"); containerColorVal != nil {
					create.ContainerColor = containerColorVal.String()
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		t, err := tabOps.CreateTab(create)
		if err != nil || t == nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		// Fire onCreated event.
		c.FireTabCreated(t)
		if callback != nil {
			_, _ = callback(vm.ToValue(map[string]interface{}{
				"id":             t.ID,
				"title":          t.Title,
				"url":            t.URL,
				"active":         t.IsActive,
				"windowId":       t.WindowID,
			}))
		}
		return goja.Undefined()
	})

	// chrome.tabs.remove
	_ = tabsObj.Set("remove", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		tabID := int(call.Argument(0).ToInteger())
		callback, _ := goja.AssertFunction(call.Argument(1))
		_ = tabOps.RemoveTab(tabID)
		// Fire onRemoved event.
		c.FireTabRemoved(tabID)
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.tabs.sendMessage - send a message to the content script in a specific tab.
	_ = tabsObj.Set("sendMessage", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewGoError(fmt.Errorf("sendMessage requires tabId and message arguments")))
		}
		tabID := int(call.Argument(0).ToInteger())
		msg := Message{
			Sender:  ext.ID,
			Target:  ext.ID,
			Type:    "background",
			Payload: call.Argument(1).Export(),
			TabID:   tabID,
		}
		if err := router.Route(msg); err != nil {
			panic(vm.NewGoError(fmt.Errorf("tabs.sendMessage: %w", err)))
		}
		// Deliver to content script sandbox for this extension.
		if err := router.DeliverToSandbox(ext.ID, msg); err != nil {
			// No sandbox registered; message is still routed to background handlers.
		}
		return goja.Undefined()
	})

	// chrome.tabs.onUpdated event
	onUpdatedObj := vm.NewObject()
	_ = onUpdatedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		// Store listener for tab update notifications (wired by shell).
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.tabEventListeners["onUpdated"] = append(c.tabEventListeners["onUpdated"], cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = tabsObj.Set("onUpdated", onUpdatedObj)

	// chrome.tabs.onCreated event
	onCreatedObj := vm.NewObject()
	_ = onCreatedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.tabEventListeners["onCreated"] = append(c.tabEventListeners["onCreated"], cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = tabsObj.Set("onCreated", onCreatedObj)

	// chrome.tabs.onRemoved event
	onRemovedObj := vm.NewObject()
	_ = onRemovedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.tabEventListeners["onRemoved"] = append(c.tabEventListeners["onRemoved"], cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = tabsObj.Set("onRemoved", onRemovedObj)

	// chrome.tabs.onActivated event
	onActivatedObj := vm.NewObject()
	_ = onActivatedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.tabEventListeners["onActivated"] = append(c.tabEventListeners["onActivated"], cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = tabsObj.Set("onActivated", onActivatedObj)

	_ = chromeObj.Set("tabs", tabsObj)
	return nil
}

// FireTabCreated fires the chrome.tabs.onCreated event to all registered
// listeners for this extension.
func (c *ChromeAPI) FireTabCreated(tab *TabInfo) {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.tabEventListeners["onCreated"]))
	copy(listeners, c.tabEventListeners["onCreated"])
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		_, _ = cb(goja.Undefined(), c.vm.ToValue(map[string]interface{}{
			"id":             tab.ID,
			"title":          tab.Title,
			"url":            tab.URL,
			"favicon":        tab.Favicon,
			"containerId":    tab.ContainerID,
			"containerName":  tab.ContainerName,
			"containerColor": tab.ContainerColor,
			"active":         tab.IsActive,
			"windowId":       tab.WindowID,
		}))
	}
}

// FireTabRemoved fires the chrome.tabs.onRemoved event to all registered
// listeners for this extension.
func (c *ChromeAPI) FireTabRemoved(tabID int) {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.tabEventListeners["onRemoved"]))
	copy(listeners, c.tabEventListeners["onRemoved"])
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		_, _ = cb(goja.Undefined(), c.vm.ToValue(float64(tabID)))
	}
}

// FireTabActivated fires the chrome.tabs.onActivated event to all registered
// listeners for this extension.
func (c *ChromeAPI) FireTabActivated(tabID int) {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.tabEventListeners["onActivated"]))
	copy(listeners, c.tabEventListeners["onActivated"])
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		_, _ = cb(goja.Undefined(), c.vm.ToValue(map[string]interface{}{
			"tabId": float64(tabID),
		}))
	}
}

// FireTabUpdated fires the chrome.tabs.onUpdated event to all registered
// listeners for this extension.
func (c *ChromeAPI) FireTabUpdated(tabID int, changes map[string]interface{}) {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.tabEventListeners["onUpdated"]))
	copy(listeners, c.tabEventListeners["onUpdated"])
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		_, _ = cb(goja.Undefined(), c.vm.ToValue(map[string]interface{}{
			"tabId":      tabID,
			"changes":    changes,
			"status":     "complete",
		}))
	}
}

// FireWebRequestEvent fires a chrome.webRequest event to all registered
// listeners for this extension. It is called by the shell when sauce
// notifies WebRequest observers of network activity.
func (c *ChromeAPI) FireWebRequestEvent(event string, details WebRequestDetails) {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.webRequestEventListeners[event]))
	copy(listeners, c.webRequestEventListeners[event])
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		_, _ = cb(goja.Undefined(), c.vm.ToValue(map[string]interface{}{
			"requestId":    details.RequestID,
			"url":          details.URL,
			"method":       details.Method,
			"headers":      details.Headers,
			"statusCode":   details.StatusCode,
			"statusLine":   details.StatusLine,
			"responseHeaders": details.ResponseHeaders,
			"initiator":    details.Initiator,
			"tabId":        details.TabID,
			"timestamp":    details.Timestamp,
		}))
	}
}

// FireDownloadEvent fires a chrome.downloads event to all registered
// listeners for this extension. It is called by the shell when download
// state changes (created, changed, erased).
func (c *ChromeAPI) FireDownloadEvent(event string, item DownloadItem) {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.downloadEventListeners[event]))
	copy(listeners, c.downloadEventListeners[event])
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		if event == "onErased" {
			_, _ = cb(goja.Undefined(), c.vm.ToValue(item.ID))
			continue
		}
		_, _ = cb(goja.Undefined(), c.vm.ToValue(map[string]interface{}{
			"id":           item.ID,
			"url":          item.URL,
			"filename":     item.Filename,
			"state":        item.State,
			"totalBytes":   item.TotalBytes,
			"bytesReceived": item.BytesReceived,
			"mime":         item.Mime,
			"startTime":    item.StartTime,
			"endTime":      item.EndTime,
			"paused":       item.Paused,
			"error":        item.Error,
		}))
	}
}

// FireCookieChanged fires the chrome.cookies.onChanged event to all
// registered listeners for this extension.
func (c *ChromeAPI) FireCookieChanged(change CookieChange) {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.cookieEventListeners))
	copy(listeners, c.cookieEventListeners)
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		_, _ = cb(goja.Undefined(), c.vm.ToValue(map[string]interface{}{
			"cookie": map[string]interface{}{
				"name":       change.Cookie.Name,
				"value":      change.Cookie.Value,
				"domain":     change.Cookie.Domain,
				"path":       change.Cookie.Path,
				"secure":     change.Cookie.Secure,
				"httpOnly":   change.Cookie.HttpOnly,
				"sameSite":   change.Cookie.SameSite,
				"session":    change.Cookie.Session,
				"expiration": change.Cookie.Expiration,
				"storeId":    change.Cookie.StoreID,
			},
			"cause":     change.Cause,
			"removed":   change.Removed,
		}))
	}
}

// CookieChange describes a cookie change event.
type CookieChange struct {
	Cookie Cookie
	Cause  string // "expired", "explicit", "overwrite"
	Removed bool
}

// bindWindows installs chrome.windows into the VM.
func (c *ChromeAPI) bindWindows(vm *goja.Runtime, ext *Extension, winOps WindowOperations) error {
	chrome := vm.Get("chrome")
	var chromeObj *goja.Object
	if chrome == nil || chrome.ToObject(vm) == nil {
		chromeObj = vm.NewObject()
		_ = vm.Set("chrome", chromeObj)
	} else {
		chromeObj = chrome.ToObject(vm)
	}

	windowsObj := vm.NewObject()

	// chrome.windows.getCurrent
	_ = windowsObj.Set("getCurrent", func(call goja.FunctionCall) goja.Value {
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		win, err := winOps.GetCurrent()
		if err != nil || win == nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(map[string]interface{}{
				"id":        win.ID,
				"title":     win.Title,
				"focused":   win.IsFocused,
				"width":     win.Width,
				"height":    win.Height,
			}))
		}
		return goja.Undefined()
	})

	// chrome.windows.getAll
	_ = windowsObj.Set("getAll", func(call goja.FunctionCall) goja.Value {
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		wins, err := winOps.GetAll()
		if err != nil {
			wins = nil
		}
		result := make([]interface{}, 0, len(wins))
		for _, w := range wins {
			result = append(result, map[string]interface{}{
				"id":        w.ID,
				"title":     w.Title,
				"focused":   w.IsFocused,
				"width":     w.Width,
				"height":    w.Height,
			})
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(result))
		}
		return goja.Undefined()
	})

	// chrome.windows.create
	_ = windowsObj.Set("create", func(call goja.FunctionCall) goja.Value {
		create := WindowCreate{}
		if len(call.Arguments) > 0 {
			createObj := call.Argument(0).ToObject(vm)
			if createObj != nil {
				if titleVal := createObj.Get("title"); titleVal != nil {
					create.Title = titleVal.String()
				}
				if urlVal := createObj.Get("url"); urlVal != nil {
					create.URL = urlVal.String()
				}
				if widthVal := createObj.Get("width"); widthVal != nil {
					create.Width = int(widthVal.ToInteger())
				}
				if heightVal := createObj.Get("height"); heightVal != nil {
					create.Height = int(heightVal.ToInteger())
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		win, err := winOps.Create(create)
		if err != nil || win == nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(map[string]interface{}{
				"id":        win.ID,
				"title":     win.Title,
				"focused":   win.IsFocused,
				"width":     win.Width,
				"height":    win.Height,
			}))
		}
		return goja.Undefined()
	})

	// chrome.windows.remove
	_ = windowsObj.Set("remove", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		windowID := int(call.Argument(0).ToInteger())
		callback, _ := goja.AssertFunction(call.Argument(1))
		_ = winOps.Remove(windowID)
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	_ = chromeObj.Set("windows", windowsObj)
	return nil
}

// bindContextualIdentities installs chrome.contextualIdentities into the VM.
func (c *ChromeAPI) bindContextualIdentities(vm *goja.Runtime, ext *Extension, containerOps ContainerOperations) error {
	chrome := vm.Get("chrome")
	var chromeObj *goja.Object
	if chrome == nil || chrome.ToObject(vm) == nil {
		chromeObj = vm.NewObject()
		_ = vm.Set("chrome", chromeObj)
	} else {
		chromeObj = chrome.ToObject(vm)
	}

	ciObj := vm.NewObject()

	// chrome.contextualIdentities.query
	_ = ciObj.Set("query", func(call goja.FunctionCall) goja.Value {
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		containers := containerOps.QueryContainers()
		result := make([]interface{}, 0, len(containers))
		for _, c := range containers {
			result = append(result, map[string]interface{}{
				"id":    c.ID,
				"name":  c.Name,
				"color": c.Color,
				"icon":  c.Icon,
			})
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(result))
		}
		return goja.Undefined()
	})

	// chrome.contextualIdentities.get
	_ = ciObj.Set("get", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		id := call.Argument(0).String()
		callback, _ := goja.AssertFunction(call.Argument(1))
		c, err := containerOps.GetContainer(id)
		if err != nil || c == nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(map[string]interface{}{
				"id":    c.ID,
				"name":  c.Name,
				"color": c.Color,
				"icon":  c.Icon,
			}))
		}
		return goja.Undefined()
	})

	// chrome.contextualIdentities.create
	_ = ciObj.Set("create", func(call goja.FunctionCall) goja.Value {
		name := ""
		color := ""
		if len(call.Arguments) > 0 {
			createObj := call.Argument(0).ToObject(vm)
			if createObj != nil {
				if nameVal := createObj.Get("name"); nameVal != nil {
					name = nameVal.String()
				}
				if colorVal := createObj.Get("color"); colorVal != nil {
					color = colorVal.String()
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		c, err := containerOps.CreateContainer(name, color)
		if err != nil || c == nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(map[string]interface{}{
				"id":    c.ID,
				"name":  c.Name,
				"color": c.Color,
				"icon":  c.Icon,
			}))
		}
		return goja.Undefined()
	})

	// chrome.contextualIdentities.update
	_ = ciObj.Set("update", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		id := call.Argument(0).String()
		name := ""
		color := ""
		if len(call.Arguments) > 1 {
			updateObj := call.Argument(1).ToObject(vm)
			if updateObj != nil {
				if nameVal := updateObj.Get("name"); nameVal != nil {
					name = nameVal.String()
				}
				if colorVal := updateObj.Get("color"); colorVal != nil {
					color = colorVal.String()
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		c, err := containerOps.UpdateContainer(id, name, color)
		if err != nil || c == nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(map[string]interface{}{
				"id":    c.ID,
				"name":  c.Name,
				"color": c.Color,
				"icon":  c.Icon,
			}))
		}
		return goja.Undefined()
	})

	// chrome.contextualIdentities.remove
	_ = ciObj.Set("remove", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		id := call.Argument(0).String()
		callback, _ := goja.AssertFunction(call.Argument(1))
		_ = containerOps.RemoveContainer(id)
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.contextualIdentities.onCreated event
	onCreatedObj := vm.NewObject()
	_ = onCreatedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.containerEventListeners["onCreated"] = append(c.containerEventListeners["onCreated"], cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = ciObj.Set("onCreated", onCreatedObj)

	// chrome.contextualIdentities.onRemoved event
	onRemovedObj := vm.NewObject()
	_ = onRemovedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.containerEventListeners["onRemoved"] = append(c.containerEventListeners["onRemoved"], cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = ciObj.Set("onRemoved", onRemovedObj)

	// chrome.contextualIdentities.onUpdated event
	onUpdatedObj := vm.NewObject()
	_ = onUpdatedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.containerEventListeners["onUpdated"] = append(c.containerEventListeners["onUpdated"], cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = ciObj.Set("onUpdated", onUpdatedObj)

	_ = chromeObj.Set("contextualIdentities", ciObj)
	return nil
}

// FireContainerCreated fires the chrome.contextualIdentities.onCreated event.
func (c *ChromeAPI) FireContainerCreated(container *ContainerInfo) {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.containerEventListeners["onCreated"]))
	copy(listeners, c.containerEventListeners["onCreated"])
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		_, _ = cb(goja.Undefined(), c.vm.ToValue(map[string]interface{}{
			"id":    container.ID,
			"name":  container.Name,
			"color": container.Color,
			"icon":  container.Icon,
		}))
	}
}

// FireContainerRemoved fires the chrome.contextualIdentities.onRemoved event.
func (c *ChromeAPI) FireContainerRemoved(containerID string) {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.containerEventListeners["onRemoved"]))
	copy(listeners, c.containerEventListeners["onRemoved"])
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		_, _ = cb(goja.Undefined(), c.vm.ToValue(map[string]interface{}{
			"id": containerID,
		}))
	}
}

// FireContainerUpdated fires the chrome.contextualIdentities.onUpdated event.
func (c *ChromeAPI) FireContainerUpdated(container *ContainerInfo) {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.containerEventListeners["onUpdated"]))
	copy(listeners, c.containerEventListeners["onUpdated"])
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		_, _ = cb(goja.Undefined(), c.vm.ToValue(map[string]interface{}{
			"id":    container.ID,
			"name":  container.Name,
			"color": container.Color,
			"icon":  container.Icon,
		}))
	}
}

// FireContextMenuClicked fires the chrome.contextMenus.onClicked event.
func (c *ChromeAPI) FireContextMenuClicked(itemID string) {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.contextMenuEventListeners))
	copy(listeners, c.contextMenuEventListeners)
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		_, _ = cb(goja.Undefined(), c.vm.ToValue(map[string]interface{}{
			"menuItemId": itemID,
		}))
	}
}

// FirePermissionAdded fires the chrome.permissions.onAdded event.
func (c *ChromeAPI) FirePermissionAdded(permissions []string) {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.permissionAddedListeners))
	copy(listeners, c.permissionAddedListeners)
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		_, _ = cb(goja.Undefined(), c.vm.ToValue(permissions))
	}
}

// FirePermissionRemoved fires the chrome.permissions.onRemoved event.
func (c *ChromeAPI) FirePermissionRemoved(permissions []string) {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.permissionRemovedListeners))
	copy(listeners, c.permissionRemovedListeners)
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		_, _ = cb(goja.Undefined(), c.vm.ToValue(permissions))
	}
}

// bindCookies installs chrome.cookies into the VM.
func (c *ChromeAPI) bindCookies(vm *goja.Runtime, ext *Extension, cookieOps CookieOperations) error {
	chrome := vm.Get("chrome")
	var chromeObj *goja.Object
	if chrome == nil || chrome.ToObject(vm) == nil {
		chromeObj = vm.NewObject()
		_ = vm.Set("chrome", chromeObj)
	} else {
		chromeObj = chrome.ToObject(vm)
	}

	cookiesObj := vm.NewObject()

	// chrome.cookies.get
	_ = cookiesObj.Set("get", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		details := call.Argument(0).ToObject(vm)
		if details == nil {
			return goja.Undefined()
		}
		urlVal := details.Get("url")
		nameVal := details.Get("name")
		storeIDVal := details.Get("storeId")

		url := ""
		if urlVal != nil {
			url = urlVal.String()
		}
		name := ""
		if nameVal != nil {
			name = nameVal.String()
		}
		storeID := ""
		if storeIDVal != nil {
			storeID = storeIDVal.String()
		}

		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		cookies, err := cookieOps.GetCookies(storeID, url)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		for _, cookie := range cookies {
			if cookie.Name == name {
				if callback != nil {
					_, _ = callback(vm.ToValue(map[string]interface{}{
						"name":       cookie.Name,
						"value":      cookie.Value,
						"domain":     cookie.Domain,
						"path":       cookie.Path,
						"secure":     cookie.Secure,
						"httpOnly":   cookie.HttpOnly,
						"sameSite":   cookie.SameSite,
						"session":    cookie.Session,
						"expiration": cookie.Expiration,
						"storeId":    cookie.StoreID,
					}))
				}
				return goja.Undefined()
			}
		}
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.cookies.set
	_ = cookiesObj.Set("set", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		details := call.Argument(0).ToObject(vm)
		if details == nil {
			return goja.Undefined()
		}
		cookie := Cookie{}
		if nameVal := details.Get("name"); nameVal != nil {
			cookie.Name = nameVal.String()
		}
		if valueVal := details.Get("value"); valueVal != nil {
			cookie.Value = valueVal.String()
		}
		if domainVal := details.Get("domain"); domainVal != nil {
			cookie.Domain = domainVal.String()
		}
		if pathVal := details.Get("path"); pathVal != nil {
			cookie.Path = pathVal.String()
		}
		if secureVal := details.Get("secure"); secureVal != nil {
			cookie.Secure = secureVal.ToBoolean()
		}
		if httpOnlyVal := details.Get("httpOnly"); httpOnlyVal != nil {
			cookie.HttpOnly = httpOnlyVal.ToBoolean()
		}
		if sameSiteVal := details.Get("sameSite"); sameSiteVal != nil {
			cookie.SameSite = sameSiteVal.String()
		}
		if sessionVal := details.Get("session"); sessionVal != nil {
			cookie.Session = sessionVal.ToBoolean()
		}
		if expirationVal := details.Get("expiration"); expirationVal != nil {
			cookie.Expiration = expirationVal.String()
		}
		if storeIDVal := details.Get("storeId"); storeIDVal != nil {
			cookie.StoreID = storeIDVal.String()
		}

		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		if err := cookieOps.SetCookie(cookie.StoreID, cookie); err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(map[string]interface{}{
				"name":       cookie.Name,
				"value":      cookie.Value,
				"domain":     cookie.Domain,
				"path":       cookie.Path,
				"secure":     cookie.Secure,
				"httpOnly":   cookie.HttpOnly,
				"sameSite":   cookie.SameSite,
				"session":    cookie.Session,
				"expiration": cookie.Expiration,
				"storeId":    cookie.StoreID,
			}))
		}
		return goja.Undefined()
	})

	// chrome.cookies.remove
	_ = cookiesObj.Set("remove", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		details := call.Argument(0).ToObject(vm)
		if details == nil {
			return goja.Undefined()
		}
		cookie := Cookie{}
		if urlVal := details.Get("url"); urlVal != nil {
			// URL is used to identify the cookie but not stored in Cookie struct
			_ = urlVal.String()
		}
		if nameVal := details.Get("name"); nameVal != nil {
			cookie.Name = nameVal.String()
		}
		if storeIDVal := details.Get("storeId"); storeIDVal != nil {
			cookie.StoreID = storeIDVal.String()
		}

		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		if err := cookieOps.RemoveCookie(cookie.StoreID, cookie); err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(map[string]interface{}{
				"name":    cookie.Name,
				"storeId": cookie.StoreID,
			}))
		}
		return goja.Undefined()
	})

	// chrome.cookies.getAll
	_ = cookiesObj.Set("getAll", func(call goja.FunctionCall) goja.Value {
		storeID := ""
		if len(call.Arguments) > 0 {
			details := call.Argument(0).ToObject(vm)
			if details != nil {
				if storeIDVal := details.Get("storeId"); storeIDVal != nil {
					storeID = storeIDVal.String()
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		cookies, err := cookieOps.GetAllCookies(storeID)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		result := make([]interface{}, 0, len(cookies))
		for _, cookie := range cookies {
			result = append(result, map[string]interface{}{
				"name":       cookie.Name,
				"value":      cookie.Value,
				"domain":     cookie.Domain,
				"path":       cookie.Path,
				"secure":     cookie.Secure,
				"httpOnly":   cookie.HttpOnly,
				"sameSite":   cookie.SameSite,
				"session":    cookie.Session,
				"expiration": cookie.Expiration,
				"storeId":    cookie.StoreID,
			})
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(result))
		}
		return goja.Undefined()
	})

	// chrome.cookies.removeAll (clear all cookies for a store)
	_ = cookiesObj.Set("removeAll", func(call goja.FunctionCall) goja.Value {
		storeID := ""
		if len(call.Arguments) > 0 {
			details := call.Argument(0).ToObject(vm)
			if details != nil {
				if storeIDVal := details.Get("storeId"); storeIDVal != nil {
					storeID = storeIDVal.String()
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		if err := cookieOps.ClearCookies(storeID); err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.cookies.getAllCookieStores
	_ = cookiesObj.Set("getAllCookieStores", func(call goja.FunctionCall) goja.Value {
		var callback goja.Callable
		if len(call.Arguments) > 0 {
			callback, _ = goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		}
		stores, err := cookieOps.GetAllCookieStores()
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		result := make([]interface{}, 0, len(stores))
		for _, s := range stores {
			result = append(result, map[string]interface{}{
				"id":       s.ID,
				"incognito": s.Incognito,
			})
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(result))
		}
		return vm.ToValue(result)
	})

	// chrome.cookies.onChanged event
	onChangedObj := vm.NewObject()
	_ = onChangedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.cookieEventListeners = append(c.cookieEventListeners, cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = cookiesObj.Set("onChanged", onChangedObj)

	_ = chromeObj.Set("cookies", cookiesObj)
	return nil
}

// EnsureChromePrefix ensures the chrome global object exists in the VM.
func EnsureChromePrefix(vm *goja.Runtime) *goja.Object {
	chrome := vm.Get("chrome")
	if chrome != nil && chrome.ToObject(vm) != nil {
		return chrome.ToObject(vm)
	}
	chromeObj := vm.NewObject()
	_ = vm.Set("chrome", chromeObj)
	return chromeObj
}

// bindScripting installs chrome.scripting into the VM.
func (c *ChromeAPI) bindScripting(vm *goja.Runtime, ext *Extension, ops ScriptingOperations) error {
	chrome := vm.Get("chrome")
	var chromeObj *goja.Object
	if chrome == nil || chrome.ToObject(vm) == nil {
		chromeObj = vm.NewObject()
		_ = vm.Set("chrome", chromeObj)
	} else {
		chromeObj = chrome.ToObject(vm)
	}

	scriptingObj := vm.NewObject()

	// chrome.scripting.executeScript
	_ = scriptingObj.Set("executeScript", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("executeScript requires a script argument")))
		}
		script := call.Argument(0).String()
		world := ISOLATEDWorld
		if len(call.Arguments) > 1 {
			worldStr := call.Argument(1).String()
			if worldStr == "MAIN" {
				world = MAINWorld
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		result, err := ops.ExecuteScript(ext.ID, world, script)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(result))
		}
		return goja.Undefined()
	})

	// chrome.scripting.executeScriptFile
	_ = scriptingObj.Set("executeScriptFile", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 3 {
			panic(vm.NewGoError(fmt.Errorf("executeScriptFile requires extPath, world, and jsPath arguments")))
		}
		extPath := call.Argument(0).String()
		worldStr := call.Argument(1).String()
		jsPath := call.Argument(2).String()
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		result, err := ops.ExecuteScriptFile(ext.ID, extPath, worldStr, jsPath)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(result))
		}
		return goja.Undefined()
	})

	// chrome.scripting.insertCSS
	_ = scriptingObj.Set("insertCSS", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("insertCSS requires a CSS argument")))
		}
		css := call.Argument(0).String()
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		if err := ops.InjectCSS(ext.ID, css); err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.scripting.removeCSS
	_ = scriptingObj.Set("removeCSS", func(call goja.FunctionCall) goja.Value {
		var callback goja.Callable
		if len(call.Arguments) > 0 {
			callback, _ = goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		}
		if err := ops.RemoveCSS(ext.ID); err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.scripting.registerContentScripts
	_ = scriptingObj.Set("registerContentScripts", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("registerContentScripts requires a scripts argument")))
		}
		scriptsObj := call.Argument(0).ToObject(vm)
		var scripts []ContentScript
		if scriptsObj != nil {
			if arr, ok := scriptsObj.ToObject(vm).Export().([]interface{}); ok {
				for _, v := range arr {
					if s, ok := v.(map[string]interface{}); ok {
						script := ContentScript{
							ExtID: ext.ID,
						}
						if id, ok := s["id"].(string); ok {
							script.ID = id
						}
						if matches, ok := s["matches"].([]interface{}); ok {
							for _, m := range matches {
								if matchStr, ok := m.(string); ok {
									script.Matches = append(script.Matches, matchStr)
								}
							}
						}
						if js, ok := s["js"].([]interface{}); ok {
							for _, j := range js {
								if jsStr, ok := j.(string); ok {
									script.JS = append(script.JS, jsStr)
								}
							}
						}
						if css, ok := s["css"].([]interface{}); ok {
							for _, c := range css {
								if cssStr, ok := c.(string); ok {
									script.CSS = append(script.CSS, cssStr)
								}
							}
						}
						if runAt, ok := s["runAt"].(string); ok {
							script.RunAt = runAt
						}
						scripts = append(scripts, script)
					}
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		if err := ops.RegisterContentScripts(ext.ID, scripts); err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.scripting.unregisterContentScripts
	_ = scriptingObj.Set("unregisterContentScripts", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("unregisterContentScripts requires a scriptIds argument")))
		}
		idsObj := call.Argument(0).ToObject(vm)
		var scriptIDs []string
		if idsObj != nil {
			if arr, ok := idsObj.Export().([]interface{}); ok {
				for _, v := range arr {
					if idStr, ok := v.(string); ok {
						scriptIDs = append(scriptIDs, idStr)
					}
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		if err := ops.UnregisterContentScripts(ext.ID, scriptIDs); err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	_ = chromeObj.Set("scripting", scriptingObj)
	return nil
}

// bindContextMenus installs chrome.contextMenus into the VM.
func (c *ChromeAPI) bindContextMenus(vm *goja.Runtime, ext *Extension, ops ContextMenuOperations) error {
	chrome := vm.Get("chrome")
	var chromeObj *goja.Object
	if chrome == nil || chrome.ToObject(vm) == nil {
		chromeObj = vm.NewObject()
		_ = vm.Set("chrome", chromeObj)
	} else {
		chromeObj = chrome.ToObject(vm)
	}

	menusObj := vm.NewObject()

	// chrome.contextMenus.create
	_ = menusObj.Set("create", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("create requires a menu item argument")))
		}
		itemObj := call.Argument(0).ToObject(vm)
		item := ContextMenuItem{}
		if itemObj != nil {
			if idVal := itemObj.Get("id"); idVal != nil {
				item.ID = idVal.String()
			}
			if titleVal := itemObj.Get("title"); titleVal != nil {
				item.Title = titleVal.String()
			}
			if contextsVal := itemObj.Get("contexts"); contextsVal != nil {
				if arr, ok := contextsVal.Export().([]interface{}); ok {
					for _, v := range arr {
						if ctxStr, ok := v.(string); ok {
							item.Contexts = append(item.Contexts, ctxStr)
						}
					}
				}
			}
			if checkedVal := itemObj.Get("checked"); checkedVal != nil {
				item.Checked = checkedVal.ToBoolean()
			}
			if enabledVal := itemObj.Get("enabled"); enabledVal != nil {
				item.Enabled = enabledVal.ToBoolean()
			}
			if visibleVal := itemObj.Get("visible"); visibleVal != nil {
				item.Visible = visibleVal.ToBoolean()
			}
			if parentIdVal := itemObj.Get("parentId"); parentIdVal != nil {
				item.ParentID = parentIdVal.String()
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		id, err := ops.CreateContextMenu(item)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(id))
		}
		return goja.Undefined()
	})

	// chrome.contextMenus.update
	_ = menusObj.Set("update", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewGoError(fmt.Errorf("update requires an id and menu item argument")))
		}
		id := call.Argument(0).String()
		itemObj := call.Argument(1).ToObject(vm)
		item := ContextMenuItem{}
		if itemObj != nil {
			if titleVal := itemObj.Get("title"); titleVal != nil {
				item.Title = titleVal.String()
			}
			if checkedVal := itemObj.Get("checked"); checkedVal != nil {
				item.Checked = checkedVal.ToBoolean()
			}
			if enabledVal := itemObj.Get("enabled"); enabledVal != nil {
				item.Enabled = enabledVal.ToBoolean()
			}
			if visibleVal := itemObj.Get("visible"); visibleVal != nil {
				item.Visible = visibleVal.ToBoolean()
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		if err := ops.UpdateContextMenu(id, item); err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.contextMenus.remove
	_ = menusObj.Set("remove", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("remove requires an id argument")))
		}
		id := call.Argument(0).String()
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		if err := ops.RemoveContextMenu(id); err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.contextMenus.removeAll
	_ = menusObj.Set("removeAll", func(call goja.FunctionCall) goja.Value {
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		if err := ops.RemoveAllContextMenus(); err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.contextMenus.onClicked event
	onClickedObj := vm.NewObject()
	_ = onClickedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.contextMenuEventListeners = append(c.contextMenuEventListeners, cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = menusObj.Set("onClicked", onClickedObj)

	_ = chromeObj.Set("contextMenus", menusObj)
	return nil
}

// bindPermissions installs chrome.permissions into the VM.
func (c *ChromeAPI) bindPermissions(vm *goja.Runtime, ext *Extension, ops PermissionOperations) error {
	chrome := vm.Get("chrome")
	var chromeObj *goja.Object
	if chrome == nil || chrome.ToObject(vm) == nil {
		chromeObj = vm.NewObject()
		_ = vm.Set("chrome", chromeObj)
	} else {
		chromeObj = chrome.ToObject(vm)
	}

	permsObj := vm.NewObject()

	// chrome.permissions.request
	_ = permsObj.Set("request", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("request requires a permissions argument")))
		}
		permsObj := call.Argument(0).ToObject(vm)
		var permissions []string
		if permsObj != nil {
			if arr, ok := permsObj.Export().([]interface{}); ok {
				for _, v := range arr {
					if permStr, ok := v.(string); ok {
						permissions = append(permissions, permStr)
					}
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		granted, err := ops.RequestPermissions(ext.ID, permissions)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(granted))
		}
		return goja.Undefined()
	})

	// chrome.permissions.contains
	_ = permsObj.Set("contains", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("contains requires a permissions argument")))
		}
		permsObj := call.Argument(0).ToObject(vm)
		var permissions []string
		if permsObj != nil {
			if arr, ok := permsObj.Export().([]interface{}); ok {
				for _, v := range arr {
					if permStr, ok := v.(string); ok {
						permissions = append(permissions, permStr)
					}
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		has, err := ops.ContainsPermissions(ext.ID, permissions)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(has))
		}
		return goja.Undefined()
	})

	// chrome.permissions.getAll
	_ = permsObj.Set("getAll", func(call goja.FunctionCall) goja.Value {
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		all, err := ops.GetAllPermissions(ext.ID)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(all))
		}
		return goja.Undefined()
	})

	// chrome.permissions.onAdded event
	onAddedObj := vm.NewObject()
	_ = onAddedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.permissionAddedListeners = append(c.permissionAddedListeners, cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = permsObj.Set("onAdded", onAddedObj)

	// chrome.permissions.onRemoved event
	onRemovedObj := vm.NewObject()
	_ = onRemovedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.permissionRemovedListeners = append(c.permissionRemovedListeners, cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = permsObj.Set("onRemoved", onRemovedObj)

	_ = chromeObj.Set("permissions", permsObj)
	return nil
}

// bindAlarms installs chrome.alarms into the VM.
func (c *ChromeAPI) bindAlarms(vm *goja.Runtime, ext *Extension, ops AlarmOperations) error {
	chrome := vm.Get("chrome")
	var chromeObj *goja.Object
	if chrome == nil || chrome.ToObject(vm) == nil {
		chromeObj = vm.NewObject()
		_ = vm.Set("chrome", chromeObj)
	} else {
		chromeObj = chrome.ToObject(vm)
	}

	alarmsObj := vm.NewObject()

	// chrome.alarms.create
	_ = alarmsObj.Set("create", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("create requires a name argument")))
		}
		name := call.Argument(0).String()
		alarm := Alarm{Name: name}
		if len(call.Arguments) > 1 {
			alarmObj := call.Argument(1).ToObject(vm)
			if alarmObj != nil {
				if whenVal := alarmObj.Get("when"); whenVal != nil {
					alarm.When = whenVal.ToFloat()
				}
				if periodVal := alarmObj.Get("periodInMinutes"); periodVal != nil {
					alarm.PeriodInMinutes = periodVal.ToFloat()
				}
				if delayVal := alarmObj.Get("delayInMinutes"); delayVal != nil {
					alarm.Scheduled = delayVal.ToFloat()
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		if err := ops.CreateAlarm(ext.ID, name, alarm); err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.alarms.get
	_ = alarmsObj.Set("get", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("get requires a name argument")))
		}
		name := call.Argument(0).String()
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		alarm, err := ops.GetAlarm(ext.ID, name)
		if err != nil || alarm == nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(map[string]interface{}{
				"name":             alarm.Name,
				"scheduled":        alarm.Scheduled,
				"when":             alarm.When,
				"periodInMinutes":  alarm.PeriodInMinutes,
			}))
		}
		return goja.Undefined()
	})

	// chrome.alarms.getAll
	_ = alarmsObj.Set("getAll", func(call goja.FunctionCall) goja.Value {
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		all, err := ops.GetAllAlarms(ext.ID)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		result := make([]interface{}, 0, len(all))
		for _, a := range all {
			result = append(result, map[string]interface{}{
				"name":             a.Name,
				"scheduled":        a.Scheduled,
				"when":             a.When,
				"periodInMinutes":  a.PeriodInMinutes,
			})
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(result))
		}
		return goja.Undefined()
	})

	// chrome.alarms.clear
	_ = alarmsObj.Set("clear", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("clear requires a name argument")))
		}
		name := call.Argument(0).String()
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		_ = ops.ClearAlarm(ext.ID, name)
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.alarms.clearAll
	_ = alarmsObj.Set("clearAll", func(call goja.FunctionCall) goja.Value {
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		_ = ops.ClearAllAlarms(ext.ID)
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.alarms.onAlarm event
	onAlarmObj := vm.NewObject()
	_ = onAlarmObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.alarmEventListeners["onAlarm"] = append(c.alarmEventListeners["onAlarm"], cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = alarmsObj.Set("onAlarm", onAlarmObj)

	_ = chromeObj.Set("alarms", alarmsObj)
	return nil
}

// FireAlarm fires the chrome.alarms.onAlarm event to all registered
// listeners for this extension.
func (c *ChromeAPI) FireAlarm(name string) {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.alarmEventListeners["onAlarm"]))
	copy(listeners, c.alarmEventListeners["onAlarm"])
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		_, _ = cb(goja.Undefined(), c.vm.ToValue(map[string]interface{}{
			"name": name,
		}))
	}
}

// bindI18n installs chrome.i18n into the VM.
func (c *ChromeAPI) bindI18n(vm *goja.Runtime, ext *Extension, ops I18nOperations) error {
	chrome := vm.Get("chrome")
	var chromeObj *goja.Object
	if chrome == nil || chrome.ToObject(vm) == nil {
		chromeObj = vm.NewObject()
		_ = vm.Set("chrome", chromeObj)
	} else {
		chromeObj = chrome.ToObject(vm)
	}

	i18nObj := vm.NewObject()

	// chrome.i18n.getMessage
	_ = i18nObj.Set("getMessage", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("getMessage requires a message name argument")))
		}
		messageName := call.Argument(0).String()
		var substitutions []string
		if len(call.Arguments) > 1 {
			subsObj := call.Argument(1).ToObject(vm)
			if subsObj != nil {
				if arr, ok := subsObj.Export().([]interface{}); ok {
					for _, v := range arr {
						if subStr, ok := v.(string); ok {
							substitutions = append(substitutions, subStr)
						}
					}
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		msg, err := ops.GetMessage(ext.ID, messageName, substitutions...)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(msg))
		}
		return goja.Undefined()
	})

	// chrome.i18n.getAcceptLanguages
	_ = i18nObj.Set("getAcceptLanguages", func(call goja.FunctionCall) goja.Value {
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		langs, err := ops.GetAcceptLanguages(ext.ID)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(langs))
		}
		return goja.Undefined()
	})

	_ = chromeObj.Set("i18n", i18nObj)
	return nil
}

// bindWebRequest installs chrome.webRequest into the VM.
func (c *ChromeAPI) bindWebRequest(vm *goja.Runtime, ext *Extension, ops WebRequestOperations) error {
	chrome := vm.Get("chrome")
	var chromeObj *goja.Object
	if chrome == nil || chrome.ToObject(vm) == nil {
		chromeObj = vm.NewObject()
		_ = vm.Set("chrome", chromeObj)
	} else {
		chromeObj = chrome.ToObject(vm)
	}

	webRequestObj := vm.NewObject()

	// chrome.webRequest.onBeforeRequest
	onBeforeRequest := vm.NewObject()
	_ = onBeforeRequest.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return goja.Undefined()
		}
		callback, _ := goja.AssertFunction(call.Argument(0))
		details := call.Argument(1).ToObject(vm)
		urlPatterns := []string{}
		if typesVal := details.Get("urls"); typesVal != nil {
			arr := typesVal.ToObject(vm)
			if arr != nil {
				for _, key := range arr.Keys() {
					urlPatterns = append(urlPatterns, arr.Get(key).String())
				}
			}
		}
		wrDetails := WebRequestDetails{URLPatterns: urlPatterns}
		if err := ops.AddListener(ext.ID, "onBeforeRequest", wrDetails); err != nil {
			return goja.Undefined()
		}
		c.mu.Lock()
		c.webRequestEventListeners["onBeforeRequest"] = append(c.webRequestEventListeners["onBeforeRequest"], callback)
		c.mu.Unlock()
		return goja.Undefined()
	})
	_ = onBeforeRequest.Set("removeListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		_ = call.Argument(0)
		if err := ops.RemoveListener(ext.ID, "onBeforeRequest"); err != nil {
			return goja.Undefined()
		}
		c.mu.Lock()
		c.webRequestEventListeners["onBeforeRequest"] = nil
		c.mu.Unlock()
		return goja.Undefined()
	})
	_ = webRequestObj.Set("onBeforeRequest", onBeforeRequest)

	// chrome.webRequest.onBeforeSendHeaders
	onBeforeSendHeaders := vm.NewObject()
	_ = onBeforeSendHeaders.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return goja.Undefined()
		}
		callback, _ := goja.AssertFunction(call.Argument(0))
		details := call.Argument(1).ToObject(vm)
		urlPatterns := []string{}
		if typesVal := details.Get("urls"); typesVal != nil {
			arr := typesVal.ToObject(vm)
			if arr != nil {
				for _, key := range arr.Keys() {
					urlPatterns = append(urlPatterns, arr.Get(key).String())
				}
			}
		}
		wrDetails := WebRequestDetails{URLPatterns: urlPatterns}
		if err := ops.AddListener(ext.ID, "onBeforeSendHeaders", wrDetails); err != nil {
			return goja.Undefined()
		}
		c.mu.Lock()
		c.webRequestEventListeners["onBeforeSendHeaders"] = append(c.webRequestEventListeners["onBeforeSendHeaders"], callback)
		c.mu.Unlock()
		return goja.Undefined()
	})
	_ = onBeforeSendHeaders.Set("removeListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		_ = call.Argument(0)
		if err := ops.RemoveListener(ext.ID, "onBeforeSendHeaders"); err != nil {
			return goja.Undefined()
		}
		c.mu.Lock()
		c.webRequestEventListeners["onBeforeSendHeaders"] = nil
		c.mu.Unlock()
		return goja.Undefined()
	})
	_ = webRequestObj.Set("onBeforeSendHeaders", onBeforeSendHeaders)

	// chrome.webRequest.onHeadersReceived
	onHeadersReceived := vm.NewObject()
	_ = onHeadersReceived.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return goja.Undefined()
		}
		callback, _ := goja.AssertFunction(call.Argument(0))
		details := call.Argument(1).ToObject(vm)
		urlPatterns := []string{}
		if typesVal := details.Get("urls"); typesVal != nil {
			arr := typesVal.ToObject(vm)
			if arr != nil {
				for _, key := range arr.Keys() {
					urlPatterns = append(urlPatterns, arr.Get(key).String())
				}
			}
		}
		wrDetails := WebRequestDetails{URLPatterns: urlPatterns}
		if err := ops.AddListener(ext.ID, "onHeadersReceived", wrDetails); err != nil {
			return goja.Undefined()
		}
		c.mu.Lock()
		c.webRequestEventListeners["onHeadersReceived"] = append(c.webRequestEventListeners["onHeadersReceived"], callback)
		c.mu.Unlock()
		return goja.Undefined()
	})
	_ = onHeadersReceived.Set("removeListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		_ = call.Argument(0)
		if err := ops.RemoveListener(ext.ID, "onHeadersReceived"); err != nil {
			return goja.Undefined()
		}
		c.mu.Lock()
		c.webRequestEventListeners["onHeadersReceived"] = nil
		c.mu.Unlock()
		return goja.Undefined()
	})
	_ = webRequestObj.Set("onHeadersReceived", onHeadersReceived)

	// chrome.webRequest.onCompleted
	onCompleted := vm.NewObject()
	_ = onCompleted.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return goja.Undefined()
		}
		callback, _ := goja.AssertFunction(call.Argument(0))
		details := call.Argument(1).ToObject(vm)
		urlPatterns := []string{}
		if typesVal := details.Get("urls"); typesVal != nil {
			arr := typesVal.ToObject(vm)
			if arr != nil {
				for _, key := range arr.Keys() {
					urlPatterns = append(urlPatterns, arr.Get(key).String())
				}
			}
		}
		wrDetails := WebRequestDetails{URLPatterns: urlPatterns}
		if err := ops.AddListener(ext.ID, "onCompleted", wrDetails); err != nil {
			return goja.Undefined()
		}
		c.mu.Lock()
		c.webRequestEventListeners["onCompleted"] = append(c.webRequestEventListeners["onCompleted"], callback)
		c.mu.Unlock()
		return goja.Undefined()
	})
	_ = onCompleted.Set("removeListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		_ = call.Argument(0)
		if err := ops.RemoveListener(ext.ID, "onCompleted"); err != nil {
			return goja.Undefined()
		}
		c.mu.Lock()
		c.webRequestEventListeners["onCompleted"] = nil
		c.mu.Unlock()
		return goja.Undefined()
	})
	_ = webRequestObj.Set("onCompleted", onCompleted)

	// chrome.webRequest.onErrorOccurred
	onErrorOccurred := vm.NewObject()
	_ = onErrorOccurred.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return goja.Undefined()
		}
		callback, _ := goja.AssertFunction(call.Argument(0))
		details := call.Argument(1).ToObject(vm)
		urlPatterns := []string{}
		if typesVal := details.Get("urls"); typesVal != nil {
			arr := typesVal.ToObject(vm)
			if arr != nil {
				for _, key := range arr.Keys() {
					urlPatterns = append(urlPatterns, arr.Get(key).String())
				}
			}
		}
		wrDetails := WebRequestDetails{URLPatterns: urlPatterns}
		if err := ops.AddListener(ext.ID, "onErrorOccurred", wrDetails); err != nil {
			return goja.Undefined()
		}
		c.mu.Lock()
		c.webRequestEventListeners["onErrorOccurred"] = append(c.webRequestEventListeners["onErrorOccurred"], callback)
		c.mu.Unlock()
		return goja.Undefined()
	})
	_ = onErrorOccurred.Set("removeListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		_ = call.Argument(0)
		if err := ops.RemoveListener(ext.ID, "onErrorOccurred"); err != nil {
			return goja.Undefined()
		}
		c.mu.Lock()
		c.webRequestEventListeners["onErrorOccurred"] = nil
		c.mu.Unlock()
		return goja.Undefined()
	})
	_ = webRequestObj.Set("onErrorOccurred", onErrorOccurred)

	_ = chromeObj.Set("webRequest", webRequestObj)
	return nil
}

// bindNotifications installs chrome.notifications into the VM.
func (c *ChromeAPI) bindNotifications(vm *goja.Runtime, ext *Extension, ops NotificationOperations) error {
	chrome := vm.Get("chrome")
	var chromeObj *goja.Object
	if chrome == nil || chrome.ToObject(vm) == nil {
		chromeObj = vm.NewObject()
		_ = vm.Set("chrome", chromeObj)
	} else {
		chromeObj = chrome.ToObject(vm)
	}

	notifObj := vm.NewObject()

	// chrome.notifications.create
	_ = notifObj.Set("create", func(call goja.FunctionCall) goja.Value {
		var id string
		if len(call.Arguments) > 0 {
			id = call.Argument(0).String()
		}
		options := NotificationOptions{}
		if len(call.Arguments) > 1 {
			opts := call.Argument(1).ToObject(vm)
			if opts != nil {
				if v := opts.Get("title"); v != nil {
					options.Title = v.String()
				}
				if v := opts.Get("message"); v != nil {
					options.Message = v.String()
				}
				if v := opts.Get("iconUrl"); v != nil {
					options.IconURL = v.String()
				}
				if v := opts.Get("context"); v != nil {
					options.Context = v.String()
				}
				if v := opts.Get("priority"); v != nil {
					options.Priority = int(v.ToInteger())
				}
				if v := opts.Get("timestamp"); v != nil {
					options.Timestamp = v.ToFloat()
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		newID, err := ops.CreateNotification(ext.ID, id, options)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(newID))
		}
		return vm.ToValue(newID)
	})

	// chrome.notifications.clear
	_ = notifObj.Set("clear", func(call goja.FunctionCall) goja.Value {
		id := ""
		if len(call.Arguments) > 0 {
			id = call.Argument(0).String()
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		wasCleared, err := ops.ClearNotification(ext.ID, id)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(wasCleared))
		}
		return vm.ToValue(wasCleared)
	})

	// chrome.notifications.getAll
	_ = notifObj.Set("getAll", func(call goja.FunctionCall) goja.Value {
		var callback goja.Callable
		if len(call.Arguments) > 0 {
			callback, _ = goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		}
		notifications, err := ops.GetAllNotifications(ext.ID)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		result := make([]interface{}, 0, len(notifications))
		for _, n := range notifications {
			result = append(result, map[string]interface{}{
				"id":        n.ID,
				"title":     n.Title,
				"message":   n.Message,
				"iconUrl":   n.IconURL,
				"context":   n.Context,
				"priority":  n.Priority,
				"timestamp": n.Timestamp,
			})
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(result))
		}
		return vm.ToValue(result)
	})

	// chrome.notifications.onClicked event
	onClickedObj := vm.NewObject()
	_ = onClickedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.notificationClickedListeners = append(c.notificationClickedListeners, cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = notifObj.Set("onClicked", onClickedObj)

	// chrome.notifications.onClosed event
	onClosedObj := vm.NewObject()
	_ = onClosedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.notificationClosedListeners = append(c.notificationClosedListeners, cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = notifObj.Set("onClosed", onClosedObj)

	_ = chromeObj.Set("notifications", notifObj)
	return nil
}

// FireNotificationClicked fires the chrome.notifications.onClicked event.
func (c *ChromeAPI) FireNotificationClicked(notificationID string) {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.notificationClickedListeners))
	copy(listeners, c.notificationClickedListeners)
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		_, _ = cb(goja.Undefined(), c.vm.ToValue(notificationID))
	}
}

// FireNotificationClosed fires the chrome.notifications.onClosed event.
func (c *ChromeAPI) FireNotificationClosed(notificationID string) {
	c.mu.Lock()
	listeners := make([]goja.Callable, len(c.notificationClosedListeners))
	copy(listeners, c.notificationClosedListeners)
	c.mu.Unlock()

	c.vmMu.lock()
	defer c.vmMu.unlock()

	for _, cb := range listeners {
		_, _ = cb(goja.Undefined(), c.vm.ToValue(notificationID))
	}
}

// bindDownloads installs chrome.downloads into the VM.
func (c *ChromeAPI) bindDownloads(vm *goja.Runtime, ext *Extension, ops DownloadOperations) error {
	chrome := vm.Get("chrome")
	var chromeObj *goja.Object
	if chrome == nil || chrome.ToObject(vm) == nil {
		chromeObj = vm.NewObject()
		_ = vm.Set("chrome", chromeObj)
	} else {
		chromeObj = chrome.ToObject(vm)
	}

	dlObj := vm.NewObject()

	// chrome.downloads.download
	_ = dlObj.Set("download", func(call goja.FunctionCall) goja.Value {
		options := DownloadOptions{}
		if len(call.Arguments) > 0 {
			opts := call.Argument(0).ToObject(vm)
			if opts != nil {
				if v := opts.Get("url"); v != nil {
					options.URL = v.String()
				}
				if v := opts.Get("filename"); v != nil {
					options.Filename = v.String()
				}
				if v := opts.Get("saveAs"); v != nil {
					options.SaveAs = v.ToBoolean()
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		id, err := ops.Download(ext.ID, options)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(id))
		}
		return vm.ToValue(id)
	})

	// chrome.downloads.cancel
	_ = dlObj.Set("cancel", func(call goja.FunctionCall) goja.Value {
		id := 0
		if len(call.Arguments) > 0 {
			id = int(call.Argument(0).ToInteger())
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		if err := ops.Cancel(id); err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(true))
		}
		return vm.ToValue(true)
	})

	// chrome.downloads.pause
	_ = dlObj.Set("pause", func(call goja.FunctionCall) goja.Value {
		id := 0
		if len(call.Arguments) > 0 {
			id = int(call.Argument(0).ToInteger())
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		if err := ops.Pause(id); err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(true))
		}
		return vm.ToValue(true)
	})

	// chrome.downloads.resume
	_ = dlObj.Set("resume", func(call goja.FunctionCall) goja.Value {
		id := 0
		if len(call.Arguments) > 0 {
			id = int(call.Argument(0).ToInteger())
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		if err := ops.Resume(id); err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(true))
		}
		return vm.ToValue(true)
	})

	// chrome.downloads.get
	_ = dlObj.Set("get", func(call goja.FunctionCall) goja.Value {
		id := 0
		if len(call.Arguments) > 0 {
			id = int(call.Argument(0).ToInteger())
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		item, err := ops.Get(id)
		if err != nil || item == nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(map[string]interface{}{
				"id":           item.ID,
				"url":          item.URL,
				"filename":     item.Filename,
				"state":        item.State,
				"totalBytes":   item.TotalBytes,
				"bytesReceived": item.BytesReceived,
				"mime":         item.Mime,
				"startTime":    item.StartTime,
				"endTime":      item.EndTime,
				"paused":       item.Paused,
				"error":        item.Error,
			}))
		}
		return goja.Undefined()
	})

	// chrome.downloads.getAll
	_ = dlObj.Set("getAll", func(call goja.FunctionCall) goja.Value {
		var callback goja.Callable
		if len(call.Arguments) > 0 {
			callback, _ = goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		}
		items, err := ops.GetAll()
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		result := make([]interface{}, 0, len(items))
		for _, item := range items {
			result = append(result, map[string]interface{}{
				"id":           item.ID,
				"url":          item.URL,
				"filename":     item.Filename,
				"state":        item.State,
				"totalBytes":   item.TotalBytes,
				"bytesReceived": item.BytesReceived,
				"mime":         item.Mime,
				"startTime":    item.StartTime,
				"endTime":      item.EndTime,
				"paused":       item.Paused,
				"error":        item.Error,
			})
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(result))
		}
		return vm.ToValue(result)
	})

	// chrome.downloads.erase
	_ = dlObj.Set("erase", func(call goja.FunctionCall) goja.Value {
		id := 0
		if len(call.Arguments) > 0 {
			id = int(call.Argument(0).ToInteger())
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		if err := ops.Erase(id); err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.downloads.search
	_ = dlObj.Set("search", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		queryObj := call.Argument(0).ToObject(vm)
		q := DownloadQuery{}
		if queryObj != nil {
			if v := queryObj.Get("id"); v != nil {
				i := int(v.ToInteger())
				q.ID = &i
			}
			if v := queryObj.Get("url"); v != nil {
				q.URL = v.String()
			}
			if v := queryObj.Get("filename"); v != nil {
				q.Filename = v.String()
			}
			if v := queryObj.Get("state"); v != nil {
				q.State = v.String()
			}
			if v := queryObj.Get("paused"); v != nil {
				b := v.ToBoolean()
				q.Paused = &b
			}
			if v := queryObj.Get("error"); v != nil {
				q.Error = v.String()
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		items, err := ops.Search(q)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		result := make([]interface{}, 0, len(items))
		for _, item := range items {
			result = append(result, map[string]interface{}{
				"id":           item.ID,
				"url":          item.URL,
				"filename":     item.Filename,
				"state":        item.State,
				"totalBytes":   item.TotalBytes,
				"bytesReceived": item.BytesReceived,
				"mime":         item.Mime,
				"startTime":    item.StartTime,
				"endTime":      item.EndTime,
				"paused":       item.Paused,
				"error":        item.Error,
			})
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(result))
		}
		return goja.Undefined()
	})

	// chrome.downloads.onCreated event
	onCreatedObj := vm.NewObject()
	_ = onCreatedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.downloadEventListeners["onCreated"] = append(c.downloadEventListeners["onCreated"], cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = dlObj.Set("onCreated", onCreatedObj)

	// chrome.downloads.onChanged event
	onChangedObj := vm.NewObject()
	_ = onChangedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.downloadEventListeners["onChanged"] = append(c.downloadEventListeners["onChanged"], cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = dlObj.Set("onChanged", onChangedObj)

	// chrome.downloads.onErased event
	onErasedObj := vm.NewObject()
	_ = onErasedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				c.mu.Lock()
				c.downloadEventListeners["onErased"] = append(c.downloadEventListeners["onErased"], cb)
				c.mu.Unlock()
			}
		}
		return goja.Undefined()
	})
	_ = dlObj.Set("onErased", onErasedObj)

	_ = chromeObj.Set("downloads", dlObj)
	return nil
}

// bindBookmarks installs chrome.bookmarks into the VM.
func (c *ChromeAPI) bindBookmarks(vm *goja.Runtime, ext *Extension, ops BookmarkOperations) error {
	chrome := vm.Get("chrome")
	var chromeObj *goja.Object
	if chrome == nil || chrome.ToObject(vm) == nil {
		chromeObj = vm.NewObject()
		_ = vm.Set("chrome", chromeObj)
	} else {
		chromeObj = chrome.ToObject(vm)
	}

	bookmarksObj := vm.NewObject()

	// chrome.bookmarks.create
	_ = bookmarksObj.Set("create", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("create requires a bookmark argument")))
		}
		bmObj := call.Argument(0).ToObject(vm)
		bm := Bookmark{}
		if bmObj != nil {
			if titleVal := bmObj.Get("title"); titleVal != nil {
				bm.Title = titleVal.String()
			}
			if urlVal := bmObj.Get("url"); urlVal != nil {
				bm.URL = urlVal.String()
			}
			if parentIDVal := bmObj.Get("parentId"); parentIDVal != nil {
				bm.ParentID = parentIDVal.String()
			}
			if indexVal := bmObj.Get("index"); indexVal != nil {
				bm.Index = int(indexVal.ToInteger())
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		created, err := ops.CreateBookmark(bm)
		if err != nil || created == nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(map[string]interface{}{
				"id":       created.ID,
				"title":    created.Title,
				"url":      created.URL,
				"parentId": created.ParentID,
				"index":    created.Index,
			}))
		}
		return goja.Undefined()
	})

	// chrome.bookmarks.get
	_ = bookmarksObj.Set("get", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("get requires an id argument")))
		}
		id := call.Argument(0).String()
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		bm, err := ops.GetBookmark(id)
		if err != nil || bm == nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(map[string]interface{}{
				"id":       bm.ID,
				"title":    bm.Title,
				"url":      bm.URL,
				"parentId": bm.ParentID,
				"index":    bm.Index,
			}))
		}
		return goja.Undefined()
	})

	// chrome.bookmarks.getChildren
	_ = bookmarksObj.Set("getChildren", func(call goja.FunctionCall) goja.Value {
		parentID := ""
		if len(call.Arguments) > 0 {
			parentID = call.Argument(0).String()
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		children, err := ops.GetChildren(parentID)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		result := make([]interface{}, 0, len(children))
		for _, bm := range children {
			result = append(result, map[string]interface{}{
				"id":       bm.ID,
				"title":    bm.Title,
				"url":      bm.URL,
				"parentId": bm.ParentID,
				"index":    bm.Index,
			})
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(result))
		}
		return goja.Undefined()
	})

	// chrome.bookmarks.move
	_ = bookmarksObj.Set("move", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("move requires an id argument")))
		}
		id := call.Argument(0).String()
		moveObj := call.Argument(1).ToObject(vm)
		parentID := ""
		index := 0
		if moveObj != nil {
			if pid := moveObj.Get("parentId"); pid != nil {
				parentID = pid.String()
			}
			if idx := moveObj.Get("index"); idx != nil {
				index = int(idx.ToInteger())
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		bm, err := ops.MoveBookmark(id, parentID, index)
		if err != nil || bm == nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(map[string]interface{}{
				"id":       bm.ID,
				"title":    bm.Title,
				"url":      bm.URL,
				"parentId": bm.ParentID,
				"index":    bm.Index,
			}))
		}
		return goja.Undefined()
	})

	// chrome.bookmarks.remove
	_ = bookmarksObj.Set("remove", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("remove requires an id argument")))
		}
		id := call.Argument(0).String()
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		_ = ops.RemoveBookmark(id)
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.bookmarks.update
	_ = bookmarksObj.Set("update", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("update requires an id argument")))
		}
		id := call.Argument(0).String()
		updates := Bookmark{}
		if len(call.Arguments) > 1 {
			updObj := call.Argument(1).ToObject(vm)
			if updObj != nil {
				if titleVal := updObj.Get("title"); titleVal != nil {
					updates.Title = titleVal.String()
				}
				if urlVal := updObj.Get("url"); urlVal != nil {
					updates.URL = urlVal.String()
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		bm, err := ops.UpdateBookmark(id, updates)
		if err != nil || bm == nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(map[string]interface{}{
				"id":       bm.ID,
				"title":    bm.Title,
				"url":      bm.URL,
				"parentId": bm.ParentID,
				"index":    bm.Index,
			}))
		}
		return goja.Undefined()
	})

	_ = chromeObj.Set("bookmarks", bookmarksObj)
	return nil
}

// bindHistory installs chrome.history into the VM.
func (c *ChromeAPI) bindHistory(vm *goja.Runtime, ext *Extension, ops HistoryOperations) error {
	chrome := vm.Get("chrome")
	var chromeObj *goja.Object
	if chrome == nil || chrome.ToObject(vm) == nil {
		chromeObj = vm.NewObject()
		_ = vm.Set("chrome", chromeObj)
	} else {
		chromeObj = chrome.ToObject(vm)
	}

	historyObj := vm.NewObject()

	// chrome.history.addUrl
	_ = historyObj.Set("addUrl", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("addUrl requires a url argument")))
		}
		url := call.Argument(0).String()
		title := ""
		if len(call.Arguments) > 1 {
			details := call.Argument(1).ToObject(vm)
			if details != nil {
				if titleVal := details.Get("title"); titleVal != nil {
					title = titleVal.String()
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		if err := ops.AddUrl(url, title); err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.history.deleteUrl
	_ = historyObj.Set("deleteUrl", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("deleteUrl requires a url argument")))
		}
		url := call.Argument(0).String()
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		if err := ops.DeleteUrl(url); err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.history.search
	_ = historyObj.Set("search", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("search requires a query argument")))
		}
		query := call.Argument(0).String()
		maxResults := 100
		if len(call.Arguments) > 1 {
			opts := call.Argument(1).ToObject(vm)
			if opts != nil {
				if mr := opts.Get("maxResults"); mr != nil {
					maxResults = int(mr.ToInteger())
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		results, err := ops.Search(query, maxResults)
		if err != nil {
			if callback != nil {
				_, _ = callback(goja.Undefined())
			}
			return goja.Undefined()
		}
		result := make([]interface{}, 0, len(results))
		for _, entry := range results {
			result = append(result, map[string]interface{}{
				"id":         entry.ID,
				"url":        entry.URL,
				"title":      entry.Title,
				"visitTime":  entry.VisitTime,
				"typed":      entry.Typed,
			})
		}
		if callback != nil {
			_, _ = callback(vm.ToValue(result))
		}
		return goja.Undefined()
	})

	_ = chromeObj.Set("history", historyObj)
	return nil
}

// bindAction installs chrome.action into the VM.
func (c *ChromeAPI) bindAction(vm *goja.Runtime, ext *Extension, ops ActionOperations) error {
	chrome := vm.Get("chrome")
	var chromeObj *goja.Object
	if chrome == nil || chrome.ToObject(vm) == nil {
		chromeObj = vm.NewObject()
		_ = vm.Set("chrome", chromeObj)
	} else {
		chromeObj = chrome.ToObject(vm)
	}

	actionObj := vm.NewObject()

	// chrome.action.setBadgeText
	_ = actionObj.Set("setBadgeText", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("setBadgeText requires a details argument")))
		}
		details := call.Argument(0).ToObject(vm)
		text := ""
		tabID := -1
		if details != nil {
			if textVal := details.Get("text"); textVal != nil {
				text = textVal.String()
			}
			if tabIDVal := details.Get("tabId"); tabIDVal != nil {
				tabID = int(tabIDVal.ToInteger())
			}
		}
		if err := ops.SetBadgeText(ext.ID, text, tabID); err != nil {
			return goja.Undefined()
		}
		return goja.Undefined()
	})

	// chrome.action.setBadgeColor
	_ = actionObj.Set("setBadgeColor", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("setBadgeColor requires a details argument")))
		}
		details := call.Argument(0).ToObject(vm)
		color := ""
		tabID := -1
		if details != nil {
			if colorVal := details.Get("color"); colorVal != nil {
				color = colorVal.String()
			}
			if tabIDVal := details.Get("tabId"); tabIDVal != nil {
				tabID = int(tabIDVal.ToInteger())
			}
		}
		if err := ops.SetBadgeColor(ext.ID, color, tabID); err != nil {
			return goja.Undefined()
		}
		return goja.Undefined()
	})

	// chrome.action.setIcon
	_ = actionObj.Set("setIcon", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("setIcon requires a details argument")))
		}
		details := call.Argument(0).ToObject(vm)
		path := ""
		tabID := -1
		if details != nil {
			if pathVal := details.Get("path"); pathVal != nil {
				path = pathVal.String()
			}
			if tabIDVal := details.Get("tabId"); tabIDVal != nil {
				tabID = int(tabIDVal.ToInteger())
			}
		}
		if err := ops.SetIcon(ext.ID, path, tabID); err != nil {
			return goja.Undefined()
		}
		return goja.Undefined()
	})

	// chrome.action.setPopup
	_ = actionObj.Set("setPopup", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("setPopup requires a details argument")))
		}
		details := call.Argument(0).ToObject(vm)
		popup := ""
		tabID := -1
		if details != nil {
			if popupVal := details.Get("popup"); popupVal != nil {
				popup = popupVal.String()
			}
			if tabIDVal := details.Get("tabId"); tabIDVal != nil {
				tabID = int(tabIDVal.ToInteger())
			}
		}
		if err := ops.SetPopup(ext.ID, popup, tabID); err != nil {
			return goja.Undefined()
		}
		return goja.Undefined()
	})

	// chrome.action.getPopup
	_ = actionObj.Set("getPopup", func(call goja.FunctionCall) goja.Value {
		tabID := -1
		if len(call.Arguments) > 0 {
			details := call.Argument(0).ToObject(vm)
			if details != nil {
				if tabIDVal := details.Get("tabId"); tabIDVal != nil {
					tabID = int(tabIDVal.ToInteger())
				}
			}
		}
		callback, _ := goja.AssertFunction(call.Argument(len(call.Arguments) - 1))
		popup, _ := ops.GetPopup(ext.ID, tabID)
		if callback != nil {
			_, _ = callback(vm.ToValue(map[string]interface{}{
				"popup": popup,
			}))
		}
		return goja.Undefined()
	})

	_ = chromeObj.Set("action", actionObj)
	return nil
}
