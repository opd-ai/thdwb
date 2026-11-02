package debug

import (
	"bufio"
	"encoding/json"
	"net"
	"testing"
	"time"

	goja "github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

func TestDebugProtocolRegisterAndList(t *testing.T) {
	dp := NewDebugProtocol()
	vm := goja.New()
	dp.RegisterVM("ext1", vm)
	defer dp.UnregisterVM("ext1")

	// Start listener on a random port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()

	// Override the listener for testing.
	dp.mu.Lock()
	dp.listener = ln
	dp.running = true
	dp.mu.Unlock()

	go dp.acceptLoop()
	defer dp.Close()

	// Connect a client.
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	require.NoError(t, err)
	defer conn.Close()

	// Send list extensions request.
	req := map[string]interface{}{"id": 1, "method": "Debug.listExtensions"}
	enc := json.NewEncoder(conn)
	require.NoError(t, enc.Encode(req))

	// Read response.
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	require.NoError(t, err)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(line), &resp))
	require.Equal(t, float64(1), resp["id"])
	result, ok := resp["result"].(map[string]interface{})
	require.True(t, ok)
	exts, ok := result["extensions"].([]interface{})
	require.True(t, ok)
	require.Contains(t, exts, "ext1")
}

func TestDebugProtocolEvaluate(t *testing.T) {
	dp := NewDebugProtocol()
	vm := goja.New()
	_, err := vm.RunString(`var x = 42;`)
	require.NoError(t, err)
	dp.RegisterVM("ext1", vm)
	defer dp.UnregisterVM("ext1")

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()

	dp.mu.Lock()
	dp.listener = ln
	dp.running = true
	dp.mu.Unlock()

	go dp.acceptLoop()
	defer dp.Close()

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	require.NoError(t, err)
	defer conn.Close()

	req := map[string]interface{}{
		"id":      2,
		"method":  "Debug.evaluate",
		"params":  map[string]interface{}{"extensionId": "ext1", "expression": "x + 1"},
	}
	enc := json.NewEncoder(conn)
	require.NoError(t, enc.Encode(req))

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	require.NoError(t, err)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(line), &resp))
	require.Equal(t, float64(2), resp["id"])
	result, ok := resp["result"].(map[string]interface{})
	require.True(t, ok)
	require.Contains(t, result["result"], "43")
}

func TestDebugProtocolVMNotFound(t *testing.T) {
	dp := NewDebugProtocol()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()

	dp.mu.Lock()
	dp.listener = ln
	dp.running = true
	dp.mu.Unlock()

	go dp.acceptLoop()
	defer dp.Close()

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	require.NoError(t, err)
	defer conn.Close()

	req := map[string]interface{}{
		"id":      3,
		"method":  "Debug.evaluate",
		"params":  map[string]interface{}{"extensionId": "missing", "expression": "1"},
	}
	enc := json.NewEncoder(conn)
	require.NoError(t, enc.Encode(req))

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	require.NoError(t, err)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(line), &resp))
	require.Equal(t, float64(3), resp["id"])
	errObj, ok := resp["error"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "extension VM not found", errObj["message"])
}

func TestDebugProtocolClose(t *testing.T) {
	dp := NewDebugProtocol()
	require.NoError(t, dp.Listen("127.0.0.1:0"))
	require.NoError(t, dp.Close())
}
