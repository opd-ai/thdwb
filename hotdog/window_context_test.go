package hotdog

import (
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"

	profiler "github.com/danfragoso/thdwb/profiler"
	"github.com/dop251/goja"
)

func TestWindowContextJSRuntimeIsolation(t *testing.T) {
	settings := &Settings{}
	buildInfo := &BuildInfo{}
	prof := profiler.CreateProfiler()

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
	if val != nil && !goja.IsUndefined(val) {
		t.Fatalf("wc2 should not have access to wc1's variables, got: %v", val)
	}

	// Clean up
	wc1.Destroy()
	wc2.Destroy()
}

func TestWindowContextJSRuntimeBasicExecution(t *testing.T) {
	settings := &Settings{}
	buildInfo := &BuildInfo{}
	prof := profiler.CreateProfiler()

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

func TestDOMMutationsFromJS(t *testing.T) {
	settings := &Settings{}
	buildInfo := &BuildInfo{}
	prof := profiler.CreateProfiler()

	wc := NewWindowContext(settings, buildInfo, prof)

	// Parse a basic HTML document with html and body
	htmlContent := `<html><head></head><body></body></html>`
	// Use ketchup parser to parse the HTML
	parsedDoc, err := parseHTMLForTest(htmlContent, wc)
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}
	wc.ActiveDocument = parsedDoc

	// Initialize JS runtime with DOM bindings
	err = wc.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init JS runtime: %v", err)
	}

	runtime := wc.GetJSRuntime()

	// Test innerHTML setter
	_, err = runtime.RunString(`
		var div = document.createElement("div");
		div.innerHTML = "<span>Hello</span><span>World</span>";
		document.body.appendChild(div);
	`)
	if err != nil {
		t.Fatalf("Failed to set innerHTML: %v", err)
	}

	// Verify innerHTML was set
	val, err := runtime.RunString("document.body.innerHTML")
	if err != nil {
		t.Fatalf("Failed to get innerHTML: %v", err)
	}
	t.Logf("body innerHTML: %v", val.String())

	// Test textContent setter
	_, err = runtime.RunString(`
		var p = document.createElement("p");
		p.textContent = "Hello World";
		document.body.appendChild(p);
	`)
	if err != nil {
		t.Fatalf("Failed to set textContent: %v", err)
	}

	// Test insertBefore
	_, err = runtime.RunString(`
		var parent = document.createElement("div");
		var child1 = document.createElement("span");
		child1.textContent = "First";
		var child2 = document.createElement("span");
		child2.textContent = "Second";
		parent.appendChild(child1);
		parent.appendChild(child2);
		var child3 = document.createElement("span");
		child3.textContent = "Inserted";
		parent.insertBefore(child3, child2);
		document.body.appendChild(parent);
	`)
	if err != nil {
		t.Fatalf("Failed to insertBefore: %v", err)
	}

	// Test replaceChild
	_, err = runtime.RunString(`
		var parent = document.createElement("div");
		var oldChild = document.createElement("span");
		oldChild.textContent = "Old";
		parent.appendChild(oldChild);
		var newChild = document.createElement("span");
		newChild.textContent = "New";
		parent.replaceChild(newChild, oldChild);
		document.body.appendChild(parent);
	`)
	if err != nil {
		t.Fatalf("Failed to replaceChild: %v", err)
	}

	// Test cloneNode
	_, err = runtime.RunString(`
		var original = document.createElement("div");
		original.textContent = "Original";
		var clone = original.cloneNode(true);
		clone.textContent = "Clone";
		document.body.appendChild(original);
		document.body.appendChild(clone);
	`)
	if err != nil {
		t.Fatalf("Failed to cloneNode: %v", err)
	}

	wc.Destroy()
}

// parseHTMLForTest parses HTML using the ketchup parser for testing.
func parseHTMLForTest(htmlContent string, windowCtx *WindowContext) (*Document, error) {
	// Use the existing HTML parser from ketchup
	// This is a simplified version - in reality we'd use the ketchup package
	htmlRoot, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	// Set document URL to match window context's origin for origin checking
	docURL := windowCtx.GetOrigin()
	if docURL == nil || docURL.String() == "" {
		docURL, _ = url.Parse("http://localhost")
	}

	doc := &Document{
		Title:       "",
		ContentType: "text/html",
		URL:         docURL,
		RawDocument: htmlContent,
		HTMLRoot:    htmlRoot,
		StyleSheets: make([]*StyleElement, 0),
	}

	// Convert html.Node tree to NodeDOM tree
	doc.DOM = convertHTMLNodeToNodeDOMForTest(htmlRoot, doc, windowCtx)
	if doc.DOM != nil {
		doc.DOM.Document = doc
	}

	return doc, nil
}

