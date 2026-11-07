package shell

import (
	"testing"

	"github.com/danfragoso/thdwb/browser/extensions"
	mustard "github.com/danfragoso/thdwb/mustard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtensionSettingsRebuildUnregistersButtons (audit R-17) verifies that
// rebuilding the extensions settings page does not leak button registrations.
// DetachWidget clears the frame's child list but NOT the window's
// registeredButtons slice, so each Rebuild appended three more stale buttons
// per extension -- hit targets that ProcessButtons keeps walking and whose
// callbacks fire on detached widgets.
func TestExtensionSettingsRebuildUnregistersButtons(t *testing.T) {
	window := newTestWindow(t)
	extManager := NewExtensionManager()
	extMgr := extensions.NewManager("")
	require.NoError(t, extMgr.AddExtension(&extensions.Extension{
		ID: "ext1", Name: "One", Version: "1.0",
	}))

	page := NewExtensionSettingsPage(window, extManager, extMgr, NewBrowserSettings())
	afterCreate := window.RegisteredButtonCount()
	require.Greater(t, afterCreate, 0, "one extension row should register buttons")

	// Rebuild repeatedly: the count must stay flat, not grow by 3 each time.
	for i := 0; i < 5; i++ {
		page.Rebuild()
	}
	assert.Equal(t, afterCreate, window.RegisteredButtonCount(),
		"registered buttons must not accumulate across rebuilds")

	// Removing the extension and rebuilding must drop the buttons entirely.
	require.NoError(t, extMgr.Uninstall("ext1"))
	page.Rebuild()
	assert.Equal(t, 0, window.RegisteredButtonCount(),
		"uninstalled extension's buttons must be unregistered")
}

// TestRegisteredButtonCountBasic covers the mustard accessor itself.
func TestRegisteredButtonCountBasic(t *testing.T) {
	window := newTestWindow(t)
	assert.Equal(t, 0, window.RegisteredButtonCount())

	btn := mustard.CreateButtonWidget("x", nil)
	window.RegisterButton(btn, func() {})
	assert.Equal(t, 1, window.RegisteredButtonCount())

	window.UnregisterButton(btn)
	assert.Equal(t, 0, window.RegisteredButtonCount())
}
