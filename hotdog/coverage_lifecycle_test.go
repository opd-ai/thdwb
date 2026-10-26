package hotdog

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/danfragoso/thdwb/assets"
	"github.com/danfragoso/thdwb/profiler"
)

func writeTempFile(t *testing.T, path, content string) error {
	t.Helper()
	return os.WriteFile(path, []byte(content), 0o600)
}

func newBareWindowContext(t *testing.T) *WindowContext {
	t.Helper()
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	t.Cleanup(wc.Destroy)
	return wc
}

// waitForCounter polls get until it returns want or the deadline elapses.
func waitForCounter(t *testing.T, get func() int, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if get() == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for counter to reach %d, last seen %d", want, get())
}

// waitForJS polls the window's JS runtime until expr evaluates to want.
func waitForJS(t *testing.T, wc *WindowContext, expr, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		v, err := wc.RunJS(expr)
		if err == nil && v.String() == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q == %q", expr, want)
}

func TestWindowContextEvents(t *testing.T) {
	wc := newBareWindowContext(t)

	var got, nodeGot interface{}
	handler := func(data interface{}) { got = data }
	wc.RegisterEvent("test", handler)
	wc.EmitEvent("test", "hello")
	if got != "hello" {
		t.Fatalf("EmitEvent handler data = %v, want hello", got)
	}

	// Removing the handler should stop delivery.
	wc.RemoveEventByKey("test", functionKey(handler))
	got = nil
	wc.EmitEvent("test", "again")
	if got != nil {
		t.Fatalf("handler fired after removal, got %v", got)
	}

	// Node-scoped events: nil node is a no-op, matched node delivers.
	node := &NodeDOM{Type: NodeTypeElement, Element: "div"}
	wc.registerNodeEvent(nil, "click", 0, handler)
	wc.registerNodeEvent(node, "click", 1, func(data interface{}) { nodeGot = data })
	wc.EmitNodeEvent(node, "click", 42)
	if nodeGot != 42 {
		t.Fatalf("node event data = %v, want 42", nodeGot)
	}
	wc.RemoveNodeEventByKey(nil, "click", 0)
	wc.RemoveNodeEventByKey(node, "click", 1)
	nodeGot = nil
	wc.EmitNodeEvent(node, "click", 1)
	if nodeGot != nil {
		t.Fatalf("node handler fired after removal, got %v", nodeGot)
	}

	// Inactive windows must not emit.
	wc.SetActive(false)
	got = nil
	handler2 := func(data interface{}) { got = data }
	wc.RegisterEvent("test2", handler2)
	wc.EmitEvent("test2", "x")
	if got != nil {
		t.Fatalf("EmitEvent fired while inactive")
	}
	var nilWC *WindowContext
	if nilWC.IsActive() {
		t.Fatal("IsActive on nil window context should be false")
	}
}

func TestWindowContextMutationObservers(t *testing.T) {
	wc := newBareWindowContext(t)

	// Trigger is a no-op with nil runtime; verify registration/filtering.
	other := &NodeDOM{Type: NodeTypeElement, Element: "span"}
	target := &NodeDOM{Type: NodeTypeElement, Element: "div"}
	mo := &MutationObserver{observed: map[*NodeDOM]struct{}{other: {}}}
	mo2 := &MutationObserver{observed: map[*NodeDOM]struct{}{target: {}}}
	wc.RegisterMutationObserver(nil)
	wc.RegisterMutationObserver(mo)
	wc.RegisterMutationObserver(mo2)
	if len(wc.mutationObservers) != 2 {
		t.Fatalf("registered observers = %d, want 2", len(wc.mutationObservers))
	}
	wc.NotifyMutationObservers(nil, "childList")
	wc.NotifyMutationObservers(target, "attributes")
	wc.NotifyMutationObservers(other, "childList")

	wc.UnregisterMutationObserver(nil)
	wc.UnregisterMutationObserver(mo)
	if _, stillThere := wc.mutationObservers[mo]; stillThere {
		t.Fatal("observer still registered after unregister")
	}
}