// convertHTMLNodeToNodeDOMForTest converts an html.Node to a NodeDOM for testing.
func convertHTMLNodeToNodeDOMForTest(htmlNode *html.Node, doc *Document, windowCtx *WindowContext) *NodeDOM {
	switch htmlNode.Type {
	case html.DocumentNode:
		var result *NodeDOM
		for child := htmlNode.FirstChild; child != nil; child = child.NextSibling {
			childDOM := convertHTMLNodeToNodeDOMForTest(child, doc, windowCtx)
			if childDOM != nil {
				result = childDOM
				break
			}
		}
		return result
	case html.ElementNode:
		nodeDOM := &NodeDOM{
			Type:       NodeTypeElement,
			Element:    htmlNode.Data,
			Children:   make([]*NodeDOM, 0),
			Attributes: make([]*Attribute, 0),
			Document:   doc,
			HTMLNode:   htmlNode,
			WindowCtx:  windowCtx,
		}
		for _, attr := range htmlNode.Attr {
			nodeDOM.Attributes = append(nodeDOM.Attributes, &Attribute{
				Name:  attr.Key,
				Value: attr.Val,
			})
		}
		for child := htmlNode.FirstChild; child != nil; child = child.NextSibling {
			childDOM := convertHTMLNodeToNodeDOMForTest(child, doc, windowCtx)
			if childDOM != nil {
				nodeDOM.appendChildNode(childDOM)
			}
		}
		return nodeDOM
	case html.TextNode:
		return &NodeDOM{
			Type:      NodeTypeText,
			Content:   htmlNode.Data,
			Document:  doc,
			WindowCtx: windowCtx,
		}
	case html.CommentNode:
		return &NodeDOM{
			Type:      NodeTypeComment,
			Content:   htmlNode.Data,
			Document:  doc,
			WindowCtx: windowCtx,
		}
	default:
		return nil
	}
}

