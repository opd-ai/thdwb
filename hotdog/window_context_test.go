package hotdog

import (
	"testing"

	profiler "github.com/danfragoso/thdwb/profiler"
)

func TestWindowContextJSRuntimeIsolation(t *testing.T) {
	settings := &Settings{}
	buildInfo := &BuildInfo{}
	prof := profiler.NewProfiler()

	// Create two separate window contexts
	wc1 := NewWindowContext(settings, buildInfo, prof)
	wc2 := NewWindowContext(settings, buildInfo, prof)

	// Each should have its own Goja runtime
	runtime1 := wc1.GetJSRuntime()
	runtime2 := wc2.GetJSRuntime()

	if runtime1 == nil {
		t.Fatal("wc1 should have a Goja runtime")
	}
	if runtime2 == nil {
		t.Fatal("wc2 should have a Goja runtime")
	}

	// They should be different instances
	if runtime1 == runtime2 {
		t.Fatal("wc1 and wc2 should have different Goja runtime instances")
	}

	// Set a variable in runtime1
	runtime1.Set("testVar", 42)

	// Verify runtime2 doesn't have that variable
	val := runtime2.Get("testVar")
	if val != nil && !val.IsUndefined() {
		t.Fatalf("wc2 should not have access to wc1's variables, got: %v", val)
	}

	// Clean up
	wc1.Destroy()
	wc2.Destroy()
}

func TestWindowContextJSRuntimeBasicExecution(t *testing.T) {
	settings := &Settings{}
	buildInfo := &BuildInfo{}
	prof := profiler.NewProfiler()

	wc := NewWindowContext(settings, buildInfo, prof)
	runtime := wc.GetJSRuntime()

	// Execute simple JavaScript
	val, err := runtime.RunString("1 + 2")
	if err != nil {
		t.Fatalf("Failed to execute JS: %v", err)
	}

	result := val.ToInteger()
	if result != 3 {
		t.Fatalf("Expected 3, got %d", result)
	}

	// Test variable assignment
	_, err = runtime.RunString("var x = 42; x")
	if err != nil {
		t.Fatalf("Failed to execute JS: %v", err)
	}

	val = runtime.Get("x")
	if val.ToInteger() != 42 {
		t.Fatalf("Expected x=42, got %d", val.ToInteger())
	}

	wc.Destroy()
}
