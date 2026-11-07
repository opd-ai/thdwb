package shell

import (
	"testing"

	"github.com/danfragoso/thdwb/assets"
	"github.com/danfragoso/thdwb/hotdog"
	"github.com/danfragoso/thdwb/ketchup"
	profiler "github.com/danfragoso/thdwb/profiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFrameTickDrainsReflowQueue (audit C-16) verifies the shell's per-frame
// callback drains the reflow queue that JS DOM mutations fill. Before the fix
// DrainReflowQueue had zero non-test callers: script-driven mutations queued
// invalidated roots forever (probe P13: 1000 appends -> 1001 roots retained)
// and the page never repainted until the next navigation.
func TestFrameTickDrainsReflowQueue(t *testing.T) {
	bs := NewBrowserShell(nil, nil)

	tab := bs.tabManager.Add("T", "https://reflow.example", "", "", "")
	wc := hotdog.NewWindowContext(&hotdog.Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()

	doc := ketchup.ParseHTMLDocument(`<!DOCTYPE html><html><body><div id="root"></div></body></html>`, wc)
	require.NotNil(t, doc)
	wc.ActiveDocument = doc
	require.NoError(t, wc.InitJSRuntime())
	bs.tabWindowContexts[tab.ID] = wc

	// A JS mutation queues a reflow.
	_, err := wc.RunJS(`
		var r = document.getElementById("root");
		r.appendChild(document.createElement("span"));
	`)
	require.NoError(t, err)

	// The queue now holds the invalidated root.
	assert.NotEmpty(t, wc.DrainReflowQueue(), "mutation should queue a reflow")

	// Re-queue, then let the frame tick consume it.
	_, err = wc.RunJS(`document.getElementById("root").appendChild(document.createElement("i"));`)
	require.NoError(t, err)

	bs.frameTick() // headless: webContent is nil, but the drain must still run

	// After frameTick the queue is empty -- the drain has a real caller.
	assert.Empty(t, wc.DrainReflowQueue(), "frameTick must drain the reflow queue")
}

// TestFrameTickNoActiveTabIsNoop guards the nil paths so a tick with no tab or
// no context never panics.
func TestFrameTickNoActiveTabIsNoop(t *testing.T) {
	bs := NewBrowserShell(nil, nil)
	assert.NotPanics(t, bs.frameTick)

	tab := bs.tabManager.Add("T", "about:blank", "", "", "")
	// No window context registered for the tab.
	assert.NotPanics(t, bs.frameTick)
	_ = tab
}
