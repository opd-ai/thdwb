package extensions

import (
	"os"
	"runtime"
	"testing"

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
	router.RegisterHandler("ext2", func(msg Message) error {
		received = append(received, msg)
		return nil
	})

	runJS(t, vm, `chrome.runtime.sendMessage({hello: "world"});`)

	require.Len(t, received, 1)
	require.Equal(t, "ext1", received[0].Sender)
	require.Equal(t, "world", received[0].Payload.(map[string]interface{})["hello"])
}

func TestChromeAPIRuntimeOnMessage(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	router := NewMessageRouter()
	vm := goja.New()
	api := NewChromeAPI(ext, vm, router)

	require.NoError(t, api.Bind())

	// Register onMessage listener at the Go level.
	var received []interface{}
	router.RegisterHandler(ext.ID, func(msg Message) error {
		received = append(received, msg.Payload)
		return nil
	})

	// Route a message to this extension.
	err := router.Route(Message{
		Sender:  "ext2",
		Target:  "ext1",
		Type:    "background",
		Payload: map[string]interface{}{"msg": "hi"},
	})
	require.NoError(t, err)

	require.Len(t, received, 1)
	require.Equal(t, "hi", received[0].(map[string]interface{})["msg"])
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

	// Send a message from ext1's VM to ext2's VM.
	vm1 := ext1.BackgroundVM
	_ = ext2.BackgroundVM

	// Register onMessage listener in ext2 at the Go level.
	var received []interface{}
	router := m.MessageRouter()
	router.RegisterHandler(ext2.ID, func(msg Message) error {
		received = append(received, msg.Payload)
		return nil
	})

	// Send a message from ext1's VM.
	runJS(t, vm1, `chrome.runtime.sendMessage({hello: "world"});`)

	require.Len(t, received, 1)
	require.Equal(t, "world", received[0].(map[string]interface{})["hello"])
}
