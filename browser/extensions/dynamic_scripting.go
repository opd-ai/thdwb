package extensions

import (
	"fmt"
	"sync"
)

// ScriptingWorld defines the JavaScript world for dynamic script execution.
type ScriptingWorld int

const (
	// ISOLATEDWorld executes the script in an isolated JavaScript world
	// that cannot access the page's JavaScript variables or functions.
	ISOLATEDWorld ScriptingWorld = iota
	// MAINWorld executes the script in the same JavaScript world as the
	// page, sharing globals and DOM objects.
	MAINWorld
)

// DynamicScripting provides chrome.scripting.executeScript functionality,
// allowing extensions to dynamically inject JavaScript into pages.
type DynamicScripting struct {
	mu       sync.Mutex
	loader   *ContentScriptLoader
	sandboxes map[string]*ContentScriptSandbox
	router   *MessageRouter
}

// NewDynamicScripting creates a new dynamic scripting manager.
func NewDynamicScripting(loader *ContentScriptLoader, router *MessageRouter) *DynamicScripting {
	return &DynamicScripting{
		loader:    loader,
		sandboxes: make(map[string]*ContentScriptSandbox),
		router:    router,
	}
}

// RegisterSandbox associates a content script sandbox with the dynamic
// scripting manager for a given extension.
func (d *DynamicScripting) RegisterSandbox(extID string, sandbox *ContentScriptSandbox) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sandboxes[extID] = sandbox
}

// ExecuteScript runs a script in the specified world for the given extension.
// The script source is provided directly (not loaded from a file).
func (d *DynamicScripting) ExecuteScript(extID string, world ScriptingWorld, script string) (interface{}, error) {
	d.mu.Lock()
	sandbox, ok := d.sandboxes[extID]
	d.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("no sandbox for extension %s", extID)
	}

	// Set the world mode on the sandbox.
	sandbox.SetMode(SandboxMode(world))

	// Execute the script in the sandbox.
	val, err := sandbox.Run(script)
	if err != nil {
		return nil, fmt.Errorf("execute script: %w", err)
	}

	return val.Export(), nil
}

// ExecuteScriptFile loads a script file from the extension directory and
// executes it in the specified world.
func (d *DynamicScripting) ExecuteScriptFile(extID, extPath, worldStr, jsPath string) (interface{}, error) {
	d.mu.Lock()
	sandbox, ok := d.sandboxes[extID]
	d.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("no sandbox for extension %s", extID)
	}

	world := ISOLATEDWorld
	if worldStr == "MAIN" {
		world = MAINWorld
	}
	sandbox.SetMode(SandboxMode(world))

	if err := sandbox.LoadScript(extPath, jsPath); err != nil {
		return nil, fmt.Errorf("execute script file: %w", err)
	}

	return nil, nil
}

// InjectCSS injects CSS into a page for the given extension. CSS is stored
// and can be applied when the page loads.
func (d *DynamicScripting) InjectCSS(extID, css string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// CSS injection is handled by the content script loader and applied
	// during page load. For dynamic injection, we register a synthetic
	// content script that matches all URLs.
	_ = extID
	_ = css
	return fmt.Errorf("CSS injection not yet implemented")
}

// RemoveCSS removes previously injected CSS for the given extension.
func (d *DynamicScripting) RemoveCSS(extID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_ = extID
	return fmt.Errorf("CSS removal not yet implemented")
}

// ExecuteScriptTab executes a script in a specific tab. TabID is reserved
// for future multi-tab support.
func (d *DynamicScripting) ExecuteScriptTab(extID string, tabID int, world ScriptingWorld, script string) (interface{}, error) {
	_ = tabID
	return d.ExecuteScript(extID, world, script)
}
