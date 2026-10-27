package extensions

import (
	"testing"

	goja "github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

func TestDebugOnMessage(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	router := NewMessageRouter()
	vm := goja.New()
	api := NewChromeAPI(ext, vm, router)

	require.NoError(t, api.Bind())

	// Check if chrome.runtime.onMessage exists.
	runJS(t, vm, `var onMsgExists = typeof chrome.runtime.onMessage !== "undefined";`)
	onMsgExists := getGlobal(t, vm, "onMsgExists").ToBoolean()
	t.Logf("onMessage exists: %v", onMsgExists)
	require.True(t, onMsgExists)

	// Check if addListener is a function.
	runJS(t, vm, `var addListenerExists = typeof chrome.runtime.onMessage.addListener === "function";`)
	addListenerExists := getGlobal(t, vm, "addListenerExists").ToBoolean()
	t.Logf("addListener exists: %v", addListenerExists)
	require.True(t, addListenerExists)

	// Register onMessage listener via JS.
	runJS(t, vm, `
		var received = null;
		chrome.runtime.onMessage.addListener(function(msg) {
			received = msg;
			print("Listener called with: " + JSON.stringify(msg));
		});
	`)

	// Check if received is still null after registration.
	receivedAfterReg := getGlobal(t, vm, "received")
	t.Logf("received after reg: %v (null=%v undef=%v)", receivedAfterReg, goja.IsNull(receivedAfterReg), goja.IsUndefined(receivedAfterReg))

	router.Route(Message{
		Sender:  "ext2",
		Target:  "ext1",
		Type:    "background",
		Payload: map[string]interface{}{"msg": "hi"},
	})

	received := getGlobal(t, vm, "received")
	t.Logf("received after route: %v (null=%v undef=%v)", received, goja.IsNull(received), goja.IsUndefined(received))
	if !goja.IsNull(received) && !goja.IsUndefined(received) {
		obj := received.ToObject(vm)
		t.Logf("received.msg: %v", obj.Get("msg"))
	}
}
