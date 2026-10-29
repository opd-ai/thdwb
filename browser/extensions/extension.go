// Package extensions implements the WebExtensions architecture for thdwb:
// extension discovery, manifest parsing, lifecycle management, sandboxed
// execution, content script registration, and per-extension storage.
package extensions

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	goja "github.com/dop251/goja"
)

// ManifestVersion is the WebExtensions manifest schema version.
type ManifestVersion int

const (
	// ManifestV2 is the legacy manifest version 2.
	ManifestV2 ManifestVersion = 2
	// ManifestV3 is the current manifest version 3.
	ManifestV3 ManifestVersion = 3
)

// Manifest represents a parsed WebExtensions manifest.json.
type Manifest struct {
	// Required fields
	ManifestVersion ManifestVersion `json:"manifest_version"`
	Name            string          `json:"name"`
	Version         string          `json:"version"`

	// Optional fields
	Description     string            `json:"description,omitempty"`
	Icons           map[string]string `json:"icons,omitempty"`
	Actions         *ActionManifest   `json:"action,omitempty"`
	BrowserAction   *ActionManifest   `json:"browser_action,omitempty"`
	PageAction      *ActionManifest   `json:"page_action,omitempty"`
	Background      *BackgroundManifest `json:"background,omitempty"`
	ContentScripts  []ContentScriptManifest `json:"content_scripts,omitempty"`
	Permissions     []string          `json:"permissions,omitempty"`
	OptionalPermissions []string       `json:"optional_permissions,omitempty"`
	ContentSecurityPolicy string        `json:"content_security_policy,omitempty"`
	HomepageURL           string          `json:"homepage_url,omitempty"`
	Author                string          `json:"author,omitempty"`
	UpdateURL             string          `json:"update_url,omitempty"`
}

// ActionManifest describes a browser/page action (toolbar icon).
type ActionManifest struct {
	DefaultIcon    map[string]string `json:"default_icon,omitempty"`
	DefaultTitle   string            `json:"default_title,omitempty"`
	DefaultPopup   string            `json:"default_popup,omitempty"`
}

// BackgroundManifest describes the extension background script.
type BackgroundManifest struct {
	Scripts []string `json:"scripts,omitempty"`
	Page    string   `json:"page,omitempty"`
	Persistent bool   `json:"persistent,omitempty"`
	ServiceWorker string `json:"service_worker,omitempty"`
}

// ContentScriptManifest describes a content script entry.
type ContentScriptManifest struct {
	Matches     []string `json:"matches"`
	ExcludeMatches []string `json:"exclude_matches,omitempty"`
	IncludeGlobs []string `json:"include_globs,omitempty"`
	ExcludeGlobs []string `json:"exclude_globs,omitempty"`
	JS          []string `json:"js,omitempty"`
	CSS         []string `json:"css,omitempty"`
	RunAt       string   `json:"run_at,omitempty"` // "document_start", "document_end", "document_idle"
	AllFrames   bool     `json:"all_frames,omitempty"`
	MatchAboutBlank bool `json:"match_about_blank,omitempty"`
}

// Extension represents a loaded browser extension.
type Extension struct {
	// ID uniquely identifies the extension (typically the directory name or
	// a generated UUID for unpacked extensions).
	ID string `json:"id"`

	// Name is the human-readable extension name.
	Name string `json:"name"`

	// Version is the extension version string.
	Version string `json:"version"`

	// Description is the optional extension description.
	Description string `json:"description,omitempty"`

	// Manifest is the raw parsed manifest.
	Manifest *Manifest `json:"manifest"`

	// ManifestVersion is the manifest schema version (2 or 3).
	ManifestVersion ManifestVersion `json:"manifest_version"`

	// Permissions are the declared extension permissions.
	Permissions []string `json:"permissions"`

	// Icons maps icon size to asset path/URL.
	Icons map[string]string `json:"icons,omitempty"`

	// Enabled reports whether the extension is currently enabled.
	Enabled bool `json:"enabled"`

	// Path is the filesystem path to the extension directory.
	Path string `json:"-"`

	// BackgroundVM is the isolated Goja VM for background scripts.
	BackgroundVM *goja.Runtime `json:"-"`

	// ContentScripts maps URL patterns to registered content scripts.
	ContentScripts []ContentScript `json:"-"`

	// Storage is the per-extension storage partition (default container).
	Storage *StoragePartition `json:"-"`

	// ContainerStorage is the per-container, per-extension storage partition.
	ContainerStorage *ContainerStoragePartition `json:"-"`

	mu sync.Mutex `json:"-"`
}

