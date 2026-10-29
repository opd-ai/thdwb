package debug

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkMonitorRecordAndRequests(t *testing.T) {
	nm := NewNetworkMonitor(10)

	nm.Record("GET", "https://example.com/page", 200, 1024, 50*time.Millisecond, "https://example.com", "document")
	nm.Record("GET", "https://example.com/style.css", 200, 512, 20*time.Millisecond, "https://example.com", "stylesheet")
	nm.Record("POST", "https://api.example.com/data", 500, 0, 100*time.Millisecond, "https://example.com", "xhr")

	reqs := nm.Requests()
	require.Len(t, reqs, 3)
	assert.Equal(t, "GET", reqs[0].Method)
	assert.Equal(t, "https://example.com/page", reqs[0].URL)
	assert.Equal(t, 200, reqs[0].StatusCode)
	assert.Equal(t, int64(1024), reqs[0].ResponseSize)
	assert.Equal(t, "document", reqs[0].ResourceType)
	assert.Equal(t, 500, reqs[2].StatusCode)
	assert.Equal(t, "xhr", reqs[2].ResourceType)
}

func TestNetworkMonitorMaxEntries(t *testing.T) {
	nm := NewNetworkMonitor(3)

	nm.Record("GET", "https://a.com", 200, 0, 0, "", "")
	nm.Record("GET", "https://b.com", 200, 0, 0, "", "")
	nm.Record("GET", "https://c.com", 200, 0, 0, "", "")
	nm.Record("GET", "https://d.com", 200, 0, 0, "", "") // evicts a.com

	reqs := nm.Requests()
	require.Len(t, reqs, 3)
	assert.Equal(t, "https://b.com", reqs[0].URL)
	assert.Equal(t, "https://c.com", reqs[1].URL)
	assert.Equal(t, "https://d.com", reqs[2].URL)
}

func TestNetworkMonitorFailedRequests(t *testing.T) {
	nm := NewNetworkMonitor(10)

	nm.Record("GET", "https://a.com", 200, 0, 0, "", "")
	nm.Record("GET", "https://b.com", 404, 0, 0, "", "")
	nm.Record("GET", "https://c.com", 500, 0, 0, "", "")
	nm.Record("GET", "https://d.com", 301, 0, 0, "", "")

	failed := nm.FailedRequests()
	require.Len(t, failed, 2)
	assert.Equal(t, 404, failed[0].StatusCode)
	assert.Equal(t, 500, failed[1].StatusCode)
}

func TestNetworkMonitorClearAndCount(t *testing.T) {
	nm := NewNetworkMonitor(10)

	nm.Record("GET", "https://a.com", 200, 0, 0, "", "")
	assert.Equal(t, 1, nm.Count())

	nm.Clear()
	assert.Equal(t, 0, nm.Count())
	assert.Empty(t, nm.Requests())
}