func TestCrossOriginDOMAccess(t *testing.T) {
	settings := &Settings{}
	buildInfo := &BuildInfo{}
	prof := profiler.CreateProfiler()

	wc1 := NewWindowContext(settings, buildInfo, prof)
	wc2 := NewWindowContext(settings, buildInfo, prof)

	wc1.SetOrigin("https://example.com")
	wc2.SetOrigin("https://other.com")

	// Parse HTML for both windows with a div already in body
	htmlContent := `<html><head></head><body><div id="test">Secret</div></body></html>`
	parsedDoc1, err := parseHTMLForTest(htmlContent, wc1)
	if err != nil {
		t.Fatalf("Failed to parse HTML for wc1: %v", err)
	}
	wc1.ActiveDocument = parsedDoc1

	parsedDoc2, err := parseHTMLForTest(htmlContent, wc2)
	if err != nil {
		t.Fatalf("Failed to parse HTML for wc2: %v", err)
	}
	wc2.ActiveDocument = parsedDoc2

	err = wc1.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc1 JS runtime: %v", err)
	}
	err = wc2.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc2 JS runtime: %v", err)
	}

	runtime1 := wc1.GetJSRuntime()
	wc2.GetJSRuntime() // Ensure wc2 has runtime initialized

	// Test that same-origin modification works using innerHTML
	_, err = runtime1.RunString(`
		var div = document.querySelector("#test");
		div.innerHTML = "Modified"; // Should work - same origin
	`)
	if err != nil {
		t.Fatalf("Same-origin modification failed: %v", err)
	}

	// Verify the modification
	val, err := runtime1.RunString("document.querySelector('#test').innerHTML")
	if err != nil {
		t.Fatalf("Failed to get innerHTML: %v", err)
	}
	t.Logf("innerHTML after modification: '%s'", val.String())
	if val.String() != "Modified" {
		t.Fatalf("Expected innerHTML 'Modified', got '%s'", val.String())
	}

	// Test that cross-origin access from wc2 to wc1's DOM throws SecurityError
	// We simulate this by creating a node in wc1 and trying to access it from wc2's runtime
	// Since runtimes are isolated, we test the origin check directly on the node
	node1, err := wc1.ActiveDocument.DOM.QuerySelector("#test")
	if err != nil {
		t.Fatalf("Failed to find test node in wc1: %v", err)
	}
	if node1 == nil {
		t.Fatal("Failed to find test node in wc1")
	}

	// The node belongs to wc1 (origin: https://example.com)
	// If we try to check origin against wc2's origin (https://other.com), it should fail
	// We simulate this by temporarily setting the node's WindowCtx to wc2
	originalWindowCtx := node1.WindowCtx
	node1.WindowCtx = wc2
	err = node1.checkOrigin()
	node1.WindowCtx = originalWindowCtx // Restore

	if err == nil {
		t.Fatal("Expected SecurityError for cross-origin access, got nil")
	}
	secErr, ok := err.(SecurityError)
	if !ok {
		t.Fatalf("Expected SecurityError, got %T: %v", err, err)
	}
	t.Logf("Cross-origin access correctly blocked: %v", secErr)

	// Test iframe sandbox restriction
	// Create a node that's inside a sandboxed iframe (without allow-same-origin)
	// sandbox="" (empty) means all restrictions apply, but ParseSandboxAttribute returns SandboxNone for empty string
	// So we use a sandbox value that explicitly doesn't include allow-same-origin
	iframeNode := &NodeDOM{
		Type:         NodeTypeElement,
		Element:      "iframe",
		SandboxFlags: SandboxAllowScripts | SandboxAllowForms, // Has sandbox but no allow-same-origin
		WindowCtx:    wc1,
		Document:     wc1.ActiveDocument,
	}
	childNode := &NodeDOM{
		Type:      NodeTypeElement,
		Element:   "div",
		WindowCtx: wc1,
		Document:  wc1.ActiveDocument,
		Parent:    iframeNode,
	}
	iframeNode.Children = []*NodeDOM{childNode}

	err = childNode.checkOrigin()
	if err == nil {
		t.Fatal("Expected SecurityError for sandboxed iframe access, got nil")
	}
	secErr, ok = err.(SecurityError)
	if !ok {
		t.Fatalf("Expected SecurityError, got %T: %v", err, err)
	}
	t.Logf("Sandboxed iframe access correctly blocked: %v", secErr)

	// Test that iframe with allow-same-origin works
	iframeNodeAllow := &NodeDOM{
		Type:         NodeTypeElement,
		Element:      "iframe",
		SandboxFlags: SandboxAllowSameOrigin,
		WindowCtx:    wc1,
		Document:     wc1.ActiveDocument,
	}
	childNodeAllow := &NodeDOM{
		Type:      NodeTypeElement,
		Element:   "div",
		WindowCtx: wc1,
		Document:  wc1.ActiveDocument,
		Parent:    iframeNodeAllow,
	}
	iframeNodeAllow.Children = []*NodeDOM{childNodeAllow}

	err = childNodeAllow.checkOrigin()
	if err != nil {
		t.Fatalf("Expected no error for iframe with allow-same-origin, got: %v", err)
	}
	t.Logf("Iframe with allow-same-origin correctly allows access")

	wc1.Destroy()
	wc2.Destroy()
}

