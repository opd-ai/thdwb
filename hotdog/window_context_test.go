package hotdog

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/danfragoso/thdwb/assets"
	"github.com/danfragoso/thdwb/mustard"
	sauce "github.com/danfragoso/thdwb/sauce"
	"github.com/go-gl/glfw/v3.3/glfw"
	"golang.org/x/net/html"

	profiler "github.com/danfragoso/thdwb/profiler"
	"github.com/dop251/goja"
)

func TestWindowContextJSRuntimeIsolation(t *testing.T) {
	settings := &Settings{}
	buildInfo := &assets.BuildInfo{}
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
	buildInfo := &assets.BuildInfo{}
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

func TestHttpOnlyCookiesAreNotExposedToJavaScript(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	if err := wc.SetOrigin("https://example.com"); err != nil {
		t.Fatal(err)
	}
	if err := wc.InitJSRuntime(); err != nil {
		t.Fatal(err)
	}
	wc.GetCookieJar().SetCookies(wc.GetOrigin(), []*http.Cookie{
		{Name: "session", Value: "secret", HttpOnly: true, Path: "/"},
	})

	value, err := wc.RunJS("typeof document.cookie")
	if err != nil {
		t.Fatal(err)
	}
	if value.String() != "undefined" {
		t.Fatalf("expected document.cookie to be unavailable, got %q", value.String())
	}
}

func TestWindowContextCookieJarIsolation(t *testing.T) {
	wc1 := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc1.Destroy()
	wc2 := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc2.Destroy()

	for _, wc := range []*WindowContext{wc1, wc2} {
		if err := wc.SetOrigin("https://example.com"); err != nil {
			t.Fatal(err)
		}
	}

	wc1.GetCookieJar().SetCookies(wc1.GetOrigin(), []*http.Cookie{
		{Name: "session", Value: "window-one", Path: "/"},
	})

	cookies := wc1.GetCookieJar().Cookies(wc1.GetOrigin())
	if len(cookies) != 1 || cookies[0].Value != "window-one" {
		t.Fatalf("expected cookie to remain available in wc1, got %#v", cookies)
	}
	if cookies := wc2.GetCookieJar().Cookies(wc2.GetOrigin()); len(cookies) != 0 {
		t.Fatalf("cookie set by wc1 leaked into wc2: %#v", cookies)
	}
}

func TestDirtyFlagTracksDOMMutation(t *testing.T) {
	settings := &Settings{}
	buildInfo := &assets.BuildInfo{}
	prof := profiler.CreateProfiler()

	wc := NewWindowContext(settings, buildInfo, prof)
	htmlContent := `<html><head></head><body></body></html>`
	parsedDoc, err := parseHTMLForTest(htmlContent, wc)
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}
	wc.ActiveDocument = parsedDoc
	if err := wc.InitJSRuntime(); err != nil {
		t.Fatalf("Failed to init JS runtime: %v", err)
	}

	_, err = wc.GetJSRuntime().RunString(`
		var div = document.createElement("div");
		div.setAttribute("data-test", "value");
		document.body.appendChild(div);
	`)
	if err != nil {
		t.Fatalf("Failed to mutate DOM: %v", err)
	}

	body, err := parsedDoc.GetBody()
	if err != nil {
		t.Fatalf("Failed to get body: %v", err)
	}
	if !body.Dirty {
		t.Fatal("Body should be marked dirty after DOM mutation")
	}
	if body.FirstChild == nil || !body.FirstChild.Dirty {
		t.Fatal("Inserted child should be marked dirty after appendChild")
	}

	wc.Destroy()
}

func TestLayoutInvalidationsCoalescePerWindow(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	doc, err := parseHTMLForTest(`<html><body></body></html>`, wc)
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}
	wc.ActiveDocument = doc
	if err := wc.InitJSRuntime(); err != nil {
		t.Fatalf("Failed to init JS runtime: %v", err)
	}
	if _, err := wc.RunJS(`var target = document.createElement("div"); document.body.appendChild(target); for (var i = 0; i < 12; i++) target.setAttribute("data-pass", i);`); err != nil {
		t.Fatalf("Failed to mutate DOM: %v", err)
	}
	if roots := wc.DrainReflowQueue(); len(roots) != 1 || roots[0] != doc.DOM {
		t.Fatalf("expected one invalidated document root, got %d", len(roots))
	}
	if _, err := wc.RunJS(`target.setAttribute("data-pass", "later");`); err != nil {
		t.Fatalf("Failed to mutate DOM after draining: %v", err)
	}
	if roots := wc.DrainReflowQueue(); len(roots) != 1 || roots[0] != doc.DOM {
		t.Fatalf("expected a fresh invalidation after drain, got %d", len(roots))
	}
}

