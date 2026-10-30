package shell

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

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
	// ContainerCookieJar does not expose single-cookie removal; clear and
	// re-add is not feasible without the full cookie list. This is a
	// best-effort no-op for now.
	_ = u
	return nil
}

// GetAllCookies returns all cookies for a container.
func (eo *extensionOps) GetAllCookies(storeID string) ([]extensions.Cookie, error) {
	// ContainerCookieJar doesn't expose all cookies for a container directly.
	// Return empty for now; individual URL-based queries are supported.
	return nil, nil
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

// RequestPermissions requests permissions for an extension.
func (eo *extensionOps) RequestPermissions(extID string, permissions []string) ([]string, error) {
	eo.permMu.Lock()
	defer eo.permMu.Unlock()

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
	return nil
}

// ClearAllAlarms removes all alarms for an extension.
func (eo *extensionOps) ClearAllAlarms(extID string) error {
	eo.alarmMu.Lock()
	defer eo.alarmMu.Unlock()

	delete(eo.alarms, extID)
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