func TestQuerySelectorRespectsIframeBoundaries(t *testing.T) {
	settings := &Settings{}
	buildInfo := &BuildInfo{}
	prof := profiler.CreateProfiler()

	wc := NewWindowContext(settings, buildInfo, prof)

	// Parse HTML with an iframe containing content
	htmlContent := `<html><head></head><body>
		<div id="main-div">Main Content</div>
		<iframe src="iframe.html" sandbox="allow-scripts">
			<div id="iframe-div">Iframe Content</div>
		</iframe>
		<div id="after-iframe">After Iframe</div>
	</body></html>`
	parsedDoc, err := parseHTMLForTest(htmlContent, wc)
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}
	wc.ActiveDocument = parsedDoc

	// Initialize JS runtime with DOM bindings
	err = wc.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init JS runtime: %v", err)
	}

	runtime := wc.GetJSRuntime()

	// Test querySelector from document - should find main-div but not iframe-div
	val, err := runtime.RunString(`document.querySelector("#main-div")`)
	if err != nil {
		t.Fatalf("Failed to querySelector #main-div: %v", err)
	}
	if goja.IsNull(val) || goja.IsUndefined(val) {
		t.Fatal("Expected to find #main-div, got null")
	}
	t.Logf("Found #main-div: %v", val.ToObject(runtime).Get("id"))

	// Should NOT find iframe-div (inside iframe)
	val, err = runtime.RunString(`document.querySelector("#iframe-div")`)
	if err != nil {
		t.Fatalf("Failed to querySelector #iframe-div: %v", err)
	}
	if !goja.IsNull(val) && !goja.IsUndefined(val) {
		t.Fatal("Expected NOT to find #iframe-div (inside iframe), but found it")
	}
	t.Log("Correctly did not find #iframe-div inside iframe")

	// Should find after-iframe
	val, err = runtime.RunString(`document.querySelector("#after-iframe")`)
	if err != nil {
		t.Fatalf("Failed to querySelector #after-iframe: %v", err)
	}
	if goja.IsNull(val) || goja.IsUndefined(val) {
		t.Fatal("Expected to find #after-iframe, got null")
	}
	t.Logf("Found #after-iframe: %v", val.ToObject(runtime).Get("id"))

	// Test querySelectorAll - should only find elements in main document
	val, err = runtime.RunString(`document.querySelectorAll("div").length`)
	if err != nil {
		t.Fatalf("Failed to querySelectorAll div: %v", err)
	}
	length := val.ToInteger()
	// Should find 2 divs: main-div, after-iframe
	// But NOT the div inside the iframe, and iframe itself is not a div
	if length != 2 {
		t.Fatalf("Expected 2 div elements in main document, got %d", length)
	}
	t.Logf("Found %d div elements in main document (excluding iframe content)", length)

	// Test querySelector from element - should not cross iframe boundary
	val, err = runtime.RunString(`
		var body = document.body;
		body.querySelector("#iframe-div")
	`)
	if err != nil {
		t.Fatalf("Failed to querySelector from body: %v", err)
	}
	if !goja.IsNull(val) && !goja.IsUndefined(val) {
		t.Fatal("Expected NOT to find #iframe-div from body.querySelector, but found it")
	}
	t.Log("Correctly did not find #iframe-div from body.querySelector")

	wc.Destroy()
}

