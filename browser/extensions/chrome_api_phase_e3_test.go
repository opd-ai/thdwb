package extensions

import (
	"testing"

	goja "github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

// mockScriptingOps implements ScriptingOperations for testing.
type mockScriptingOps struct {
	executeErr    error
	executeResult interface{}
	injectErr     error
	removeErr     error
	executeCalls  int
	injectCalls   int
	removeCalls   int
}

func (m *mockScriptingOps) ExecuteScript(extID string, world ScriptingWorld, script string) (interface{}, error) {
	m.executeCalls++
	if m.executeErr != nil {
		return nil, m.executeErr
	}
	return m.executeResult, nil
}

func (m *mockScriptingOps) ExecuteScriptFile(extID, extPath, worldStr, jsPath string) (interface{}, error) {
	m.executeCalls++
	if m.executeErr != nil {
		return nil, m.executeErr
	}
	return m.executeResult, nil
}

func (m *mockScriptingOps) InjectCSS(extID, css string) error {
	m.injectCalls++
	return m.injectErr
}

func (m *mockScriptingOps) RemoveCSS(extID string) error {
	m.removeCalls++
	return m.removeErr
}

func (m *mockScriptingOps) ExecuteScriptTab(extID string, tabID int, world ScriptingWorld, script string) (interface{}, error) {
	m.executeCalls++
	if m.executeErr != nil {
		return nil, m.executeErr
	}
	return m.executeResult, nil
}

// mockContextMenuOps implements ContextMenuOperations for testing.
type mockContextMenuOps struct {
	items            map[string]ContextMenuItem
	createErr        error
	updateErr        error
	removeErr        error
	removeAllErr     error
	createCalls      int
	updateCalls      int
	removeCalls      int
	removeAllCalls   int
}

func (m *mockContextMenuOps) CreateContextMenu(item ContextMenuItem) (string, error) {
	m.createCalls++
	if m.createErr != nil {
		return "", m.createErr
	}
	if m.items == nil {
		m.items = make(map[string]ContextMenuItem)
	}
	if item.ID == "" {
		item.ID = "menu-1"
	}
	m.items[item.ID] = item
	return item.ID, nil
}

func (m *mockContextMenuOps) UpdateContextMenu(id string, item ContextMenuItem) error {
	m.updateCalls++
	if m.updateErr != nil {
		return m.updateErr
	}
	if m.items != nil {
		m.items[id] = item
	}
	return nil
}

func (m *mockContextMenuOps) RemoveContextMenu(id string) error {
	m.removeCalls++
	if m.removeErr != nil {
		return m.removeErr
	}
	delete(m.items, id)
	return nil
}

func (m *mockContextMenuOps) RemoveAllContextMenus() error {
	m.removeAllCalls++
	if m.removeAllErr != nil {
		return m.removeAllErr
	}
	m.items = make(map[string]ContextMenuItem)
	return nil
}

// mockPermissionOps implements PermissionOperations for testing.
type mockPermissionOps struct {
	granted     []string
	contains    []string
	all         []string
	requestErr  error
	containsErr error
	getAllErr   error
}

func (m *mockPermissionOps) RequestPermissions(extID string, permissions []string) ([]string, error) {
	if m.requestErr != nil {
		return nil, m.requestErr
	}
	return m.granted, nil
}

func (m *mockPermissionOps) ContainsPermissions(extID string, permissions []string) ([]string, error) {
	if m.containsErr != nil {
		return nil, m.containsErr
	}
	return m.contains, nil
}

func (m *mockPermissionOps) GetAllPermissions(extID string) ([]string, error) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	return m.all, nil
}

// mockAlarmOps implements AlarmOperations for testing.
type mockAlarmOps struct {
	alarms      map[string]*Alarm
	createErr   error
	getErr      error
	getAllErr   error
	clearErr    error
	clearAllErr error
	createCalls int
	clearCalls  int
	clearAllCalls int
}

