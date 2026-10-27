package shell

import (
	"testing"

	mustard "github.com/danfragoso/thdwb/mustard"
)

// newTestWindow builds a headless mustard window for UI tests. The ebitengine
// driver's NewWindow needs no display, so this works in CI.
func newTestWindow(t *testing.T) *mustard.WindowBase {
	t.Helper()
	w, err := mustard.CreateNewWindow(nil, "test", 1024, 768, false)
	if err != nil {
		t.Fatalf("create test window: %v", err)
	}
	wb, ok := w.(*mustard.WindowBase)
	if !ok {
		t.Fatalf("window is %T, want *mustard.WindowBase", w)
	}
	return wb
}
