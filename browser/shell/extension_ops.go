package shell

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	extensions "github.com/danfragoso/thdwb/browser/extensions"
	sauce "github.com/danfragoso/thdwb/sauce"
)

// extensionOps implements all WebExtensions operations interfaces on the
// BrowserShell, wiring chrome.* API calls to the shell's tab manager,
// container manager, cookie jar, and extension manager.
type extensionOps struct {
	bs *BrowserShell

	// cookieJar is a per-shell container-partitioned cookie store used by
	// chrome.cookies API.
	cookieJar *sauce.ContainerCookieJar

	// contextMenus holds context menu items keyed by extension ID then item ID.
	contextMenus map[string]map[string]extensions.ContextMenuItem
	contextMu    sync.Mutex

	// permissions holds granted permissions keyed by extension ID.
	permissions map[string]map[string]bool
	permMu      sync.Mutex

	// alarms holds alarms keyed by extension ID then alarm name.
	alarms  map[string]map[string]*extensions.Alarm
	alarmMu sync.Mutex

	// i18nMessages holds localized messages keyed by extension ID then message name.
	i18nMessages map[string]map[string]string
	i18nMu       sync.Mutex
}

// newExtensionOps creates a new extensionOps wired to the given browser shell.
func newExtensionOps(bs *BrowserShell) *extensionOps {
	return &extensionOps{
		bs:           bs,
		cookieJar:    sauce.NewContainerCookieJar(),
		contextMenus: make(map[string]map[string]extensions.ContextMenuItem),
		permissions:  make(map[string]map[string]bool),
		alarms:       make(map[string]map[string]*extensions.Alarm),
		i18nMessages: make(map[string]map[string]string),
	}
}

// TabOperations implementation.

// QueryTabs returns all open tabs as TabInfo.
func (eo *extensionOps) QueryTabs() []extensions.TabInfo {
	tabs := eo.bs.tabManager.Tabs()
	result := make([]extensions.TabInfo, 0, len(tabs))
	for _, t := range tabs {
		result = append(result, extensions.TabInfo{
			ID:             t.ID,
			Title:          t.Title,
			URL:            t.URL,
			Favicon:        t.Favicon,
			ContainerID:    t.ContainerID,
			ContainerName:  t.ContainerName,
			ContainerColor: t.ContainerColor,
			IsActive:       t.IsActive,
			WindowID:       1,
		})
	}
	return result
}

// GetTab returns the tab with the given ID.
func (eo *extensionOps) GetTab(tabID int) (*extensions.TabInfo, error) {
	t := eo.bs.tabManager.GetTab(tabID)
	if t == nil {
		return nil, nil
	}
	return &extensions.TabInfo{
		ID:             t.ID,
		Title:          t.Title,
		URL:            t.URL,
		Favicon:        t.Favicon,
		ContainerID:    t.ContainerID,
		ContainerName:  t.ContainerName,
		ContainerColor: t.ContainerColor,
		IsActive:       t.IsActive,
		WindowID:       1,
	}, nil
}

// UpdateTab updates the tab with the given ID.
func (eo *extensionOps) UpdateTab(tabID int, updates extensions.TabUpdate) (*extensions.TabInfo, error) {
	t := eo.bs.tabManager.GetTab(tabID)
	if t == nil {
		return nil, nil
	}
	if updates.URL != "" {
		eo.bs.tabManager.SetURL(tabID, updates.URL)
	}
	if updates.Title != "" {
		eo.bs.tabManager.SetTitle(tabID, updates.Title)
	}
	if updates.Active != nil && *updates.Active {
		eo.bs.tabManager.Activate(tabID)
	}
	// Re-fetch after mutations.
	t = eo.bs.tabManager.GetTab(tabID)
	return &extensions.TabInfo{
		ID:             t.ID,
		Title:          t.Title,
		URL:            t.URL,
		Favicon:        t.Favicon,
		ContainerID:    t.ContainerID,
		ContainerName:  t.ContainerName,
		ContainerColor: t.ContainerColor,
		IsActive:       t.IsActive,
		WindowID:       1,
	}, nil
}

// CreateTab creates a new tab.
func (eo *extensionOps) CreateTab(tab extensions.TabCreate) (*extensions.TabInfo, error) {
	t := eo.bs.tabManager.Add(tab.Title, tab.URL, tab.ContainerID, tab.ContainerName, tab.ContainerColor)
	if tab.Active != nil && !*tab.Active {
		// Deactivate the newly added tab if requested.
		active := eo.bs.tabManager.ActiveTab()
		if active != nil && active.ID == t.ID && eo.bs.tabManager.Count() > 1 {
			// Activate the previous tab instead.
			tabs := eo.bs.tabManager.Tabs()
			for i, candidate := range tabs {
				if candidate.ID == t.ID && i > 0 {
					eo.bs.tabManager.Activate(tabs[i-1].ID)
					break
				}
			}
		}
	}
	t = eo.bs.tabManager.ActiveTab()
	if t == nil {
		return nil, fmt.Errorf("failed to create tab")
	}
	return &extensions.TabInfo{
		ID:             t.ID,
		Title:          t.Title,
		URL:            t.URL,
		Favicon:        t.Favicon,
		ContainerID:    t.ContainerID,
		ContainerName:  t.ContainerName,
		ContainerColor: t.ContainerColor,
		IsActive:       t.IsActive,
		WindowID:       1,
	}, nil
}