func (m *mockAlarmOps) CreateAlarm(extID string, name string, alarm Alarm) error {
	m.createCalls++
	if m.createErr != nil {
		return m.createErr
	}
	if m.alarms == nil {
		m.alarms = make(map[string]*Alarm)
	}
	m.alarms[name] = &alarm
	return nil
}

func (m *mockAlarmOps) GetAlarm(extID string, name string) (*Alarm, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.alarms == nil {
		return nil, nil
	}
	return m.alarms[name], nil
}

func (m *mockAlarmOps) GetAllAlarms(extID string) ([]Alarm, error) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	if m.alarms == nil {
		return nil, nil
	}
	out := make([]Alarm, 0, len(m.alarms))
	for _, a := range m.alarms {
		out = append(out, *a)
	}
	return out, nil
}

func (m *mockAlarmOps) ClearAlarm(extID string, name string) error {
	m.clearCalls++
	if m.clearErr != nil {
		return m.clearErr
	}
	delete(m.alarms, name)
	return nil
}

func (m *mockAlarmOps) ClearAllAlarms(extID string) error {
	m.clearAllCalls++
	if m.clearAllErr != nil {
		return m.clearAllErr
	}
	m.alarms = make(map[string]*Alarm)
	return nil
}

// mockI18nOps implements I18nOperations for testing.
type mockI18nOps struct {
	messages map[string]string
	langs    []string
	getErr   error
	langsErr error
}

func (m *mockI18nOps) GetMessage(extID string, messageName string, substitutions ...string) (string, error) {
	if m.getErr != nil {
		return "", m.getErr
	}
	if m.messages == nil {
		return messageName, nil
	}
	return m.messages[messageName], nil
}

func (m *mockI18nOps) GetAcceptLanguages(extID string) ([]string, error) {
	if m.langsErr != nil {
		return nil, m.langsErr
	}
	return m.langs, nil
}

// TestScriptingOpsDirect tests the ScriptingOperations mock directly.
func TestScriptingOpsDirect(t *testing.T) {
	ops := &mockScriptingOps{executeResult: float64(42)}

	result, err := ops.ExecuteScript("ext1", ISOLATEDWorld, "1+1")
	require.NoError(t, err)
	require.Equal(t, float64(42), result)
	require.Equal(t, 1, ops.executeCalls)

	result, err = ops.ExecuteScriptFile("ext1", "/p", "ISOLATED", "script.js")
	require.NoError(t, err)
	require.Equal(t, float64(42), result)
	require.Equal(t, 2, ops.executeCalls)

	err = ops.InjectCSS("ext1", "body { color: red }")
	require.NoError(t, err)
	require.Equal(t, 1, ops.injectCalls)

	err = ops.RemoveCSS("ext1")
	require.NoError(t, err)
	require.Equal(t, 1, ops.removeCalls)

	result, err = ops.ExecuteScriptTab("ext1", 42, MAINWorld, "1+1")
	require.NoError(t, err)
	require.Equal(t, float64(42), result)
	require.Equal(t, 3, ops.executeCalls)
}

// TestScriptingOpsExecuteError tests error propagation.
func TestScriptingOpsExecuteError(t *testing.T) {
	ops := &mockScriptingOps{executeErr: &errTest{msg: "script failed"}}

	_, err := ops.ExecuteScript("ext1", ISOLATEDWorld, "bad")
	require.Error(t, err)
	require.Contains(t, err.Error(), "script failed")
}

type errTest struct {
	msg string
}