func TestWindowContextLayoutCache(t *testing.T) {
	wc := newBareWindowContext(t)
	if _, ok := wc.GetLayoutCache("missing"); ok {
		t.Fatal("GetLayoutCache returned hit for missing key")
	}
	wc.SetLayoutCache("k", 42)
	val, ok := wc.GetLayoutCache("k")
	if !ok || val.(int) != 42 {
		t.Fatalf("GetLayoutCache = %v, %v; want 42, true", val, ok)
	}
	wc.ClearLayoutCache()
	if _, ok := wc.GetLayoutCache("k"); ok {
		t.Fatal("GetLayoutCache returned hit after clear")
	}
}

func TestWindowContextFocusAndInputState(t *testing.T) {
	wc := newBareWindowContext(t)
	if wc.GetFocusedElement() != nil {
		t.Fatal("focused element should start nil")
	}
	a := &NodeDOM{Type: NodeTypeElement, Element: "a", WindowCtx: wc}
	b := &NodeDOM{Type: NodeTypeElement, Element: "b", WindowCtx: wc}
	wc.SetFocusedElement(a)
	if wc.GetFocusedElement() != a {
		t.Fatal("SetFocusedElement did not set focus")
	}
	// Re-setting the same element is a no-op path.
	wc.SetFocusedElement(a)
	wc.SetFocusedElement(b)
	if wc.GetFocusedElement() != b {
		t.Fatal("focus did not move to b")
	}
	wc.SetFocusedElement(nil)
	if wc.GetFocusedElement() != nil {
		t.Fatal("focus was not cleared")
	}

	wc.SetInputState("cursor", 7)
	val, ok := wc.GetInputState("cursor")
	if !ok || val.(int) != 7 {
		t.Fatalf("GetInputState = %v, %v; want 7, true", val, ok)
	}
	if _, ok := wc.GetInputState("nope"); ok {
		t.Fatal("GetInputState hit for missing key")
	}
}

func TestWindowContextHTTPClientAndReflow(t *testing.T) {
	wc := newBareWindowContext(t)
	if wc.GetHTTPClient() == nil {
		t.Fatal("GetHTTPClient returned nil")
	}
	if wc.GetCookieJar() == nil {
		t.Fatal("GetCookieJar returned nil")
	}
	if wc.GetSessionStorageManager() == nil {
		t.Fatal("GetSessionStorageManager returned nil")
	}

	// queueReflow: nil node, duplicate root dedup, drain.
	if _, ok := wc.queueReflow(nil); ok {
		t.Fatal("queueReflow(nil) should not queue")
	}
	root := &NodeDOM{Type: NodeTypeElement, Element: "html"}
	child := &NodeDOM{Type: NodeTypeElement, Element: "body", Parent: root}
	root.Children = append(root.Children, child)
	queuedRoot, first := wc.queueReflow(child)
	if !first || queuedRoot != root {
		t.Fatalf("queueReflow = %v, %v; want root, true", queuedRoot, first)
	}
	if _, dup := wc.queueReflow(root); dup {
		t.Fatal("duplicate reflow for same root reported new")
	}
	drained := wc.DrainReflowQueue()
	if len(drained) != 1 || drained[0] != root {
		t.Fatalf("DrainReflowQueue = %v, want [root]", drained)
	}
	if len(wc.DrainReflowQueue()) != 0 {
		t.Fatal("DrainReflowQueue returned drained items twice")
	}
}

func TestWindowContextScheduleAfterDestroy(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	wc.Destroy()
	if id := wc.ScheduleAnimationFrame(func() {}); id != 0 {
		t.Fatalf("ScheduleAnimationFrame after Destroy = %d, want 0", id)
	}
	if wc.IsActive() {
		t.Fatal("destroyed window should be inactive")
	}
	wc.SetActive(true) // destroyed: no-op
	if wc.IsActive() {
		t.Fatal("SetActive(true) must not reactivate a destroyed window")
	}
	if got, err := wc.RunJS("1"); got != nil || err == nil {
		t.Fatalf("RunJS after Destroy = %v, %v; want nil, error", got, err)
	}
	if err := wc.SetJSValue("x", 1); err == nil {
		t.Fatal("SetJSValue after Destroy should fail")
	}
}

