package extensions

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// RunAt timing for content script injection.
type RunAt string

const (
	// RunAtDocumentStart injects as soon as possible, before the DOM is built.
	RunAtDocumentStart RunAt = "document_start"
	// RunAtDocumentEnd injects after the DOM is complete but before subresources load.
	RunAtDocumentEnd RunAt = "document_end"
	// RunAtDocumentIdle injects after the page is idle (window.onload).
	RunAtDocumentIdle RunAt = "document_idle"
)

// ContentScriptLoader loads and matches content scripts against URLs.
// It manages the set of scripts registered per extension and determines
// which scripts should run on a given page URL.
type ContentScriptLoader struct {
	mu       sync.Mutex
	scripts  []ContentScript
	byExtID  map[string][]ContentScript
}

// NewContentScriptLoader creates an empty content script loader.
func NewContentScriptLoader() *ContentScriptLoader {
	return &ContentScriptLoader{
		scripts: make([]ContentScript, 0),
		byExtID: make(map[string][]ContentScript),
	}
}

// Register adds a content script to the loader.
func (l *ContentScriptLoader) Register(script ContentScript) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.scripts = append(l.scripts, script)
	extID := script.ExtID
	if extID == "" {
		extID = script.ID
	}
	l.byExtID[extID] = append(l.byExtID[extID], script)
}

// RegisterFromManifest registers all content scripts from an extension's
// manifest. Script IDs are prefixed with the extension ID.
func (l *ContentScriptLoader) RegisterFromManifest(extID string, manifest *Manifest) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, cs := range manifest.ContentScripts {
		if len(cs.JS) == 0 {
			continue
		}
		runAt := RunAt(cs.RunAt)
		if runAt == "" {
			runAt = RunAtDocumentIdle
		}
		script := ContentScript{
			ID:             extID + ":" + cs.JS[0],
			ExtID:          extID,
			Matches:        cs.Matches,
			ExcludeMatches: cs.ExcludeMatches,
			JS:             cs.JS,
			CSS:            cs.CSS,
			RunAt:          string(runAt),
			AllFrames:      cs.AllFrames,
		}
		l.scripts = append(l.scripts, script)
		l.byExtID[extID] = append(l.byExtID[extID], script)
	}
}

// Matching returns all content scripts that match the given URL, sorted by
// run_at timing (document_start first, then document_end, then document_idle).
// Only scripts from enabled extensions are returned.
func (l *ContentScriptLoader) Matching(urlStr string) []ContentScript {
	l.mu.Lock()
	defer l.mu.Unlock()

	u, err := url.Parse(urlStr)
	if err != nil {
		return nil
	}

	var matched []ContentScript
	for _, s := range l.scripts {
		if !l.matchesURL(s, u) {
			continue
		}
		matched = append(matched, s)
	}

	sort.Slice(matched, func(i, j int) bool {
		return runAtOrder(matched[i].RunAt) < runAtOrder(matched[j].RunAt)
	})

	return matched
}

// MatchingForExtension returns all content scripts for the given extension
// that match the given URL.
func (l *ContentScriptLoader) MatchingForExtension(extID, urlStr string) []ContentScript {
	l.mu.Lock()
	defer l.mu.Unlock()

	u, err := url.Parse(urlStr)
	if err != nil {
		return nil
	}

	scripts, ok := l.byExtID[extID]
	if !ok {
		return nil
	}

	var matched []ContentScript
	for _, s := range scripts {
		if !l.matchesURL(s, u) {
			continue
		}
		matched = append(matched, s)
	}

	sort.Slice(matched, func(i, j int) bool {
		return runAtOrder(matched[i].RunAt) < runAtOrder(matched[j].RunAt)
	})

	return matched
}

// Count returns the total number of registered content scripts.
func (l *ContentScriptLoader) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.scripts)
}

// CountForExtension returns the number of content scripts for the given extension.
func (l *ContentScriptLoader) CountForExtension(extID string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.byExtID[extID])
}

