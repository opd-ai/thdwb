package hotdog

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// pollJS polls a JS expression until it evaluates to want.
func pollJS(t *testing.T, wc *WindowContext, expr, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		v, err := wc.RunJS(expr)
		if err == nil && v.String() == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for JS %q == %q", expr, want)
}

func TestWindowLocationProperties(t *testing.T) {
	wc := newBindingTestContext(t, `<html><body></body></html>`)
	assertJSResult(t, wc, `var l=window.location(); l.protocol+l.host+l.hostname+l.port+l.pathname+l.origin`, "https:binding.examplebinding.example/https://binding.example")

	// A bare context still reports a (non-nil) empty origin, so href is the
	// empty URL string and origin collapses to "://".
	wc2 := newBareWindowContext(t)
	if err := wc2.InitJSRuntime(); err != nil {
		t.Fatal(err)
	}
	assertJSResult(t, wc2, `var l=window.location(); l.href+":"+l.origin`, ":://")
}

func TestWindowGetStyleWithPopulatedStylesheet(t *testing.T) {
	wc := newBindingTestContext(t, `<html><body><div id="s"></div></body></html>`)
	// Populate the node's Stylesheet from Go, then read it via JS .style().
	body, err := wc.ActiveDocument.GetBody()
	if err != nil {
		t.Fatal(err)
	}
	div := body.Children[0]
	div.Style = &Stylesheet{
		Color:           &ColorRGBA{R: 1, G: 0, B: 0, A: 1},
		BackgroundColor: &ColorRGBA{R: 0, G: 1, B: 0, A: 0.5},
		FontSize:        16,
		FontWeight:      700,
		FontFamily:      "serif",
		Display:         "block",
		Position:        "relative",
		Width:           100,
		Height:          50,
	}
	assertJSResult(t, wc, `var s=document.getElementById("s").style(); s.color+"|"+s.backgroundColor+"|"+s.fontSize+"|"+s.fontWeight+"|"+s.fontFamily+"|"+s.display+"|"+s.position+"|"+s.width+"|"+s.height`, "rgba(255, 0, 0, 1.000000)|rgba(0, 255, 0, 0.500000)|16.000000px|700|serif|block|relative|100.000000px|50.000000px")

	// A node with no stylesheet returns null (already asserted elsewhere,
	// keep the null branch covered here too).
	bare := wc.ActiveDocument.CreateElement("span")
	div.appendChildNode(bare)
	assertJSResult(t, wc, `document.getElementById("s").lastChild().style()===null`, "true")
}

