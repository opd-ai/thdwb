package profiler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPageLoadTrackerEviction (audit R-18) navigates well past the cap: every
// http(s) navigation appends an entry and nothing evicted them, so a long
// session grew loads/order without bound. The tracker must retain at most
// maxPageLoads entries, dropping the oldest.
func TestPageLoadTrackerEviction(t *testing.T) {
	plt := NewPageLoadTracker()

	total := maxPageLoads + 57
	for i := 0; i < total; i++ {
		plt.StartLoad(fmt.Sprintf("https://site%d.example", i))
	}

	assert.Len(t, plt.AllLoads(), maxPageLoads)

	// Oldest evicted, newest retained.
	assert.Nil(t, plt.GetLoad("https://site0.example"), "oldest entry must be evicted")
	assert.NotNil(t, plt.GetLoad(fmt.Sprintf("https://site%d.example", total-1)))

	// AllLoads must not contain nil holes (order and map stay in sync).
	for _, m := range plt.AllLoads() {
		require.NotNil(t, m)
	}
}

// TestPageLoadTrackerRevisitDedups checks that re-navigating to the same URL
// updates the existing entry instead of appending a duplicate order slot.
func TestPageLoadTrackerRevisitDedups(t *testing.T) {
	plt := NewPageLoadTracker()
	first := plt.StartLoad("https://a.example")
	second := plt.StartLoad("https://a.example")

	assert.Len(t, plt.AllLoads(), 1)
	assert.Same(t, first, second, "revisit must reuse the metrics object")
	assert.True(t, !second.NavigationStart.Before(first.NavigationStart))
}