func (e *errTest) Error() string {
	return e.msg
}
func TestContextMenuOpsDirect(t *testing.T) {
	ops := &mockContextMenuOps{}

	id, err := ops.CreateContextMenu(ContextMenuItem{ID: "menu1", Title: "Test"})
	require.NoError(t, err)
	require.Equal(t, "menu1", id)
	require.Equal(t, 1, ops.createCalls)

	err = ops.UpdateContextMenu("menu1", ContextMenuItem{Title: "Updated"})
	require.NoError(t, err)
	require.Equal(t, 1, ops.updateCalls)

	err = ops.RemoveContextMenu("menu1")
	require.NoError(t, err)
	require.Equal(t, 1, ops.removeCalls)

	err = ops.RemoveAllContextMenus()
	require.NoError(t, err)
	require.Equal(t, 1, ops.removeAllCalls)
}

// TestPermissionOpsDirect tests the PermissionOperations mock directly.
func TestPermissionOpsDirect(t *testing.T) {
	ops := &mockPermissionOps{granted: []string{"tabs", "storage"}}

	granted, err := ops.RequestPermissions("ext1", []string{"tabs", "storage"})
	require.NoError(t, err)
	require.Equal(t, []string{"tabs", "storage"}, granted)

	ops.contains = []string{"tabs"}
	contains, err := ops.ContainsPermissions("ext1", []string{"tabs"})
	require.NoError(t, err)
	require.Equal(t, []string{"tabs"}, contains)

	ops.all = []string{"tabs", "storage", "cookies"}
	all, err := ops.GetAllPermissions("ext1")
	require.NoError(t, err)
	require.Equal(t, []string{"tabs", "storage", "cookies"}, all)
}

// TestAlarmOpsDirect tests the AlarmOperations mock directly.
func TestAlarmOpsDirect(t *testing.T) {
	ops := &mockAlarmOps{}

	err := ops.CreateAlarm("ext1", "a1", Alarm{Name: "a1", PeriodInMinutes: 10})
	require.NoError(t, err)
	require.Equal(t, 1, ops.createCalls)

	alarm, err := ops.GetAlarm("ext1", "a1")
	require.NoError(t, err)
	require.NotNil(t, alarm)
	require.Equal(t, "a1", alarm.Name)
	require.Equal(t, float64(10), alarm.PeriodInMinutes)

	all, err := ops.GetAllAlarms("ext1")
	require.NoError(t, err)
	require.Len(t, all, 1)

	err = ops.ClearAlarm("ext1", "a1")
	require.NoError(t, err)
	require.Equal(t, 1, ops.clearCalls)

	all, err = ops.GetAllAlarms("ext1")
	require.NoError(t, err)
	require.Empty(t, all)

	err = ops.ClearAllAlarms("ext1")
	require.NoError(t, err)
	require.Equal(t, 1, ops.clearAllCalls)
}

// TestI18nOpsDirect tests the I18nOperations mock directly.
func TestI18nOpsDirect(t *testing.T) {
	ops := &mockI18nOps{messages: map[string]string{"hello": "Hello, World!"}}

	msg, err := ops.GetMessage("ext1", "hello")
	require.NoError(t, err)
	require.Equal(t, "Hello, World!", msg)

	ops.langs = []string{"en", "es", "fr"}
	langs, err := ops.GetAcceptLanguages("ext1")
	require.NoError(t, err)
	require.Equal(t, []string{"en", "es", "fr"}, langs)
}

// TestChromeAPIScriptingBind tests chrome.scripting JS binding.
func TestChromeAPIScriptingBind(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	vm := goja.New()
	scriptingOps := &mockScriptingOps{executeResult: float64(42)}
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetScriptingOperations(scriptingOps)

	require.NoError(t, api.Bind())

	// Verify chrome.scripting exists.
	chrome := vm.Get("chrome").ToObject(vm)
	scripting := chrome.Get("scripting")
	require.NotNil(t, scripting)

	// Verify executeScript is a function.
	executeScript := scripting.ToObject(vm).Get("executeScript")
	_, ok := goja.AssertFunction(executeScript)
	require.True(t, ok)

	// Verify insertCSS is a function.
	insertCSS := scripting.ToObject(vm).Get("insertCSS")
	_, ok = goja.AssertFunction(insertCSS)
	require.True(t, ok)

	// Verify removeCSS is a function.
	removeCSS := scripting.ToObject(vm).Get("removeCSS")
	_, ok = goja.AssertFunction(removeCSS)
	require.True(t, ok)
}