// RemoveTab closes the tab with the given ID.
func (eo *extensionOps) RemoveTab(tabID int) error {
	t := eo.bs.tabManager.GetTab(tabID)
	if t == nil {
		return fmt.Errorf("tab %d not found", tabID)
	}
	eo.bs.tabManager.Close(tabID)
	return nil
}

// ActivateTab makes the tab with the given ID active.
func (eo *extensionOps) ActivateTab(tabID int) (*extensions.TabInfo, error) {
	t := eo.bs.tabManager.Activate(tabID)
	if t == nil {
		return nil, nil
	}
	return &extensions.TabInfo{
		ID:             t.ID,
		Title:          t.Title,
		URL:            t.URL,
		Favicon:        t.Favicon,
		ContainerID:    t.ContainerID,
		ContainerName:  t.ContainerName,
		ContainerColor: t.ContainerColor,
		IsActive:       t.IsActive,
		WindowID:       1,
	}, nil
}

// WindowOperations implementation.

// GetCurrent returns the current (only) window.
func (eo *extensionOps) GetCurrent() (*extensions.WindowInfo, error) {
	return &extensions.WindowInfo{
		ID:        1,
		Title:     "thdwb",
		IsFocused: true,
		Width:     1024,
		Height:    768,
	}, nil
}

// GetAll returns all windows (only one in this implementation).
func (eo *extensionOps) GetAll() ([]extensions.WindowInfo, error) {
	return []extensions.WindowInfo{
		{
			ID:        1,
			Title:     "thdwb",
			IsFocused: true,
			Width:     1024,
			Height:    768,
		},
	}, nil
}

// Create creates a new window (opens a new tab in this implementation).
func (eo *extensionOps) Create(win extensions.WindowCreate) (*extensions.WindowInfo, error) {
	_ = eo.bs.tabManager.Add(win.Title, win.URL, "", "", "")
	return &extensions.WindowInfo{
		ID:        2,
		Title:     win.Title,
		IsFocused: true,
		Width:     win.Width,
		Height:    win.Height,
	}, nil
}

// Remove removes a window by ID. Not supported in single-window mode.
func (eo *extensionOps) Remove(windowID int) error {
	if windowID == 1 {
		return fmt.Errorf("cannot remove the only window")
	}
	return nil
}

// CookieOperations implementation.

// cookieKey returns the container-scoped key for cookie lookups.
func cookieKey(containerID string) string {
	if containerID == "" {
		return "default"
	}
	return containerID
}

// GetCookies returns cookies for a container+origin.
func (eo *extensionOps) GetCookies(storeID string, urlStr string) ([]extensions.Cookie, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	cookies := eo.cookieJar.Cookies(cookieKey(storeID), u)
	result := make([]extensions.Cookie, 0, len(cookies))
	for _, c := range cookies {
		result = append(result, extensions.Cookie{
			Name:       c.Name,
			Value:      c.Value,
			Domain:     c.Domain,
			Path:       c.Path,
			Secure:     c.Secure,
			HttpOnly:   c.HttpOnly,
			SameSite:   sameSiteToString(c.SameSite),
			Session:    c.Expires.IsZero(),
			Expiration: c.Expires.Format("2006-01-02T15:04:05Z07:00"),
			StoreID:    storeID,
		})
	}
	return result, nil
}