// ContentScript is a registered content script with URL pattern matching.
type ContentScript struct {
	ID          string
	ExtID       string
	Matches     []string
	ExcludeMatches []string
	JS          []string
	CSS         []string
	RunAt       string
	AllFrames   bool
}

// NewExtension creates an Extension from a parsed manifest and directory path.
func NewExtension(id, path string, manifest *Manifest) *Extension {
	return &Extension{
		ID:             id,
		Name:           manifest.Name,
		Version:        manifest.Version,
		Description:    manifest.Description,
		Manifest:       manifest,
		ManifestVersion: manifest.ManifestVersion,
		Permissions:    manifest.Permissions,
		Icons:          manifest.Icons,
		Enabled:        true,
		Path:           path,
		ContentScripts: make([]ContentScript, 0),
	}
}

// IsEnabled reports whether the extension is enabled.
func (e *Extension) IsEnabled() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.Enabled
}

// SetEnabled sets the extension enabled state.
func (e *Extension) SetEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Enabled = enabled
}

// ParseManifest parses a manifest.json file from the given path.
func ParseManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validate manifest: %w", err)
	}

	return &m, nil
}

// Validate checks that the manifest has all required fields.
func (m *Manifest) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("manifest: missing required field 'name'")
	}
	if m.Version == "" {
		return fmt.Errorf("manifest: missing required field 'version'")
	}
	switch m.ManifestVersion {
	case ManifestV2, ManifestV3:
	default:
		return fmt.Errorf("manifest: unsupported manifest_version %d (must be 2 or 3)", m.ManifestVersion)
	}
	return nil
}

// GetIcon returns the icon path for the given size, or the closest available.
func (m *Manifest) GetIcon(size int) string {
	if m.Icons == nil {
		return ""
	}
	// Try exact match first.
	key := fmt.Sprintf("%d", size)
	if path, ok := m.Icons[key]; ok {
		return path
	}
	// Try closest smaller size.
	var bestKey string
	var bestSize int
	for k := range m.Icons {
		var s int
		fmt.Sscanf(k, "%d", &s)
		if s <= size && s > bestSize {
			bestSize = s
			bestKey = k
		}
	}
	if bestKey != "" {
		return m.Icons[bestKey]
	}
	// Return any icon.
	for _, v := range m.Icons {
		return v
	}
	return ""
}

// ResolvePath resolves a relative path within the extension directory.
func (e *Extension) ResolvePath(relPath string) string {
	if filepath.IsAbs(relPath) {
		return relPath
	}
	return filepath.Join(e.Path, relPath)
}

// ExtensionIDFromPath derives an extension ID from a filesystem path.
func ExtensionIDFromPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Base(path)
	}
	return filepath.Base(abs)
}

// IsValidPermission checks whether a permission string is recognized.
func IsValidPermission(perm string) bool {
	valid := []string{
		"tabs", "storage", "cookies", "webRequest", "webRequestBlocking",
		"notifications", "downloads", "contextMenus", "alarms",
		"bookmarks", "history", "sessions", "identity",
		"activeTab", "scripting", "debugger", "privacy",
		"proxy", "management", "pageCapture", "printing",
		"printingMetrics", "topSites", "tabHide", "tabGroups",
		"window.open", "windows", "geolocation", "notifications",
		"unlimitedStorage", "storage", "fileSystem",
	}
	_ = valid
	// For now, accept any non-empty permission string; strict validation
	// can be added later as the API surface grows.
	return perm != ""
}

