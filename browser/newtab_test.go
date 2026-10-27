package main

import (
	"strings"
	"testing"

	ketchup "github.com/danfragoso/thdwb/ketchup"
	pages "github.com/danfragoso/thdwb/pages"
)

// TestNewTabPageParses verifies the new tab page renders HTML that the
// ketchup parser turns into a well-formed DOM: a heading, a form with a text
// input, and at least one link per default shortcut.
func TestNewTabPageParses(t *testing.T) {
	body := pages.RenderNewTabPage(nil)

	doc := ketchup.ParseHTMLDocument(body, nil)
	if doc == nil || doc.DOM == nil {
		t.Fatal("new tab page failed to parse")
	}

	h1 := firstByTag(doc.DOM, "h1")
	if h1 == nil || elementText(h1) != "New Tab" {
		t.Errorf("h1 text = %q, want \"New Tab\"", elementText(h1))
	}

	if form := firstByTag(doc.DOM, "form"); form == nil {
		t.Error("search form missing from new tab page DOM")
	}
	if input := firstByTag(doc.DOM, "input"); input == nil {
		t.Error("search input missing from new tab page DOM")
	}

	var links int
	collectByTag(doc.DOM, "a", &links, nil)
	if links == 0 {
		t.Error("no links found in new tab page DOM; expected default shortcuts")
	}
}

// TestNewTabPageWithBookmarksParses verifies caller-supplied bookmarks and
// containers survive the parse.
func TestNewTabPageWithBookmarksParses(t *testing.T) {
	body := pages.RenderNewTabPage(&pages.NewTabData{
		Bookmarks: []pages.NewTabShortcut{
			{Name: "Bookmark One", URL: "https://example.com/b1"},
		},
		Containers: []pages.NewTabShortcut{
			{Name: "Work", URL: "thdwb://container/work/"},
		},
	})

	doc := ketchup.ParseHTMLDocument(body, nil)
	if doc == nil || doc.DOM == nil {
		t.Fatal("new tab page with bookmarks failed to parse")
	}

	var text strings.Builder
	collectText(doc.DOM, &text)
	for _, want := range []string{"Bookmark One", "Work"} {
		if !strings.Contains(text.String(), want) {
			t.Errorf("DOM text missing %q", want)
		}
	}
}