func TestWindowContextAnimationFrameCancelAndInactivate(t *testing.T) {
	wc := newBareWindowContext(t)
	if wc.CancelAnimationFrame(999) {
		t.Fatal("CancelAnimationFrame reported success for unknown id")
	}
	// ran is written by the timer goroutine and read by the test goroutine.
	var ran atomic.Int32
	id := wc.ScheduleAnimationFrame(func() { ran.Add(1) })
	// Cancel before the 16ms timer fires.
	if !wc.CancelAnimationFrame(id) {
		t.Fatal("CancelAnimationFrame failed for live id")
	}
	time.Sleep(30 * time.Millisecond)
	if got := ran.Load(); got != 0 {
		t.Fatalf("cancelled frame callback ran %d times", got)
	}

	// SetActive(false) mid-queue defers remaining callbacks until resume.
	wc.SetActive(false)
	id2 := wc.ScheduleAnimationFrame(func() { ran.Add(1) })
	wc.SetActive(true)
	waitForCounter(t, func() int { return int(ran.Load()) }, 1, 2*time.Second)
	if wc.CancelAnimationFrame(id2) {
		t.Fatal("CancelAnimationFrame succeeded for already-run id")
	}
}

func TestWindowContextRunJSInactive(t *testing.T) {
	wc := newBareWindowContext(t)
	wc.SetActive(false)
	if _, err := wc.RunJS("1"); err == nil {
		t.Fatal("RunJS succeeded on inactive window")
	}
	if err := wc.SetJSValue("x", 1); err == nil {
		t.Fatal("SetJSValue succeeded on inactive window")
	}
}

func TestGetOriginNilAndDestroy(t *testing.T) {
	wc := newBareWindowContext(t)
	if wc.GetOrigin().String() != "" {
		t.Fatalf("default origin = %q, want empty", wc.GetOrigin().String())
	}
	if err := wc.SetOrigin("https://example.org:8443"); err != nil {
		t.Fatal(err)
	}
	got := wc.GetOrigin()
	if got.Scheme != "https" || got.Host != "example.org:8443" {
		t.Fatalf("SetOrigin produced %+v", got)
	}
}

func TestFunctionKey(t *testing.T) {
	if functionKey(nil) != 0 {
		t.Fatal("functionKey(nil) != 0")
	}
	if functionKey(42) != 0 {
		t.Fatal("functionKey(non-func) != 0")
	}
	a := func() {}
	b := func() {}
	if functionKey(a) == 0 {
		t.Fatal("functionKey(func) == 0")
	}
	// A deterministic key must be stable across separate invocations.
	first, second := functionKey(a), functionKey(a)
	if first != second {
		t.Fatal("functionKey unstable for same function")
	}
	if functionKey(a) == functionKey(b) {
		t.Fatal("functionKey collides for distinct functions")
	}
}

func TestLoadSettings(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.json")
	if err := writeTempFile(t, valid, `{"homepage":"thdwb://home","windowWidth":800,"windowHeight":600,"hiDPI":true,"experimentalLayout":true}`); err != nil {
		t.Fatal(err)
	}
	s := LoadSettings(valid)
	if s.Homepage != "thdwb://home" || s.WindowWidth != 800 || !s.ExperimentalLayout {
		t.Fatalf("LoadSettings(valid) = %+v", s)
	}

	invalid := filepath.Join(dir, "bad.json")
	if err := writeTempFile(t, invalid, `{not json`); err != nil {
		t.Fatal(err)
	}
	s2 := LoadSettings(invalid)
	if s2 == nil {
		t.Fatal("LoadSettings(invalid) returned nil")
	}

	missing := filepath.Join(dir, "missing.json")
	s3 := LoadSettings(missing)
	if s3 == nil {
		t.Fatal("LoadSettings(missing) returned nil")
	}
}