func TestStoragePartitioningByOrigin(t *testing.T) {
	settings := &Settings{}
	buildInfo := &BuildInfo{}
	prof := profiler.CreateProfiler()

	// Create two window contexts with the same origin
	origin1, _ := url.Parse("https://example.com:443")
	origin2, _ := url.Parse("https://other.com:443")

	wc1 := NewWindowContext(settings, buildInfo, prof)
	wc1.SetOrigin(origin1.String())

	wc2 := NewWindowContext(settings, buildInfo, prof)
	wc2.SetOrigin(origin1.String()) // Same origin as wc1

	wc3 := NewWindowContext(settings, buildInfo, prof)
	wc3.SetOrigin(origin2.String()) // Different origin

	// Initialize JS runtimes
	err := wc1.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc1 JS runtime: %v", err)
	}
	err = wc2.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc2 JS runtime: %v", err)
	}
	err = wc3.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc3 JS runtime: %v", err)
	}

	runtime1 := wc1.GetJSRuntime()
	runtime2 := wc2.GetJSRuntime()
	runtime3 := wc3.GetJSRuntime()

	// Test localStorage is shared across windows of same origin
	_, err = runtime1.RunString(`localStorage.setItem("testKey", "valueFromWc1")`)
	if err != nil {
		t.Fatalf("wc1 failed to set localStorage: %v", err)
	}

	val, err := runtime2.RunString(`localStorage.getItem("testKey")`)
	if err != nil {
		t.Fatalf("wc2 failed to get localStorage: %v", err)
	}
	if val.String() != "valueFromWc1" {
		t.Fatalf("Expected localStorage to be shared across windows of same origin. Got: %s", val.String())
	}

	// Test localStorage is NOT shared across different origins
	val, err = runtime3.RunString(`localStorage.getItem("testKey")`)
	if err != nil {
		t.Fatalf("wc3 failed to get localStorage: %v", err)
	}
	if !goja.IsNull(val) && !goja.IsUndefined(val) {
		t.Fatalf("Expected localStorage to be isolated by origin. Got: %s", val.String())
	}

	// Test sessionStorage is per-window but partitioned by origin
	_, err = runtime1.RunString(`sessionStorage.setItem("sessionKey", "sessionFromWc1")`)
	if err != nil {
		t.Fatalf("wc1 failed to set sessionStorage: %v", err)
	}

	val, err = runtime2.RunString(`sessionStorage.getItem("sessionKey")`)
	if err != nil {
		t.Fatalf("wc2 failed to get sessionStorage: %v", err)
	}
	// sessionStorage should NOT be shared between windows even of same origin
	if !goja.IsNull(val) && !goja.IsUndefined(val) {
		t.Fatalf("Expected sessionStorage to be per-window. Got: %s", val.String())
	}

	// Test sessionStorage in wc1 is accessible
	val, err = runtime1.RunString(`sessionStorage.getItem("sessionKey")`)
	if err != nil {
		t.Fatalf("wc1 failed to get sessionStorage: %v", err)
	}
	if val.String() != "sessionFromWc1" {
		t.Fatalf("Expected sessionStorage to work in same window. Got: %s", val.String())
	}

	// Test sessionStorage in wc3 (different origin) is isolated
	val, err = runtime3.RunString(`sessionStorage.getItem("sessionKey")`)
	if err != nil {
		t.Fatalf("wc3 failed to get sessionStorage: %v", err)
	}
	if !goja.IsNull(val) && !goja.IsUndefined(val) {
		t.Fatalf("Expected sessionStorage to be isolated by origin. Got: %s", val.String())
	}

	// Test localStorage length
	val, err = runtime1.RunString(`localStorage.length`)
	if err != nil {
		t.Fatalf("wc1 failed to get localStorage.length: %v", err)
	}
	if val.ToInteger() != 1 {
		t.Fatalf("Expected localStorage.length == 1, got %d", val.ToInteger())
	}

	// Test sessionStorage length
	val, err = runtime1.RunString(`sessionStorage.length`)
	if err != nil {
		t.Fatalf("wc1 failed to get sessionStorage.length: %v", err)
	}
	if val.ToInteger() != 1 {
		t.Fatalf("Expected sessionStorage.length == 1, got %d", val.ToInteger())
	}

	// Test localStorage clear
	_, err = runtime1.RunString(`localStorage.clear()`)
	if err != nil {
		t.Fatalf("wc1 failed to clear localStorage: %v", err)
	}

	val, err = runtime2.RunString(`localStorage.length`)
	if err != nil {
		t.Fatalf("wc2 failed to get localStorage.length after clear: %v", err)
	}
	if val.ToInteger() != 0 {
		t.Fatalf("Expected localStorage.length == 0 after clear, got %d", val.ToInteger())
	}
	// Test sessionStorage clear
	_, err = runtime1.RunString(`sessionStorage.clear()`)
	if err != nil {
		t.Fatalf("wc1 failed to clear sessionStorage: %v", err)
	}

	val, err = runtime1.RunString(`sessionStorage.length`)
	if err != nil {
		t.Fatalf("wc1 failed to get sessionStorage.length after clear: %v", err)
	}
	if val.ToInteger() != 0 {
		t.Fatalf("Expected sessionStorage.length == 0 after clear, got %d", val.ToInteger())
	}

	// Clean up
	wc1.Destroy()
	wc2.Destroy()
	wc3.Destroy()

	// Verify sessionStorage is cleared on window destroy
	// Create new window with same origin
	wc4 := NewWindowContext(settings, buildInfo, prof)
	wc4.SetOrigin(origin1.String())
	err = wc4.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc4 JS runtime: %v", err)
	}
	runtime4 := wc4.GetJSRuntime()

	val, err = runtime4.RunString(`sessionStorage.getItem("sessionKey")`)
	if err != nil {
		t.Fatalf("wc4 failed to get sessionStorage: %v", err)
	}
	if !goja.IsNull(val) && !goja.IsUndefined(val) {
		t.Fatalf("Expected sessionStorage to be cleared after window destroy. Got: %s", val.String())
	}

	// But localStorage should persist across window destroy
	val, err = runtime4.RunString(`localStorage.setItem("persistKey", "persistValue")`)
	if err != nil {
		t.Fatalf("wc4 failed to set localStorage: %v", err)
	}
	wc4.Destroy()

	wc5 := NewWindowContext(settings, buildInfo, prof)
	wc5.SetOrigin(origin1.String())
	err = wc5.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc5 JS runtime: %v", err)
	}
	runtime5 := wc5.GetJSRuntime()

	val, err = runtime5.RunString(`localStorage.getItem("persistKey")`)
	if err != nil {
		t.Fatalf("wc5 failed to get localStorage: %v", err)
	}
	if val.String() != "persistValue" {
		t.Fatalf("Expected localStorage to persist across window destroy. Got: %s", val.String())
	}

	wc5.Destroy()
}

