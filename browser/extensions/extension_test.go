package extensions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeManifest(t *testing.T, dir string, content string) string {
	t.Helper()
	path := filepath.Join(dir, "manifest.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestParseManifestV3(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, `{
		"manifest_version": 3,
		"name": "Test Extension",
		"version": "1.0.0",
		"description": "A test extension",
		"permissions": ["tabs", "storage"],
		"icons": { "16": "icon16.png", "48": "icon48.png" },
		"action": {
			"default_title": "Test",
			"default_popup": "popup.html"
		},
		"background": {
			"service_worker": "background.js",
			"persistent": false
		}
	}`)

	m, err := ParseManifest(path)
	require.NoError(t, err)
	require.Equal(t, ManifestV3, m.ManifestVersion)
	require.Equal(t, "Test Extension", m.Name)
	require.Equal(t, "1.0.0", m.Version)
	require.Equal(t, "A test extension", m.Description)
	require.Contains(t, m.Permissions, "tabs")
	require.Contains(t, m.Permissions, "storage")
	require.Equal(t, "icon16.png", m.Icons["16"])
	require.NotNil(t, m.Actions)
	require.Equal(t, "Test", m.Actions.DefaultTitle)
	require.Equal(t, "popup.html", m.Actions.DefaultPopup)
	require.NotNil(t, m.Background)
	require.Equal(t, "background.js", m.Background.ServiceWorker)
	require.False(t, m.Background.Persistent)
}

func TestParseManifestV2(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, `{
		"manifest_version": 2,
		"name": "Legacy Extension",
		"version": "2.0",
		"browser_action": {
			"default_title": "Legacy",
			"default_popup": "popup.html"
		},
		"background": {
			"scripts": ["background.js"],
			"persistent": true
		},
		"content_scripts": [
			{
				"matches": ["*://example.com/*"],
				"js": ["content.js"],
				"run_at": "document_end"
			}
		]
	}`)

	m, err := ParseManifest(path)
	require.NoError(t, err)
	require.Equal(t, ManifestV2, m.ManifestVersion)
	require.Equal(t, "Legacy Extension", m.Name)
	require.NotNil(t, m.BrowserAction)
	require.Equal(t, "Legacy", m.BrowserAction.DefaultTitle)
	require.NotNil(t, m.Background)
	require.Equal(t, []string{"background.js"}, m.Background.Scripts)
	require.True(t, m.Background.Persistent)
	require.Len(t, m.ContentScripts, 1)
	require.Equal(t, "*://example.com/*", m.ContentScripts[0].Matches[0])
	require.Equal(t, "document_end", m.ContentScripts[0].RunAt)
}

func TestParseManifestMissingName(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, `{
		"manifest_version": 3,
		"version": "1.0"
	}`)

	_, err := ParseManifest(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "name")
}

func TestParseManifestMissingVersion(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, `{
		"manifest_version": 3,
		"name": "Test"
	}`)

	_, err := ParseManifest(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "version")
}

func TestParseManifestInvalidVersion(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, `{
		"manifest_version": 1,
		"name": "Test",
		"version": "1.0"
	}`)

	_, err := ParseManifest(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported")
}

func TestNewExtension(t *testing.T) {
	m := &Manifest{
		ManifestVersion: ManifestV3,
		Name:            "My Ext",
		Version:         "1.0",
		Description:     "desc",
		Permissions:     []string{"tabs"},
		Icons:           map[string]string{"16": "i.png"},
	}
	ext := NewExtension("ext1", "/path/to/ext", m)

	require.Equal(t, "ext1", ext.ID)
	require.Equal(t, "My Ext", ext.Name)
	require.Equal(t, "1.0", ext.Version)
	require.Equal(t, "desc", ext.Description)
	require.Equal(t, ManifestV3, ext.ManifestVersion)
	require.True(t, ext.Enabled)
	require.Equal(t, "/path/to/ext", ext.Path)
	require.NotNil(t, ext.ContentScripts)
}

func TestExtensionEnableDisable(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "T", Version: "1.0"}
	ext := NewExtension("e1", "/p", m)

	require.True(t, ext.IsEnabled())
	ext.SetEnabled(false)
	require.False(t, ext.IsEnabled())
	ext.SetEnabled(true)
	require.True(t, ext.IsEnabled())
}

func TestResolvePath(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "T", Version: "1.0"}
	ext := NewExtension("e1", "/home/user/ext", m)

	require.Equal(t, "/home/user/ext/foo.js", ext.ResolvePath("foo.js"))
	require.Equal(t, "/abs/path.js", ext.ResolvePath("/abs/path.js"))
}

func TestExtensionIDFromPath(t *testing.T) {
	id := ExtensionIDFromPath("/home/user/extensions/my-ext")
	require.Equal(t, "my-ext", id)

	id = ExtensionIDFromPath("relative/path/ext")
	require.Equal(t, "ext", id)
}

func TestGetIcon(t *testing.T) {
	m := &Manifest{
		ManifestVersion: ManifestV3,
		Name:            "T",
		Version:         "1.0",
		Icons:           map[string]string{"16": "s.png", "48": "m.png", "128": "l.png"},
	}

	require.Equal(t, "s.png", m.GetIcon(16))
	require.Equal(t, "m.png", m.GetIcon(48))
	require.Equal(t, "s.png", m.GetIcon(32)) // closest smaller (16)
	require.Equal(t, "l.png", m.GetIcon(200)) // largest available
	require.Equal(t, "", (&Manifest{ManifestVersion: ManifestV3, Name: "T", Version: "1.0"}).GetIcon(16))
}

func TestIsValidMatchPattern(t *testing.T) {
	require.True(t, IsValidMatchPattern("*://example.com/*"))
	require.True(t, IsValidMatchPattern("https://example.com/*"))
	require.True(t, IsValidMatchPattern("http://*.example.com/path"))
	require.False(t, IsValidMatchPattern("not-a-pattern"))
	require.False(t, IsValidMatchPattern("://example.com"))
	require.False(t, IsValidMatchPattern(""))
}

func TestMatchPattern(t *testing.T) {
	ok, err := MatchPattern("*://example.com/*", "https://example.com/page")
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = MatchPattern("https://example.com/*", "http://example.com/page")
	require.NoError(t, err)
	require.False(t, ok)

	ok, err = MatchPattern("http://*.example.com/path", "http://sub.example.com/path")
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = MatchPattern("https://example.com/*", "https://other.com/page")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestSortPermissions(t *testing.T) {
	perms := []string{"storage", "tabs", "cookies"}
	sorted := SortPermissions(perms)
	require.Equal(t, []string{"cookies", "storage", "tabs"}, sorted)
	// Original should be unchanged.
	require.Equal(t, []string{"storage", "tabs", "cookies"}, perms)
}

func TestExtensionJSON(t *testing.T) {
	m := &Manifest{ManifestVersion: ManifestV3, Name: "T", Version: "1.0"}
	ext := NewExtension("e1", "/p", m)
	ext.Permissions = []string{"tabs"}

	data, err := json.Marshal(ext)
	require.NoError(t, err)
	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, "e1", decoded["id"])
	require.Equal(t, "T", decoded["name"])
	require.Equal(t, "1.0", decoded["version"])
}