func TestHistoryOperations(t *testing.T) {
	h := &History{}
	if h.Last() != nil || h.PageCount() != 0 || len(h.AllPages()) != 0 || len(h.NextPages()) != 0 {
		t.Fatalf("empty history not empty: %+v", h)
	}
	u1, _ := url.Parse("https://a.example/")
	u2, _ := url.Parse("https://b.example/")
	h.Push(u1)
	h.Push(u2)
	if h.PageCount() != 2 || h.Last() != u2 || len(h.AllPages()) != 2 {
		t.Fatalf("history after pushes = count %d last %v all %v", h.PageCount(), h.Last(), h.AllPages())
	}
	h.Pop()
	if h.Last() != u1 || len(h.NextPages()) != 1 {
		t.Fatalf("history after pop = last %v next %v", h.Last(), h.NextPages())
	}
	h.PopNext()
	if h.Last() != u2 || len(h.NextPages()) != 0 {
		t.Fatalf("history after popNext = last %v next %v", h.Last(), h.NextPages())
	}
	// Push clears the forward stack.
	h.Pop()
	if len(h.NextPages()) != 1 {
		t.Fatal("expected one next page")
	}
	h.Push(u1)
	if len(h.NextPages()) != 0 {
		t.Fatal("Push did not clear next pages")
	}
	// Pop on empty history is a no-op.
	empty := &History{}
	empty.Pop()
	empty.PopNext()
	if empty.PageCount() != 0 {
		t.Fatal("no-op history mutations changed state")
	}
}

func TestRenderBoxGetRectAndErrors(t *testing.T) {
	box := &RenderBox{Top: 1, Left: 2, Width: 3, Height: 4}
	top, left, width, height := box.GetRect()
	if top != 1 || left != 2 || width != 3 || height != 4 {
		t.Fatalf("GetRect = %v %v %v %v", top, left, width, height)
	}
	e := NoSuchElementError("x")
	if e.Error() != `no such element: "x"` {
		t.Fatalf("Error() = %q", e.Error())
	}
}

func TestResourceAndImageCaches(t *testing.T) {
	resCache := &ResourceCache{}
	if resCache.GetResource("k") != nil {
		t.Fatal("empty resource cache returned a hit")
	}
	res := &Resource{Key: "k", Body: "b"}
	resCache.AddResource(res)
	if resCache.GetResource("k") != res {
		t.Fatal("resource cache miss after add")
	}
	if resCache.GetResource("other") != nil {
		t.Fatal("resource cache hit for wrong key")
	}

	imgCache := &ImgCache{}
	if imgCache.GetImage("img") != nil {
		t.Fatal("empty image cache returned a hit")
	}
	imgCache.AddImage("img", []byte{1, 2})
	cached := imgCache.GetImage("img")
	if cached == nil || string(cached.Image) != string([]byte{1, 2}) {
		t.Fatalf("image cache = %+v", cached)
	}
	if imgCache.GetImage("nope") != nil {
		t.Fatal("image cache hit for wrong key")
	}
}