// RemoveForExtension removes all content scripts for the given extension.
func (l *ContentScriptLoader) RemoveForExtension(extID string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.byExtID, extID)

	var remaining []ContentScript
	for _, s := range l.scripts {
		if s.ExtID != extID {
			remaining = append(remaining, s)
		}
	}
	l.scripts = remaining
}

// RemoveScript removes the content script with the given ID.
func (l *ContentScriptLoader) RemoveScript(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var extID string
	for i, s := range l.scripts {
		if s.ID == id {
			extID = s.ExtID
			l.scripts = append(l.scripts[:i], l.scripts[i+1:]...)
			break
		}
	}

	if extID == "" {
		return
	}

	scripts, ok := l.byExtID[extID]
	if !ok {
		return
	}

	var remaining []ContentScript
	for _, s := range scripts {
		if s.ID != id {
			remaining = append(remaining, s)
		}
	}

	if len(remaining) == 0 {
		delete(l.byExtID, extID)
	} else {
		l.byExtID[extID] = remaining
	}
}

// matchesURL checks whether a content script matches the given URL.
func (l *ContentScriptLoader) matchesURL(s ContentScript, u *url.URL) bool {
	for _, m := range s.Matches {
		ok, err := MatchPattern(m, u.String())
		if err != nil || !ok {
			continue
		}
		// Check exclude patterns.
		excluded := false
		for _, em := range s.ExcludeMatches {
			ok, err := MatchPattern(em, u.String())
			if err == nil && ok {
				excluded = true
				break
			}
		}
		if !excluded {
			return true
		}
	}
	return false
}

// runAtOrder returns a sort order for run_at timing values.
func runAtOrder(runAt string) int {
	switch RunAt(runAt) {
	case RunAtDocumentStart:
		return 0
	case RunAtDocumentEnd:
		return 1
	case RunAtDocumentIdle:
		return 2
	default:
		return 2
	}
}

// ContentScriptSource represents the loaded source code for a content script.
type ContentScriptSource struct {
	// ID is the content script ID.
	ID string `json:"id"`

	// ExtID is the extension ID this script belongs to.
	ExtID string `json:"ext_id"`

	// JS is the loaded JavaScript source code.
	JS []string `json:"js"`

	// CSS is the loaded CSS source code.
	CSS []string `json:"css"`

	// RunAt is the injection timing.
	RunAt string `json:"run_at"`

	// AllFrames indicates whether to inject into all frames.
	AllFrames bool `json:"all_frames"`
}

// LoadSource reads the JS and CSS files for a content script from the
// extension directory and returns their contents.
func LoadSource(extPath string, script ContentScript) (*ContentScriptSource, error) {
	jsSources := make([]string, 0, len(script.JS))
	for _, js := range script.JS {
		absPath := js
		if !filepath.IsAbs(js) {
			absPath = filepath.Join(extPath, js)
		}
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("read content script %s: %w", js, err)
		}
		jsSources = append(jsSources, string(data))
	}

	cssSources := make([]string, 0, len(script.CSS))
	for _, css := range script.CSS {
		absPath := css
		if !filepath.IsAbs(css) {
			absPath = filepath.Join(extPath, css)
		}
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("read content script CSS %s: %w", css, err)
		}
		cssSources = append(cssSources, string(data))
	}

	return &ContentScriptSource{
		ID:        script.ID,
		ExtID:     script.ExtID,
		JS:        jsSources,
		CSS:       cssSources,
		RunAt:     script.RunAt,
		AllFrames: script.AllFrames,
	}, nil
}

// EnsureExtID sets the ExtID field on all scripts that lack it, deriving
// it from the script ID prefix (everything before the first colon).
func (l *ContentScriptLoader) EnsureExtIDs() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for i := range l.scripts {
		if l.scripts[i].ExtID == "" {
			if idx := strings.Index(l.scripts[i].ID, ":"); idx >= 0 {
				l.scripts[i].ExtID = l.scripts[i].ID[:idx]
			}
		}
	}
	for extID, scripts := range l.byExtID {
		for i := range scripts {
			if scripts[i].ExtID == "" {
				scripts[i].ExtID = extID
			}
		}
	}
}
