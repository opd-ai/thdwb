package autoupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	assert.Equal(t, ChannelStable, c.Channel)
	assert.Equal(t, 6*time.Hour, c.CheckInterval)
	require.NotNil(t, c.HTTPClient)
	assert.Equal(t, os.TempDir(), c.DownloadDir)
	assert.Equal(t, "thdwb", c.RolloutSeed)
}

func TestUpdateResultEligible(t *testing.T) {
	r := &UpdateResult{
		UpdateAvailable: true,
		CurrentVersion:  "0.1.0",
		LatestVersion:   "0.2.0",
		Channel:         ChannelStable,
		Eligible:        true,
	}
	assert.True(t, r.UpdateAvailable)
	assert.True(t, r.Eligible)
}

func TestChannelConstants(t *testing.T) {
	assert.Equal(t, "stable", string(ChannelStable))
	assert.Equal(t, "beta", string(ChannelBeta))
	assert.Equal(t, "nightly", string(ChannelNightly))
}

func TestSortedChannels(t *testing.T) {
	chans := SortedChannels()
	require.Len(t, chans, 3)
	assert.Equal(t, ChannelStable, chans[0])
	assert.Equal(t, ChannelBeta, chans[1])
	assert.Equal(t, ChannelNightly, chans[2])
}

func TestChannelDescription(t *testing.T) {
	assert.Contains(t, ChannelDescription(ChannelStable), "Stable")
	assert.Contains(t, ChannelDescription(ChannelBeta), "Beta")
	assert.Contains(t, ChannelDescription(ChannelNightly), "Nightly")
	assert.Contains(t, ChannelDescription("custom"), "custom")
}

func TestIsNewerVersion(t *testing.T) {
	assert.True(t, isNewerVersion("0.2.0", "0.1.0"))
	assert.True(t, isNewerVersion("1.0.0", "0.9.9"))
	assert.True(t, isNewerVersion("0.1.1", "0.1.0"))
	assert.False(t, isNewerVersion("0.1.0", "0.1.0"))
	assert.False(t, isNewerVersion("0.1.0", "0.2.0"))
	assert.False(t, isNewerVersion("1.0.0", "1.0.1"))
}

func TestParseVersion(t *testing.T) {
	assert.Equal(t, []int{1, 2, 3}, parseVersion("1.2.3"))
	assert.Equal(t, []int{1, 2, 3}, parseVersion("v1.2.3"))
	assert.Equal(t, []int{0, 1, 0}, parseVersion("0.1.0"))
}

func TestCompareVersionParts(t *testing.T) {
	assert.Equal(t, 1, compareVersionParts([]int{1, 2}, []int{1, 1}))
	assert.Equal(t, -1, compareVersionParts([]int{1, 1}, []int{1, 2}))
	assert.Equal(t, 0, compareVersionParts([]int{1, 2}, []int{1, 2}))
	assert.Equal(t, -1, compareVersionParts([]int{1}, []int{1, 2}))
	assert.Equal(t, 1, compareVersionParts([]int{1, 2}, []int{1}))
}

func TestIsInRollout(t *testing.T) {
	// 100% rollout includes everyone.
	assert.True(t, isInRollout("seed", "1.0.0", 100))
	// 0% rollout includes nobody.
	assert.False(t, isInRollout("seed", "1.0.0", 0))
	// Same seed+version should be deterministic.
	assert.Equal(t, isInRollout("seed", "1.0.0", 50), isInRollout("seed", "1.0.0", 50))
	// Different versions may differ.
	_ = isInRollout("seed", "2.0.0", 50)
}

func TestCheckForUpdatesNoManifestURL(t *testing.T) {
	u := NewUpdater(&Config{
		CurrentVersion: "0.1.0",
		Channel:        ChannelStable,
	})
	result, err := u.CheckForUpdates()
	require.NoError(t, err)
	assert.False(t, result.UpdateAvailable)
	assert.Equal(t, "0.1.0", result.CurrentVersion)
}