func TestCSPAllowEvalAndStyle(t *testing.T) {
	var nilCSP *ContentSecurityPolicy
	if !nilCSP.AllowEval() || !nilCSP.AllowInlineScript() || !nilCSP.AllowInlineStyle() {
		t.Fatal("nil CSP must allow everything")
	}

	noEval := NewContentSecurityPolicy("script-src 'self'")
	if noEval.AllowEval() {
		t.Fatal("CSP without 'unsafe-eval' allows eval")
	}
	withEval := NewContentSecurityPolicy("script-src 'unsafe-eval'")
	if !withEval.AllowEval() {
		t.Fatal("CSP with 'unsafe-eval' must allow eval")
	}
	evalViaDefault := NewContentSecurityPolicy("default-src 'unsafe-eval'")
	if !evalViaDefault.AllowEval() {
		t.Fatal("default-src 'unsafe-eval' must allow eval")
	}
	evalBlockedByAdditional := NewContentSecurityPolicy("script-src 'unsafe-eval'")
	evalBlockedByAdditional.additional = append(evalBlockedByAdditional.additional, NewContentSecurityPolicy("script-src 'self'"))
	if evalBlockedByAdditional.AllowEval() {
		t.Fatal("additional CSP without 'unsafe-eval' must block eval")
	}

	noInlineStyle := NewContentSecurityPolicy("style-src 'self'")
	if noInlineStyle.AllowInlineStyle() {
		t.Fatal("CSP style-src 'self' must block inline style")
	}
	withInlineStyle := NewContentSecurityPolicy("style-src 'unsafe-inline'")
	if !withInlineStyle.AllowInlineStyle() {
		t.Fatal("CSP style-src 'unsafe-inline' must allow inline style")
	}
	styleViaDefault := NewContentSecurityPolicy("default-src 'unsafe-inline'")
	if !styleViaDefault.AllowInlineStyle() {
		t.Fatal("default-src 'unsafe-inline' must allow inline style")
	}
	styleBlockedByAdditional := NewContentSecurityPolicy("style-src 'unsafe-inline'")
	styleBlockedByAdditional.additional = append(styleBlockedByAdditional.additional, NewContentSecurityPolicy("style-src 'self'"))
	if styleBlockedByAdditional.AllowInlineStyle() {
		t.Fatal("additional CSP must block inline style")
	}

	// AllowInlineScript with additional policies.
	inlineScriptBlockedByAdditional := NewContentSecurityPolicy("script-src 'unsafe-inline'")
	inlineScriptBlockedByAdditional.additional = append(inlineScriptBlockedByAdditional.additional, NewContentSecurityPolicy("default-src 'self'"))
	if inlineScriptBlockedByAdditional.AllowInlineScript() {
		t.Fatal("additional CSP must block inline script")
	}
}

func TestCSPSourceChecks(t *testing.T) {
	doc := &Document{URL: mustParseURL(t, "https://app.example/")}
	doc.ApplyContentSecurityPolicyHeaders(http.Header{
		"Content-Security-Policy": []string{"default-src 'self' https://css.example; script-src 'unsafe-inline'; img-src 'none'"},
	})
	csp := doc.CSP
	if !csp.AllowInlineScript() {
		t.Fatal("script-src 'unsafe-inline' must allow inline script")
	}
	// 'self' resolves against the document origin.
	if !csp.CheckStyleSource("https://app.example/a.css") {
		t.Fatal("CheckStyleSource rejected same-origin URL")
	}
	if csp.CheckStyleSource("https://evil.example/a.css") {
		t.Fatal("CheckStyleSource accepted unallowed origin")
	}
	// Explicit cross-origin source is allowed by default-src.
	if !csp.CheckStyleSource("https://css.example/site.css") {
		t.Fatal("CheckStyleSource rejected allowed cross-origin source")
	}
	// 'none' blocks all image sources.
	if csp.CheckImageSource("https://app.example/p.png") {
		t.Fatal("img-src 'none' must block all image sources")
	}

	// Scheme source and "*" source.
	schemeCSP := NewContentSecurityPolicy("script-src https:")
	if !schemeCSP.CheckScriptSource("https://any.example/x.js") {
		t.Fatal("scheme source 'https:' must allow https URLs")
	}
	if schemeCSP.CheckScriptSource("http://insecure.example/x.js") {
		t.Fatal("scheme source 'https:' must not allow http URLs")
	}
	wildCSP := NewContentSecurityPolicy("script-src *")
	if !wildCSP.CheckScriptSource("https://any.example/x.js") {
		t.Fatal("'*' must allow http(s) sources")
	}
	// Path-scoped source.
	pathCSP := NewContentSecurityPolicy("script-src https://cdn.example/lib/")
	if !pathCSP.CheckScriptSource("https://cdn.example/lib/app.js") {
		t.Fatal("path-scoped source rejected matching path")
	}
	if pathCSP.CheckScriptSource("https://cdn.example/other/app.js") {
		t.Fatal("path-scoped source accepted non-matching path")
	}
}
