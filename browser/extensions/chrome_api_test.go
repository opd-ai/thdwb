package extensions

import (
	"os"
	"runtime"
	"testing"
	"time"

	goja "github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

// mockTabOps implements TabOperations for testing.
type mockTabOps struct {
	tabs      []TabInfo
	getErr    error
	updateErr error
	createErr error
	removeErr error
	activateErr error
}

func (m *mockTabOps) QueryTabs() []TabInfo {
	return m.tabs
}

func (m *mockTabOps) GetTab(tabID int) (*TabInfo, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, t := range m.tabs {
		if t.ID == tabID {
			return &t, nil
		}
	}
	return nil, nil
}

func (m *mockTabOps) UpdateTab(tabID int, updates TabUpdate) (*TabInfo, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	for _, t := range m.tabs {
		if t.ID == tabID {
			if updates.URL != "" {
				t.URL = updates.URL
			}
			if updates.Title != "" {
				t.Title = updates.Title
			}
			return &t, nil
		}
	}
	return nil, nil
}

func (m *mockTabOps) CreateTab(tab TabCreate) (*TabInfo, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	t := TabInfo{
		ID:        len(m.tabs) + 1,
		Title:     tab.Title,
		URL:       tab.URL,
		IsActive:  true,
		WindowID:  1,
	}
	m.tabs = append(m.tabs, t)
	return &t, nil
}

func (m *mockTabOps) RemoveTab(tabID int) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	for i, t := range m.tabs {
		if t.ID == tabID {
			m.tabs = append(m.tabs[:i], m.tabs[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockTabOps) ActivateTab(tabID int) (*TabInfo, error) {
	if m.activateErr != nil {
		return nil, m.activateErr
	}
	for _, t := range m.tabs {
		if t.ID == tabID {
			return &t, nil
		}
	}
	return nil, nil
}

// mockWinOps implements WindowOperations for testing.
type mockWinOps struct {
	windows   []WindowInfo
	getErr    error
	createErr error
	removeErr error
}

func (m *mockWinOps) GetCurrent() (*WindowInfo, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if len(m.windows) == 0 {
		return nil, nil
	}
	return &m.windows[0], nil
}

func (m *mockWinOps) GetAll() ([]WindowInfo, error) {
	return m.windows, nil
}

func (m *mockWinOps) Create(win WindowCreate) (*WindowInfo, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	w := WindowInfo{
		ID:        len(m.windows) + 1,
		Title:     win.Title,
		IsFocused: true,
		Width:     win.Width,
		Height:    win.Height,
	}
	m.windows = append(m.windows, w)
	return &w, nil
}

func (m *mockWinOps) Remove(windowID int) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	for i, w := range m.windows {
		if w.ID == windowID {
			m.windows = append(m.windows[:i], m.windows[i+1:]...)
			return nil
		}
	}
	return nil
}

// mockContainerOps implements ContainerOperations for testing.
type mockContainerOps struct {
	containers []ContainerInfo
	getErr     error
	createErr  error
	updateErr  error
	removeErr  error
}

func (m *mockContainerOps) QueryContainers() []ContainerInfo {
	return m.containers
}

func (m *mockContainerOps) GetContainer(id string) (*ContainerInfo, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, c := range m.containers {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, nil
}

func (m *mockContainerOps) CreateContainer(name, color string) (*ContainerInfo, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	c := ContainerInfo{
		ID:    "c" + string(rune(len(m.containers)+1)),
		Name:  name,
		Color: color,
	}
	m.containers = append(m.containers, c)
	return &c, nil
}

func (m *mockContainerOps) UpdateContainer(id, name, color string) (*ContainerInfo, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	for i, c := range m.containers {
		if c.ID == id {
			if name != "" {
				c.Name = name
			}
			if color != "" {
				c.Color = color
			}
			m.containers[i] = c
			return &c, nil
		}
	}
	return nil, nil
}

func (m *mockContainerOps) RemoveContainer(id string) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	for i, c := range m.containers {
		if c.ID == id {
			m.containers = append(m.containers[:i], m.containers[i+1:]...)
			return nil
		}
	}
	return nil
}

// mockCookieOps implements CookieOperations for testing.
type mockCookieOps struct {
	cookies   map[string][]Cookie // key: storeID
	stores    []CookieStore
	getErr    error
	setErr    error
	remErr    error
	clearErr  error
	storesErr error
}

func (m *mockCookieOps) GetCookies(storeID string, url string) ([]Cookie, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.cookies[storeID], nil
}

func (m *mockCookieOps) SetCookie(storeID string, cookie Cookie) error {
	if m.setErr != nil {
		return m.setErr
	}
	if m.cookies == nil {
		m.cookies = make(map[string][]Cookie)
	}
	m.cookies[storeID] = append(m.cookies[storeID], cookie)
	return nil
}

func (m *mockCookieOps) RemoveCookie(storeID string, cookie Cookie) error {
	if m.remErr != nil {
		return m.remErr
	}
	cookies := m.cookies[storeID]
	for i, c := range cookies {
		if c.Name == cookie.Name {
			m.cookies[storeID] = append(cookies[:i], cookies[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockCookieOps) GetAllCookies(storeID string) ([]Cookie, error) {
	return m.cookies[storeID], nil
}

func (m *mockCookieOps) ClearCookies(storeID string) error {
	if m.clearErr != nil {
		return m.clearErr
	}
	delete(m.cookies, storeID)
	return nil
}

func (m *mockCookieOps) GetAllCookieStores() ([]CookieStore, error) {
	if m.storesErr != nil {
		return nil, m.storesErr
	}
	return m.stores, nil
}

// runJS executes JS source in the VM and returns the last expression value.
func runJS(t *testing.T, vm *goja.Runtime, src string) goja.Value {
	t.Helper()
	val, err := vm.RunString(src)
	require.NoError(t, err)
	return val
}

// getGlobal gets a global variable value from the VM.
func getGlobal(t *testing.T, vm *goja.Runtime, name string) goja.Value {
	t.Helper()
	return vm.Get(name)
}

func TestChromeAPIRuntimeID(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())

	require.NoError(t, api.Bind())

	runJS(t, vm, `var rid = chrome.runtime.id;`)
	require.Equal(t, "ext1", getGlobal(t, vm, "rid").String())
}

func TestChromeAPIRuntimeGetManifest(t *testing.T) {
	m := &Manifest{
		ManifestVersion: ManifestV3,
		Name:            "Test Ext",
		Version:         "1.0.0",
		Description:     "A test extension",
		Permissions:     []string{"tabs", "storage"},
		Icons:           map[string]string{"16": "icon16.png"},
	}
	ext := NewExtension("ext1", "/p", m)
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())

	require.NoError(t, api.Bind())

	runJS(t, vm, `var manifest = chrome.runtime.getManifest();`)
	manifest := getGlobal(t, vm, "manifest").ToObject(vm)
	require.Equal(t, "Test Ext", manifest.Get("name").String())
	require.Equal(t, "1.0.0", manifest.Get("version").String())
	require.Equal(t, "A test extension", manifest.Get("description").String())
	perms := manifest.Get("permissions").ToObject(vm)
	require.Equal(t, "tabs", perms.Get("0").String())
	require.Equal(t, "storage", perms.Get("1").String())
}

func TestChromeAPIRuntimeGetURL(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/home/user/ext", m)
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())

	require.NoError(t, api.Bind())

	runJS(t, vm, `var url = chrome.runtime.getURL("foo.js");`)
	require.Equal(t, "/home/user/ext/foo.js", getGlobal(t, vm, "url").String())
}

func TestChromeAPIRuntimeGetPlatformInfo(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())

	require.NoError(t, api.Bind())

	runJS(t, vm, `var platform = chrome.runtime.getPlatformInfo();`)
	platform := getGlobal(t, vm, "platform").ToObject(vm)
	require.Equal(t, runtime.GOOS, platform.Get("os").String())
	require.Equal(t, runtime.GOARCH, platform.Get("arch").String())
	require.Equal(t, runtime.GOOS+"/"+runtime.GOARCH, platform.Get("platform").String())
}

func TestChromeAPIRuntimeSendMessage(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	router := NewMessageRouter()
	vm := goja.New()
	api := NewChromeAPI(ext, vm, router)

	require.NoError(t, api.Bind())

	var received []Message
	router.RegisterHandler(ext.ID, func(msg Message) error {
		received = append(received, msg)
		return nil
	})

	runJS(t, vm, `chrome.runtime.sendMessage({hello: "world"});`)

	require.Len(t, received, 1)
	require.Equal(t, ext.ID, received[0].Sender)
	require.Equal(t, "world", received[0].Payload.(map[string]interface{})["hello"])
}

func TestChromeAPIRuntimeOnMessage(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	router := NewMessageRouter()
	vm := goja.New()
	api := NewChromeAPI(ext, vm, router)

	require.NoError(t, api.Bind())

	// Register onMessage listener at the Go level for this extension.
	var received []interface{}
	router.RegisterHandler(ext.ID, func(msg Message) error {
		received = append(received, msg.Payload)
		return nil
	})

	// Route a message from this extension to itself (same extension).
	err := router.Route(Message{
		Sender:  ext.ID,
		Target:  ext.ID,
		Type:    "background",
		Payload: map[string]interface{}{"msg": "hi"},
	})
	require.NoError(t, err)

	require.Len(t, received, 1)
	require.Equal(t, "hi", received[0].(map[string]interface{})["msg"])

	// Cross-extension messaging is blocked.
	err = router.Route(Message{
		Sender:  "ext2",
		Target:  ext.ID,
		Type:    "background",
		Payload: map[string]interface{}{"msg": "blocked"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cross-extension messaging blocked")
	require.Len(t, received, 1) // no additional delivery
}

func TestChromeAPIStorageLocal(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	ext.Storage = NewStoragePartition("ext1", "")
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())

	require.NoError(t, api.Bind())

	runJS(t, vm, `
		chrome.storage.local.set({theme: "dark"});
		var theme = chrome.storage.local.get("theme");
		var all = chrome.storage.local.get();
		chrome.storage.local.remove("theme");
		var afterRemove = chrome.storage.local.get("theme");
		chrome.storage.local.set({a: "1", b: "2"});
		chrome.storage.local.clear();
		var afterClear = chrome.storage.local.get();
	`)

	theme := getGlobal(t, vm, "theme").ToObject(vm)
	require.Equal(t, "dark", theme.Get("theme").String())

	all := getGlobal(t, vm, "all").ToObject(vm)
	require.Equal(t, "dark", all.Get("theme").String())

	afterRemove := getGlobal(t, vm, "afterRemove").ToObject(vm)
	require.Equal(t, 0, len(afterRemove.Keys()))

	afterClear := getGlobal(t, vm, "afterClear").ToObject(vm)
	require.Equal(t, 0, len(afterClear.Keys()))
}

func TestChromeAPIStorageSync(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	ext.Storage = NewStoragePartition("ext1", "")
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())

	require.NoError(t, api.Bind())

	runJS(t, vm, `
		chrome.storage.sync.set({key: "value"});
		var val = chrome.storage.sync.get("key");
	`)
	v := getGlobal(t, vm, "val").ToObject(vm)
	require.Equal(t, "value", v.Get("key").String())
}

func TestChromeAPIStorageManaged(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	ext.Storage = NewStoragePartition("ext1", "")
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())

	require.NoError(t, api.Bind())

	runJS(t, vm, `
		chrome.storage.managed.set({policy: "strict"});
		var pol = chrome.storage.managed.get("policy");
	`)
	p := getGlobal(t, vm, "pol").ToObject(vm)
	require.Equal(t, "strict", p.Get("policy").String())
}

func TestChromeAPITabsQuery(t *testing.T) {
	tabOps := &mockTabOps{
		tabs: []TabInfo{
			{ID: 1, Title: "Tab 1", URL: "https://example.com", IsActive: true, WindowID: 1},
			{ID: 2, Title: "Tab 2", URL: "https://example.org", IsActive: false, WindowID: 1},
		},
	}

	tabs := tabOps.QueryTabs()
	require.Len(t, tabs, 2)
	require.Equal(t, 1, tabs[0].ID)
	require.Equal(t, "Tab 1", tabs[0].Title)
	require.Equal(t, true, tabs[0].IsActive)
}

func TestChromeAPITabsGet(t *testing.T) {
	tabOps := &mockTabOps{
		tabs: []TabInfo{
			{ID: 1, Title: "Tab 1", URL: "https://example.com", IsActive: true, WindowID: 1},
		},
	}

	tab, err := tabOps.GetTab(1)
	require.NoError(t, err)
	require.NotNil(t, tab)
	require.Equal(t, 1, tab.ID)
	require.Equal(t, "Tab 1", tab.Title)
	require.Equal(t, "https://example.com", tab.URL)
}

func TestChromeAPITabsGetNotFound(t *testing.T) {
	tabOps := &mockTabOps{}

	tab, err := tabOps.GetTab(999)
	require.NoError(t, err)
	require.Nil(t, tab)
}

func TestChromeAPITabsUpdate(t *testing.T) {
	tabOps := &mockTabOps{
		tabs: []TabInfo{
			{ID: 1, Title: "Tab 1", URL: "https://example.com", IsActive: true, WindowID: 1},
		},
	}

	tab, err := tabOps.UpdateTab(1, TabUpdate{URL: "https://updated.com"})
	require.NoError(t, err)
	require.NotNil(t, tab)
	require.Equal(t, "https://updated.com", tab.URL)
}

func TestChromeAPITabsCreate(t *testing.T) {
	tabOps := &mockTabOps{}

	tab, err := tabOps.CreateTab(TabCreate{URL: "https://newtab.com", Title: "New Tab"})
	require.NoError(t, err)
	require.NotNil(t, tab)
	require.Equal(t, "https://newtab.com", tab.URL)
	require.Equal(t, "New Tab", tab.Title)
	require.Equal(t, 1, tab.ID)
}

func TestChromeAPITabsRemove(t *testing.T) {
	tabOps := &mockTabOps{
		tabs: []TabInfo{
			{ID: 1, Title: "Tab 1", URL: "https://example.com", IsActive: true, WindowID: 1},
		},
	}

	require.NoError(t, tabOps.RemoveTab(1))
	require.Empty(t, tabOps.QueryTabs())
}

func TestChromeAPIWindowsGetCurrent(t *testing.T) {
	winOps := &mockWinOps{
		windows: []WindowInfo{
			{ID: 1, Title: "Window 1", IsFocused: true, Width: 1024, Height: 768},
		},
	}

	win, err := winOps.GetCurrent()
	require.NoError(t, err)
	require.NotNil(t, win)
	require.Equal(t, 1, win.ID)
	require.Equal(t, "Window 1", win.Title)
	require.Equal(t, true, win.IsFocused)
	require.Equal(t, 1024, win.Width)
	require.Equal(t, 768, win.Height)
}

func TestChromeAPIWindowsGetAll(t *testing.T) {
	winOps := &mockWinOps{
		windows: []WindowInfo{
			{ID: 1, Title: "Window 1", IsFocused: true, Width: 1024, Height: 768},
			{ID: 2, Title: "Window 2", IsFocused: false, Width: 800, Height: 600},
		},
	}

	wins, err := winOps.GetAll()
	require.NoError(t, err)
	require.Len(t, wins, 2)
	require.Equal(t, 1, wins[0].ID)
	require.Equal(t, 2, wins[1].ID)
}

func TestChromeAPIWindowsCreate(t *testing.T) {
	winOps := &mockWinOps{}

	win, err := winOps.Create(WindowCreate{Title: "New Window", Width: 800, Height: 600})
	require.NoError(t, err)
	require.NotNil(t, win)
	require.Equal(t, "New Window", win.Title)
	require.Equal(t, 800, win.Width)
	require.Equal(t, 600, win.Height)
	require.Equal(t, 1, win.ID)
}

func TestChromeAPIWindowsRemove(t *testing.T) {
	winOps := &mockWinOps{
		windows: []WindowInfo{
			{ID: 1, Title: "Window 1", IsFocused: true, Width: 1024, Height: 768},
		},
	}

	require.NoError(t, winOps.Remove(1))
	wins, err := winOps.GetAll()
	require.NoError(t, err)
	require.Empty(t, wins)
}

func TestChromeAPIManagerLifecycle(t *testing.T) {
	root := t.TempDir()
	storageDir := root + "/storage"
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	writeExtDir(t, root, "ext-lifecycle", `{
		"manifest_version": 3,
		"name": "Lifecycle Test",
		"version": "1.0",
		"background": {
			"scripts": ["background.js"],
			"persistent": false
		}
	}`)
	require.NoError(t, os.WriteFile(root+"/ext-lifecycle/background.js", []byte(`chrome.runtime.id = "ext-lifecycle";`), 0o644))

	m := NewManager(storageDir)
	m.AddWatchDir(root)
	_, err := m.Scan()
	require.NoError(t, err)

	// Enable the extension - should initialize Chrome API bindings.
	require.NoError(t, m.Enable("ext-lifecycle"))
	require.True(t, m.IsEnabled("ext-lifecycle"))

	ext := m.Get("ext-lifecycle")
	require.NotNil(t, ext.BackgroundVM)

	// Verify chrome API is bound.
	vm := ext.BackgroundVM
	chrome := vm.Get("chrome")
	require.NotNil(t, chrome)
	runtimeObj := chrome.ToObject(vm).Get("runtime").ToObject(vm)
	idVal := runtimeObj.Get("id")
	require.Equal(t, "ext-lifecycle", idVal.String())

	// Disable - should clean up.
	require.NoError(t, m.Disable("ext-lifecycle"))
	require.False(t, m.IsEnabled("ext-lifecycle"))
}

func TestChromeAPIManagerMessageRouting(t *testing.T) {
	root := t.TempDir()
	storageDir := root + "/storage"
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	writeExtDir(t, root, "ext-msg1", `{
		"manifest_version": 3,
		"name": "Msg Ext 1",
		"version": "1.0"
	}`)
	writeExtDir(t, root, "ext-msg2", `{
		"manifest_version": 3,
		"name": "Msg Ext 2",
		"version": "1.0"
	}`)

	m := NewManager(storageDir)
	m.AddWatchDir(root)
	_, err := m.Scan()
	require.NoError(t, err)

	require.NoError(t, m.Enable("ext-msg1"))
	require.NoError(t, m.Enable("ext-msg2"))

	ext1 := m.Get("ext-msg1")
	ext2 := m.Get("ext-msg2")

	// Send a message from ext1's VM to ext1's own handlers (same extension).
	vm1 := ext1.BackgroundVM

	// Register onMessage listener in ext1 at the Go level.
	var received1 []interface{}
	router := m.MessageRouter()
	router.RegisterHandler(ext1.ID, func(msg Message) error {
		received1 = append(received1, msg.Payload)
		return nil
	})

	// Register onMessage listener in ext2 at the Go level.
	var received2 []interface{}
	router.RegisterHandler(ext2.ID, func(msg Message) error {
		received2 = append(received2, msg.Payload)
		return nil
	})

	// Send a message from ext1's VM (broadcast within ext1).
	runJS(t, vm1, `chrome.runtime.sendMessage({hello: "world"});`)

	require.Len(t, received1, 1)
	require.Equal(t, "world", received1[0].(map[string]interface{})["hello"])
	require.Empty(t, received2) // ext2 should not receive ext1's messages

	// Send another message from ext1's VM.
	runJS(t, vm1, `chrome.runtime.sendMessage({hello: "cross"});`)

	require.Len(t, received1, 2)
	require.Equal(t, "cross", received1[1].(map[string]interface{})["hello"])
	require.Empty(t, received2) // ext2 still receives nothing
}

func TestChromeAPIContextualIdentitiesQuery(t *testing.T) {
	containerOps := &mockContainerOps{
		containers: []ContainerInfo{
			{ID: "c1", Name: "Work", Color: "#ff0000"},
			{ID: "c2", Name: "Personal", Color: "#00ff00"},
		},
	}

	containers := containerOps.QueryContainers()
	require.Len(t, containers, 2)
	require.Equal(t, "c1", containers[0].ID)
	require.Equal(t, "Work", containers[0].Name)
	require.Equal(t, "#ff0000", containers[0].Color)
	require.Equal(t, "c2", containers[1].ID)
	require.Equal(t, "Personal", containers[1].Name)
}

func TestChromeAPIContextualIdentitiesGet(t *testing.T) {
	containerOps := &mockContainerOps{
		containers: []ContainerInfo{
			{ID: "c1", Name: "Work", Color: "#ff0000"},
		},
	}

	c, err := containerOps.GetContainer("c1")
	require.NoError(t, err)
	require.NotNil(t, c)
	require.Equal(t, "c1", c.ID)
	require.Equal(t, "Work", c.Name)
	require.Equal(t, "#ff0000", c.Color)

	c, err = containerOps.GetContainer("nonexistent")
	require.NoError(t, err)
	require.Nil(t, c)
}

func TestChromeAPIContextualIdentitiesCreate(t *testing.T) {
	containerOps := &mockContainerOps{}

	c, err := containerOps.CreateContainer("Shopping", "#0000ff")
	require.NoError(t, err)
	require.NotNil(t, c)
	require.Equal(t, "Shopping", c.Name)
	require.Equal(t, "#0000ff", c.Color)
	require.NotEmpty(t, c.ID)

	require.Len(t, containerOps.containers, 1)
}

func TestChromeAPIContextualIdentitiesUpdate(t *testing.T) {
	containerOps := &mockContainerOps{
		containers: []ContainerInfo{
			{ID: "c1", Name: "Work", Color: "#ff0000"},
		},
	}

	c, err := containerOps.UpdateContainer("c1", "Work2", "")
	require.NoError(t, err)
	require.NotNil(t, c)
	require.Equal(t, "Work2", c.Name)
	require.Equal(t, "#ff0000", c.Color)

	c, err = containerOps.UpdateContainer("nonexistent", "x", "y")
	require.NoError(t, err)
	require.Nil(t, c)
}

func TestChromeAPIContextualIdentitiesRemove(t *testing.T) {
	containerOps := &mockContainerOps{
		containers: []ContainerInfo{
			{ID: "c1", Name: "Work", Color: "#ff0000"},
		},
	}

	require.NoError(t, containerOps.RemoveContainer("c1"))
	require.Len(t, containerOps.containers, 0)

	require.NoError(t, containerOps.RemoveContainer("nonexistent"))
}

func TestChromeAPICookiesGet(t *testing.T) {
	cookieOps := &mockCookieOps{
		cookies: map[string][]Cookie{
			"c1": {
				{Name: "session", Value: "abc123", Domain: "example.com", Path: "/", StoreID: "c1"},
			},
		},
	}

	cookies, err := cookieOps.GetCookies("c1", "https://example.com/")
	require.NoError(t, err)
	require.Len(t, cookies, 1)
	require.Equal(t, "session", cookies[0].Name)
	require.Equal(t, "abc123", cookies[0].Value)
}

func TestChromeAPICookiesGetNotFound(t *testing.T) {
	cookieOps := &mockCookieOps{}

	cookies, err := cookieOps.GetCookies("c1", "https://example.com/")
	require.NoError(t, err)
	require.Nil(t, cookies)
}

func TestChromeAPICookiesSet(t *testing.T) {
	cookieOps := &mockCookieOps{}

	err := cookieOps.SetCookie("c1", Cookie{
		Name:  "session",
		Value: "abc123",
		StoreID: "c1",
	})
	require.NoError(t, err)
	require.Len(t, cookieOps.cookies["c1"], 1)
	require.Equal(t, "session", cookieOps.cookies["c1"][0].Name)
}

func TestChromeAPICookiesRemove(t *testing.T) {
	cookieOps := &mockCookieOps{
		cookies: map[string][]Cookie{
			"c1": {
				{Name: "session", Value: "abc123", Domain: "example.com", Path: "/", StoreID: "c1"},
			},
		},
	}

	require.NoError(t, cookieOps.RemoveCookie("c1", Cookie{Name: "session", StoreID: "c1"}))
	require.Len(t, cookieOps.cookies["c1"], 0)
}

func TestChromeAPICookiesGetAll(t *testing.T) {
	cookieOps := &mockCookieOps{
		cookies: map[string][]Cookie{
			"c1": {
				{Name: "session", Value: "abc123", Domain: "example.com", Path: "/", StoreID: "c1"},
				{Name: "theme", Value: "dark", Domain: "example.com", Path: "/", StoreID: "c1"},
			},
		},
	}

	cookies, err := cookieOps.GetAllCookies("c1")
	require.NoError(t, err)
	require.Len(t, cookies, 2)
}

func TestChromeAPICookiesClear(t *testing.T) {
	cookieOps := &mockCookieOps{
		cookies: map[string][]Cookie{
			"c1": {
				{Name: "session", Value: "abc123", Domain: "example.com", Path: "/", StoreID: "c1"},
			},
		},
	}

	require.NoError(t, cookieOps.ClearCookies("c1"))
	require.Len(t, cookieOps.cookies["c1"], 0)
}

// mockWebRequestOps implements WebRequestOperations for testing.
type mockWebRequestOps struct {
	addErr    error
	removeErr error
	addCalls  int
	removeCalls int
	listeners map[string]bool
}

func (m *mockWebRequestOps) AddListener(extID string, event string, details WebRequestDetails) error {
	m.addCalls++
	if m.listeners == nil {
		m.listeners = make(map[string]bool)
	}
	if m.addErr != nil {
		return m.addErr
	}
	m.listeners[event] = true
	return nil
}

func (m *mockWebRequestOps) RemoveListener(extID string, event string) error {
	m.removeCalls++
	if m.removeErr != nil {
		return m.removeErr
	}
	delete(m.listeners, event)
	return nil
}

// mockNotificationOps implements NotificationOperations for testing.
type mockNotificationOps struct {
	notifications map[string]Notification
	createErr     error
	clearErr      error
	getAllErr     error
	createCalls   int
	clearCalls    int
	getAllCalls   int
}

func (m *mockNotificationOps) CreateNotification(extID string, id string, options NotificationOptions) (string, error) {
	m.createCalls++
	if m.createErr != nil {
		return "", m.createErr
	}
	if m.notifications == nil {
		m.notifications = make(map[string]Notification)
	}
	n := Notification{
		ID:        id,
		Title:     options.Title,
		Message:   options.Message,
		IconURL:   options.IconURL,
		Context:   options.Context,
		Priority:  options.Priority,
		Timestamp: options.Timestamp,
	}
	m.notifications[id] = n
	return id, nil
}

func (m *mockNotificationOps) ClearNotification(extID string, id string) (bool, error) {
	m.clearCalls++
	if m.clearErr != nil {
		return false, m.clearErr
	}
	if _, ok := m.notifications[id]; ok {
		delete(m.notifications, id)
		return true, nil
	}
	return false, nil
}

func (m *mockNotificationOps) GetAllNotifications(extID string) ([]Notification, error) {
	m.getAllCalls++
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	result := make([]Notification, 0, len(m.notifications))
	for _, n := range m.notifications {
		result = append(result, n)
	}
	return result, nil
}

// mockDownloadOps implements DownloadOperations for testing.
type mockDownloadOps struct {
	downloads    map[int]DownloadItem
	nextID       int
	downloadErr  error
	cancelErr    error
	pauseErr     error
	resumeErr    error
	getErr       error
	getAllErr    error
	eraseErr     error
	cancelCalls  int
	pauseCalls   int
	resumeCalls  int
	getCalls     int
	getAllCalls  int
	eraseCalls   int
}

func (m *mockDownloadOps) Download(extID string, options DownloadOptions) (int, error) {
	if m.downloadErr != nil {
		return 0, m.downloadErr
	}
	if m.downloads == nil {
		m.downloads = make(map[int]DownloadItem)
	}
	m.nextID++
	item := DownloadItem{
		ID:         m.nextID,
		URL:        options.URL,
		Filename:   options.Filename,
		State:      "in_progress",
		StartTime:  float64(time.Now().UnixMilli()),
	}
	m.downloads[m.nextID] = item
	return m.nextID, nil
}

func (m *mockDownloadOps) Cancel(downloadID int) error {
	m.cancelCalls++
	if m.cancelErr != nil {
		return m.cancelErr
	}
	if item, ok := m.downloads[downloadID]; ok {
		item.State = "interrupted"
		item.Error = "cancelled"
		m.downloads[downloadID] = item
	}
	return nil
}

func (m *mockDownloadOps) Pause(downloadID int) error {
	m.pauseCalls++
	if m.pauseErr != nil {
		return m.pauseErr
	}
	if item, ok := m.downloads[downloadID]; ok {
		item.Paused = true
		m.downloads[downloadID] = item
	}
	return nil
}

func (m *mockDownloadOps) Resume(downloadID int) error {
	m.resumeCalls++
	if m.resumeErr != nil {
		return m.resumeErr
	}
	if item, ok := m.downloads[downloadID]; ok {
		item.Paused = false
		m.downloads[downloadID] = item
	}
	return nil
}

func (m *mockDownloadOps) Get(downloadID int) (*DownloadItem, error) {
	m.getCalls++
	if m.getErr != nil {
		return nil, m.getErr
	}
	if item, ok := m.downloads[downloadID]; ok {
		return &item, nil
	}
	return nil, nil
}

func (m *mockDownloadOps) GetAll() ([]DownloadItem, error) {
	m.getAllCalls++
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	result := make([]DownloadItem, 0, len(m.downloads))
	for _, item := range m.downloads {
		result = append(result, item)
	}
	return result, nil
}

func (m *mockDownloadOps) Erase(downloadID int) error {
	m.eraseCalls++
	if m.eraseErr != nil {
		return m.eraseErr
	}
	delete(m.downloads, downloadID)
	return nil
}

func TestChromeAPIWebRequestAddListener(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	webRequestOps := &mockWebRequestOps{}
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetWebRequestOperations(webRequestOps)

	require.NoError(t, api.Bind())

	runJS(t, vm, `
		chrome.webRequest.onBeforeRequest.addListener(
			function(details) { return {cancel: false}; },
			{urls: ["<all_urls>"]}
		);
	`)

	require.Equal(t, 1, webRequestOps.addCalls)
	require.True(t, webRequestOps.listeners["onBeforeRequest"])
}

func TestChromeAPIWebRequestRemoveListener(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	webRequestOps := &mockWebRequestOps{listeners: map[string]bool{"onBeforeRequest": true}}
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetWebRequestOperations(webRequestOps)

	require.NoError(t, api.Bind())

	runJS(t, vm, `
		chrome.webRequest.onBeforeRequest.removeListener(function() {});
	`)

	require.Equal(t, 1, webRequestOps.removeCalls)
	require.False(t, webRequestOps.listeners["onBeforeRequest"])
}

func TestChromeAPINotificationsCreate(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	notifOps := &mockNotificationOps{}
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetNotificationOperations(notifOps)

	require.NoError(t, api.Bind())

	runJS(t, vm, `
		var id = chrome.notifications.create("notif-1", {
			title: "Test",
			message: "Hello",
			iconUrl: "icon.png",
			priority: 1
		});
	`)

	require.Equal(t, 1, notifOps.createCalls)
	require.Equal(t, "notif-1", getGlobal(t, vm, "id").String())
	require.Equal(t, "Test", notifOps.notifications["notif-1"].Title)
	require.Equal(t, "Hello", notifOps.notifications["notif-1"].Message)
	require.Equal(t, 1, notifOps.notifications["notif-1"].Priority)
}

func TestChromeAPINotificationsClear(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	notifOps := &mockNotificationOps{
		notifications: map[string]Notification{
			"notif-1": {ID: "notif-1", Title: "Test", Message: "Hello"},
		},
	}
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetNotificationOperations(notifOps)

	require.NoError(t, api.Bind())

	runJS(t, vm, `
		var cleared = chrome.notifications.clear("notif-1");
	`)

	require.Equal(t, 1, notifOps.clearCalls)
	require.True(t, getGlobal(t, vm, "cleared").ToBoolean())
	_, exists := notifOps.notifications["notif-1"]
	require.False(t, exists)
}

func TestChromeAPINotificationsGetAll(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	notifOps := &mockNotificationOps{
		notifications: map[string]Notification{
			"notif-1": {ID: "notif-1", Title: "Test", Message: "Hello"},
			"notif-2": {ID: "notif-2", Title: "Test2", Message: "World"},
		},
	}
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetNotificationOperations(notifOps)

	require.NoError(t, api.Bind())

	runJS(t, vm, `
		var all = chrome.notifications.getAll();
	`)

	require.Equal(t, 1, notifOps.getAllCalls)
	all := getGlobal(t, vm, "all").ToObject(vm)
	require.Equal(t, 2, len(all.Keys()))
}

func TestChromeAPIDownloadsDownload(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	dlOps := &mockDownloadOps{}
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetDownloadOperations(dlOps)

	require.NoError(t, api.Bind())

	runJS(t, vm, `
		var id = chrome.downloads.download({
			url: "https://example.com/file.zip",
			filename: "file.zip",
			saveAs: false
		});
	`)

	require.Equal(t, 1, dlOps.nextID)
	require.Equal(t, int64(dlOps.nextID), getGlobal(t, vm, "id").ToInteger())
	require.Equal(t, "https://example.com/file.zip", dlOps.downloads[1].URL)
	require.Equal(t, "file.zip", dlOps.downloads[1].Filename)
	require.Equal(t, "in_progress", dlOps.downloads[1].State)
}

func TestChromeAPIDownloadsCancel(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	dlOps := &mockDownloadOps{
		downloads: map[int]DownloadItem{
			1: {ID: 1, URL: "https://example.com/file.zip", State: "in_progress"},
		},
		nextID: 1,
	}
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetDownloadOperations(dlOps)

	require.NoError(t, api.Bind())

	runJS(t, vm, `
		chrome.downloads.cancel(1);
	`)

	require.Equal(t, 1, dlOps.cancelCalls)
	require.Equal(t, "interrupted", dlOps.downloads[1].State)
	require.Equal(t, "cancelled", dlOps.downloads[1].Error)
}

func TestChromeAPIDownloadsGetAll(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	dlOps := &mockDownloadOps{
		downloads: map[int]DownloadItem{
			1: {ID: 1, URL: "https://example.com/file.zip", State: "complete"},
			2: {ID: 2, URL: "https://example.com/image.png", State: "in_progress"},
		},
		nextID: 2,
	}
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetDownloadOperations(dlOps)

	require.NoError(t, api.Bind())

	runJS(t, vm, `
		var all = chrome.downloads.getAll();
	`)

	require.Equal(t, 1, dlOps.getAllCalls)
	all := getGlobal(t, vm, "all").ToObject(vm)
	require.Equal(t, 2, len(all.Keys()))
}

func TestChromeAPICookiesGetAllCookieStores(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	cookieOps := &mockCookieOps{
		stores: []CookieStore{
			{ID: "c1", Incognito: false},
			{ID: "c2", Incognito: true},
		},
	}
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetCookieOperations(cookieOps)

	require.NoError(t, api.Bind())

	runJS(t, vm, `
		var stores = chrome.cookies.getAllCookieStores();
	`)

	stores := getGlobal(t, vm, "stores").ToObject(vm)
	require.Equal(t, 2, len(stores.Keys()))
	require.Equal(t, "c1", stores.Get("0").ToObject(vm).Get("id").String())
	require.Equal(t, "c2", stores.Get("1").ToObject(vm).Get("id").String())
	require.True(t, stores.Get("1").ToObject(vm).Get("incognito").ToBoolean())
}

func TestChromeAPITabsOnCreated(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	tabOps := &mockTabOps{}
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetTabOperations(tabOps)

	require.NoError(t, api.Bind())

	// Register onCreated listener.
	runJS(t, vm, `
		chrome.tabs.onCreated.addListener(function(tab) {
			__createdTab = tab;
		});
	`)

	// Create a tab.
	tab, err := tabOps.CreateTab(TabCreate{URL: "https://example.com", Title: "Example"})
	require.NoError(t, err)
	require.NotNil(t, tab)

	// Fire the event manually (as the shell would).
	api.FireTabCreated(tab)

	createdTabVal := getGlobal(t, vm, "__createdTab")
	require.NotNil(t, createdTabVal)
	require.NotEqual(t, goja.Undefined(), createdTabVal)
	createdTab := createdTabVal.ToObject(vm)
	require.NotNil(t, createdTab)
	require.Equal(t, float64(tab.ID), createdTab.Get("id").ToFloat())
	require.Equal(t, "https://example.com", createdTab.Get("url").String())
	require.Equal(t, "Example", createdTab.Get("title").String())
}

func TestChromeAPITabsOnRemoved(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	tabOps := &mockTabOps{
		tabs: []TabInfo{{ID: 1, Title: "Tab 1", URL: "https://example.com"}},
	}
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetTabOperations(tabOps)

	require.NoError(t, api.Bind())

	// Register onRemoved listener.
	runJS(t, vm, `
		chrome.tabs.onRemoved.addListener(function(tabId) {
			__removedTabId = tabId;
		});
	`)

	// Remove a tab.
	require.NoError(t, tabOps.RemoveTab(1))

	// Fire the event manually (as the shell would).
	api.FireTabRemoved(1)

	removedIDVal := getGlobal(t, vm, "__removedTabId")
	require.NotNil(t, removedIDVal)
	require.NotEqual(t, goja.Undefined(), removedIDVal)
	removedID := removedIDVal.ToInteger()
	require.Equal(t, int64(1), removedID)
}

func TestChromeAPITabsOnActivated(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	tabOps := &mockTabOps{
		tabs: []TabInfo{{ID: 1, Title: "Tab 1", URL: "https://example.com"}},
	}
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetTabOperations(tabOps)

	require.NoError(t, api.Bind())

	// Register onActivated listener.
	runJS(t, vm, `
		chrome.tabs.onActivated.addListener(function(info) {
			__activatedTabId = info.tabId;
		});
	`)

	// Activate a tab.
	_, err := tabOps.ActivateTab(1)
	require.NoError(t, err)

	// Fire the event manually (as the shell would).
	api.FireTabActivated(1)

	activatedIDVal := getGlobal(t, vm, "__activatedTabId")
	require.NotNil(t, activatedIDVal)
	require.NotEqual(t, goja.Undefined(), activatedIDVal)
	activatedID := activatedIDVal.ToInteger()
	require.Equal(t, int64(1), activatedID)
}

func TestChromeAPIRuntimeOnInstalled(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())

	require.NoError(t, api.Bind())

	// Register onInstalled listener.
	runJS(t, vm, `
		chrome.runtime.onInstalled.addListener(function(details) {
			__installReason = details.reason;
		});
	`)

	// Fire the event manually (as the shell would).
	api.FireRuntimeInstalled("install")

	reasonVal := getGlobal(t, vm, "__installReason")
	require.NotNil(t, reasonVal)
	require.NotEqual(t, goja.Undefined(), reasonVal)
	reason := reasonVal.String()
	require.Equal(t, "install", reason)
}

func TestChromeAPIRuntimeOnStartup(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	vm := goja.New()
	api := NewChromeAPI(ext, vm, NewMessageRouter())

	require.NoError(t, api.Bind())

	// Register onStartup listener.
	runJS(t, vm, `
		chrome.runtime.onStartup.addListener(function() {
			__startupFired = true;
		});
	`)

	// Fire the event manually (as the shell would).
	api.FireRuntimeStartup()

	startedVal := getGlobal(t, vm, "__startupFired")
	require.NotNil(t, startedVal)
	require.NotEqual(t, goja.Undefined(), startedVal)
	started := startedVal.ToBoolean()
	require.True(t, started)
}
