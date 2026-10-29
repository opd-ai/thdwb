// Package autoupdate implements automatic update infrastructure for the
// thdwb browser itself. It checks a version manifest endpoint, downloads
// new releases, verifies checksums, and supports staged rollouts.
package autoupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// UpdateChannel is the release channel for staged rollouts.
type UpdateChannel string

const (
	ChannelStable   UpdateChannel = "stable"
	ChannelBeta     UpdateChannel = "beta"
	ChannelNightly  UpdateChannel = "nightly"
)

// UpdateCheckResponse is the response from the browser update manifest.
type UpdateCheckResponse struct {
	// Version is the latest available version.
	Version string `json:"version"`

	// Channel is the release channel.
	Channel UpdateChannel `json:"channel"`

	// URL is the download URL for the new release.
	URL string `json:"url"`

	// SHA256 is the expected SHA-256 checksum of the download.
	SHA256 string `json:"sha256,omitempty"`

	// Notes are release notes.
	Notes string `json:"notes,omitempty"`

	// MinVersion is the minimum current version eligible for this update.
	MinVersion string `json:"min_version,omitempty"`

	// RolloutPercentage is the percentage of users eligible (0-100).
	RolloutPercentage int `json:"rollout_percentage,omitempty"`

	// Mandatory indicates the update is required.
	Mandatory bool `json:"mandatory,omitempty"`
}

// UpdateResult describes the outcome of an update check.
type UpdateResult struct {
	// UpdateAvailable indicates whether a newer version is available.
	UpdateAvailable bool

	// CurrentVersion is the currently installed version.
	CurrentVersion string

	// LatestVersion is the latest version available.
	LatestVersion string

	// Channel is the release channel.
	Channel UpdateChannel

	// Mandatory indicates the update is required.
	Mandatory bool

	// Notes are release notes.
	Notes string

	// DownloadURL is the URL to download the update.
	DownloadURL string

	// SHA256 is the expected checksum of the download.
	SHA256 string

	// Eligible indicates whether this client is eligible for the update
	// based on rollout percentage and version constraints.
	Eligible bool
}

// Config holds auto-update configuration.
type Config struct {
	// CurrentVersion is the currently installed browser version.
	CurrentVersion string

	// Channel is the release channel to subscribe to.
	Channel UpdateChannel

	// ManifestURL is the URL to fetch the update manifest from.
	ManifestURL string

	// CheckInterval is how often to check for updates.
	CheckInterval time.Duration

	// HTTPClient is the HTTP client used for update checks and downloads.
	HTTPClient *http.Client

	// DownloadDir is where updates are downloaded.
	DownloadDir string

	// RolloutSeed is a stable seed for rollout percentage calculation.
	RolloutSeed string
}

// DefaultConfig returns a default auto-update configuration.
func DefaultConfig() *Config {
	return &Config{
		Channel:        ChannelStable,
		CheckInterval:  6 * time.Hour,
		HTTPClient:     &http.Client{Timeout: 30 * time.Second},
		DownloadDir:    os.TempDir(),
		RolloutSeed:    "thdwb",
	}
}

// Updater checks for and applies browser updates.
type Updater struct {
	mu        sync.Mutex
	config    *Config
	lastCheck time.Time
}

// NewUpdater creates a new auto-update updater.
func NewUpdater(config *Config) *Updater {
	if config == nil {
		config = DefaultConfig()
	}
	return &Updater{
		config: config,
	}
}

// CheckForUpdates checks whether a browser update is available.
// It fetches the manifest, compares versions, and evaluates rollout eligibility.
func (u *Updater) CheckForUpdates() (*UpdateResult, error) {
	u.mu.Lock()
	config := u.config
	u.mu.Unlock()

	if config.ManifestURL == "" {
		return &UpdateResult{
			UpdateAvailable: false,
			CurrentVersion:  config.CurrentVersion,
			Channel:         config.Channel,
		}, nil
	}

	resp, err := config.HTTPClient.Get(config.ManifestURL)
	if err != nil {
		return nil, fmt.Errorf("fetch update manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update manifest returned %d", resp.StatusCode)
	}

	var manifest UpdateCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode update manifest: %w", err)
	}

	// Filter by channel.
	if manifest.Channel != "" && manifest.Channel != config.Channel {
		return &UpdateResult{
			UpdateAvailable: false,
			CurrentVersion:  config.CurrentVersion,
			LatestVersion:   manifest.Version,
			Channel:         config.Channel,
			Eligible:        false,
		}, nil
	}

	// Compare versions.
	if !isNewerVersion(manifest.Version, config.CurrentVersion) {
		return &UpdateResult{
			UpdateAvailable: false,
			CurrentVersion:  config.CurrentVersion,
			LatestVersion:   manifest.Version,
			Channel:         manifest.Channel,
			Mandatory:       manifest.Mandatory,
			Notes:           manifest.Notes,
			Eligible:        false,
		}, nil
	}

	// Check minimum version constraint.
	if manifest.MinVersion != "" && !isNewerVersion(config.CurrentVersion, manifest.MinVersion) &&
		config.CurrentVersion != manifest.MinVersion {
		return &UpdateResult{
			UpdateAvailable: false,
			CurrentVersion:  config.CurrentVersion,
			LatestVersion:   manifest.Version,
			Channel:         manifest.Channel,
			Mandatory:       manifest.Mandatory,
			Notes:           manifest.Notes,
			Eligible:        false,
		}, nil
	}

	// Check rollout percentage.
	eligible := true
	if manifest.RolloutPercentage == 0 {
		eligible = false
	} else if manifest.RolloutPercentage < 100 {
		eligible = isInRollout(config.RolloutSeed, config.CurrentVersion, manifest.RolloutPercentage)
	}

	u.mu.Lock()
	u.lastCheck = time.Now()
	u.mu.Unlock()

	return &UpdateResult{
		UpdateAvailable: true,
		CurrentVersion:  config.CurrentVersion,
		LatestVersion:   manifest.Version,
		Channel:         manifest.Channel,
		Mandatory:       manifest.Mandatory,
		Notes:           manifest.Notes,
		DownloadURL:     manifest.URL,
		SHA256:          manifest.SHA256,
		Eligible:        eligible,
	}, nil
}