func TestCheckForUpdatesUpdateAvailable(t *testing.T) {
	manifest := `{
		"version": "0.2.0",
		"channel": "stable",
		"url": "https://example.com/thdwb-0.2.0.zip",
		"sha256": "abc123",
		"notes": "Bug fixes and improvements",
		"rollout_percentage": 100
	}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, manifest)
	}))
	defer server.Close()

	u := NewUpdater(&Config{
		CurrentVersion: "0.1.0",
		Channel:        ChannelStable,
		ManifestURL:    server.URL,
		HTTPClient:     server.Client(),
	})
	result, err := u.CheckForUpdates()
	require.NoError(t, err)
	assert.True(t, result.UpdateAvailable)
	assert.Equal(t, "0.1.0", result.CurrentVersion)
	assert.Equal(t, "0.2.0", result.LatestVersion)
	assert.True(t, result.Eligible)
	assert.Equal(t, "Bug fixes and improvements", result.Notes)
	assert.Equal(t, "https://example.com/thdwb-0.2.0.zip", result.DownloadURL)
}

func TestCheckForUpdatesNoUpdateAvailable(t *testing.T) {
	manifest := `{
		"version": "0.1.0",
		"channel": "stable",
		"url": "https://example.com/thdwb-0.1.0.zip"
	}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, manifest)
	}))
	defer server.Close()

	u := NewUpdater(&Config{
		CurrentVersion: "0.1.0",
		Channel:        ChannelStable,
		ManifestURL:    server.URL,
		HTTPClient:     server.Client(),
	})
	result, err := u.CheckForUpdates()
	require.NoError(t, err)
	assert.False(t, result.UpdateAvailable)
	assert.Equal(t, "0.1.0", result.LatestVersion)
}

func TestCheckForUpdatesChannelMismatch(t *testing.T) {
	manifest := `{
		"version": "0.2.0",
		"channel": "beta",
		"url": "https://example.com/thdwb-0.2.0.zip"
	}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, manifest)
	}))
	defer server.Close()

	u := NewUpdater(&Config{
		CurrentVersion: "0.1.0",
		Channel:        ChannelStable,
		ManifestURL:    server.URL,
		HTTPClient:     server.Client(),
	})
	result, err := u.CheckForUpdates()
	require.NoError(t, err)
	assert.False(t, result.UpdateAvailable)
	assert.False(t, result.Eligible)
}

func TestCheckForUpdatesRolloutPercentage(t *testing.T) {
	manifest := `{
		"version": "0.2.0",
		"channel": "stable",
		"url": "https://example.com/thdwb-0.2.0.zip",
		"rollout_percentage": 0
	}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, manifest)
	}))
	defer server.Close()

	u := NewUpdater(&Config{
		CurrentVersion: "0.1.0",
		Channel:        ChannelStable,
		ManifestURL:    server.URL,
		RolloutSeed:    "test-seed",
		HTTPClient:     server.Client(),
	})
	result, err := u.CheckForUpdates()
	require.NoError(t, err)
	assert.True(t, result.UpdateAvailable)
	// With 0% rollout, nobody is eligible.
	assert.False(t, result.Eligible)
}

func TestCheckForUpdatesMandatory(t *testing.T) {
	manifest := `{
		"version": "0.2.0",
		"channel": "stable",
		"url": "https://example.com/thdwb-0.2.0.zip",
		"mandatory": true,
		"rollout_percentage": 100
	}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, manifest)
	}))
	defer server.Close()

	u := NewUpdater(&Config{
		CurrentVersion: "0.1.0",
		Channel:        ChannelStable,
		ManifestURL:    server.URL,
		HTTPClient:     server.Client(),
	})
	result, err := u.CheckForUpdates()
	require.NoError(t, err)
	assert.True(t, result.UpdateAvailable)
	assert.True(t, result.Mandatory)
}

func TestVerifyChecksum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.bin")
	require.NoError(t, os.WriteFile(path, []byte("hello world"), 0o644))

	h := sha256Sum([]byte("hello world"))
	assert.NoError(t, verifyChecksum(path, h))
	assert.Error(t, verifyChecksum(path, "badchecksum"))
}

func TestDownloadUpdateNoURL(t *testing.T) {
	u := NewUpdater(DefaultConfig())
	_, err := u.DownloadUpdate(&UpdateResult{})
	assert.Error(t, err)
}

func TestLastCheck(t *testing.T) {
	u := NewUpdater(DefaultConfig())
	assert.Equal(t, time.Time{}, u.LastCheck())
}

func sha256Sum(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
