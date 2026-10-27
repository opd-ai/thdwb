package pages

import (
	"strings"
	"testing"
)

func TestRenderNewTabPageDefaults(t *testing.T) {
	body := RenderNewTabPage(nil)

	for _, want := range []string{
		"New Tab",
		"<form",
		`name="q"`,
		"https://duckduckgo.com/",
		"Search the web",
		"Shortcuts",
		"Wikipedia",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("RenderNewTabPage(nil) missing %q", want)
		}
	}

	// No optional sections should render with default data.
	for _, absent := range []string{"Bookmarks", "Recently Visited", "Containers"} {
		if strings.Contains(body, absent) {
			t.Errorf("RenderNewTabPage(nil) should not contain section %q", absent)
		}
	}
}

func TestRenderNewTabPageCustomData(t *testing.T) {
	data := &NewTabData{
		SearchAction: "https://example.com/search",
		SearchHint:   "Ask",
		Shortcuts: []NewTabShortcut{
			{Name: "Example", URL: "https://example.com/"},
		},
		Bookmarks: []NewTabShortcut{
			{Name: "Bookmark One", URL: "https://example.com/b1"},
		},
		RecentlyVisited: []NewTabShortcut{
			{Name: "recent.test", URL: "https://recent.test/page"},
		},
		Containers: []NewTabShortcut{
			{Name: "Work", URL: "thdwb://container/work/"},
		},
	}

	body := RenderNewTabPage(data)

	for _, want := range []string{
		"Ask",
		"Example",
		"Bookmarks",
		"Bookmark One",
		"Recently Visited",
		"https://recent.test/page",
		"Containers",
		"thdwb://container/work/",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("RenderNewTabPage(data) missing %q", want)
		}
	}

	// Custom shortcuts replace the defaults.
	if strings.Contains(body, "Wikipedia") {
		t.Error("custom shortcuts should replace default shortcuts")
	}
}

func TestRenderNewTabPageEscapesHTML(t *testing.T) {
	data := &NewTabData{
		Shortcuts: []NewTabShortcut{
			{Name: "<script>alert(1)</script>", URL: "https://x.test/"},
		},
	}

	body := RenderNewTabPage(data)
	if strings.Contains(body, "<script>") {
		t.Fatal("shortcut names must be HTML-escaped in the rendered page")
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Error("escaped shortcut name not found in rendered page")
	}
}