func TestStorageOriginKey(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"https://example.com", "https://example.com:443"},
		{"http://example.com", "http://example.com:80"},
		{"https://example.com:8443", "https://example.com:8443"},
		{"http://example.com:8080", "http://example.com:8080"},
		{"https://sub.example.com", "https://sub.example.com:443"},
	}

	for _, tc := range testCases {
		u, err := url.Parse(tc.input)
		if err != nil {
			t.Fatalf("Failed to parse URL %s: %v", tc.input, err)
		}
		key := originKey(u)
		if key != tc.expected {
			t.Errorf("originKey(%s) = %s, want %s", tc.input, key, tc.expected)
		}
	}

	// Test nil origin
	nilKey := originKey(nil)
	if nilKey != "null" {
		t.Errorf("originKey(nil) = %s, want 'null'", nilKey)
	}
}

// TestPostMessageTargetOriginValidation tests postMessage targetOrigin validation and delivery.
func TestPostMessageTargetOriginValidation(t *testing.T) {
	settings := &Settings{}
	buildInfo := &BuildInfo{}
	prof := profiler.CreateProfiler()

	origin1, _ := url.Parse("https://example.com:443")
	origin2, _ := url.Parse("https://other.com:443")

	wc1 := NewWindowContext(settings, buildInfo, prof)
	wc1.SetOrigin(origin1.String())

	wc2 := NewWindowContext(settings, buildInfo, prof)
	wc2.SetOrigin(origin2.String())

	err := wc1.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc1 JS runtime: %v", err)
	}
	err = wc2.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc2 JS runtime: %v", err)
	}

	runtime1 := wc1.GetJSRuntime()
	runtime2 := wc2.GetJSRuntime()

	// Test 1: postMessage with "*" targetOrigin delivers to all windows
	t.Run("wildcard targetOrigin delivers to all windows", func(t *testing.T) {
		var receivedMsg interface{}
		runtime2.Set("onMessageHandler", func(call goja.FunctionCall) goja.Value {
			receivedMsg = call.Arguments[0].Export()
			return goja.Undefined()
		})
		_, err := runtime2.RunString(`window.addEventListener("message", onMessageHandler)`)
		if err != nil {
			t.Fatalf("Failed to add event listener: %v", err)
		}

		_, err = runtime1.RunString(`window.postMessage("hello from wc1", "*")`)
		if err != nil {
			t.Fatalf("Failed to postMessage: %v", err)
		}

		if receivedMsg == nil {
			t.Fatal("wc2 did not receive message")
		}
		msgMap, ok := receivedMsg.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected message object, got %T", receivedMsg)
		}
		if msgMap["data"] != "hello from wc1" {
			t.Errorf("Expected data 'hello from wc1', got %v", msgMap["data"])
		}
		if msgMap["origin"] != "https://example.com:443" {
			t.Errorf("Expected origin 'https://example.com:443', got %v", msgMap["origin"])
		}
	})

	wc1.Destroy()
	wc2.Destroy()
}

