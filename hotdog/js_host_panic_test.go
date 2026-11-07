package hotdog

import (
	"testing"

	"github.com/danfragoso/thdwb/assets"
	profiler "github.com/danfragoso/thdwb/profiler"
	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunJSContainsHostPanic (audit C-8) verifies that a panic raised by bound
// Go code is converted to an error at the RunJS boundary instead of crashing
// the calling goroutine. goja re-panics Go-originated panics as
// UncatchableException -- even a JS try/catch cannot contain them -- so without
// the recover in RunJS a single buggy DOM binding kills the whole browser.
func TestRunJSContainsHostPanic(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	require.NoError(t, wc.LoadHTML("<html><body></body></html>", "https://panic.example"))

	require.NoError(t, wc.SetJSValue("boom", func(call goja.FunctionCall) goja.Value {
		panic("host binding exploded")
	}))

	v, err := wc.RunJS(`boom();`)
	require.Error(t, err, "RunJS must return an error, not panic")
	assert.Nil(t, v)
	assert.Contains(t, err.Error(), "host panic")
	assert.Contains(t, err.Error(), "host binding exploded")

	// Even a JS try/catch cannot save a Go panic -- the recover must be at the
	// host boundary. Same script wrapped in try/catch must still error cleanly.
	v, err = wc.RunJS(`try { boom(); } catch (e) { /* swallowed in JS */ } 42;`)
	require.Error(t, err)
	assert.Nil(t, v)

	// The runtime stays usable after a contained panic.
	v, err = wc.RunJS(`1 + 1;`)
	require.NoError(t, err)
	assert.EqualValues(t, 2, v.ToInteger())
}

// TestSetJSValueContainsHostPanic covers Set, which can also panic on a
// misbehaving marshaling hook.
func TestSetJSValueContainsHostPanic(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	require.NoError(t, wc.LoadHTML("<html><body></body></html>", "https://panic.example"))

	require.NoError(t, wc.SetJSValue("x", 1))
	// A value whose Go marshaling panics must not take down the caller.
	require.NoError(t, wc.SetJSValue("y", func(call goja.FunctionCall) goja.Value {
		return goja.Undefined()
	}))
}