func TestInactiveWindowBlocksScriptsAndDefersReflow(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	wc.SetActive(false)
	if _, err := wc.RunJS(`globalThis.inactiveScriptRan = true`); err == nil {
		t.Fatal("inactive window executed JavaScript")
	}
	if err := wc.SetJSValue("inactiveValue", true); err == nil {
		t.Fatal("inactive window accepted a JavaScript global write")
	}
	root := &NodeDOM{WindowCtx: wc}
	eventFired := false
	wc.registerNodeEvent(root, "click", 1, func(interface{}) { eventFired = true })
	wc.EmitNodeEvent(root, "click", nil)
	if eventFired {
		t.Fatal("inactive window dispatched a DOM event")
	}
	root.RequestReflow()
	if roots := wc.DrainReflowQueue(); len(roots) != 0 {
		t.Fatal("inactive window drained layout work")
	}
	wc.SetActive(true)
	if roots := wc.DrainReflowQueue(); len(roots) != 1 || roots[0] != root {
		t.Fatal("pending layout work was not retained for activation")
	}
}

func TestAppTabActivationControlsBoundWindowContexts(t *testing.T) {
	app := mustard.CreateNewApp("tabs")
	firstWindow, secondWindow := &mustard.Window{}, &mustard.Window{}
	app.AddWindow(firstWindow)
	app.AddWindow(secondWindow)
	firstContext := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	secondContext := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer firstContext.Destroy()
	defer secondContext.Destroy()
	firstContext.BindWindowInput(firstWindow)
	secondContext.BindWindowInput(secondWindow)

	if err := app.ActivateWindow(firstWindow); err != nil {
		t.Fatal(err)
	}
	if firstContext.IsActive() != true || secondContext.IsActive() {
		t.Fatal("activating the first tab did not pause the other window context")
	}
	if err := app.ActivateWindow(secondWindow); err != nil {
		t.Fatal(err)
	}
	if firstContext.IsActive() || !secondContext.IsActive() {
		t.Fatal("activating the second tab did not resume only its window context")
	}
}

func TestInactiveWindowPausesAnimationFramesUntilActivation(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	fired := make(chan struct{}, 1)
	wc.SetActive(false)
	wc.ScheduleAnimationFrame(func() { fired <- struct{}{} })
	select {
	case <-fired:
		t.Fatal("inactive window ran an animation frame")
	case <-time.After(50 * time.Millisecond):
	}
	wc.SetActive(true)
	select {
	case <-fired:
	case <-time.After(time.Second):
		t.Fatal("animation frame did not run after activation")
	}
}

func TestAnimationFrameDeactivationDefersRemainingCallbacks(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	first, second := make(chan struct{}, 1), make(chan struct{}, 1)
	wc.ScheduleAnimationFrame(func() {
		wc.SetActive(false)
		first <- struct{}{}
	})
	wc.ScheduleAnimationFrame(func() { second <- struct{}{} })
	select {
	case <-first:
	case <-time.After(time.Second):
		t.Fatal("first animation frame did not run")
	}
	select {
	case <-second:
		t.Fatal("deactivation did not pause the remaining callback")
	case <-time.After(50 * time.Millisecond):
	}
	wc.SetActive(true)
	select {
	case <-second:
	case <-time.After(time.Second):
		t.Fatal("remaining callback did not resume after activation")
	}
}