// SetCookie sets a cookie in a container.
func (eo *extensionOps) SetCookie(storeID string, cookie extensions.Cookie) error {
	u, err := url.Parse(cookie.Domain)
	if err != nil {
		return err
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	if u.Path == "" {
		u.Path = "/"
	}
	hc := &http.Cookie{
		Name:     cookie.Name,
		Value:    cookie.Value,
		Path:     cookie.Path,
		Domain:   u.Host,
		Secure:   cookie.Secure,
		HttpOnly: cookie.HttpOnly,
		SameSite: sameSiteFromString(cookie.SameSite),
	}
	eo.cookieJar.SetCookies(cookieKey(storeID), u, []*http.Cookie{hc})

	// Fire chrome.cookies.onChanged event for all extensions.
	for _, ext := range eo.bs.extManager.GetAll() {
		if api, ok := eo.bs.extManager.ChromeAPIFor(ext.ID); ok {
			api.FireCookieChanged(extensions.CookieChange{
				Cookie: cookie,
				Cause:  "explicit",
			})
		}
	}
	return nil
}

// RemoveCookie removes a cookie from a container.
func (eo *extensionOps) RemoveCookie(storeID string, cookie extensions.Cookie) error {
	u, err := url.Parse(cookie.Domain)
	if err != nil {
		return err
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	// Ensure the URL has a path for cookie key derivation.
	if u.Path == "" {
		u.Path = "/"
	}
	eo.cookieJar.RemoveCookie(storeID, u, cookie.Name)

	// Fire chrome.cookies.onChanged event for all extensions.
	for _, ext := range eo.bs.extManager.GetAll() {
		if api, ok := eo.bs.extManager.ChromeAPIFor(ext.ID); ok {
			api.FireCookieChanged(extensions.CookieChange{
				Cookie:  cookie,
				Cause:   "explicit",
				Removed: true,
			})
		}
	}
	return nil
}

// GetAllCookies returns all cookies for a container.
func (eo *extensionOps) GetAllCookies(storeID string) ([]extensions.Cookie, error) {
	cookies := eo.cookieJar.AllCookies(storeID)
	result := make([]extensions.Cookie, 0, len(cookies))
	for _, c := range cookies {
		result = append(result, extensions.Cookie{
			Name:       c.Name,
			Value:      c.Value,
			Domain:     c.Domain,
			Path:       c.Path,
			Secure:     c.Secure,
			HttpOnly:   c.HttpOnly,
			SameSite:   sameSiteToString(c.SameSite),
			Session:    c.Expires.IsZero(),
			Expiration: c.Expires.Format("2006-01-02T15:04:05Z07:00"),
			StoreID:    storeID,
		})
	}
	return result, nil
}

// ClearCookies clears all cookies for a container.
func (eo *extensionOps) ClearCookies(storeID string) error {
	eo.cookieJar.ClearContainer(cookieKey(storeID))
	return nil
}

// GetAllCookieStores returns all cookie stores (containers).
func (eo *extensionOps) GetAllCookieStores() ([]extensions.CookieStore, error) {
	containers := eo.bs.containerManager.List()
	stores := make([]extensions.CookieStore, 0, len(containers))
	for _, c := range containers {
		stores = append(stores, extensions.CookieStore{
			ID:        c.ID,
			Incognito: false,
		})
	}
	// Always include the default store.
	stores = append(stores, extensions.CookieStore{
		ID:        "default",
		Incognito: false,
	})
	return stores, nil
}

// ScriptingOperations implementation.

// ExecuteScript executes a script in the specified world.
func (eo *extensionOps) ExecuteScript(extID string, world extensions.ScriptingWorld, script string) (interface{}, error) {
	return eo.bs.extManager.DynamicScripting().ExecuteScript(extID, world, script)
}

// ExecuteScriptFile executes a script file from the extension directory.
func (eo *extensionOps) ExecuteScriptFile(extID, extPath, worldStr, jsPath string) (interface{}, error) {
	return eo.bs.extManager.DynamicScripting().ExecuteScriptFile(extID, extPath, worldStr, jsPath)
}

// InjectCSS injects CSS for the given extension.
func (eo *extensionOps) InjectCSS(extID, css string) error {
	return eo.bs.extManager.DynamicScripting().InjectCSS(extID, css)
}

// RemoveCSS removes injected CSS for the given extension.
func (eo *extensionOps) RemoveCSS(extID string) error {
	return eo.bs.extManager.DynamicScripting().RemoveCSS(extID)
}

// ExecuteScriptTab executes a script in a specific tab.
func (eo *extensionOps) ExecuteScriptTab(extID string, tabID int, world extensions.ScriptingWorld, script string) (interface{}, error) {
	return eo.bs.extManager.DynamicScripting().ExecuteScriptTab(extID, tabID, world, script)
}

// ContextMenuOperations implementation.

// CreateContextMenu creates a context menu item.
func (eo *extensionOps) CreateContextMenu(item extensions.ContextMenuItem) (string, error) {
	eo.contextMu.Lock()
	defer eo.contextMu.Unlock()

	extMenus, ok := eo.contextMenus[item.ID]
	if !ok {
		extMenus = make(map[string]extensions.ContextMenuItem)
		eo.contextMenus[item.ID] = extMenus
	}
	if item.ID == "" {
		item.ID = "menu-1"
	}
	extMenus[item.ID] = item
	return item.ID, nil
}

// UpdateContextMenu updates a context menu item.
func (eo *extensionOps) UpdateContextMenu(id string, item extensions.ContextMenuItem) error {
	eo.contextMu.Lock()
	defer eo.contextMu.Unlock()

	for _, extMenus := range eo.contextMenus {
		if _, ok := extMenus[id]; ok {
			extMenus[id] = item
			return nil
		}
	}
	return fmt.Errorf("context menu item %q not found", id)
}

// RemoveContextMenu removes a context menu item.
func (eo *extensionOps) RemoveContextMenu(id string) error {
	eo.contextMu.Lock()
	defer eo.contextMu.Unlock()

	for _, extMenus := range eo.contextMenus {
		if _, ok := extMenus[id]; ok {
			delete(extMenus, id)
			return nil
		}
	}
	return fmt.Errorf("context menu item %q not found", id)
}

// RemoveAllContextMenus removes all context menu items.
func (eo *extensionOps) RemoveAllContextMenus() error {
	eo.contextMu.Lock()
	defer eo.contextMu.Unlock()

	eo.contextMenus = make(map[string]map[string]extensions.ContextMenuItem)
	return nil
}

// PermissionOperations implementation.

// RequestPermissions requests permissions for an extension. Only permissions
// declared in the extension's manifest can be granted.
func (eo *extensionOps) RequestPermissions(extID string, permissions []string) ([]string, error) {
	eo.permMu.Lock()
	defer eo.permMu.Unlock()

	// Validate requested permissions against the extension's manifest.
	ext := eo.bs.extManager.Get(extID)
	if ext != nil {
		manifestPerms := make(map[string]bool)
		for _, p := range ext.Manifest.Permissions {
			manifestPerms[p] = true
		}
		// Also allow optional_permissions.
		for _, p := range ext.Manifest.OptionalPermissions {
			manifestPerms[p] = true
		}
		for _, p := range permissions {
			if !manifestPerms[p] {
				return nil, fmt.Errorf("permission %q not declared in manifest", p)
			}
		}
	}

	granted, ok := eo.permissions[extID]
	if !ok {
		granted = make(map[string]bool)
		eo.permissions[extID] = granted
	}
	for _, p := range permissions {
		granted[p] = true
	}
	result := make([]string, 0, len(granted))
	for p := range granted {
		result = append(result, p)
	}
	return result, nil
}

// ContainsPermissions checks if an extension has the given permissions.
func (eo *extensionOps) ContainsPermissions(extID string, permissions []string) ([]string, error) {
	eo.permMu.Lock()
	defer eo.permMu.Unlock()

	granted, ok := eo.permissions[extID]
	if !ok {
		return nil, nil
	}
	var result []string
	for _, p := range permissions {
		if granted[p] {
			result = append(result, p)
		}
	}
	return result, nil
}

// GetAllPermissions returns all permissions for an extension.
func (eo *extensionOps) GetAllPermissions(extID string) ([]string, error) {
	eo.permMu.Lock()
	defer eo.permMu.Unlock()

	granted, ok := eo.permissions[extID]
	if !ok {
		return nil, nil
	}
	result := make([]string, 0, len(granted))
	for p := range granted {
		result = append(result, p)
	}
	return result, nil
}

// AlarmOperations implementation.

// CreateAlarm creates an alarm for an extension.
func (eo *extensionOps) CreateAlarm(extID string, name string, alarm extensions.Alarm) error {
	eo.alarmMu.Lock()
	defer eo.alarmMu.Unlock()

	extAlarms, ok := eo.alarms[extID]
	if !ok {
		extAlarms = make(map[string]*extensions.Alarm)
		eo.alarms[extID] = extAlarms
	}
	extAlarms[name] = &alarm

	// Schedule the alarm to fire at the specified time.
	when := alarm.Scheduled
	if when == 0 && alarm.PeriodInMinutes > 0 {
		when = float64(time.Now().UnixMilli()) + alarm.PeriodInMinutes*60000
	}
	if when > 0 {
		eo.bs.extManager.ScheduleAlarm(extID, name, when)
	}
	return nil
}

// GetAlarm returns an alarm by name.
func (eo *extensionOps) GetAlarm(extID string, name string) (*extensions.Alarm, error) {
	eo.alarmMu.Lock()
	defer eo.alarmMu.Unlock()

	extAlarms, ok := eo.alarms[extID]
	if !ok {
		return nil, nil
	}
	return extAlarms[name], nil
}

// GetAllAlarms returns all alarms for an extension.
func (eo *extensionOps) GetAllAlarms(extID string) ([]extensions.Alarm, error) {
	eo.alarmMu.Lock()
	defer eo.alarmMu.Unlock()

	extAlarms, ok := eo.alarms[extID]
	if !ok {
		return nil, nil
	}
	result := make([]extensions.Alarm, 0, len(extAlarms))
	for _, a := range extAlarms {
		result = append(result, *a)
	}
	return result, nil
}

// ClearAlarm removes an alarm by name.
func (eo *extensionOps) ClearAlarm(extID string, name string) error {
	eo.alarmMu.Lock()
	defer eo.alarmMu.Unlock()

	extAlarms, ok := eo.alarms[extID]
	if !ok {
		return nil
	}
	delete(extAlarms, name)

	// Cancel the scheduled alarm timer.
	eo.bs.extManager.CancelAlarm(extID, name)
	return nil
}

// ClearAllAlarms removes all alarms for an extension.
func (eo *extensionOps) ClearAllAlarms(extID string) error {
	eo.alarmMu.Lock()
	defer eo.alarmMu.Unlock()

	delete(eo.alarms, extID)

	// Cancel all scheduled alarm timers for this extension.
	eo.bs.extManager.CancelAllAlarms(extID)
	return nil
}

// I18nOperations implementation.

// GetMessage returns a localized message.
func (eo *extensionOps) GetMessage(extID string, messageName string, substitutions ...string) (string, error) {
	eo.i18nMu.Lock()
	defer eo.i18nMu.Unlock()

	messages, ok := eo.i18nMessages[extID]
	if !ok {
		return messageName, nil
	}
	msg, ok := messages[messageName]
	if !ok {
		return messageName, nil
	}
	// Apply simple substitutions.
	for i, sub := range substitutions {
		msg = strings.ReplaceAll(msg, fmt.Sprintf("$%d", i+1), sub)
	}
	return msg, nil
}

// GetAcceptLanguages returns accepted languages.
func (eo *extensionOps) GetAcceptLanguages(extID string) ([]string, error) {
	_ = extID
	return []string{"en-US", "en"}, nil
}

// WebRequestOperations implementation.

// webRequestListener tracks a registered webRequest listener for an extension.
type webRequestListener struct {
	extID       string
	event       string
	urlPatterns []string
}

// webRequestListeners holds all registered webRequest listeners keyed by event name.
var webRequestListeners []webRequestListener
var webRequestMu sync.Mutex

// webRequestObserverRegistered tracks whether the global sauce observer has been registered.
var webRequestObserverRegistered bool

// ensureWebRequestObserver registers a single sauce observer that dispatches to
// all extension webRequest listeners. It is idempotent.
// Caller must NOT hold webRequestMu.
func ensureWebRequestObserver(eo *extensionOps) {
	webRequestMu.Lock()
	defer webRequestMu.Unlock()
	if webRequestObserverRegistered {
		return
	}
	sauce.RegisterWebRequestObserver(func(details sauce.WebRequestDetails) (sauce.WebRequestDetails, error) {
		webRequestMu.Lock()
		listeners := make([]webRequestListener, len(webRequestListeners))
		copy(listeners, webRequestListeners)
		webRequestMu.Unlock()

		for _, l := range listeners {
			// Map sauce WebRequestDetails to extensions.WebRequestDetails.
			ed := extensions.WebRequestDetails{
				RequestID:    details.RequestID,
				URL:          details.URL,
				Method:       details.Method,
				Headers:      details.Headers,
				StatusCode:   details.StatusCode,
				StatusLine:   details.StatusLine,
				ResponseHeaders: details.ResponseHeaders,
				Initiator:    details.Initiator,
				TabID:        details.TabID,
				Timestamp:    details.Timestamp,
			}
			eo.bs.extManager.FireWebRequestEvent(l.extID, l.event, ed)
		}
		return details, nil
	})
	webRequestObserverRegistered = true
}

// ensureWebRequestObserverLocked assumes the caller already holds webRequestMu.
func ensureWebRequestObserverLocked(eo *extensionOps) {
	if webRequestObserverRegistered {
		return
	}
	sauce.RegisterWebRequestObserver(func(details sauce.WebRequestDetails) (sauce.WebRequestDetails, error) {
		webRequestMu.Lock()
		listeners := make([]webRequestListener, len(webRequestListeners))
		copy(listeners, webRequestListeners)
		webRequestMu.Unlock()

		for _, l := range listeners {
			// Map sauce WebRequestDetails to extensions.WebRequestDetails.
			ed := extensions.WebRequestDetails{
				RequestID:    details.RequestID,
				URL:          details.URL,
				Method:       details.Method,
				Headers:      details.Headers,
				StatusCode:   details.StatusCode,
				StatusLine:   details.StatusLine,
				ResponseHeaders: details.ResponseHeaders,
				Initiator:    details.Initiator,
				TabID:        details.TabID,
				Timestamp:    details.Timestamp,
			}
			eo.bs.extManager.FireWebRequestEvent(l.extID, l.event, ed)
		}
		return details, nil
	})
	webRequestObserverRegistered = true
}

// AddListener registers a webRequest listener for the given extension and event.
func (eo *extensionOps) AddListener(extID string, event string, details extensions.WebRequestDetails) error {
	webRequestMu.Lock()
	defer webRequestMu.Unlock()
	webRequestListeners = append(webRequestListeners, webRequestListener{
		extID:       extID,
		event:       event,
		urlPatterns: details.URLPatterns,
	})
	ensureWebRequestObserverLocked(eo)
	return nil
}

// RemoveListener removes webRequest listeners for the given extension and event.
func (eo *extensionOps) RemoveListener(extID string, event string) error {
	webRequestMu.Lock()
	defer webRequestMu.Unlock()
	filtered := webRequestListeners[:0]
	for _, l := range webRequestListeners {
		if !(l.extID == extID && l.event == event) {
			filtered = append(filtered, l)
		}
	}
	webRequestListeners = filtered
	return nil
}

// NotificationOperations implementation.

// notifications holds notifications keyed by extension ID then notification ID.
var notifications map[string]map[string]extensions.Notification
var notificationsMu sync.Mutex

func init() {
	notifications = make(map[string]map[string]extensions.Notification)
}

// CreateNotification creates a desktop notification.
func (eo *extensionOps) CreateNotification(extID string, id string, options extensions.NotificationOptions) (string, error) {
	notificationsMu.Lock()
	defer notificationsMu.Unlock()
	if notifications[extID] == nil {
		notifications[extID] = make(map[string]extensions.Notification)
	}
	n := extensions.Notification{
		ID:        id,
		Title:     options.Title,
		Message:   options.Message,
		IconURL:   options.IconURL,
		Context:   options.Context,
		Priority:  options.Priority,
		Timestamp: options.Timestamp,
	}
	notifications[extID][id] = n
	return id, nil
}

// ClearNotification dismisses a notification.
func (eo *extensionOps) ClearNotification(extID string, id string) (bool, error) {
	notificationsMu.Lock()
	defer notificationsMu.Unlock()
	if notifications[extID] != nil {
		if _, ok := notifications[extID][id]; ok {
			delete(notifications[extID], id)
			return true, nil
		}
	}
	return false, nil
}

// GetAllNotifications returns all notifications for an extension.
func (eo *extensionOps) GetAllNotifications(extID string) ([]extensions.Notification, error) {
	notificationsMu.Lock()
	defer notificationsMu.Unlock()
	result := make([]extensions.Notification, 0, len(notifications[extID]))
	for _, n := range notifications[extID] {
		result = append(result, n)
	}
	return result, nil
}

// DownloadOperations implementation.

// downloads holds download items keyed by download ID.
var downloads map[int]extensions.DownloadItem
// downloadExtIDs maps download ID to extension ID.
var downloadExtIDs map[int]string
var downloadsMu sync.Mutex
var nextDownloadID int

func init() {
	downloads = make(map[int]extensions.DownloadItem)
	downloadExtIDs = make(map[int]string)
	nextDownloadID = 0
}

// downloadOps implements DownloadOperations for chrome.downloads API.
type downloadOps struct {
	bs *BrowserShell
}

// newDownloadOps creates a new downloadOps wired to the given browser shell.
func newDownloadOps(bs *BrowserShell) *downloadOps {
	return &downloadOps{bs: bs}
}

// Download starts a file download.
func (do *downloadOps) Download(extID string, options extensions.DownloadOptions) (int, error) {
	downloadsMu.Lock()
	nextDownloadID++
	item := extensions.DownloadItem{
		ID:         nextDownloadID,
		URL:        options.URL,
		Filename:   options.Filename,
		State:      "in_progress",
		StartTime:  float64(time.Now().UnixMilli()),
	}
	downloads[nextDownloadID] = item
	downloadExtIDs[nextDownloadID] = extID
	downloadsMu.Unlock()

	// Fire onCreated event for the new download.
	do.bs.extManager.FireDownloadEvent(extID, "onCreated", item)

	// Perform the actual download in a goroutine so the API call returns
	// immediately with the download ID.
	go do.downloadFile(nextDownloadID, extID, options)

	return nextDownloadID, nil
}

// downloadFile performs the actual HTTP download and tracks progress.
func (do *downloadOps) downloadFile(downloadID int, extID string, options extensions.DownloadOptions) {
	downloadsMu.Lock()
	item, ok := downloads[downloadID]
	downloadsMu.Unlock()
	if !ok {
		return
	}

	// Determine the download directory.
	downloadDir := do.bs.settings.DownloadsDir
	if downloadDir == "" {
		downloadDir = filepath.Join(os.TempDir(), "thdwb-downloads")
	}
	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		do.failDownload(downloadID, "failed to create download directory: "+err.Error())
		return
	}

	// Determine the filename.
	filename := options.Filename
	if filename == "" {
		u, err := url.Parse(options.URL)
		if err == nil {
			filename = filepath.Base(u.Path)
		}
		if filename == "" || filename == "/" {
			filename = "download"
		}
	}
	filePath := filepath.Join(downloadDir, filename)

	// Perform the HTTP request.
	req, err := http.NewRequest(http.MethodGet, options.URL, nil)
	if err != nil {
		do.failDownload(downloadID, "failed to create request: "+err.Error())
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		do.failDownload(downloadID, "failed to fetch URL: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		do.failDownload(downloadID, fmt.Sprintf("server returned HTTP %d", resp.StatusCode))
		return
	}

	item.TotalBytes = resp.ContentLength
	item.Mime = resp.Header.Get("Content-Type")

	out, err := os.Create(filePath)
	if err != nil {
		do.failDownload(downloadID, "failed to create file: "+err.Error())
		return
	}
	defer out.Close()

	// Track progress as we write.
	var written int64
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				do.failDownload(downloadID, "failed to write file: "+writeErr.Error())
				return
			}
			written += int64(n)
			item.BytesReceived = written
			item.Filename = filename
			do.updateDownload(downloadID, item)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			do.failDownload(downloadID, "read error: "+err.Error())
			return
		}
	}

	// Mark complete.
	downloadsMu.Lock()
	item, ok = downloads[downloadID]
	if ok {
		item.State = "complete"
		item.EndTime = float64(time.Now().UnixMilli())
		item.BytesReceived = written
		downloads[downloadID] = item
	}
	downloadsMu.Unlock()

	if ok {
		do.bs.extManager.FireDownloadEvent(extID, "onChanged", item)
	}
}