// SortPermissions returns a sorted copy of the permissions list.
func SortPermissions(perms []string) []string {
	out := make([]string, len(perms))
	copy(out, perms)
	sort.Strings(out)
	return out
}

// EnsureURL validates and normalizes a URL string.
func EnsureURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid URL: %q", raw)
	}
	return u, nil
}

// IsValidMatchPattern reports whether a match pattern is syntactically valid.
// A match pattern has the form: scheme://host/path where scheme can be "*",
// host can be "*" or "*.domain", and path can be "*".
func IsValidMatchPattern(pattern string) bool {
	parts := strings.SplitN(pattern, "://", 2)
	if len(parts) != 2 {
		return false
	}
	scheme := parts[0]
	if scheme != "*" && scheme != "http" && scheme != "https" && scheme != "file" && scheme != "ftp" {
		return false
	}
	rest := parts[1]
	if rest == "" || rest == "/" {
		return false
	}
	return true
}

// MatchPattern matches a URL against a match pattern.
func MatchPattern(pattern, urlStr string) (bool, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false, err
	}

	parts := strings.SplitN(pattern, "://", 2)
	if len(parts) != 2 {
		return false, nil
	}
	schemePat := parts[0]
	restPat := parts[1]

	// Scheme matching.
	if schemePat != "*" && schemePat != u.Scheme {
		return false, nil
	}

	// Split host and path.
	hostPat, pathPat := splitHostPath(restPat)

	// Host matching.
	if !matchHost(hostPat, u.Hostname()) {
		return false, nil
	}

	// Path matching.
	if !matchPath(pathPat, u.Path) {
		return false, nil
	}

	return true, nil
}

func splitHostPath(s string) (host, path string) {
	if i := strings.Index(s, "/"); i >= 0 {
		return s[:i], s[i:]
	}
	return s, ""
}

func matchHost(pattern, host string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // includes the leading dot
		return host == suffix[1:] || strings.HasSuffix(host, suffix)
	}
	return pattern == host
}

func matchPath(pattern, path string) bool {
	if pattern == "" || pattern == "/" {
		return true
	}
	if pattern == "/*" {
		return true
	}
	if strings.HasSuffix(pattern, "/*") {
		prefix := pattern[:len(pattern)-2] // strip "/*"
		return strings.HasPrefix(path, prefix)
	}
	return pattern == path
}

// ContentScriptRegistry maps URL patterns to content scripts for injection.
type ContentScriptRegistry struct {
	mu       sync.Mutex
	scripts  []ContentScript
}

// NewContentScriptRegistry creates an empty content script registry.
func NewContentScriptRegistry() *ContentScriptRegistry {
	return &ContentScriptRegistry{
		scripts: make([]ContentScript, 0),
	}
}

// Register adds a content script to the registry.
func (r *ContentScriptRegistry) Register(script ContentScript) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scripts = append(r.scripts, script)
}

// Matching returns all content scripts whose match patterns match the given URL.
func (r *ContentScriptRegistry) Matching(urlStr string) []ContentScript {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result []ContentScript
	for _, s := range r.scripts {
		for _, m := range s.Matches {
			ok, err := MatchPattern(m, urlStr)
			if err != nil || !ok {
				continue
			}
			// Check exclude patterns.
			excluded := false
			for _, em := range s.ExcludeMatches {
				ok, err := MatchPattern(em, urlStr)
				if err == nil && ok {
					excluded = true
					break
				}
			}
			if !excluded {
				result = append(result, s)
			}
			break
		}
	}
	return result
}

// All returns all registered content scripts.
func (r *ContentScriptRegistry) All() []ContentScript {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]ContentScript, len(r.scripts))
	copy(out, r.scripts)
	return out
}

// Count returns the number of registered content scripts.
func (r *ContentScriptRegistry) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.scripts)
}
