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
	URL      string `json:"url,omitempty"`
	Title    string `json:"title,omitempty"`
	Active   *bool  `json:"active,omitempty"`
	Pinned   *bool  `json:"pinned,omitempty"`
	Muted    *bool  `json:"muted,omitempty"`
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
	GetTab(tabID int) (*TabInfo, error)
	UpdateTab(tabID int, updates TabUpdate) (*TabInfo, error)
	CreateTab(tab TabCreate) (*TabInfo, error)
	RemoveTab(tabID int) error
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
	platform      PlatformInfo
}

// NewChromeAPI creates a ChromeAPI binding for the given extension and VM.
func NewChromeAPI(ext *Extension, vm *goja.Runtime, router *MessageRouter) *ChromeAPI {
	return &ChromeAPI{
		ext:    ext,
		vm:     vm,
		router: router,
		platform: PlatformInfo{
			OS:           runtime.GOOS,
			Arch:         runtime.GOARCH,
			Platform:     runtime.GOOS + "/" + runtime.GOARCH,
			BrowserVersion: "thdwb dev",
		},
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

	_ = chromeObj.Set("runtime", runtimeObj)
	return nil
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
					keyStr, _ := key.(string)
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
					keyStr, _ := key.(string)
					_ = sp.Remove(area, keyStr)
				}
			}
			return goja.Undefined()
		})

		_ = areaObj.Set("clear", func(call goja.FunctionCall) goja.Value {
			_ = sp.Clear(area)
			return goja.Undefined()
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
		tabs := tabOps.QueryTabs()
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
		if callback != nil {
			_, _ = callback(goja.Undefined())
		}
		return goja.Undefined()
	})

	// chrome.tabs.onUpdated event
	onUpdatedObj := vm.NewObject()
	_ = onUpdatedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		// Store listener for tab update notifications (wired by shell).
		_ = call
		return goja.Undefined()
	})
	_ = tabsObj.Set("onUpdated", onUpdatedObj)

	_ = chromeObj.Set("tabs", tabsObj)
	return nil
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

	_ = chromeObj.Set("contextualIdentities", ciObj)
	return nil
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
						item.Contexts = append(item.Contexts, v.(string))
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
		// Listener storage wired by shell via message router.
		_ = call
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
					permissions = append(permissions, v.(string))
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
					permissions = append(permissions, v.(string))
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
		_ = call
		return goja.Undefined()
	})
	_ = permsObj.Set("onAdded", onAddedObj)

	// chrome.permissions.onRemoved event
	onRemovedObj := vm.NewObject()
	_ = onRemovedObj.Set("addListener", func(call goja.FunctionCall) goja.Value {
		_ = call
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
		_ = call
		return goja.Undefined()
	})
	_ = alarmsObj.Set("onAlarm", onAlarmObj)

	_ = chromeObj.Set("alarms", alarmsObj)
	return nil
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
						substitutions = append(substitutions, v.(string))
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
		_ = callback
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
		_ = callback
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
		_ = callback
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
		_ = callback
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
		_ = callback
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

	_ = chromeObj.Set("notifications", notifObj)
	return nil
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

	_ = chromeObj.Set("downloads", dlObj)
	return nil
}