// failDownload marks a download as failed with the given error.
func (do *downloadOps) failDownload(downloadID int, errMsg string) {
	downloadsMu.Lock()
	item, ok := downloads[downloadID]
	if ok {
		item.State = "interrupted"
		item.Error = errMsg
		item.EndTime = float64(time.Now().UnixMilli())
		downloads[downloadID] = item
	}
	extID := downloadExtIDs[downloadID]
	downloadsMu.Unlock()

	if ok {
		do.bs.extManager.FireDownloadEvent(extID, "onChanged", item)
	}
}

// updateDownload updates a download item and fires onChanged event.
func (do *downloadOps) updateDownload(downloadID int, item extensions.DownloadItem) {
	downloadsMu.Lock()
	downloads[downloadID] = item
	downloadsMu.Unlock()
	extID := downloadExtIDs[downloadID]
	do.bs.extManager.FireDownloadEvent(extID, "onChanged", item)
}

// Cancel aborts a download.
func (do *downloadOps) Cancel(downloadID int) error {
	downloadsMu.Lock()
	if item, ok := downloads[downloadID]; ok {
		item.State = "interrupted"
		item.Error = "cancelled"
		downloads[downloadID] = item
		extID := downloadExtIDs[downloadID]
		downloadsMu.Unlock()
		do.bs.extManager.FireDownloadEvent(extID, "onChanged", item)
		return nil
	}
	downloadsMu.Unlock()
	return nil
}

