package suspension

import (
	"testing"
	"time"

	"github.com/danfragoso/thdwb/browser/shell"
	"github.com/stretchr/testify/require"
)

func TestTabSuspenderSuspendAndRestore(t *testing.T) {
	ts := NewTabSuspender(DefaultSuspendPolicy())

	tab := &shell.Tab{ID: 1, URL: "https://example.com", Title: "Example"}
	require.False(t, tab.Suspended)

	require.NoError(t, ts.Suspend(tab, "inactive"))
	require.True(t, tab.Suspended)
	require.NotNil(t, tab.SuspendedAt)
	require.True(t, ts.IsSuspended(1))

	require.NoError(t, ts.Restore(tab))
	require.False(t, tab.Suspended)
	require.Nil(t, tab.SuspendedAt)
	require.False(t, ts.IsSuspended(1))
}

func TestTabSuspenderAlreadySuspended(t *testing.T) {
	ts := NewTabSuspender(DefaultSuspendPolicy())

	tab := &shell.Tab{ID: 1, URL: "https://example.com"}
	require.NoError(t, ts.Suspend(tab, "inactive"))
	require.NoError(t, ts.Suspend(tab, "inactive")) // should be no-op
	require.Equal(t, 1, ts.Count())
}

func TestTabSuspenderProtectActiveTab(t *testing.T) {
	policy := DefaultSuspendPolicy()
	policy.ProtectActiveTab = true
	ts := NewTabSuspender(policy)

	tab := &shell.Tab{ID: 1, URL: "https://example.com", IsActive: true}
	err := ts.Suspend(tab, "inactive")
	require.NoError(t, err)
	require.False(t, tab.Suspended)
	require.False(t, ts.IsSuspended(1))
	require.Equal(t, 0, ts.Count())
}

func TestTabSuspenderMaxSuspendedTabs(t *testing.T) {
	policy := DefaultSuspendPolicy()
	policy.MaxSuspendedTabs = 2
	policy.ProtectActiveTab = false
	ts := NewTabSuspender(policy)

	tab1 := &shell.Tab{ID: 1, URL: "https://example1.com"}
	tab2 := &shell.Tab{ID: 2, URL: "https://example2.com"}
	tab3 := &shell.Tab{ID: 3, URL: "https://example3.com"}

	require.NoError(t, ts.Suspend(tab1, "inactive"))
	require.NoError(t, ts.Suspend(tab2, "inactive"))
	err := ts.Suspend(tab3, "inactive")
	require.NoError(t, err) // at capacity, no-op
	require.False(t, tab3.Suspended)
	require.Equal(t, 2, ts.Count())
}

func TestTabSuspenderSuspendInactive(t *testing.T) {
	policy := DefaultSuspendPolicy()
	policy.InactiveThreshold = time.Minute
	policy.ProtectActiveTab = false
	ts := NewTabSuspender(policy)

	now := time.Now()
	old := now.Add(-2 * time.Minute)

	activeTab := &shell.Tab{ID: 1, URL: "https://active.com", IsActive: true}
	inactiveTab := &shell.Tab{ID: 2, URL: "https://inactive.com"}

	ts.RecordActivity(1)
	ts.lastActivity[2] = old

	suspended := ts.SuspendInactive([]*shell.Tab{activeTab, inactiveTab}, now)
	require.Contains(t, suspended, 2)
	require.NotContains(t, suspended, 1)
	require.True(t, inactiveTab.Suspended)
	require.False(t, activeTab.Suspended)
}

func TestTabSuspenderRestoreAll(t *testing.T) {
	ts := NewTabSuspender(DefaultSuspendPolicy())

	tab1 := &shell.Tab{ID: 1, URL: "https://example1.com"}
	tab2 := &shell.Tab{ID: 2, URL: "https://example2.com"}

	require.NoError(t, ts.Suspend(tab1, "inactive"))
	require.NoError(t, ts.Suspend(tab2, "inactive"))

	restored := ts.RestoreAll([]*shell.Tab{tab1, tab2})
	require.Contains(t, restored, 1)
	require.Contains(t, restored, 2)
	require.False(t, tab1.Suspended)
	require.False(t, tab2.Suspended)
	require.Equal(t, 0, ts.Count())
}

func TestTabSuspenderSuspensionInfo(t *testing.T) {
	ts := NewTabSuspender(DefaultSuspendPolicy())

	tab := &shell.Tab{ID: 1, URL: "https://example.com"}
	now := time.Now()
	require.NoError(t, ts.Suspend(tab, "memory-pressure"))

	info := ts.SuspensionInfoFor(1)
	require.NotNil(t, info)
	require.Equal(t, 1, info.TabID)
	require.Equal(t, "https://example.com", info.URL)
	require.Equal(t, "memory-pressure", info.Reason)
	require.True(t, info.SuspendedAt.After(now.Add(-time.Second)))
	require.True(t, info.SuspendedAt.Before(now.Add(time.Second)))
}

func TestTabSuspenderSuspendedTabs(t *testing.T) {
	ts := NewTabSuspender(DefaultSuspendPolicy())

	tab1 := &shell.Tab{ID: 1, URL: "https://example1.com"}
	tab2 := &shell.Tab{ID: 2, URL: "https://example2.com"}

	require.NoError(t, ts.Suspend(tab1, "inactive"))
	require.NoError(t, ts.Suspend(tab2, "inactive"))

	ids := ts.SuspendedTabs()
	require.Contains(t, ids, 1)
	require.Contains(t, ids, 2)
	require.Len(t, ids, 2)
}
