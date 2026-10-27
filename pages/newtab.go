package pages

import (
	"bytes"
	"html/template"
	"strings"
)

// NewTabShortcut is a named link shown on the new tab page. It is used for
// site shortcuts, bookmarks, recently visited pages, and container entries.
type NewTabShortcut struct {
	// Name is the human-readable link label.
	Name string
	// URL is the link target.
	URL string
}

// NewTabData is the data the new tab page is rendered from. Every field is
// optional; zero values fall back to reasonable defaults.
type NewTabData struct {
	// SearchAction is the search form action URL. The query is submitted by
	// the form input named "q" (e.g. "https://duckduckgo.com/").
	SearchAction string

	// SearchHint is the placeholder text shown in the search box.
	SearchHint string

	// Shortcuts is the list of site shortcuts shown on the page.
	Shortcuts []NewTabShortcut

	// Bookmarks is the list of bookmarks shown on the page.
	Bookmarks []NewTabShortcut

	// RecentlyVisited is the list of recently visited pages shown on the
	// page, most recent first. The browser shell may pass its
	// recently-closed tab stack here instead.
	RecentlyVisited []NewTabShortcut

	// Containers is the list of container tabs shown on the page.
	Containers []NewTabShortcut
}

// RenderNewTabPage renders the browser's new tab page from the given data.
// A nil data value (or empty fields) is filled with sensible defaults.
func RenderNewTabPage(data *NewTabData) string {
	if data == nil {
		data = &NewTabData{}
	}
	if strings.TrimSpace(data.SearchAction) == "" {
		data.SearchAction = "https://duckduckgo.com/"
	}
	if strings.TrimSpace(data.SearchHint) == "" {
		data.SearchHint = "Search the web"
	}
	if len(data.Shortcuts) == 0 {
		data.Shortcuts = defaultNewTabShortcuts()
	}

	buf := new(bytes.Buffer)
	t, err := template.New("newtab").Parse(newTabTemplate())
	if err != nil {
		return err.Error()
	}
	if err := t.ExecuteTemplate(buf, "newtab", data); err != nil {
		return err.Error()
	}

	return buf.String()
}

// defaultNewTabShortcuts returns the shortcuts shown when the caller does not
// provide its own.
func defaultNewTabShortcuts() []NewTabShortcut {
	return []NewTabShortcut{
		{Name: "DuckDuckGo", URL: "https://duckduckgo.com/"},
		{Name: "Wikipedia", URL: "https://www.wikipedia.org/"},
		{Name: "MDN Web Docs", URL: "https://developer.mozilla.org/"},
		{Name: "GitHub", URL: "https://github.com/"},
	}
}

// newTabTemplate returns the new tab page HTML template. The markup is
// intentionally simple (divs, headings, links, one form) so it renders
// reliably through the ketchup parser and bun layout engine.
func newTabTemplate() string {
	return `
<html lang="en">
<head>
<meta charset="UTF-8">
<title>New Tab</title>
</head>
<body>
<div>
<h1>New Tab</h1>
</div>
<div>
<form action="{{.SearchAction}}" method="get">
<input type="text" name="q" placeholder="{{.SearchHint}}">
</form>
</div>
{{if .Shortcuts}}
<div>
<h2>Shortcuts</h2>
{{range .Shortcuts}}
<div><a href="{{.URL}}">{{.Name}}</a></div>
{{end}}
</div>
{{end}}
{{if .Bookmarks}}
<div>
<h2>Bookmarks</h2>
{{range .Bookmarks}}
<div><a href="{{.URL}}">{{.Name}}</a></div>
{{end}}
</div>
{{end}}
{{if .RecentlyVisited}}
<div>
<h2>Recently Visited</h2>
{{range .RecentlyVisited}}
<div><a href="{{.URL}}">{{.Name}}</a></div>
{{end}}
</div>
{{end}}
{{if .Containers}}
<div>
<h2>Containers</h2>
{{range .Containers}}
<div><a href="{{.URL}}">{{.Name}}</a></div>
{{end}}
</div>
{{end}}
</body>
</html>
`
}
