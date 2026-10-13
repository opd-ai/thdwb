package hotdog

import (
	"testing"

	"github.com/danfragoso/thdwb/assets"
	"github.com/danfragoso/thdwb/profiler"
)

func newBindingTestContext(t *testing.T, markup string) *WindowContext {
	t.Helper()
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	t.Cleanup(wc.Destroy)
	if err := wc.SetOrigin("https://binding.example"); err != nil {
		t.Fatal(err)
	}
	doc, err := parseHTMLForTest(markup, wc)
	if err != nil {
		t.Fatal(err)
	}
	wc.ActiveDocument = doc
	if err := wc.InitJSRuntime(); err != nil {
		t.Fatal(err)
	}
	return wc
}

func TestDocumentBindingQueriesAndCreation(t *testing.T) {
	wc := newBindingTestContext(t, `<html><head><title>Page</title></head><body><p id="one" class="entry"></p><p class="entry"></p></body></html>`)
	wc.ActiveDocument.Title = "Page"
	assertJSResult(t, wc, `document.querySelectorAll("p").length + ":" + document.getElementById("one").tagName + ":" + document.getElementsByClassName("entry").length + ":" + document.getElementsByTagName("p").length`, "2:P:2:2")
	assertJSResult(t, wc, `document.documentElement.tagName + ":" + document.title()`, "HTML:Page")

	// Test createTextNode and appendChild - the text node will be appended after existing children
	result, err := wc.RunJS(`var text = document.createTextNode("new"); document.body.appendChild(text); document.body.lastChild().textContent`)
	if err != nil {
		t.Fatal(err)
	}
	if result.String() != "new" {
		t.Fatalf("JS result = %q, want \"new\"", result.String())
	}
}

func TestElementBindingTraversalMutationAndStyle(t *testing.T) {
	wc := newBindingTestContext(t, `<html><body><div id="target" class="old"></div></body></html>`)

	// Test full element binding traversal, mutation, and style
	script := `var n=document.getElementById("target"), first=document.createElement("span"), last=document.createElement("span"); first.textContent="first"; last.textContent="last"; n.appendChild(first); n.appendChild(last); var before=document.createElement("b"); n.insertBefore(before,first); var replacement=document.createElement("i"); n.replaceChild(replacement,before); n.removeChild(replacement); var copy=n.cloneNode(true); n.setAttribute("data-x","yes"); var attr=n.getAttribute("data-x"); n.removeAttribute("class"); attr+":"+n.childNodes().length+":"+first.parentNode().tagName+":"+first.nextSibling().firstChild().textContent+":"+last.previousSibling().firstChild().textContent+":"+copy.childNodes().length+":"+(n.style()===null)`
	if got, err := wc.RunJS(script); err != nil || got.String() != "yes:2:DIV:last:first:2:true" {
		t.Fatalf("element binding result = %v, %v", got, err)
	}
}

func TestWindowLocationStorageConsoleAndFetchValidation(t *testing.T) {
	wc := newBindingTestContext(t, `<html><body></body></html>`)
	assertJSResult(t, wc, `localStorage.clear(); sessionStorage.clear(); localStorage.setItem("b","2"); localStorage.setItem("a","1"); sessionStorage.setItem("s","v"); console.log("test"); console.warn("test"); console.error("test"); window.location().origin+":"+localStorage.key(0)+":"+localStorage.getItem("a")+":"+sessionStorage.getItem("s")+":"+localStorage.length`, "https://binding.example:a:1:v:2")
	if _, err := wc.RunJS(`fetch()`); err == nil {
		t.Fatal("fetch() without a resource did not fail")
	}
	if _, err := wc.RunJS(`fetch("http://[")`); err != nil {
		t.Fatalf("fetch() rejected invalid URL synchronously: %v", err)
	}
}

func TestXMLHttpRequestValidationAndConfiguration(t *testing.T) {
	wc := newBindingTestContext(t, `<html><body></body></html>`)
	if _, err := wc.RunJS(`var x=new XMLHttpRequest(); x.setRequestHeader("x","y")`); err == nil {
		t.Fatal("setRequestHeader() accepted an unopened request")
	}
	if _, err := wc.RunJS(`x.send()`); err == nil {
		t.Fatal("send() accepted an unopened request")
	}
	assertJSResult(t, wc, `var x=new XMLHttpRequest(), listener=function(){}; x.open("post","http://localhost"); x.setRequestHeader("Content-Type","text/plain"); x.withCredentials=true; x.addEventListener("load",listener); x.removeEventListener("load",listener); x.onreadystatechange=listener; x.status+":"+x.readyState+":"+x.withCredentials`, "0:1:true")
}
