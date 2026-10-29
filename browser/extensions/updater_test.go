package extensions

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdaterCheckForUpdatesNoUpdateURL(t *testing.T) {
	root := t.TempDir()
	storageDir := filepath.Join(root, "storage")
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	m := NewManager(storageDir)
	inst := NewInstaller(m, root)
	updater := NewUpdater(m, inst)

	ext := &Extension{
		ID:      "test-ext",
		Name:    "Test",
		Version: "1.0",
		Manifest: &Manifest{
			ManifestVersion: ManifestV3,
			Name:            "Test",
			Version:         "1.0",
		},
	}
	m.extensions["test-ext"] = ext

	result, err := updater.CheckForUpdates(&UpdateCheckRequest{
		ExtensionID:   "test-ext",
		CurrentVersion: "1.0",
	})
	require.NoError(t, err)
	require.False(t, result.UpdateAvailable)
	require.Equal(t, "1.0", result.Version)
}

func TestUpdaterCheckForUpdatesUpdateAvailable(t *testing.T) {
	root := t.TempDir()
	storageDir := filepath.Join(root, "storage")
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	m := NewManager(storageDir)
	inst := NewInstaller(m, root)
	updater := NewUpdater(m, inst)

	ext := &Extension{
		ID:      "test-ext",
		Name:    "Test",
		Version: "1.0",
		Manifest: &Manifest{
			ManifestVersion: ManifestV3,
			Name:            "Test",
			Version:         "1.0",
			UpdateURL:       "http://example.com/update",
		},
	}
	m.extensions["test-ext"] = ext

	// Mock update server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"version": "2.0",
			"update_url": "http://example.com/update.xpi",
			"sha256": "abc123",
			"notes": "Bug fixes"
		}`))
	}))
	defer server.Close()

	// Override update URL to point to test server.
	ext.Manifest.UpdateURL = server.URL

	result, err := updater.CheckForUpdates(&UpdateCheckRequest{
		ExtensionID:   "test-ext",
		CurrentVersion: "1.0",
		UpdateURL:     server.URL,
	})
	require.NoError(t, err)
	require.True(t, result.UpdateAvailable)
	require.Equal(t, "2.0", result.LatestVersion)
	require.Equal(t, "1.0", result.Version)
	require.False(t, result.Updated)
}

func TestUpdaterCheckForUpdatesNoUpdateAvailable(t *testing.T) {
	root := t.TempDir()
	storageDir := filepath.Join(root, "storage")
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	m := NewManager(storageDir)
	inst := NewInstaller(m, root)
	updater := NewUpdater(m, inst)

	ext := &Extension{
		ID:      "test-ext",
		Name:    "Test",
		Version: "2.0",
		Manifest: &Manifest{
			ManifestVersion: ManifestV3,
			Name:            "Test",
			Version:         "2.0",
			UpdateURL:       "http://example.com/update",
		},
	}
	m.extensions["test-ext"] = ext

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"version": "1.0",
			"update_url": "http://example.com/update.xpi"
		}`))
	}))
	defer server.Close()

	ext.Manifest.UpdateURL = server.URL

	result, err := updater.CheckForUpdates(&UpdateCheckRequest{
		ExtensionID:   "test-ext",
		CurrentVersion: "2.0",
		UpdateURL:     server.URL,
	})
	require.NoError(t, err)
	require.False(t, result.UpdateAvailable)
	require.Equal(t, "2.0", result.Version)
	require.Equal(t, "1.0", result.LatestVersion)
}

func TestUpdaterCheckForUpdatesUnknownExtension(t *testing.T) {
	m := NewManager("")
	inst := NewInstaller(m, "")
	updater := NewUpdater(m, inst)

	_, err := updater.CheckForUpdates(&UpdateCheckRequest{
		ExtensionID: "nonexistent",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestCompareVersions(t *testing.T) {
	require.Equal(t, -1, compareVersions("1.0", "2.0"))
	require.Equal(t, 1, compareVersions("2.0", "1.0"))
	require.Equal(t, 0, compareVersions("1.0", "1.0"))
	require.Equal(t, -1, compareVersions("1.0", "1.1"))
	require.Equal(t, 1, compareVersions("1.1", "1.0"))
	require.Equal(t, -1, compareVersions("1.0.0", "1.0.1"))
	require.Equal(t, 1, compareVersions("1.0.1", "1.0.0"))
	require.Equal(t, 0, compareVersions("1.0.0", "1.0.0"))
	require.Equal(t, -1, compareVersions("1.0", "1.0.0"))
	require.Equal(t, 1, compareVersions("1.0.0", "1.0"))
}

func TestVerifyChecksum(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("hello world"), 0o644))

	// Compute expected checksum.
	data, _ := os.ReadFile(filePath)
	expected := hexEncodeSHA256(data)

	require.NoError(t, verifyChecksum(filePath, expected))

	// Wrong checksum should fail.
	err := verifyChecksum(filePath, "wrong")
	require.Error(t, err)
	require.Contains(t, err.Error(), "checksum mismatch")
}

func hexEncodeSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func TestUpdaterNoManager(t *testing.T) {
	updater := NewUpdater(nil, nil)
	_, err := updater.CheckForUpdates(&UpdateCheckRequest{
		ExtensionID: "test",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no extension manager configured")
}
