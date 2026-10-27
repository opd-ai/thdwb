package pages

import (
	"bytes"
	"html/template"
	"strings"
	"unicode"
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
	if err := t.ExecuteTemplate(buf, "newtab", newTabTemplateData{
		SearchAction:    safeURL(data.SearchAction),
		SearchHint:      data.SearchHint,
		Shortcuts:       toTemplateShortcuts(data.Shortcuts),
		Bookmarks:       toTemplateShortcuts(data.Bookmarks),
		RecentlyVisited: toTemplateShortcuts(data.RecentlyVisited),
		Containers:      toTemplateShortcuts(data.Containers),
	}); err != nil {
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

// templateShortcut is a shortcut mapped into the template's URL attribute
// context. Names stay plain strings (HTML-escaped by html/template); URLs are
// marked safe after passing safeURL so internal thdwb:// schemes render
// unescaped while executable schemes are dropped.
type templateShortcut struct {
	Name string
	URL  template.URL
}

// newTabTemplateData is the shape the template receives.
type newTabTemplateData struct {
	SearchAction    template.URL
	SearchHint      string
	Shortcuts       []templateShortcut
	Bookmarks       []templateShortcut
	RecentlyVisited []templateShortcut
	Containers      []templateShortcut
}

func toTemplateShortcuts(shortcuts []NewTabShortcut) []templateShortcut {
	out := make([]templateShortcut, 0, len(shortcuts))
	for _, s := range shortcuts {
		out = append(out, templateShortcut{Name: s.Name, URL: safeURL(s.URL)})
	}
	return out
}

// unsafeURLSchemes are executable or local schemes that must never appear in
// a link or form action generated from data supplied by the browser's own
// state (bookmarks, history, container entries, settings). A page rendered
// through the new tab page with one of these targets could execute script on
// navigation.
var unsafeURLSchemes = []string{"javascript:", "data:", "vbscript:", "file://"}

// safeURL validates a link target for use in a generated page and marks it
// safe for html/template URL contexts. Without the template.URL wrapper,
// html/template rejects unknown schemes (such as thdwb://) in URL attribute
// contexts and renders them as the inert "#ZgotmplZ" placeholder, which would
// break every internal link on the new tab page. Unsafe targets return an
// empty URL, rendering a dead link instead of an executable one.
func safeURL(rawURL string) template.URL {
	// Control and space characters are ignored by browsers when parsing URL
	// attributes (they are stripped before scheme resolution), so the scheme
	// check must run against a copy with them removed: otherwise
	// " javascript:alert(1)" would slip past the prefix check and render as
	// an executable link.
	candidate := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return -1
		}
		return r
	}, strings.ToLower(rawURL))
	for _, scheme := range unsafeURLSchemes {
		if strings.HasPrefix(candidate, scheme) {
			return ""
		}
	}
	return template.URL(rawURL)
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
