package extensions

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// UpdateCheckRequest represents a request to check for updates for an
// installed extension.
type UpdateCheckRequest struct {
	// ExtensionID is the ID of the extension to check.
	ExtensionID string

	// CurrentVersion is the currently installed version.
	CurrentVersion string

	// UpdateURL is the URL to fetch update metadata from.
	UpdateURL string
}

// UpdateCheckResponse is the response from an update check endpoint.
type UpdateCheckResponse struct {
	// Version is the latest available version.
	Version string `json:"version"`

	// UpdateURL is the URL to download the updated .xpi package.
	UpdateURL string `json:"update_url"`

	// SHA256 is the expected SHA-256 checksum of the downloaded package.
	SHA256 string `json:"sha256,omitempty"`

	// Notes are optional release notes.
	Notes string `json:"notes,omitempty"`
}

// UpdateResult describes the outcome of an update check or download.
type UpdateResult struct {
	// Updated indicates whether an update was found and applied.
	Updated bool

	// Extension is the updated extension (if updated).
	Extension *Extension

	// Package is the downloaded update package (if updated).
	Package *XPIPackage

	// Version is the version after the update check.
	Version string

	// UpdateAvailable indicates whether a newer version is available.
	UpdateAvailable bool

	// LatestVersion is the latest version available (if different).
	LatestVersion string
}

// Updater checks for and applies updates to installed extensions. It polls
// update URLs, downloads new .xpi packages, verifies checksums, and
// installs them via the Installer.
type Updater struct {
	mu        sync.Mutex
	manager   *Manager
	installer *Installer
	httpClient *http.Client
	checkInterval time.Duration
}

// NewUpdater creates an updater bound to the given manager and installer.
func NewUpdater(manager *Manager, installer *Installer) *Updater {
	return &Updater{
		manager:      manager,
		installer:    installer,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		checkInterval: 6 * time.Hour,
	}
}

// SetCheckInterval sets the interval between automatic update checks.
func (u *Updater) SetCheckInterval(d time.Duration) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.checkInterval = d
}

// CheckForUpdates checks whether an update is available for the given
// extension without downloading or installing it.
func (u *Updater) CheckForUpdates(req *UpdateCheckRequest) (*UpdateResult, error) {
	u.mu.Lock()
	manager := u.manager
	httpClient := u.httpClient
	u.mu.Unlock()

	if manager == nil {
		return nil, fmt.Errorf("updater: no extension manager configured")
	}

	ext := manager.Get(req.ExtensionID)
	if ext == nil {
		return nil, fmt.Errorf("updater: extension %s not found", req.ExtensionID)
	}

	updateURL := req.UpdateURL
	if updateURL == "" {
		updateURL = ext.Manifest.UpdateURL
	}
	if updateURL == "" {
		return &UpdateResult{
			Updated:         false,
			Extension:       ext,
			Version:         ext.Version,
			UpdateAvailable: false,
		}, nil
	}

	// Fetch update metadata.
	resp, err := httpClient.Get(updateURL)
	if err != nil {
		return nil, fmt.Errorf("updater: fetch update metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("updater: update endpoint returned %d", resp.StatusCode)
	}

	var updateResp UpdateCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
		return nil, fmt.Errorf("updater: decode update response: %w", err)
	}

	// Compare versions.
	current := ext.Version
	latest := updateResp.Version
	if latest == "" {
		return &UpdateResult{
			Updated:         false,
			Extension:       ext,
			Version:         current,
			UpdateAvailable: false,
		}, nil
	}

	// Simple semantic version comparison.
	if compareVersions(latest, current) <= 0 {
		return &UpdateResult{
			Updated:         false,
			Extension:       ext,
			Version:         current,
			UpdateAvailable: false,
			LatestVersion:   latest,
		}, nil
	}

	return &UpdateResult{
		Updated:         false,
		Extension:       ext,
		Version:         current,
		UpdateAvailable: true,
		LatestVersion:   latest,
	}, nil
}

