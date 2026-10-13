package hotdog

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/danfragoso/thdwb/assets"
	"github.com/danfragoso/thdwb/profiler"
)

func TestCSPHeadersAndSourceMatching(t *testing.T) {
	pageURL, err := url.Parse("https://example.com/page/index.html")
	if err != nil {
		t.Fatal(err)
	}
	document := &Document{URL: pageURL}
	headers := make(http.Header)
	headers.Add("Content-Security-Policy", "default-src 'none'; img-src 'self' https://images.example")
	headers.Add("Content-Security-Policy", "img-src 'self' https://images.example")
	document.ApplyContentSecurityPolicyHeaders(headers)

	for _, source := range []string{
		"/images/logo.png",
		"https://example.com:443/image.png",
		"https://images.example/path/image.png",
	} {
		if err := document.CheckCSPResource("img", source); err != nil {
			t.Errorf("expected %q to be allowed: %v", source, err)
		}
	}
	for _, source := range []string{
		"https://blocked.example/image.png",
		"data:image/png;base64,AAAA",
	} {
		if err := document.CheckCSPResource("img", source); err == nil {
			t.Errorf("expected %q to be blocked", source)
		}
	}

	headers.Set("Content-Security-Policy", "img-src *")
	document.ApplyContentSecurityPolicyHeaders(headers)
	if err := document.CheckCSPResource("img", "data:image/png;base64,AAAA"); err == nil {
		t.Fatal("expected wildcard source to reject data URLs")
	}

	document.ApplyContentSecurityPolicyHeaders(nil)
	if document.CSP != nil {
		t.Fatal("expected removing the CSP response header to clear the policy")
	}
}

func TestCSPValidatesMarkupAndResources(t *testing.T) {
	pageURL, err := url.Parse("https://example.com/index.html")
	if err != nil {
		t.Fatal(err)
	}
	document := &Document{URL: pageURL}
	headers := make(http.Header)
	headers.Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'")
	document.ApplyContentSecurityPolicyHeaders(headers)

	blockedMarkup := []struct {
		markup    string
		directive string
	}{
		{`<script>alert(1)</script>`, "script-src"},
		{`<div onclick="run()"></div>`, "script-src"},
		{`<div style="color:red"></div>`, "style-src"},
		{`<style>body { color: red }</style>`, "style-src"},
		{`<script src="https://evil.example/app.js"></script>`, "script-src"},
		{`<img src="https://evil.example/image.png">`, "img-src"},
		{`<link rel="stylesheet" href="https://evil.example/site.css">`, "style-src"},
	}
	for _, test := range blockedMarkup {
		err := document.ValidateMarkup(test.markup)
		if err == nil || !strings.Contains(err.Error(), test.directive) {
			t.Errorf("ValidateMarkup(%q) error = %v, want a %s violation", test.markup, err, test.directive)
		}
	}

	for _, markup := range []string{
		`<script src="/app.js"></script>`,
		`<img src="https://example.com/image.png">`,
		`<link rel="stylesheet" href="/site.css">`,
	} {
		if err := document.ValidateMarkup(markup); err != nil {
			t.Errorf("expected %q to be allowed: %v", markup, err)
		}
	}
}

func TestCSPBlocksJavaScriptDOMMutations(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	if err := wc.SetOrigin("https://example.com"); err != nil {
		t.Fatal(err)
	}
	document, err := parseHTMLForTest("<html><body></body></html>", wc)
	if err != nil {
		t.Fatal(err)
	}
	headers := make(http.Header)
	headers.Set("Content-Security-Policy", "default-src 'self'")
	document.ResponseHeaders = headers
	wc.ActiveDocument = document
	if err := wc.InitJSRuntime(); err != nil {
		t.Fatal(err)
	}

	for _, script := range []string{
		`var image = document.createElement("img"); image.setAttribute("src", "https://evil.example/image.png");`,
		`document.body.innerHTML = '<div onclick="run()"></div>';`,
	} {
		if _, err := wc.RunJS(script); err == nil || !strings.Contains(err.Error(), "SecurityError") {
			t.Errorf("RunJS(%q) error = %v, want SecurityError", script, err)
		}
	}
}
