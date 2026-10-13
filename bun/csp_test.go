package bun

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/danfragoso/thdwb/hotdog"
)

func TestFetchNodeImageBlocksDisallowedCSPSource(t *testing.T) {
	pageURL, err := url.Parse("https://example.com/index.html")
	if err != nil {
		t.Fatal(err)
	}
	document := &hotdog.Document{URL: pageURL}
	headers := make(http.Header)
	headers.Set("Content-Security-Policy", "img-src 'self'")
	document.ApplyContentSecurityPolicyHeaders(headers)
	node := &hotdog.NodeDOM{
		Element:    "img",
		Attributes: []*hotdog.Attribute{{Name: "src", Value: "https://evil.example/image.png"}},
		Document:   document,
	}

	if _, err := fetchNodeImage(node); err == nil {
		t.Fatal("expected image fetch to be blocked by CSP")
	} else if _, ok := err.(hotdog.SecurityError); !ok {
		t.Fatalf("expected SecurityError, got %T: %v", err, err)
	}
}
