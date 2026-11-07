package hotdog

import (
	"fmt"
	"testing"

	"github.com/danfragoso/thdwb/assets"
	profiler "github.com/danfragoso/thdwb/profiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDocumentsHistoryIsBounded (audit R-9) navigates well past the cap and
// asserts the retained-document list stays bounded: each entry pins a full DOM
// plus parsed tree, so an unbounded append leaked every visited page until the
// tab closed.
func TestDocumentsHistoryIsBounded(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()

	const navigations = 120
	for i := 0; i < navigations; i++ {
		require.NoError(t, wc.LoadHTML(
			fmt.Sprintf("<html><head><title>p%d</title></head><body>x</body></html>", i),
			fmt.Sprintf("https://nav%d.example", i)))
	}

	assert.Len(t, wc.Documents, maxDocumentHistory)
	// The most recent document must be the retained tail.
	assert.Same(t, wc.ActiveDocument, wc.Documents[len(wc.Documents)-1])
}