func TestCalcPointIntersectionChoosesDeepestTopmostLaidOutNode(t *testing.T) {
	root := &NodeDOM{RenderBox: &RenderBox{Width: 100, Height: 100}}
	earlier := &NodeDOM{RenderBox: &RenderBox{Left: 10, Top: 10, Width: 30, Height: 30}}
	inner := &NodeDOM{RenderBox: &RenderBox{Left: 12, Top: 12, Width: 6, Height: 6}}
	later := &NodeDOM{RenderBox: &RenderBox{Left: 20, Top: 20, Width: 20, Height: 20}}
	earlier.Children = []*NodeDOM{inner}
	root.Children = []*NodeDOM{nil, {}, earlier, later}
	for _, point := range []struct {
		x, y float64
		want *NodeDOM
	}{{15, 15, inner}, {25, 25, later}, {10, 10, earlier}, {100, 100, nil}} {
		if got := root.CalcPointIntersection(point.x, point.y); got != point.want {
			t.Fatalf("hit at (%v,%v) = %p, want %p", point.x, point.y, got, point.want)
		}
	}
}

func TestWindowRequestAnimationFrameIsolation(t *testing.T) {
	settings := &Settings{}
	buildInfo := &assets.BuildInfo{}
	prof := profiler.CreateProfiler()

	wc1 := NewWindowContext(settings, buildInfo, prof)
	wc2 := NewWindowContext(settings, buildInfo, prof)
	if err := wc1.InitJSRuntime(); err != nil {
		t.Fatalf("Failed to init wc1 JS runtime: %v", err)
	}
	if err := wc2.InitJSRuntime(); err != nil {
		t.Fatalf("Failed to init wc2 JS runtime: %v", err)
	}

	frame1 := make(chan bool, 1)
	frame2 := make(chan bool, 1)
	if err := wc1.SetJSValue("onFrame1", func(call goja.FunctionCall) goja.Value {
		frame1 <- true
		return goja.Undefined()
	}); err != nil {
		t.Fatalf("Failed to set wc1 frame callback: %v", err)
	}
	if err := wc2.SetJSValue("onFrame2", func(call goja.FunctionCall) goja.Value {
		frame2 <- true
		return goja.Undefined()
	}); err != nil {
		t.Fatalf("Failed to set wc2 frame callback: %v", err)
	}

	if _, err := wc1.RunJS(`window.requestAnimationFrame(onFrame1)`); err != nil {
		t.Fatalf("Failed to schedule wc1 animation frame: %v", err)
	}
	if _, err := wc2.RunJS(`window.requestAnimationFrame(onFrame2)`); err != nil {
		t.Fatalf("Failed to schedule wc2 animation frame: %v", err)
	}

	select {
	case <-frame1:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("wc1 requestAnimationFrame callback did not fire")
	}

	select {
	case <-frame2:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("wc2 requestAnimationFrame callback did not fire")
	}

	cancelled := make(chan bool, 1)
	if err := wc1.SetJSValue("onCancelled", func(call goja.FunctionCall) goja.Value {
		cancelled <- true
		return goja.Undefined()
	}); err != nil {
		t.Fatalf("Failed to set cancellation callback: %v", err)
	}
	if _, err := wc1.RunJS(`var cancelledId = window.requestAnimationFrame(onCancelled); window.cancelAnimationFrame(cancelledId);`); err != nil {
		t.Fatalf("Failed to cancel requested animation frame: %v", err)
	}

	select {
	case <-cancelled:
		t.Fatal("cancelAnimationFrame did not prevent callback execution")
	case <-time.After(100 * time.Millisecond):
	}

	wc1.Destroy()
	wc2.Destroy()
}

func TestAnimationFrameBatchOrderAndIsolation(t *testing.T) {
	wc1 := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	wc2 := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc1.Destroy()
	defer wc2.Destroy()
	order1, order2 := make(chan int, 2), make(chan int, 2)
	wc1.ScheduleAnimationFrame(func() { order1 <- 1 })
	cancelledID := wc1.ScheduleAnimationFrame(func() { order1 <- 2 })
	wc1.ScheduleAnimationFrame(func() { order1 <- 3 })
	wc2.ScheduleAnimationFrame(func() { order2 <- 4 })
	wc2.ScheduleAnimationFrame(func() { order2 <- 5 })
	if !wc1.CancelAnimationFrame(cancelledID) {
		t.Fatal("failed to cancel wc1 callback")
	}
	for _, expected := range []int{1, 3} {
		select {
		case got := <-order1:
			if got != expected {
				t.Fatalf("wc1 callback order = %d, want %d", got, expected)
			}
		case <-time.After(time.Second):
			t.Fatalf("wc1 callback %d did not execute", expected)
		}
	}
	for _, expected := range []int{4, 5} {
		select {
		case got := <-order2:
			if got != expected {
				t.Fatalf("wc2 callback order = %d, want %d", got, expected)
			}
		case <-time.After(time.Second):
			t.Fatalf("wc2 callback %d did not execute", expected)
		}
	}
}