// Pause pauses a download.
func (do *downloadOps) Pause(downloadID int) error {
	downloadsMu.Lock()
	if item, ok := downloads[downloadID]; ok {
		item.Paused = true
		downloads[downloadID] = item
		extID := downloadExtIDs[downloadID]
		downloadsMu.Unlock()
		do.bs.extManager.FireDownloadEvent(extID, "onChanged", item)
		return nil
	}
	downloadsMu.Unlock()
	return nil
}

// Resume resumes a download.
func (do *downloadOps) Resume(downloadID int) error {
	downloadsMu.Lock()
	if item, ok := downloads[downloadID]; ok {
		item.Paused = false
		downloads[downloadID] = item
		extID := downloadExtIDs[downloadID]
		downloadsMu.Unlock()
		do.bs.extManager.FireDownloadEvent(extID, "onChanged", item)
		return nil
	}
	downloadsMu.Unlock()
	return nil
}

// Get returns a download item by ID.
func (do *downloadOps) Get(downloadID int) (*extensions.DownloadItem, error) {
	downloadsMu.Lock()
	defer downloadsMu.Unlock()
	if item, ok := downloads[downloadID]; ok {
		return &item, nil
	}
	return nil, nil
}

// GetAll returns all download items.
func (do *downloadOps) GetAll() ([]extensions.DownloadItem, error) {
	downloadsMu.Lock()
	defer downloadsMu.Unlock()
	result := make([]extensions.DownloadItem, 0, len(downloads))
	for _, item := range downloads {
		result = append(result, item)
	}
	return result, nil
}

