package extensions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDynamicScriptingCreation(t *testing.T) {
	loader := NewContentScriptLoader()
	router := NewMessageRouter()
	d := NewDynamicScripting(loader, router)
	require.NotNil(t, d)
}

func TestDynamicScriptingExecuteScriptNoSandbox(t *testing.T) {
	loader := NewContentScriptLoader()
	router := NewMessageRouter()
	d := NewDynamicScripting(loader, router)

	_, err := d.ExecuteScript("nonexistent", ISOLATEDWorld, "1+1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no sandbox for extension")
}

func TestDynamicScriptingExecuteScript(t *testing.T) {
	loader := NewContentScriptLoader()
	router := NewMessageRouter()
	d := NewDynamicScripting(loader, router)

	sandbox := NewContentScriptSandbox("ext1")
	d.RegisterSandbox("ext1", sandbox)

	result, err := d.ExecuteScript("ext1", ISOLATEDWorld, "1 + 2")
	require.NoError(t, err)
	switch v := result.(type) {
	case float64:
		require.Equal(t, float64(3), v)
	case int64:
		require.Equal(t, int64(3), v)
	default:
		require.Fail(t, "unexpected type")
	}
}

func TestDynamicScriptingExecuteScriptMainWorld(t *testing.T) {
	loader := NewContentScriptLoader()
	router := NewMessageRouter()
	d := NewDynamicScripting(loader, router)

	sandbox := NewContentScriptSandbox("ext1")
	d.RegisterSandbox("ext1", sandbox)

	result, err := d.ExecuteScript("ext1", MAINWorld, "var x = 21 * 2; x")
	require.NoError(t, err)
	switch v := result.(type) {
	case float64:
		require.Equal(t, float64(42), v)
	case int64:
		require.Equal(t, int64(42), v)
	default:
		require.Fail(t, "unexpected type")
	}
}

func TestDynamicScriptingExecuteScriptFile(t *testing.T) {
	loader := NewContentScriptLoader()
	router := NewMessageRouter()
	d := NewDynamicScripting(loader, router)

	sandbox := NewContentScriptSandbox("ext1")
	d.RegisterSandbox("ext1", sandbox)

	// Create a temp script file.
	dir := t.TempDir()
	scriptContent := `var result = 100 / 4;`
	writeFile(t, dir, "script.js", scriptContent)

	result, err := d.ExecuteScriptFile("ext1", dir, "ISOLATED", "script.js")
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestDynamicScriptingExecuteScriptFileMissing(t *testing.T) {
	loader := NewContentScriptLoader()
	router := NewMessageRouter()
	d := NewDynamicScripting(loader, router)

	sandbox := NewContentScriptSandbox("ext1")
	d.RegisterSandbox("ext1", sandbox)

	_, err := d.ExecuteScriptFile("ext1", "/nonexistent", "ISOLATED", "script.js")
	require.Error(t, err)
	require.Contains(t, err.Error(), "execute script file")
}

func TestDynamicScriptingExecuteScriptTab(t *testing.T) {
	loader := NewContentScriptLoader()
	router := NewMessageRouter()
	d := NewDynamicScripting(loader, router)

	sandbox := NewContentScriptSandbox("ext1")
	d.RegisterSandbox("ext1", sandbox)

	result, err := d.ExecuteScriptTab("ext1", 42, ISOLATEDWorld, "7 * 6")
	require.NoError(t, err)
	switch v := result.(type) {
	case float64:
		require.Equal(t, float64(42), v)
	case int64:
		require.Equal(t, int64(42), v)
	default:
		require.Fail(t, "unexpected type")
	}
}

func TestDynamicScriptingInjectCSSNotImplemented(t *testing.T) {
	loader := NewContentScriptLoader()
	router := NewMessageRouter()
	d := NewDynamicScripting(loader, router)

	err := d.InjectCSS("ext1", "body { color: red }")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not yet implemented")
}

func TestDynamicScriptingRemoveCSSNotImplemented(t *testing.T) {
	loader := NewContentScriptLoader()
	router := NewMessageRouter()
	d := NewDynamicScripting(loader, router)

	err := d.RemoveCSS("ext1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not yet implemented")
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