// TestPostMessageSpecificTargetOrigin tests specific targetOrigin delivery.
func TestPostMessageSpecificTargetOrigin(t *testing.T) {
	settings := &Settings{}
	buildInfo := &BuildInfo{}
	prof := profiler.CreateProfiler()

	origin1, _ := url.Parse("https://example.com:443")
	origin2, _ := url.Parse("https://other.com:443")

	wc1 := NewWindowContext(settings, buildInfo, prof)
	wc1.SetOrigin(origin1.String())

	wc2 := NewWindowContext(settings, buildInfo, prof)
	wc2.SetOrigin(origin2.String())

	err := wc1.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc1 JS runtime: %v", err)
	}
	err = wc2.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc2 JS runtime: %v", err)
	}

	runtime1 := wc1.GetJSRuntime()
	runtime2 := wc2.GetJSRuntime()

	// Test: postMessage with specific targetOrigin delivers only to matching origin
	var receivedMsg2 interface{}
	runtime2.Set("onMessageHandler2", func(call goja.FunctionCall) goja.Value {
		receivedMsg2 = call.Arguments[0].Export()
		return goja.Undefined()
	})
	_, err = runtime2.RunString(`window.addEventListener("message", onMessageHandler2)`)
	if err != nil {
		t.Fatalf("Failed to add event listener: %v", err)
	}

	// Send message from wc1 with wc2's origin as targetOrigin
	_, err = runtime1.RunString(`window.postMessage("hello to wc2", "https://other.com:443")`)
	if err != nil {
		t.Fatalf("Failed to postMessage: %v", err)
	}

	// Check that wc2 received the message
	if receivedMsg2 == nil {
		t.Fatal("wc2 did not receive message with specific targetOrigin")
	}
	msgMap, ok := receivedMsg2.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected message object, got %T", receivedMsg2)
	}
	if msgMap["data"] != "hello to wc2" {
		t.Errorf("Expected data 'hello to wc2', got %v", msgMap["data"])
	}

	wc1.Destroy()
	wc2.Destroy()
}

// TestPostMessageNonMatchingTargetOrigin tests that non-matching targetOrigin does not deliver.
func TestPostMessageNonMatchingTargetOrigin(t *testing.T) {
	settings := &Settings{}
	buildInfo := &BuildInfo{}
	prof := profiler.CreateProfiler()

	origin1, _ := url.Parse("https://example.com:443")
	origin2, _ := url.Parse("https://other.com:443")

	wc1 := NewWindowContext(settings, buildInfo, prof)
	wc1.SetOrigin(origin1.String())

	wc2 := NewWindowContext(settings, buildInfo, prof)
	wc2.SetOrigin(origin2.String())

	err := wc1.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc1 JS runtime: %v", err)
	}
	err = wc2.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc2 JS runtime: %v", err)
	}

	runtime1 := wc1.GetJSRuntime()
	runtime2 := wc2.GetJSRuntime()

	// Test: postMessage with non-matching targetOrigin does not deliver
	var receivedMsg3 interface{}
	runtime2.Set("onMessageHandler3", func(call goja.FunctionCall) goja.Value {
		receivedMsg3 = call.Arguments[0].Export()
		return goja.Undefined()
	})
	_, err = runtime2.RunString(`window.addEventListener("message", onMessageHandler3)`)
	if err != nil {
		t.Fatalf("Failed to add event listener: %v", err)
	}

	// Send message from wc1 with a non-matching targetOrigin
	_, err = runtime1.RunString(`window.postMessage("should not deliver", "https://nonexistent.com:443")`)
	if err != nil {
		t.Fatalf("Failed to postMessage: %v", err)
	}

	// Check that wc2 did NOT receive the message
	if receivedMsg3 != nil {
		t.Fatal("wc2 should not have received message with non-matching targetOrigin")
	}

	wc1.Destroy()
	wc2.Destroy()
}

// TestPostMessageInvalidTargetOrigin tests that invalid targetOrigin throws error.
func TestPostMessageInvalidTargetOrigin(t *testing.T) {
	settings := &Settings{}
	buildInfo := &BuildInfo{}
	prof := profiler.CreateProfiler()

	origin1, _ := url.Parse("https://example.com:443")

	wc1 := NewWindowContext(settings, buildInfo, prof)
	wc1.SetOrigin(origin1.String())

	err := wc1.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc1 JS runtime: %v", err)
	}

	runtime1 := wc1.GetJSRuntime()

	// Test: Invalid targetOrigin throws error
	_, err = runtime1.RunString(`window.postMessage("test", "invalid-origin")`)
	if err == nil {
		t.Fatal("Expected error for invalid targetOrigin")
	}
	if !strings.Contains(err.Error(), "targetOrigin must be a valid origin") {
		t.Errorf("Expected targetOrigin validation error, got: %v", err)
	}

	wc1.Destroy()
}

