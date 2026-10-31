package security

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScannerInitialState(t *testing.T) {
	s := NewScanner()
	require.NotNil(t, s)
}

func TestScanSafeExtension(t *testing.T) {
	dir := t.TempDir()

	// Create a safe manifest with no dangerous permissions.
	manifest := `{
		"manifest_version": 3,
		"name": "Safe Extension",
		"version": "1.0",
		"permissions": ["storage"]
	}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))

	// Create a safe JS file with no eval.
	js := `console.log("hello");`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "background.js"), []byte(js), 0o644))

	s := NewScanner()
	result, err := s.Scan("safe-ext", "1.0", dir)
	require.NoError(t, err)
	require.True(t, result.Safe)
	require.Empty(t, result.Findings)
}

func TestScanDangerousPermissions(t *testing.T) {
	dir := t.TempDir()

	manifest := `{
		"manifest_version": 3,
		"name": "Dangerous Extension",
		"version": "1.0",
		"permissions": ["webRequestBlocking", "tabs", "nativeMessaging"]
	}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))

	js := `console.log("hello");`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "background.js"), []byte(js), 0o644))

	s := NewScanner()
	result, err := s.Scan("dangerous-ext", "1.0", dir)
	require.NoError(t, err)
	require.Greater(t, len(result.Findings), 0)

	// Check that webRequestBlocking is flagged.
	found := false
	for _, f := range result.Findings {
		if f.RuleID == "dangerous-permissions" && f.Severity == SeverityMedium {
			found = true
		}
	}
	require.True(t, found)
}

func TestScanEvalUsage(t *testing.T) {
	dir := t.TempDir()

	manifest := `{
		"manifest_version": 3,
		"name": "Eval Extension",
		"version": "1.0",
		"permissions": []
	}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))

	js := `var x = eval("1+1");`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "background.js"), []byte(js), 0o644))

	s := NewScanner()
	result, err := s.Scan("eval-ext", "1.0", dir)
	require.NoError(t, err)
	require.False(t, result.Safe)

	found := false
	for _, f := range result.Findings {
		if f.RuleID == "eval-usage" && f.Severity == SeverityHigh {
			found = true
		}
	}
	require.True(t, found)
}

func TestScanMissingManifest(t *testing.T) {
	dir := t.TempDir()
	// No manifest.json - should not error.

	s := NewScanner()
	result, err := s.Scan("no-manifest", "1.0", dir)
	require.NoError(t, err)
	require.True(t, result.Safe)
	require.Empty(t, result.Findings)
}