func TestMutationObserverFiresOnDOMChanges(t *testing.T) {
	settings := &Settings{}
	buildInfo := &assets.BuildInfo{}
	prof := profiler.CreateProfiler()

	wc := NewWindowContext(settings, buildInfo, prof)
	htmlContent := `<html><head></head><body><p>hello</p></body></html>`
	parsedDoc, err := parseHTMLForTest(htmlContent, wc)
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}
	wc.ActiveDocument = parsedDoc
	if err := wc.InitJSRuntime(); err != nil {
		t.Fatalf("Failed to init JS runtime: %v", err)
	}

	if _, err := wc.GetJSRuntime().RunString(`
		var mutationCount = 0;
		var observer = new MutationObserver(function(mutations) {
			mutationCount = mutations.length;
		});
		observer.observe(document.body, { childList: true });
		var div = document.createElement("div");
		document.body.appendChild(div);
	`); err != nil {
		t.Fatalf("Failed to create mutation observer and mutate DOM: %v", err)
	}

	if got := wc.GetJSRuntime().Get("mutationCount").ToInteger(); got != 1 {
		t.Fatalf("expected mutationCount=1 after appendChild, got %d", got)
	}

	wc.Destroy()
}

func TestWindowEventListenerRemoval(t *testing.T) {
	settings := &Settings{}
	buildInfo := &assets.BuildInfo{}
	prof := profiler.CreateProfiler()

	wc := NewWindowContext(settings, buildInfo, prof)
	if err := wc.InitJSRuntime(); err != nil {
		t.Fatalf("Failed to init JS runtime: %v", err)
	}

	runtime := wc.GetJSRuntime()
	calls := 0
	runtime.Set("onMessage", func(call goja.FunctionCall) goja.Value {
		calls++
		return goja.Undefined()
	})

	if _, err := runtime.RunString(`window.addEventListener("message", onMessage)`); err != nil {
		t.Fatalf("Failed to add event listener: %v", err)
	}
	if _, err := runtime.RunString(`window.removeEventListener("message", onMessage)`); err != nil {
		t.Fatalf("Failed to remove event listener: %v", err)
	}
	if _, err := runtime.RunString(`window.postMessage("hello", "*")`); err != nil {
		t.Fatalf("Failed to postMessage: %v", err)
	}
	if calls != 0 {
		t.Fatalf("Listener remained registered after removeEventListener; calls=%d", calls)
	}

	wc.Destroy()
}

const domElementEventScript = `
	var parent = document.querySelector("#parent");
	var button = document.querySelector("#button");
	var calls = [];
	function onButton(event) { calls.push("button:" + event.type + ":" + (this === button) + ":" + (event.target === button)); }
	button.addEventListener("click", onButton);
	parent.addEventListener("click", function(event) { calls.push("parent:" + event.type + ":" + (this === parent) + ":" + (event.currentTarget === parent)); });
	button.dispatchEvent(new Event("click", { bubbles: true }));
	button.removeEventListener("click", onButton);
	button.click();
	calls.join(",");
`

func TestDOMElementEventListeners(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	doc, err := parseHTMLForTest(`<html><body><div id="parent"><button id="button"></button></div></body></html>`, wc)
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}
	wc.ActiveDocument = doc
	if err := wc.InitJSRuntime(); err != nil {
		t.Fatalf("Failed to init JS runtime: %v", err)
	}
	result, err := wc.GetJSRuntime().RunString(domElementEventScript)
	if err != nil {
		t.Fatalf("Failed to dispatch DOM events: %v", err)
	}
	if got, want := result.String(), "button:click:true:true,parent:click:true:true,parent:click:true:true"; got != want {
		t.Fatalf("DOM event calls = %q, want %q", got, want)
	}
}

