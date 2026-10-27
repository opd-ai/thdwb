package sauce

import (
	"net/url"
	"strings"
	"testing"

	"github.com/danfragoso/thdwb/assets"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchInternalNewTabPage(t *testing.T) {
	history := &History{}
	history.Push(mustParseURL(t, "https://recent.test/page"))
	buildInfo := &assets.BuildInfo{GitRevision: "abc123", GitBranch: "main", HostInfo: "host", BuildTime: "now"}

	u, err := url.Parse("thdwb://newtab/")
	require.NoError(t, err)
	res := fetchInternalPage(u, history, buildInfo)

	assert.Equal(t, "thdwb", res.URL.Scheme)
	assert.NotEmpty(t, res.Body)
	assert.Contains(t, res.Body, "New Tab")
	assert.Contains(t, res.Body, "<form")
	assert.Contains(t, res.Body, `name="q"`)
	assert.Contains(t, res.Body, "https://recent.test/page")
}

func TestNewTabDataFromHistory(t *testing.T) {
	// nil history yields default (empty) data.
	data := newTabDataFromHistory(nil)
	require.NotNil(t, data)
	assert.Empty(t, data.RecentlyVisited)

	history := &History{}
	for i := 1; i <= 15; i++ {
		history.Push(mustParseURL(t, "https://a.test/page"+string(rune('0'+i%10))))
	}

	data = newTabDataFromHistory(history)
	assert.Len(t, data.RecentlyVisited, maxNewTabRecent)

	// Most recently visited page comes first.
	last := history.Last()
	assert.Equal(t, last.String(), data.RecentlyVisited[0].URL)
	assert.Equal(t, last.Host, data.RecentlyVisited[0].Name)
}

func TestGetResourceNewTab(t *testing.T) {
	u, err := url.Parse("thdwb://newtab/")
	require.NoError(t, err)

	res := GetResource(u, nil, nil)
	require.NotNil(t, res)
	assert.Equal(t, "thdwb", res.URL.Scheme)
	assert.True(t, strings.Contains(res.Body, "New Tab"))
}
