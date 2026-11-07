package shell

import (
	"testing"

	"github.com/danfragoso/thdwb/hotdog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCloseTabDestroysContextOnAllPaths (audit R-5) verifies that every tab
// close path releases the tab's WindowContext. Previously only the Ctrl+W
// shortcut called closeTab; TabManager.Close -- used by session restore and the
// extension tabs.remove API -- removed the tab but left the context (goja VM,
// full DOM, cookie jar, rAF timer) alive in tabWindowContexts forever.
func TestCloseTabDestroysContextOnAllPaths(t *testing.T) {
	bs := NewBrowserShell(nil, nil)

	// UI path: Ctrl+W goes through closeTab.
	tab := bs.tabManager.Add("UI", "https://one.example", "", "", "")
	wc1 := hotdog.NewWindowContext(nil, nil, nil)
	bs.tabWindowContexts[tab.ID] = wc1
	bs.closeTab(bs.tabManager.ActiveTab())
	assert.NotContains(t, bs.tabWindowContexts, tab.ID,
		"Ctrl+W close must drop the tab's WindowContext")
	assert.Nil(t, bs.tabManager.GetTab(tab.ID),
		"Ctrl+W close must also remove the tab from the manager")

	// Manager path: session restore / extension tabs.remove call Close
	// directly with the ID.
	tab2 := bs.tabManager.Add("Ext", "https://two.example", "", "", "")
	wc2 := hotdog.NewWindowContext(nil, nil, nil)
	bs.tabWindowContexts[tab2.ID] = wc2
	require.NotNil(t, bs.tabManager.Close(tab2.ID))
	assert.NotContains(t, bs.tabWindowContexts, tab2.ID,
		"TabManager.Close must destroy the WindowContext via OnClosed")

	// Contexts must be marked destroyed so lingering references cannot run JS.
	assert.False(t, wc1.IsActive())
	assert.False(t, wc2.IsActive())
}

// TestDestroyIsIdempotent guards the double-close hazard the OnClosed wiring
// introduces: destroying an already-destroyed WindowContext must be a no-op
// rather than a nil-deref panic.
func TestDestroyIsIdempotent(t *testing.T) {
	wc := hotdog.NewWindowContext(nil, nil, nil)
	require.NoError(t, wc.LoadHTML("<html><body></body></html>", "https://idem.example"))
	wc.Destroy()
	assert.NotPanics(t, wc.Destroy)
	assert.NotPanics(t, func() { _ = wc.LoadHTML("<html></html>", "https://idem.example") })
}
