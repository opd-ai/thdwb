package extensions

import (
	"testing"
	"time"

	goja "github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

// TestChromeAPIZeroArgCallsDontPanic (audit C-29) verifies every chrome.*
// binding survives being invoked with zero arguments. Pre-fix bindings
// computed call.Argument(len(call.Arguments) - 1), which indexed
// Arguments[-1]; a native-binding panic with no active JS try frame
// repanics through runWrapped and kills the process (audit probe:
// "index out of range [-1] [recovered, repanicked]").
func TestChromeAPIZeroArgCallsDontPanic(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "Test", Version: "1.0"}
	ext := NewExtension("ext1", "/p", m)
	vm := goja.New()
	vm.SetMaxCallStackSize(1000)
	api := NewChromeAPI(ext, vm, NewMessageRouter())
	require.NoError(t, api.Bind())

	chrome := vm.Get("chrome").ToObject(vm)
	require.NotNil(t, chrome)

	// Every binding exposed on the chrome object must not panic when called
	// with zero arguments. We enumerate the top-level namespaces and call
	// each function-valued property with no arguments.
	deadline := time.Now().Add(30 * time.Second)
	checked := 0
	var walk func(obj *goja.Object, path string)
	walk = func(obj *goja.Object, path string) {
		if time.Now().After(deadline) {
			return
		}
		for _, name := range obj.Keys() {
			v := obj.Get(name)
			if v == nil || v == goja.Undefined() || v == goja.Null() {
				continue
			}
			if fn, ok := goja.AssertFunction(v); ok {
				checked++
				// A goja error (e.g. "requires a message argument") is a
				// catchable JS exception -- not the C-29 native panic. We only
				// assert the call does not panic and kill the process.
				func() {
					defer func() {
						if r := recover(); r != nil {
							require.Fail(t, "zero-arg call to chrome.%s panicked natively: %v", path+"."+name, r)
						}
					}()
					fn(goja.Undefined())
				}()
			} else if o := v.ToObject(vm); o != nil {
				walk(o, path+"."+name)
			}
		}
	}
	walk(chrome, "chrome")
	require.Greater(t, checked, 0, "no chrome bindings were exercised")
}