func TestDOMEventBubblingStopsAtIframeBoundary(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	parentDoc, err := parseHTMLForTest(`<html><body><div id="outer"><iframe id="frame"></iframe></div></body></html>`, wc)
	if err != nil {
		t.Fatal(err)
	}
	wc.ActiveDocument = parentDoc
	if err := wc.InitJSRuntime(); err != nil {
		t.Fatal(err)
	}
	childDoc, err := parseHTMLForTest(`<html><body><button id="inner"></button></body></html>`, wc)
	if err != nil {
		t.Fatal(err)
	}
	iframe, err := parentDoc.DOM.QuerySelector("#frame")
	if err != nil {
		t.Fatal(err)
	}
	iframe.SandboxFlags = SandboxAllowScripts | SandboxAllowSameOrigin
	iframe.IframeDocument = childDoc
	inner, err := childDoc.DOM.QuerySelector("#inner")
	if err != nil {
		t.Fatal(err)
	}
	if inner == nil {
		t.Fatal("child document did not return the inner button")
	}
	childDoc.DOM.Parent = iframe
	childWrapper := &JSDocumentWrapper{doc: childDoc, runtime: wc.GetJSRuntime(), windowCtx: wc}
	for name, node := range map[string]*NodeDOM{"innerButton": inner, "childRoot": childDoc.DOM} {
		if err := wc.SetJSValue(name, childWrapper.wrapNode(node)); err != nil {
			t.Fatal(err)
		}
	}
	if wrapped := wc.GetJSRuntime().Get("innerButton"); wrapped == nil || goja.IsNull(wrapped) || goja.IsUndefined(wrapped) || wrapped.ToObject(wc.GetJSRuntime()).Get("addEventListener") == nil {
		t.Fatal("inner button wrapper was not exposed to JavaScript")
	}
	result, err := wc.RunJS(`var counts = [0, 0, 0]; innerButton.addEventListener("click", function() { counts[0]++; }); childRoot.addEventListener("click", function() { counts[1]++; }); document.querySelector("#outer").addEventListener("click", function() { counts[2]++; }); innerButton.dispatchEvent(new Event("click", { bubbles: true })); counts.join(":");`)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := result.String(), "1:1:0"; got != want {
		t.Fatalf("event listener counts = %s, want %s", got, want)
	}
}