// TestPostMessageSameOrigin tests same-origin postMessage delivery.
func TestPostMessageSameOrigin(t *testing.T) {
	settings := &Settings{}
	buildInfo := &BuildInfo{}
	prof := profiler.CreateProfiler()

	origin1, _ := url.Parse("https://example.com:443")

	wc1 := NewWindowContext(settings, buildInfo, prof)
	wc1.SetOrigin(origin1.String())

	wc2 := NewWindowContext(settings, buildInfo, prof)
	wc2.SetOrigin(origin1.String()) // Same origin

	err := wc1.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc1 JS runtime: %v", err)
	}
	err = wc2.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc2 JS runtime: %v", err)
	}

	runtime1 := wc1.GetJSRuntime()
	runtime2 := wc2.GetJSRuntime()

	// Test: same-origin targetOrigin delivers to same origin windows
	var receivedMsg4 interface{}
	runtime2.Set("onMessageHandler4", func(call goja.FunctionCall) goja.Value {
		receivedMsg4 = call.Arguments[0].Export()
		return goja.Undefined()
	})
	_, err = runtime2.RunString(`window.addEventListener("message", onMessageHandler4)`)
	if err != nil {
		t.Fatalf("Failed to add event listener: %v", err)
	}

	// Send message from wc1 with its own origin as targetOrigin
	_, err = runtime1.RunString(`window.postMessage("same origin msg", "https://example.com:443")`)
	if err != nil {
		t.Fatalf("Failed to postMessage: %v", err)
	}

	// Check that wc2 (same origin) received the message
	if receivedMsg4 == nil {
		t.Fatal("wc2 (same origin) did not receive message")
	}
	msgMap, ok := receivedMsg4.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected message object, got %T", receivedMsg4)
	}
	if msgMap["data"] != "same origin msg" {
		t.Errorf("Expected data 'same origin msg', got %v", msgMap["data"])
	}

	wc1.Destroy()
	wc2.Destroy()
}

// TestPostMessageCrossWindowIsolation tests cross-window isolation with wildcard.
func TestPostMessageCrossWindowIsolation(t *testing.T) {
	settings := &Settings{}
	buildInfo := &BuildInfo{}
	prof := profiler.CreateProfiler()

	origin1, _ := url.Parse("https://example.com:443")

	wc1 := NewWindowContext(settings, buildInfo, prof)
	wc1.SetOrigin(origin1.String())

	wc2 := NewWindowContext(settings, buildInfo, prof)
	wc2.SetOrigin(origin1.String()) // Same origin

	err := wc1.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc1 JS runtime: %v", err)
	}
	err = wc2.InitJSRuntime()
	if err != nil {
		t.Fatalf("Failed to init wc2 JS runtime: %v", err)
	}

	runtime1 := wc1.GetJSRuntime()
	runtime2 := wc2.GetJSRuntime()

	// Test that postMessage with "*" delivers to both windows (including sender)
	var receivedMsg1, receivedMsg2 interface{}
	runtime1.Set("handler1", func(call goja.FunctionCall) goja.Value {
		receivedMsg1 = call.Arguments[0].Export()
		return goja.Undefined()
	})
	runtime2.Set("handler2", func(call goja.FunctionCall) goja.Value {
		receivedMsg2 = call.Arguments[0].Export()
		return goja.Undefined()
	})
	_, err = runtime1.RunString(`window.addEventListener("message", handler1)`)
	if err != nil {
		t.Fatalf("Failed to add event listener: %v", err)
	}
	_, err = runtime2.RunString(`window.addEventListener("message", handler2)`)
	if err != nil {
		t.Fatalf("Failed to add event listener: %v", err)
	}

	_, err = runtime1.RunString(`window.postMessage("broadcast", "*")`)
	if err != nil {
		t.Fatalf("Failed to postMessage: %v", err)
	}

	// Both windows should receive the message
	if receivedMsg1 == nil {
		t.Fatal("wc1 (sender) did not receive broadcast message")
	}
	if receivedMsg2 == nil {
		t.Fatal("wc2 (same origin) did not receive broadcast message")
	}

	wc1.Destroy()
	wc2.Destroy()
}