// DownloadAndUpdate downloads and installs an update for the given
// extension. It verifies the SHA-256 checksum if provided.
func (u *Updater) DownloadAndUpdate(req *UpdateCheckRequest) (*UpdateResult, error) {
	u.mu.Lock()
	manager := u.manager
	installer := u.installer
	httpClient := u.httpClient
	u.mu.Unlock()

	if manager == nil {
		return nil, fmt.Errorf("updater: no extension manager configured")
	}
	if installer == nil {
		return nil, fmt.Errorf("updater: no installer configured")
	}

	ext := manager.Get(req.ExtensionID)
	if ext == nil {
		return nil, fmt.Errorf("updater: extension %s not found", req.ExtensionID)
	}

	updateURL := req.UpdateURL
	if updateURL == "" {
		updateURL = ext.Manifest.UpdateURL
	}
	if updateURL == "" {
		return nil, fmt.Errorf("updater: no update URL for extension %s", req.ExtensionID)
	}

	// Fetch update metadata.
	resp, err := httpClient.Get(updateURL)
	if err != nil {
		return nil, fmt.Errorf("updater: fetch update metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("updater: update endpoint returned %d", resp.StatusCode)
	}

	var updateResp UpdateCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
		return nil, fmt.Errorf("updater: decode update response: %w", err)
	}

	// Compare versions.
	current := ext.Version
	latest := updateResp.Version
	if latest == "" {
		return &UpdateResult{
			Updated:         false,
			Extension:       ext,
			Version:         current,
			UpdateAvailable: false,
		}, nil
	}

	if compareVersions(latest, current) <= 0 {
		return &UpdateResult{
			Updated:         false,
			Extension:       ext,
			Version:         current,
			UpdateAvailable: false,
			LatestVersion:   latest,
		}, nil
	}

	// Download the update package.
	downloadResp, err := httpClient.Get(updateResp.UpdateURL)
	if err != nil {
		return nil, fmt.Errorf("updater: download update: %w", err)
	}
	defer downloadResp.Body.Close()

	if downloadResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("updater: download returned %d", downloadResp.StatusCode)
	}

	// Save to a temp file.
	tmpFile, err := os.CreateTemp("", "thdwb-update-*.xpi")
	if err != nil {
		return nil, fmt.Errorf("updater: create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, downloadResp.Body); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("updater: write update package: %w", err)
	}
	tmpFile.Close()

	// Verify checksum if provided.
	if err := verifyChecksum(tmpPath, updateResp.SHA256); err != nil {
		return nil, fmt.Errorf("updater: checksum verification: %w", err)
	}

	// Install the update (reinstall into the same directory).
	installReq := &InstallRequest{
		PackagePath:  tmpPath,
		DestDir:      filepath.Dir(ext.Path),
		Permissions:  ext.Permissions,
		Enabled:      ext.IsEnabled(),
	}

	result, err := installer.Install(installReq)
	if err != nil {
		return nil, fmt.Errorf("updater: install update: %w", err)
	}

	// Migrate storage to the current schema version after update.
	if result.Extension.Storage != nil {
		if err := result.Extension.Storage.MigrateStorage(); err != nil {
			return nil, fmt.Errorf("updater: migrate storage: %w", err)
		}
	}
	if result.Extension.ContainerStorage != nil {
		// Container storage partitions are migrated lazily on first access.
	}

	return &UpdateResult{
		Updated:         true,
		Extension:       result.Extension,
		Package:         result.Package,
		Version:         result.Extension.Version,
		UpdateAvailable: false,
	}, nil
}

// verifyChecksum verifies the SHA-256 checksum of a file. If expected is
// empty, verification is skipped.
func verifyChecksum(path, expected string) error {
	if expected == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

// compareVersions compares two semantic version strings. Returns -1 if a < b,
// 0 if a == b, 1 if a > b.
func compareVersions(a, b string) int {
	va := parseVersion(a)
	vb := parseVersion(b)
	for i := 0; i < len(va) && i < len(vb); i++ {
		if va[i] < vb[i] {
			return -1
		}
		if va[i] > vb[i] {
			return 1
		}
	}
	if len(va) < len(vb) {
		return -1
	}
	if len(va) > len(vb) {
		return 1
	}
	return 0
}

// parseVersion parses a semantic version string into a slice of integers.
func parseVersion(v string) []int {
	var parts []int
	for _, part := range splitVersion(v) {
		var n int
		fmt.Sscanf(part, "%d", &n)
		parts = append(parts, n)
	}
	return parts
}

// splitVersion splits a version string on '.' and non-digit boundaries.
func splitVersion(v string) []string {
	var parts []string
	current := ""
	for _, c := range v {
		if c == '.' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