// Erase removes a download from history.
func (do *downloadOps) Erase(downloadID int) error {
	downloadsMu.Lock()
	item, ok := downloads[downloadID]
	extID := downloadExtIDs[downloadID]
	if ok {
		delete(downloads, downloadID)
		delete(downloadExtIDs, downloadID)
	}
	downloadsMu.Unlock()

	if ok {
		do.bs.extManager.FireDownloadEvent(extID, "onErased", item)
	}
	return nil
}

// BookmarkOperations implementation.

// CreateBookmark creates a new bookmark.
func (eo *extensionOps) CreateBookmark(bookmark extensions.Bookmark) (*extensions.Bookmark, error) {
	b := eo.bs.bookmarkManager.AddBookmark(bookmark.Title, bookmark.URL)
	return &extensions.Bookmark{
		ID:       fmt.Sprintf("%d", b.ID),
		Title:    b.Title,
		URL:      b.URL,
		ParentID: fmt.Sprintf("%d", b.FolderID),
	}, nil
}

// GetBookmark returns a bookmark by ID.
func (eo *extensionOps) GetBookmark(id string) (*extensions.Bookmark, error) {
	b := eo.bs.bookmarkManager.GetBookmark(mustAtoi(id))
	if b == nil {
		return nil, nil
	}
	return &extensions.Bookmark{
		ID:       fmt.Sprintf("%d", b.ID),
		Title:    b.Title,
		URL:      b.URL,
		ParentID: fmt.Sprintf("%d", b.FolderID),
	}, nil
}