// TestChromeAPIContextMenusBind tests chrome.contextMenus JS binding.
func TestChromeAPIContextMenusBind(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	vm := goja.New()
	menuOps := &mockContextMenuOps{}
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetContextMenuOperations(menuOps)

	require.NoError(t, api.Bind())

	chrome := vm.Get("chrome").ToObject(vm)
	menus := chrome.Get("contextMenus")
	require.NotNil(t, menus)

	create := menus.ToObject(vm).Get("create")
	_, ok := goja.AssertFunction(create)
	require.True(t, ok)

	update := menus.ToObject(vm).Get("update")
	_, ok = goja.AssertFunction(update)
	require.True(t, ok)

	remove := menus.ToObject(vm).Get("remove")
	_, ok = goja.AssertFunction(remove)
	require.True(t, ok)

	removeAll := menus.ToObject(vm).Get("removeAll")
	_, ok = goja.AssertFunction(removeAll)
	require.True(t, ok)
}

// TestChromeAPIPermissionsBind tests chrome.permissions JS binding.
func TestChromeAPIPermissionsBind(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	vm := goja.New()
	permOps := &mockPermissionOps{}
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetPermissionOperations(permOps)

	require.NoError(t, api.Bind())

	chrome := vm.Get("chrome").ToObject(vm)
	perms := chrome.Get("permissions")
	require.NotNil(t, perms)

	request := perms.ToObject(vm).Get("request")
	_, ok := goja.AssertFunction(request)
	require.True(t, ok)

	contains := perms.ToObject(vm).Get("contains")
	_, ok = goja.AssertFunction(contains)
	require.True(t, ok)

	getAll := perms.ToObject(vm).Get("getAll")
	_, ok = goja.AssertFunction(getAll)
	require.True(t, ok)
}

// TestChromeAPIAlarmsBind tests chrome.alarms JS binding.
func TestChromeAPIAlarmsBind(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	vm := goja.New()
	alarmOps := &mockAlarmOps{}
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetAlarmOperations(alarmOps)

	require.NoError(t, api.Bind())

	chrome := vm.Get("chrome").ToObject(vm)
	alarms := chrome.Get("alarms")
	require.NotNil(t, alarms)

	create := alarms.ToObject(vm).Get("create")
	_, ok := goja.AssertFunction(create)
	require.True(t, ok)

	get := alarms.ToObject(vm).Get("get")
	_, ok = goja.AssertFunction(get)
	require.True(t, ok)

	getAll := alarms.ToObject(vm).Get("getAll")
	_, ok = goja.AssertFunction(getAll)
	require.True(t, ok)

	clear := alarms.ToObject(vm).Get("clear")
	_, ok = goja.AssertFunction(clear)
	require.True(t, ok)

	clearAll := alarms.ToObject(vm).Get("clearAll")
	_, ok = goja.AssertFunction(clearAll)
	require.True(t, ok)
}

// TestChromeAPI18nBind tests chrome.i18n JS binding.
func TestChromeAPI18nBind(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	vm := goja.New()
	i18nOps := &mockI18nOps{}
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	api.SetI18nOperations(i18nOps)

	require.NoError(t, api.Bind())

	chrome := vm.Get("chrome").ToObject(vm)
	i18n := chrome.Get("i18n")
	require.NotNil(t, i18n)

	getMessage := i18n.ToObject(vm).Get("getMessage")
	_, ok := goja.AssertFunction(getMessage)
	require.True(t, ok)

	getAcceptLanguages := i18n.ToObject(vm).Get("getAcceptLanguages")
	_, ok = goja.AssertFunction(getAcceptLanguages)
	require.True(t, ok)
}