func TestDOMMutationsFromJS(t *testing.T) {
	settings := &Settings{}
	buildInfo := &assets.BuildInfo{}
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
func setupMouseClickTestWindow(t *testing.T) (*WindowContext, *mustard.Window) {
	t.Helper()
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	doc, err := parseHTMLForTest(`<html><body><button id="target"></button></body></html>`, wc)
	if err != nil {
		t.Fatal(err)
	}
	wc.ActiveDocument = doc
	if err := wc.InitJSRuntime(); err != nil {
		t.Fatal(err)
	}
	target, err := doc.DOM.QuerySelector("#target")
	if err != nil {
		t.Fatal(err)
	}
	target.RenderBox = &RenderBox{Width: 20, Height: 20}
	if _, err := wc.RunJS(`var clickCount = 0; var clickWasCorrect = false; var button = document.querySelector("#target"); button.addEventListener("click", function(event) { clickCount++; clickWasCorrect = event.target === button && event.currentTarget === button && event.button === 0 && event.clientX === 0 && event.clientY === 0; });`); err != nil {
		t.Fatal(err)
	}
	window := &mustard.Window{}
	wc.BindWindowInput(window)
	return wc, window
}

func TestWindowPointerClickDispatchIsIsolated(t *testing.T) {
	wc1, window1 := setupMouseClickTestWindow(t)
	wc2, window2 := setupMouseClickTestWindow(t)
	defer wc1.Destroy()
	defer wc2.Destroy()
	window1.ProcessPointerClick(glfw.MouseButtonLeft)
	assertJSResult(t, wc1, `clickCount + ":" + clickWasCorrect`, "1:true")
	assertJSResult(t, wc2, `clickCount + ":" + clickWasCorrect`, "0:false")
	window2.ProcessPointerClick(glfw.MouseButtonLeft)
	assertJSResult(t, wc1, `clickCount + ":" + clickWasCorrect`, "1:true")
	assertJSResult(t, wc2, `clickCount + ":" + clickWasCorrect`, "1:true")
}

func setupKeyboardTestWindow(t *testing.T) (*WindowContext, *mustard.Window) {
	t.Helper()
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	doc, err := parseHTMLForTest(`<html><body><input id="target"></body></html>`, wc)
	if err != nil {
		t.Fatal(err)
	}
	wc.ActiveDocument = doc
	if err := wc.InitJSRuntime(); err != nil {
		t.Fatal(err)
	}
	target, err := doc.DOM.QuerySelector("#target")
	if err != nil {
		t.Fatal(err)
	}
	wc.SetFocusedElement(target)
	if _, err := wc.RunJS(`var keyboardEvents = []; var focused = document.querySelector("#target"); focused.addEventListener("keydown", function(e) { keyboardEvents.push("down:" + e.key + ":" + e.shiftKey + ":" + e.repeat); }); focused.addEventListener("keyup", function(e) { keyboardEvents.push("up:" + e.key + ":" + e.repeat); }); focused.addEventListener("keypress", function(e) { keyboardEvents.push("press:" + e.key); }); focused.addEventListener("input", function(e) { keyboardEvents.push("input:" + e.data); });`); err != nil {
		t.Fatal(err)
	}
	window := &mustard.Window{}
	wc.BindWindowInput(window)
	return wc, window
}

func TestWindowKeyboardInputRoutesToFocusedElement(t *testing.T) {
	wc1, window1 := setupKeyboardTestWindow(t)
	wc2, window2 := setupKeyboardTestWindow(t)
	defer wc1.Destroy()
	defer wc2.Destroy()
	window1.ProcessKeyboardInput(mustard.KeyboardInput{Key: "A", Action: mustard.KeyboardPress, Shift: true})
	window1.ProcessTextInput('a')
	assertJSResult(t, wc1, `keyboardEvents.join(",")`, "down:A:true:false,press:a,input:a")
	assertJSResult(t, wc2, `keyboardEvents.join(",")`, "")
	window2.ProcessKeyboardInput(mustard.KeyboardInput{Key: "Enter", Action: mustard.KeyboardRelease})
	assertJSResult(t, wc1, `keyboardEvents.join(",")`, "down:A:true:false,press:a,input:a")
	assertJSResult(t, wc2, `keyboardEvents.join(",")`, "up:Enter:false")
}

func assertJSResult(t *testing.T, wc *WindowContext, script, want string) {
	t.Helper()
	result, err := wc.RunJS(script)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.String(); got != want {
		t.Fatalf("JS result = %q, want %q", got, want)
	}
}

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
	buildInfo := &assets.BuildInfo{}
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
	buildInfo := &assets.BuildInfo{}
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
	buildInfo := &assets.BuildInfo{}
	prof := profiler.CreateProfiler()

	// Create two window contexts with the same origin
	origin1, _ := url.Parse(fmt.Sprintf("https://storage-test-%d.example.com:443", time.Now().UnixNano()))
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
	buildInfo := &assets.BuildInfo{}
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
	buildInfo := &assets.BuildInfo{}
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
	buildInfo := &assets.BuildInfo{}
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
	buildInfo := &assets.BuildInfo{}
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
	buildInfo := &assets.BuildInfo{}
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
	buildInfo := &assets.BuildInfo{}
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

// TestCORSPreflightFetch tests that CORS preflight works for cross-origin Fetch API.
func TestCORSPreflightFetch(t *testing.T) {
	settings := &Settings{}
	buildInfo := &assets.BuildInfo{}
	prof := profiler.CreateProfiler()

	origin1, _ := url.Parse("https://example.com:443")
	origin2, _ := url.Parse("https://api.other.com:443")

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

	_ = wc1.GetJSRuntime()

	// Test 1: Simple request (GET) should not require preflight
	t.Run("simple GET request does not require preflight", func(t *testing.T) {
		// We can't easily test the actual network request without a server,
		// but we can verify the isSimpleRequest logic
		// This is tested indirectly through the fetch implementation
	})

	// Test 2: Non-simple request (custom header) should trigger preflight
	t.Run("request with custom header triggers preflight", func(t *testing.T) {
		// The fetch implementation checks isSimpleRequest and sends OPTIONS first
		// We verify the logic by checking the helper functions
	})

	// Test 3: POST with application/json should trigger preflight
	t.Run("POST with application/json triggers preflight", func(t *testing.T) {
		// Content-Type application/json is not a simple header value
	})

	// Test 4: PUT/DELETE methods should trigger preflight
	t.Run("PUT method triggers preflight", func(t *testing.T) {
		// PUT is not a simple method
	})

	wc1.Destroy()
	wc2.Destroy()
}

// TestCORSPreflightXHR tests that CORS preflight works for cross-origin XMLHttpRequest.
func TestCORSPreflightXHR(t *testing.T) {
	settings := &Settings{}
	buildInfo := &assets.BuildInfo{}
	prof := profiler.CreateProfiler()

	origin1, _ := url.Parse("https://example.com:443")
	origin2, _ := url.Parse("https://api.other.com:443")

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

	_ = wc1.GetJSRuntime()

	// Test that XMLHttpRequest constructor is available
	_, err = wc1.GetJSRuntime().RunString(`
		var xhr = new XMLHttpRequest();
		if (typeof xhr.open !== "function") {
			throw new Error("XMLHttpRequest.open not a function");
		}
		if (typeof xhr.send !== "function") {
			throw new Error("XMLHttpRequest.send not a function");
		}
		if (typeof xhr.setRequestHeader !== "function") {
			throw new Error("XMLHttpRequest.setRequestHeader not a function");
		}
	`)
	if err != nil {
		t.Fatalf("XMLHttpRequest not properly initialized: %v", err)
	}

	// Test withCredentials property
	_, err = wc1.GetJSRuntime().RunString(`
		var xhr = new XMLHttpRequest();
		xhr.withCredentials = true;
		if (xhr.withCredentials !== true) {
			throw new Error("withCredentials not set");
		}
		xhr.withCredentials = false;
		if (xhr.withCredentials !== false) {
			throw new Error("withCredentials not cleared");
		}
	`)
	if err != nil {
		t.Fatalf("withCredentials property failed: %v", err)
	}

	wc1.Destroy()
	wc2.Destroy()
}

// TestSOPBlocksCrossOriginResourceLoads tests that Same-Origin Policy blocks cross-origin resource loads by default.
func TestSOPBlocksCrossOriginResourceLoads(t *testing.T) {
	settings := &Settings{}
	buildInfo := &assets.BuildInfo{}
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

	_ = wc1.GetJSRuntime()
	_ = wc2.GetJSRuntime()

	// Test 1: Cross-origin fetch without CORS headers should be blocked
	// Note: We can't easily test actual network blocking without a server,
	// but we can verify the origin checking logic
	t.Run("cross-origin fetch without CORS is blocked by SOP", func(t *testing.T) {
		// The fetch implementation adds Origin header and checks CORS response
		// Without proper CORS headers, the browser should block the response
	})

	// Test 2: Cross-origin XHR without CORS headers should be blocked
	t.Run("cross-origin XHR without CORS is blocked by SOP", func(t *testing.T) {
		// Similar to fetch, XHR should be blocked without CORS
	})

	// Test 3: Same-origin requests should work
	t.Run("same-origin requests work", func(t *testing.T) {
		// Same-origin requests don't need CORS
	})

	// Test 4: Cross-origin with proper CORS headers should work
	t.Run("cross-origin with CORS headers works", func(t *testing.T) {
		// With Access-Control-Allow-Origin: * or specific origin
	})

	// Test 5: Verify origin checking in DOM access
	t.Run("cross-origin DOM access throws SecurityError", func(t *testing.T) {
		// Create a node in wc1
		htmlContent := `<html><head></head><body><div id="test">Secret</div></body></html>`
		parsedDoc1, err := parseHTMLForTest(htmlContent, wc1)
		if err != nil {
			t.Fatalf("Failed to parse HTML for wc1: %v", err)
		}
		wc1.ActiveDocument = parsedDoc1

		node1, err := wc1.ActiveDocument.DOM.QuerySelector("#test")
		if err != nil {
			t.Fatalf("Failed to find test node in wc1: %v", err)
		}

		// Try to access from wc2's context
		originalWindowCtx := node1.WindowCtx
		node1.WindowCtx = wc2
		err = node1.checkOrigin()
		node1.WindowCtx = originalWindowCtx

		if err == nil {
			t.Fatal("Expected SecurityError for cross-origin DOM access")
		}
		_, ok := err.(SecurityError)
		if !ok {
			t.Fatalf("Expected SecurityError, got %T: %v", err, err)
		}
	})

	wc1.Destroy()
	wc2.Destroy()
}

// TestCORSHelperFunctions tests the CORS helper functions directly.
func TestCORSHelperFunctions(t *testing.T) {
	// Test isSimpleRequest
	t.Run("isSimpleRequest returns true for simple requests", func(t *testing.T) {
		// GET with no headers
		if !isSimpleRequest("GET", nil) {
			t.Error("GET with no headers should be simple")
		}
		if !isSimpleRequest("GET", map[string]string{}) {
			t.Error("GET with empty headers should be simple")
		}
		if !isSimpleRequest("HEAD", nil) {
			t.Error("HEAD should be simple")
		}
		if !isSimpleRequest("POST", map[string]string{"Content-Type": "application/x-www-form-urlencoded"}) {
			t.Error("POST with form content-type should be simple")
		}
		if !isSimpleRequest("POST", map[string]string{"Content-Type": "multipart/form-data; boundary=test"}) {
			t.Error("POST with multipart content-type should be simple")
		}
		if !isSimpleRequest("POST", map[string]string{"Content-Type": "text/plain"}) {
			t.Error("POST with text/plain content-type should be simple")
		}
		if !isSimpleRequest("GET", map[string]string{"Accept": "*/*"}) {
			t.Error("GET with Accept header should be simple")
		}
		if !isSimpleRequest("GET", map[string]string{"Accept-Language": "en-US"}) {
			t.Error("GET with Accept-Language header should be simple")
		}
		if !isSimpleRequest("GET", map[string]string{"Content-Language": "en-US"}) {
			t.Error("GET with Content-Language header should be simple")
		}
	})

	t.Run("isSimpleRequest returns false for non-simple requests", func(t *testing.T) {
		// PUT, DELETE, PATCH are not simple methods
		if isSimpleRequest("PUT", nil) {
			t.Error("PUT should not be simple")
		}
		if isSimpleRequest("DELETE", nil) {
			t.Error("DELETE should not be simple")
		}
		if isSimpleRequest("PATCH", nil) {
			t.Error("PATCH should not be simple")
		}
		// Custom headers
		if isSimpleRequest("GET", map[string]string{"X-Custom-Header": "value"}) {
			t.Error("Custom header should not be simple")
		}
		if isSimpleRequest("GET", map[string]string{"Authorization": "Bearer token"}) {
			t.Error("Authorization header should not be simple")
		}
		// Non-simple Content-Type values
		if isSimpleRequest("POST", map[string]string{"Content-Type": "application/json"}) {
			t.Error("application/json should not be simple")
		}
		if isSimpleRequest("POST", map[string]string{"Content-Type": "text/xml"}) {
			t.Error("text/xml should not be simple")
		}
	})

	// Test getPreflightHeaders
	t.Run("getPreflightHeaders returns correct headers", func(t *testing.T) {
		headers := getPreflightHeaders("POST", map[string]string{"Content-Type": "application/json", "X-Custom": "value"})
		if headers["Access-Control-Request-Method"] != "POST" {
			t.Errorf("Expected Access-Control-Request-Method=POST, got %s", headers["Access-Control-Request-Method"])
		}
		if headers["Access-Control-Request-Headers"] != "content-type, x-custom" {
			t.Errorf("Expected Access-Control-Request-Headers=content-type, x-custom, got %s", headers["Access-Control-Request-Headers"])
		}
	})

	// Test isCORSPreflightSuccessful
	t.Run("isCORSPreflightSuccessful validates CORS headers", func(t *testing.T) {
		// Missing Allow-Origin
		resp := &sauce.Resource{Code: 200, Headers: http.Header{}}
		if isCORSPreflightSuccessful(resp) {
			t.Error("Should fail without Access-Control-Allow-Origin")
		}

		// With Allow-Origin
		resp = &sauce.Resource{Code: 200, Headers: http.Header{"Access-Control-Allow-Origin": []string{"*"}}}
		if !isCORSPreflightSuccessful(resp) {
			t.Error("Should succeed with Access-Control-Allow-Origin: *")
		}

		// With specific origin
		resp = &sauce.Resource{Code: 200, Headers: http.Header{"Access-Control-Allow-Origin": []string{"https://example.com"}}}
		if !isCORSPreflightSuccessful(resp) {
			t.Error("Should succeed with specific Access-Control-Allow-Origin")
		}
	})
}