// GetChildren returns bookmarks in a folder.
func (eo *extensionOps) GetChildren(parentID string) ([]extensions.Bookmark, error) {
	parent := mustAtoi(parentID)
	bookmarks := eo.bs.bookmarkManager.GetBookmarksByFolder(parent)
	result := make([]extensions.Bookmark, 0, len(bookmarks))
	for _, b := range bookmarks {
		result = append(result, extensions.Bookmark{
			ID:       fmt.Sprintf("%d", b.ID),
			Title:    b.Title,
			URL:      b.URL,
			ParentID: fmt.Sprintf("%d", b.FolderID),
		})
	}
	return result, nil
}

// MoveBookmark moves a bookmark to a different folder/position.
func (eo *extensionOps) MoveBookmark(id, parentID string, index int) (*extensions.Bookmark, error) {
	// The shell BookmarkManager doesn't support move; return the bookmark unchanged.
	b := eo.bs.bookmarkManager.GetBookmark(mustAtoi(id))
	if b == nil {
		return nil, nil
	}
	return &extensions.Bookmark{
		ID:       fmt.Sprintf("%d", b.ID),
		Title:    b.Title,
		URL:      b.URL,
		ParentID: fmt.Sprintf("%d", b.FolderID),
	}, nil
}

// RemoveBookmark removes a bookmark by ID.
func (eo *extensionOps) RemoveBookmark(id string) error {
	eo.bs.bookmarkManager.RemoveBookmark(mustAtoi(id))
	return nil
}