func TestFetchSameOriginAndCORS(t *testing.T) {
	corsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "OPTIONS":
			w.Header().Set("Access-Control-Allow-Origin", "https://app.example")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "x-custom")
			w.WriteHeader(http.StatusNoContent)
		case "DELETE":
			if r.Header.Get("X-Custom") != "yes" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", "https://app.example")
			w.Header().Set("Access-Control-Expose-Headers", "X-Extra")
			w.Header().Set("X-Extra", "hidden-otherwise")
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			// No CORS headers: cross-origin readers must see an opaque body.
			_, _ = w.Write([]byte(`plain-body`))
		}
	}))
	t.Cleanup(corsServer.Close)
	corsOrigin := corsServer.URL

	sameServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"a":1,"b":[1,2]}`))
			return
		}
		if r.URL.Path == "/err" {
			_, _ = w.Write([]byte("error: bad thing"))
			return
		}
		_, _ = w.Write([]byte("hello"))
	}))
	t.Cleanup(sameServer.Close)

	wc := newBindingTestContext(t, `<html><body></body></html>`)
	if err := wc.SetOrigin(sameServer.URL); err != nil {
		t.Fatal(err)
	}

	// Same-origin fetch: text() resolves with the body.
	_, err := wc.RunJS(`var so=""; fetch("` + sameServer.URL + `").then(function(r){so=r.status+":"+r.statusText+":"+r.ok; return r.text();}).then(function(t){so=so+":"+t;}); "started"`)
	if err != nil {
		t.Fatal(err)
	}
	pollJS(t, wc, `so`, "200:OK:true:hello")

	// json() resolves with the parsed object. Compare field values rather
	// than the raw string: JSON.stringify does not guarantee key order.
	_, err = wc.RunJS(`var js=""; fetch("` + sameServer.URL + `/json").then(function(r){return r.json();}).then(function(o){js=o.a+","+o.b[0]+","+o.b[1]+","+o.b.length;}); "started"`)
	if err != nil {
		t.Fatal(err)
	}
	pollJS(t, wc, `js`, `1,1,2,2`)

	// A 200 response whose body merely contains the word "error" still
	// resolves normally (only a code-0 network failure rejects).
	_, err = wc.RunJS(`var fe=""; fetch("` + sameServer.URL + `/err").then(function(r){fe=r.status+":"; return r.text();}).then(function(t){fe=fe+t;}); "started"`)
	if err != nil {
		t.Fatal(err)
	}
	pollJS(t, wc, `fe`, "200:error: bad thing")

	// Cross-origin without CORS headers: opaque response (status 0, empty body).
	_, err = wc.RunJS(`var co=""; fetch("` + corsOrigin + `").then(function(r){co=r.status+"|"; return r.text();}).then(function(t){co=co+"|"+t;}); "started"`)
	if err != nil {
		t.Fatal(err)
	}
	pollJS(t, wc, `co`, "0||")

	// Cross-origin with CORS + preflight (custom header forces OPTIONS).
	// The server whitelists https://app.example, so this window uses that
	// origin. Header keys keep their original case, so X-Extra is exposed
	// as-is.
	wcCors := newBindingTestContext(t, `<html><body></body></html>`)
	if err := wcCors.SetOrigin("https://app.example"); err != nil {
		t.Fatal(err)
	}
	_, err = wcCors.RunJS(`var cf=""; fetch("` + corsOrigin + `", {method:"DELETE", headers:{"X-Custom":"yes"}}).then(function(r){cf=r.status+":"+r.headers["X-Extra"]; return r.json();}).then(function(o){cf=cf+":"+JSON.stringify(o);}); "started"`)
	if err != nil {
		t.Fatal(err)
	}
	pollJS(t, wcCors, `cf`, "200:hidden-otherwise:{\"ok\":true}")
}

func TestXMLHttpRequestLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/post" {
			_, _ = w.Write([]byte("posted"))
			return
		}
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("teapot"))
	}))
	t.Cleanup(server.Close)

	wc := newBindingTestContext(t, `<html><body></body></html>`)
	if err := wc.SetOrigin(server.URL); err != nil {
		t.Fatal(err)
	}
	script := `var log=""; var x=new XMLHttpRequest(); x.open("GET","` + server.URL + `"); x.onreadystatechange=function(e){log+=x.readyState+"."}; x.addEventListener("loadend",function(e){log+="end:"+x.status+":"+x.statusText+":"+x.responseText+":"+x.response;}); x.addEventListener("load",function(e){log+="load;"}); x.send(); "started"`
	if _, err := wc.RunJS(script); err != nil {
		t.Fatal(err)
	}
	// readyState advances 3 (send) then 4 (response frame); the impl fires the
	// load event before loadend.
	pollJS(t, wc, `log`, "3.4.load;end:418:Unknown:teapot:teapot")

	// POST with body and default Content-Type.
	script2 := `var p=""; var y=new XMLHttpRequest(); y.open("POST","` + server.URL + `/post"); y.addEventListener("load",function(){p=y.status+":"+y.responseText;}); y.send("payload"); "started"`
	if _, err := wc.RunJS(script2); err != nil {
		t.Fatal(err)
	}
	pollJS(t, wc, `p`, "200:posted")

	// Unparseable URL dispatches the error listener.
	script3 := `var er=""; var z=new XMLHttpRequest(); z.open("GET","http://[::1"); z.addEventListener("error",function(){er="errored:"+z.readyState;}); z.send(); "started"`
	if _, err := wc.RunJS(script3); err != nil {
		t.Fatal(err)
	}
	pollJS(t, wc, `er`, "errored:4")
}

func TestPostMessageCrossWindow(t *testing.T) {
	// Sender window.
	sender := newBindingTestContext(t, `<html><body></body></html>`)
	// Receiver windows: one same-origin, one different.
	receiverA := newBindingTestContext(t, `<html><body></body></html>`)
	receiverB := newBindingTestContext(t, `<html><body></body></html>`)
	if err := receiverB.SetOrigin("https://other.example"); err != nil {
		t.Fatal(err)
	}
	// Make sender origin the same as receiverA.
	if err := sender.SetOrigin("https://binding.example"); err != nil {
		t.Fatal(err)
	}

	// Install listeners with a distinct global name per window.
	_, _ = sender.RunJS(`window.__sent=false;`)
	if _, err := receiverA.RunJS(`window.__recvA=""; window.addEventListener("message", function(e){ window.__recvA = e.data + "@" + e.origin; }); "ok"`); err != nil {
		t.Fatal(err)
	}
	if _, err := receiverB.RunJS(`window.__recvB=""; window.addEventListener("message", function(e){ window.__recvB = e.data + "@" + e.origin; }); "ok"`); err != nil {
		t.Fatal(err)
	}

	// Broadcast to all windows.
	if _, err := sender.RunJS(`window.postMessage("hello", "*"); "sent"`); err != nil {
		t.Fatal(err)
	}
	wait := func(wc *WindowContext, expr, want string) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			v, err := wc.RunJS(expr)
			if err == nil && v.String() == want {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatalf("%s did not receive message (want %q)", expr, want)
	}
	wait(receiverA, `window.__recvA`, "hello@https://binding.example")
	wait(receiverB, `window.__recvB`, "hello@https://binding.example")

	// Targeted postMessage only reaches matching origins.
	if _, err := sender.RunJS(`window.postMessage("only-b", "https://other.example"); "sent"`); err != nil {
		t.Fatal(err)
	}
	wait(receiverB, `window.__recvB`, "only-b@https://binding.example")
	if v, _ := receiverA.RunJS(`window.__recvA`); v.String() != "hello@https://binding.example" {
		t.Fatalf("receiverA received an off-target message: %q", v.String())
	}

	// Invalid targetOrigin is rejected.
	if _, err := sender.RunJS(`postMessage("x", "not an origin")`); err == nil {
		t.Fatal("postMessage accepted invalid targetOrigin")
	}
}

func TestRequestAnimationFrameBindings(t *testing.T) {
	wc := newBindingTestContext(t, `<html><body></body></html>`)
	// Non-function callback is rejected.
	if _, err := wc.RunJS(`requestAnimationFrame(42)`); err == nil {
		t.Fatal("requestAnimationFrame accepted non-function callback")
	}
	if _, err := wc.RunJS(`requestAnimationFrame()`); err == nil {
		t.Fatal("requestAnimationFrame accepted missing callback")
	}
	// Valid callback runs on the next frame and receives a timestamp.
	_, err := wc.RunJS(`var stamped=false; var id=requestAnimationFrame(function(t){stamped=(t>0)?t.toFixed(0):"0";}); var c1=cancelAnimationFrame(99999)===undefined; window.__id=id; "ok"`)
	if err != nil {
		t.Fatal(err)
	}
	pollJS(t, wc, `stamped==="false"?"no":"yes"`, "yes")
	// Cancel a pending frame.
	_, err = wc.RunJS(`var cancelled=false; var id2=requestAnimationFrame(function(){cancelled=true;}); cancelAnimationFrame(id2); window.__cancelled=cancelled; "ok"`)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Millisecond)
	assertJSResult(t, wc, `window.__cancelled`, "false")
}

func TestMutationObserverJSLifecycle(t *testing.T) {
	wc := newBindingTestContext(t, `<html><body><div id="t"></div></body></html>`)
	if _, err := wc.RunJS(`new MutationObserver()`); err == nil {
		t.Fatal("MutationObserver without callback should throw")
	}
	script := `var seen=[]; var mo=new MutationObserver(function(records){seen.push(records[0].type);}); mo.observe(document.getElementById("t")); document.getElementById("t").setAttribute("data-x","1"); mo.takeRecords().length+":"+seen.length;`
	// setAttribute triggers NotifyMutationObservers synchronously.
	assertJSResult(t, wc, script, "0:1")
	// disconnect: further mutations are not reported.
	if _, err := wc.RunJS(`var mo2=window.__mo; mo.disconnect();`); err != nil {
		t.Fatal(err)
	}
	if _, err := wc.RunJS(`var mo3=new MutationObserver(function(){}); window.__mo=mo3; mo3.observe(document.getElementById("t")); mo3.disconnect(); "ok"`); err != nil {
		t.Fatal(err)
	}
}
