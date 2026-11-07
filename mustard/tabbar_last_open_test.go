package mustard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestClickPastEndActivatesLastOpenTab (audit O-14) covers the fallback in
// OnTabClick when a click lands beyond the last tab. The old code activated
// tb.tabs[openCount-1], conflating the count of OPEN tabs with an index into
// the full slice: with a closed tab interleaved it could select the wrong tab
// or even a closed one.
func TestClickPastEndActivatesLastOpenTab(t *testing.T) {
	tb := CreateTabBar(40)
	a := tb.AddTab("A", "https://a.example")
	b := tb.AddTab("B", "https://b.example")
	c := tb.AddTab("C", "https://c.example")

	// Close the FIRST two: openCount == 1, and the buggy tabs[openCount-1]
	// == tabs[0] would re-activate the closed tab A.
	tb.CloseTab(a.ID)
	tb.CloseTab(b.ID)
	tb.activeTab = a

	_, isClose := tb.OnTabClick(10_000, 20) // far past any tab
	assert.False(t, isClose)
	assert.NotNil(t, tb.activeTab)
	assert.Equal(t, c.ID, tb.activeTab.ID, "must activate the last OPEN tab")
	assert.False(t, tb.activeTab.Closed)
}

// TestClickPastEndAllClosedKeepsActive verifies the nil guard: when every tab
// is closed the fallback must not panic or invent an active tab.
func TestClickPastEndAllClosedKeepsActive(t *testing.T) {
	tb := CreateTabBar(40)
	a := tb.AddTab("A", "https://a.example")
	tb.CloseTab(a.ID)
	tb.activeTab = nil

	assert.NotPanics(t, func() {
		_, _ = tb.OnTabClick(10_000, 20)
	})
	assert.Nil(t, tb.activeTab, "no open tabs -> no activation")
}