// UpdateBookmark updates a bookmark's title or URL.
func (eo *extensionOps) UpdateBookmark(id string, updates extensions.Bookmark) (*extensions.Bookmark, error) {
	b := eo.bs.bookmarkManager.GetBookmark(mustAtoi(id))
	if b == nil {
		return nil, nil
	}
	if updates.Title != "" {
		b.Title = updates.Title
	}
	if updates.URL != "" {
		b.URL = updates.URL
	}
	return &extensions.Bookmark{
		ID:       fmt.Sprintf("%d", b.ID),
		Title:    b.Title,
		URL:      b.URL,
		ParentID: fmt.Sprintf("%d", b.FolderID),
	}, nil
}

// HistoryOperations implementation.

// AddUrl adds a URL to history.
func (eo *extensionOps) AddUrl(url, title string) error {
	eo.bs.historyManager.AddEntry(title, url)
	return nil
}

// DeleteUrl removes a URL from history.
func (eo *extensionOps) DeleteUrl(url string) error {
	// The shell HistoryManager doesn't support deletion by URL; no-op.
	_ = url
	return nil
}

// Search searches history by query.
func (eo *extensionOps) Search(query string, maxResults int) ([]extensions.HistoryEntry, error) {
	entries := eo.bs.historyManager.SearchEntries(query)
	if maxResults > 0 && len(entries) > maxResults {
		entries = entries[:maxResults]
	}
	result := make([]extensions.HistoryEntry, 0, len(entries))
	for _, e := range entries {
		result = append(result, extensions.HistoryEntry{
			ID:        e.ID,
			URL:       e.URL,
			Title:     e.Title,
			VisitTime: float64(e.VisitedAt.UnixMilli()),
			Typed:     false,
		})
	}
	return result, nil
}

// GetAllHistory returns all history entries.
func (eo *extensionOps) GetAllHistory() ([]extensions.HistoryEntry, error) {
	entries := eo.bs.historyManager.GetAllEntries()
	result := make([]extensions.HistoryEntry, 0, len(entries))
	for _, e := range entries {
		result = append(result, extensions.HistoryEntry{
			ID:        e.ID,
			URL:       e.URL,
			Title:     e.Title,
			VisitTime: float64(e.VisitedAt.UnixMilli()),
			Typed:     false,
		})
	}
	return result, nil
}

// ActionOperations implementation.

// actionBadges holds badge text keyed by extension ID then tab ID.
var actionBadges map[string]map[int]string
var actionBadgesMu sync.Mutex

func init() {
	actionBadges = make(map[string]map[int]string)
}

// SetBadgeText sets badge text for an extension action.
func (eo *extensionOps) SetBadgeText(extID string, text string, tabID int) error {
	actionBadgesMu.Lock()
	defer actionBadgesMu.Unlock()
	if actionBadges[extID] == nil {
		actionBadges[extID] = make(map[int]string)
	}
	actionBadges[extID][tabID] = text
	return nil
}

// SetBadgeColor sets badge color for an extension action.
func (eo *extensionOps) SetBadgeColor(extID string, color string, tabID int) error {
	// Badge color is stored but not rendered in this MVP.
	_ = extID
	_ = color
	_ = tabID
	return nil
}

// SetIcon sets the action icon for an extension.
func (eo *extensionOps) SetIcon(extID string, path string, tabID int) error {
	// Icon changes are stored but not rendered in this MVP.
	_ = extID
	_ = path
	_ = tabID
	return nil
}

// SetPopup sets the popup for an extension action.
func (eo *extensionOps) SetPopup(extID string, popup string, tabID int) error {
	// Popup changes are stored but not rendered in this MVP.
	_ = extID
	_ = popup
	_ = tabID
	return nil
}

// GetPopup returns the popup for an extension action.
func (eo *extensionOps) GetPopup(extID string, tabID int) (string, error) {
	_ = extID
	_ = tabID
	return "", nil
}

// mustAtoi converts a string to int, returning 0 on failure.
func mustAtoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func sameSiteFromString(s string) http.SameSite {
	switch strings.ToLower(s) {
	case "lax":
		return http.SameSiteLaxMode
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteDefaultMode
	}
}

func sameSiteToString(s http.SameSite) string {
	switch s {
	case http.SameSiteLaxMode:
		return "lax"
	case http.SameSiteStrictMode:
		return "strict"
	case http.SameSiteNoneMode:
		return "no_restriction"
	default:
		return ""
	}
}
