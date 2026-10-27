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
}

// PlatformInfo describes the browser platform as reported by chrome.runtime.
type PlatformInfo struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	Platform     string `json:"platform"`
	BrowserVersion string `json:"browser_version"`
}

// ChromeAPI binds WebExtensions APIs (chrome.runtime, chrome.storage,
// chrome.tabs, chrome.windows, chrome.contextualIdentities) to an
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
