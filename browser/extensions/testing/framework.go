// Package testing provides an automated testing framework for extensions.
// It includes helpers for creating test extensions, setting up managers,
// asserting extension behavior, and running integration-style tests.
package testing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/danfragoso/thdwb/browser/extensions"
	"github.com/stretchr/testify/require"
)

// Harness is a test harness for extension testing. It provides a Manager
// and helper methods for creating and managing test extensions.
type Harness struct {
	t      *testing.T
	mgr    *extensions.Manager
	dirs   []string // temp dirs to clean up
}

// NewHarness creates a test harness with a fresh Manager.
func NewHarness(t *testing.T) *Harness {
	t.Helper()
	return &Harness{
		t:    t,
		mgr:  extensions.NewManager(t.TempDir()),
	}
}

// Manager returns the test Manager.
func (h *Harness) Manager() *extensions.Manager {
	return h.mgr
}

// CreateExtension creates a test extension with the given manifest content
// and registers it with the manager. Returns the extension ID.
func (h *Harness) CreateExtension(id, manifestContent string) *extensions.Extension {
	h.t.Helper()

	dir := h.t.TempDir()
	h.dirs = append(h.dirs, dir)

	manifestPath := filepath.Join(dir, "manifest.json")
	require.NoError(h.t, os.WriteFile(manifestPath, []byte(manifestContent), 0o644))

	m, err := extensions.ParseManifest(manifestPath)
	require.NoError(h.t, err)

	ext := extensions.NewExtension(id, dir, m)
	require.NoError(h.t, h.mgr.AddExtension(ext))

	return ext
}

// CreateExtensionV3 creates a Manifest v3 test extension.
func (h *Harness) CreateExtensionV3(id, name, version string, permissions ...string) *extensions.Extension {
	h.t.Helper()

	permJSON := "[]"
	if len(permissions) > 0 {
		perms, _ := json.Marshal(permissions)
		permJSON = string(perms)
	}

	manifest := `{
		"manifest_version": 3,
		"name": "` + name + `",
		"version": "` + version + `",
		"permissions": ` + permJSON + `
	}`
	return h.CreateExtension(id, manifest)
}

// EnableExtension enables the given extension and starts its background script.
func (h *Harness) EnableExtension(extID string) {
	h.t.Helper()
	require.NoError(h.t, h.mgr.Enable(extID))
}

// DisableExtension disables the given extension.
func (h *Harness) DisableExtension(extID string) {
	h.t.Helper()
	require.NoError(h.t, h.mgr.Disable(extID))
}

// AssertEnabled asserts that the extension is enabled.
func (h *Harness) AssertEnabled(extID string) {
	h.t.Helper()
	require.True(h.t, h.mgr.IsEnabled(extID), "extension %s should be enabled", extID)
}

// AssertDisabled asserts that the extension is disabled.
func (h *Harness) AssertDisabled(extID string) {
	h.t.Helper()
	require.False(h.t, h.mgr.IsEnabled(extID), "extension %s should be disabled", extID)
}

// AssertExtensionCount asserts the number of loaded extensions.
func (h *Harness) AssertExtensionCount(expected int) {
	h.t.Helper()
	require.Equal(h.t, expected, h.mgr.Count(), "extension count mismatch")
}

// WaitFor waits until the condition is true or timeout is reached. Returns
// an error if the condition is not met within the timeout.
func (h *Harness) WaitFor(condition func() bool, timeout time.Duration) error {
	h.t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return nil
		}
		time.Sleep(5 * time.Millisecond)
	}
	return errWaitTimeout
}

// errWaitTimeout is returned when WaitFor times out.
var errWaitTimeout = &waitTimeoutError{}

// waitTimeoutError is returned when WaitFor times out.
type waitTimeoutError struct{}

func (e *waitTimeoutError) Error() string {
	return "condition not met within timeout"
}

// Cleanup removes all temp directories created by the harness.
func (h *Harness) Cleanup() {
	for _, dir := range h.dirs {
		_ = os.RemoveAll(dir)
	}
}

// AssertExtensionExists asserts that an extension with the given ID exists.
func (h *Harness) AssertExtensionExists(extID string) {
	h.t.Helper()
	require.NotNil(h.t, h.mgr.Get(extID), "extension %s should exist", extID)
}

// AssertExtensionNotExists asserts that no extension with the given ID exists.
func (h *Harness) AssertExtensionNotExists(extID string) {
	h.t.Helper()
	require.Nil(h.t, h.mgr.Get(extID), "extension %s should not exist", extID)
}
