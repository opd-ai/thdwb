package hotdog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPostMessageDeliversPlainObject (audit C-14) posts a nested object and
// asserts the target receives an equivalent, independent copy. The values must
// survive the clone boundary intact.
func TestPostMessageDeliversPlainObject(t *testing.T) {
	sender := newABBAWindow(t, "https://clone-sender.example")
	target := newABBAWindow(t, "https://clone-target.example")
	defer sender.Destroy()
	defer target.Destroy()

	_, err := target.RunJS(`
		window.__got = null;
		window.addEventListener("message", function(e) { window.__got = e.data; });
	`)
	require.NoError(t, err)

	_, err = sender.RunJS(`window.postMessage({num: 7, str: "hi", nested: {arr: [1,2,3]}}, "https://clone-target.example")`)
	require.NoError(t, err)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		v, err := target.RunJS(`window.__got !== null ? 1 : 0`)
		if err == nil && v.ToInteger() == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	v, err := target.RunJS(`window.__got.num + ":" + window.__got.str + ":" + window.__got.nested.arr.length`)
	require.NoError(t, err)
	assert.Equal(t, "7:hi:3", v.String())

	// The sender stays fully responsive: nothing from its VM leaked into the
	// target that could drive it from another goroutine.
	_, err = sender.RunJS(`1+1`)
	require.NoError(t, err)
}

// TestPostMessageFunctionPayloadThrows (audit C-14) verifies the leak is gone.
// Previously Export() handed the target a Go closure bound to the sender's
// runtime; calling event.data.f drove the sender's VM without its lock (probe
// P16: 87 DATA RACE blocks, P15: typeof=function). Structured clone rejects
// callables -- the same DataCloneError the spec mandates -- so the function can
// never cross the boundary.
func TestPostMessageFunctionPayloadThrows(t *testing.T) {
	sender := newABBAWindow(t, "https://fn-sender.example")
	target := newABBAWindow(t, "https://fn-target.example")
	defer sender.Destroy()
	defer target.Destroy()

	_, err := target.RunJS(`
		window.__count = 0;
		window.addEventListener("message", function(e) { window.__count++; });
	`)
	require.NoError(t, err)

	v, err := sender.RunJS(`
		var outcome = "accepted";
		try { window.postMessage({f: function(){ return 42; }}, "https://fn-target.example"); }
		catch (e) { outcome = "rejected"; }
		outcome;
	`)
	require.NoError(t, err, "the clone failure must surface as a catchable JS error")
	assert.Equal(t, "rejected", v.String(), "function payloads must not cross the clone boundary")

	// Nothing was delivered, and both VMs remain usable.
	time.Sleep(50 * time.Millisecond)
	v, err = target.RunJS(`window.__count`)
	require.NoError(t, err)
	assert.EqualValues(t, 0, v.ToInteger())
	_, err = sender.RunJS(`typeof window.postMessage`)
	require.NoError(t, err)
}

// TestPostMessageCircularPayloadThrows documents the clone boundary for cycles:
// the engine must surface an error, not hang or crash.
func TestPostMessageCircularPayloadThrows(t *testing.T) {
	sender := newABBAWindow(t, "https://cycle-sender.example")
	defer sender.Destroy()

	v, err := sender.RunJS(`
		var o = {a: 1};
		o.self = o;
		var outcome = "accepted";
		try { window.postMessage(o, "*"); } catch (e) { outcome = "rejected"; }
		outcome;
	`)
	require.NoError(t, err)
	assert.Equal(t, "rejected", v.String())
}

// TestStructuredCloneUnit covers the helper directly.
func TestStructuredCloneUnit(t *testing.T) {
	out, err := structuredClone(map[string]interface{}{
		"a": 1,
		"b": []interface{}{"x", nil, true},
	})
	require.NoError(t, err)
	m, ok := out.(map[string]interface{})
	require.True(t, ok)
	assert.EqualValues(t, 1, m["a"])
	arr, ok := m["b"].([]interface{})
	require.True(t, ok)
	assert.Len(t, arr, 3)

	// Callables cannot be cloned (spec DataCloneError).
	_, err = structuredClone(map[string]interface{}{"f": func() {}})
	assert.Error(t, err)
}
