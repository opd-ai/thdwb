package extensions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentScriptSandboxCreation(t *testing.T) {
	s := NewContentScriptSandbox("ext1")
	require.NotNil(t, s)
	require.Equal(t, "ext1", s.ExtensionID())
	require.Equal(t, IsolatedWorld, s.Mode())
}

func TestContentScriptSandboxSetMode(t *testing.T) {
	s := NewContentScriptSandbox("ext1")
	s.SetMode(MainWorld)
	require.Equal(t, MainWorld, s.Mode())
	s.SetMode(IsolatedWorld)
	require.Equal(t, IsolatedWorld, s.Mode())
}

func TestContentScriptSandboxLoadScript(t *testing.T) {
	// Create a temp extension directory with a content script.
	root := t.TempDir()
	extDir := filepath.Join(root, "ext1")
	require.NoError(t, os.MkdirAll(extDir, 0o755))
	scriptContent := `console.log("hello from content script"); var x = 42;`
	require.NoError(t, os.WriteFile(filepath.Join(extDir, "content.js"), []byte(scriptContent), 0o644))

	s := NewContentScriptSandbox("ext1")
	require.NoError(t, s.LoadScript(extDir, "content.js"))
}

func TestContentScriptSandboxLoadScriptMissing(t *testing.T) {
	s := NewContentScriptSandbox("ext1")
	err := s.LoadScript("/nonexistent", "content.js")
	require.Error(t, err)
	require.Contains(t, err.Error(), "read content script")
}

func TestContentScriptSandboxRun(t *testing.T) {
	s := NewContentScriptSandbox("ext1")
	val, err := s.Run("var x = 1 + 2; x")
	require.NoError(t, err)
	require.Equal(t, float64(3), val.ToFloat())
}

func TestContentScriptSandboxDangerousGlobalsRemoved(t *testing.T) {
	s := NewContentScriptSandbox("ext1")

	// eval should be undefined.
	_, err := s.Run("typeof eval")
	require.NoError(t, err)
	require.Equal(t, "undefined", valToString(t, s, "typeof eval"))

	// Function should be undefined.
	_, err = s.Run("typeof Function")
	require.NoError(t, err)
	require.Equal(t, "undefined", valToString(t, s, "typeof Function"))
}

func TestContentScriptSandboxConsole(t *testing.T) {
	s := NewContentScriptSandbox("ext1")
	_, err := s.Run(`console.log("test message");`)
	require.NoError(t, err)
}

func TestContentScriptSandboxSendMessage(t *testing.T) {
	s := NewContentScriptSandbox("ext1")
	_, err := s.Run(`chrome.runtime.sendMessage({action: "test", data: 42});`)
	require.NoError(t, err)

	msg, ok := s.PopMessage()
	require.True(t, ok)
	require.Equal(t, "ext1", msg.Sender)
	require.Equal(t, "content-script", msg.Type)
	payload := msg.Payload.(map[string]interface{})
	require.Equal(t, "test", payload["action"])
	// JSON numbers may be int64 or float64 depending on value.
	switch v := payload["data"].(type) {
	case float64:
		require.Equal(t, float64(42), v)
	case int64:
		require.Equal(t, int64(42), v)
	default:
		require.Fail(t, "unexpected type for data")
	}
}

func TestContentScriptSandboxPostMessage(t *testing.T) {
	s := NewContentScriptSandbox("ext1")

	// No onMessage listener registered - should return an error.
	err := s.PostMessage(Message{Sender: "background", Type: "background", Payload: map[string]interface{}{"hello": "world"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no onMessage listener registered")

	// Register onMessage as a function and verify PostMessage calls it.
	_, err = s.Run(`
		var received = null;
		chrome.runtime.onMessage = function(msg) { received = msg; };
		function getReceived() { return received; }
	`)
	require.NoError(t, err)

	err = s.PostMessage(Message{Sender: "background", Type: "background", Payload: map[string]interface{}{"hello": "world"}})
	require.NoError(t, err)

	val, err := s.Run("getReceived()")
	require.NoError(t, err)
	exported := val.Export().(map[string]interface{})
	require.Equal(t, "world", exported["hello"])
}

func TestContentScriptSandboxClose(t *testing.T) {
	s := NewContentScriptSandbox("ext1")
	require.NoError(t, s.Close())
}

func valToString(t *testing.T, s *ContentScriptSandbox, expr string) string {
	t.Helper()
	val, err := s.Run(expr)
	require.NoError(t, err)
	return val.String()
}