// DownloadUpdate downloads the update package to the download directory.
// It verifies the SHA-256 checksum if provided.
func (u *Updater) DownloadUpdate(result *UpdateResult) (string, error) {
	u.mu.Lock()
	config := u.config
	u.mu.Unlock()

	if result.DownloadURL == "" {
		return "", fmt.Errorf("no download URL available")
	}

	if err := os.MkdirAll(config.DownloadDir, 0o755); err != nil {
		return "", fmt.Errorf("create download dir: %w", err)
	}

	filename := filepath.Base(result.DownloadURL)
	if filename == "" || filename == "/" {
		filename = fmt.Sprintf("thdwb-%s-%s-%s", result.LatestVersion, runtime.GOOS, runtime.GOARCH)
	}

	outPath := filepath.Join(config.DownloadDir, filename)

	resp, err := config.HTTPClient.Get(result.DownloadURL)
	if err != nil {
		return "", fmt.Errorf("download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned %d", resp.StatusCode)
	}

	out, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("create output file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("write download: %w", err)
	}

	// Verify checksum if provided.
	if result.SHA256 != "" {
		if err := verifyChecksum(outPath, result.SHA256); err != nil {
			os.Remove(outPath)
			return "", fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	return outPath, nil
}

// verifyChecksum checks that the SHA-256 of the file matches the expected value.
func verifyChecksum(path, expectedSHA256 string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	h := sha256.New()
	h.Write(data)
	sum := hex.EncodeToString(h.Sum(nil))
	if sum != expectedSHA256 {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, sum)
	}
	return nil
}

// LastCheck returns the time of the last update check.
func (u *Updater) LastCheck() time.Time {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.lastCheck
}

// SetCheckInterval sets the interval between automatic update checks.
func (u *Updater) SetCheckInterval(d time.Duration) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.config.CheckInterval = d
}

// isNewerVersion returns true if a is semantically newer than b.
func isNewerVersion(a, b string) bool {
	pa := parseVersion(a)
	pb := parseVersion(b)
	return compareVersionParts(pa, pb) > 0
}

// parseVersion parses a version string into numeric parts.
func parseVersion(v string) []int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		var n int
		fmt.Sscanf(p, "%d", &n)
		out = append(out, n)
	}
	return out
}

// compareVersionParts compares two version part slices.
func compareVersionParts(a, b []int) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

// isInRollout determines if a client is included in a staged rollout based on
// a hash of the seed + version. Returns true if the client is in the rollout.
func isInRollout(seed, version string, percentage int) bool {
	hash := sha256.Sum256([]byte(seed + "|" + version))
	// Use the first 4 bytes as a uint32, mod 100 to get 0-99.
	val := uint32(hash[0]) | uint32(hash[1])<<8 | uint32(hash[2])<<16 | uint32(hash[3])<<24
	return int(val%100) < percentage
}

// SortedChannels returns all channels in rollout order (stable, beta, nightly).
func SortedChannels() []UpdateChannel {
	return []UpdateChannel{ChannelStable, ChannelBeta, ChannelNightly}
}

// ChannelDescription returns a human-readable description for a channel.
func ChannelDescription(ch UpdateChannel) string {
	switch ch {
	case ChannelStable:
		return "Stable channel — fully tested releases"
	case ChannelBeta:
		return "Beta channel — pre-release builds with new features"
	case ChannelNightly:
		return "Nightly channel — daily builds from main branch"
	default:
		return string(ch)
	}
}
