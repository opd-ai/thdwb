package extensions

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeXPIWithManifest(t *testing.T, path, manifest string, extraFiles map[string]string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))

	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	zw := zipWriter(t, f)

	if manifest != "" {
		w, err := zw.Create("manifest.json")
		require.NoError(t, err)
		_, err = w.Write([]byte(manifest))
		require.NoError(t, err)
	}

	for name, content := range extraFiles {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, zw.Close())
	return path
}

func zipWriter(t *testing.T, f *os.File) *zipWriterWrapper {
	t.Helper()
	zw := zip.NewWriter(f)
	return &zipWriterWrapper{zw: zw}
}

type zipWriterWrapper struct {
	zw *zip.Writer
}

func (w *zipWriterWrapper) Create(name string) (interface{ Write([]byte) (int, error) }, error) {
	return w.zw.Create(name)
}

func (w *zipWriterWrapper) Close() error {
	return w.zw.Close()
}

func TestInstallerInstallFromXPI(t *testing.T) {
	root := t.TempDir()
	extDir := filepath.Join(root, "extensions")
	storageDir := filepath.Join(root, "storage")
	require.NoError(t, os.MkdirAll(extDir, 0o755))
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	xpiPath := filepath.Join(root, "test-ext.xpi")
	writeXPIWithManifest(t, xpiPath, `{
		"manifest_version": 3,
		"name": "Install Test Ext",
		"version": "1.0",
		"permissions": ["tabs", "storage"]
	}`, map[string]string{
		"background.js": "console.log('bg');",
	})

	m := NewManager(storageDir)
	inst := NewInstaller(m, extDir)

	result, err := inst.Install(&InstallRequest{
		PackagePath: xpiPath,
		Enabled:     true,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Extension)
	require.Equal(t, "install-test-ext", result.Extension.ID)
	require.Equal(t, "Install Test Ext", result.Extension.Name)
	require.Equal(t, "1.0", result.Extension.Version)
	require.Contains(t, result.GrantedPermissions, "tabs")
	require.Contains(t, result.GrantedPermissions, "storage")
	require.True(t, m.IsEnabled("install-test-ext"))
	require.True(t, m.IsInstalled("install-test-ext"))
	require.Equal(t, filepath.Join(extDir, "install-test-ext"), result.InstallDir)
}

func TestInstallerInstallDisabled(t *testing.T) {
	root := t.TempDir()
	extDir := filepath.Join(root, "extensions")
	storageDir := filepath.Join(root, "storage")
	require.NoError(t, os.MkdirAll(extDir, 0o755))
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	xpiPath := filepath.Join(root, "test-ext.xpi")
	writeXPIWithManifest(t, xpiPath, `{
		"manifest_version": 3,
		"name": "Disabled Ext",
		"version": "1.0"
	}`, nil)

	m := NewManager(storageDir)
	inst := NewInstaller(m, extDir)

	result, err := inst.Install(&InstallRequest{
		PackagePath: xpiPath,
		Enabled:     false,
	})
	require.NoError(t, err)
	require.False(t, m.IsEnabled("disabled-ext"))
	require.True(t, m.IsInstalled("disabled-ext"))
	require.NotNil(t, result.Extension)
}

func TestInstallerDuplicateInstall(t *testing.T) {
	root := t.TempDir()
	extDir := filepath.Join(root, "extensions")
	storageDir := filepath.Join(root, "storage")
	require.NoError(t, os.MkdirAll(extDir, 0o755))
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	xpiPath := filepath.Join(root, "test-ext.xpi")
	writeXPIWithManifest(t, xpiPath, `{
		"manifest_version": 3,
		"name": "Dup Ext",
		"version": "1.0"
	}`, nil)

	m := NewManager(storageDir)
	inst := NewInstaller(m, extDir)

	_, err := inst.Install(&InstallRequest{PackagePath: xpiPath, Enabled: true})
	require.NoError(t, err)

	_, err = inst.Install(&InstallRequest{PackagePath: xpiPath, Enabled: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), "already installed")
}

func TestInstallerInvalidPackage(t *testing.T) {
	root := t.TempDir()
	extDir := filepath.Join(root, "extensions")
	storageDir := filepath.Join(root, "storage")
	require.NoError(t, os.MkdirAll(extDir, 0o755))
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	m := NewManager(storageDir)
	inst := NewInstaller(m, extDir)

	_, err := inst.Install(&InstallRequest{
		PackagePath: "/nonexistent/package.xpi",
		Enabled:     true,
	})
	require.Error(t, err)
}

func TestInstallerPromptForPermissions(t *testing.T) {
	root := t.TempDir()
	xpiPath := filepath.Join(root, "test-ext.xpi")
	writeXPIWithManifest(t, xpiPath, `{
		"manifest_version": 3,
		"name": "Perm Test",
		"version": "1.0",
		"permissions": ["tabs", "cookies", "notifications"]
	}`, nil)

	inst := NewInstaller(nil, "")

	req, descriptions, err := inst.PromptForPermissions(xpiPath)
	require.NoError(t, err)
	require.NotNil(t, req)
	require.Equal(t, xpiPath, req.PackagePath)
	require.True(t, req.Enabled)
	require.Len(t, descriptions, 3)
	require.Contains(t, descriptions, "Read and modify tab titles, URLs, and tab state")
	require.Contains(t, descriptions, "Read and modify cookies")
	require.Contains(t, descriptions, "Display desktop notifications")
}

func TestInstallerGrantedPermissionsSubset(t *testing.T) {
	root := t.TempDir()
	extDir := filepath.Join(root, "extensions")
	storageDir := filepath.Join(root, "storage")
	require.NoError(t, os.MkdirAll(extDir, 0o755))
	require.NoError(t, os.MkdirAll(storageDir, 0o755))

	xpiPath := filepath.Join(root, "test-ext.xpi")
	writeXPIWithManifest(t, xpiPath, `{
		"manifest_version": 3,
		"name": "Subset Perm",
		"version": "1.0",
		"permissions": ["tabs", "storage", "cookies"]
	}`, nil)

	m := NewManager(storageDir)
	inst := NewInstaller(m, extDir)

	// User grants only "tabs" permission.
	result, err := inst.Install(&InstallRequest{
		PackagePath:  xpiPath,
		Permissions:  []string{"tabs"},
		Enabled:      true,
	})
	require.NoError(t, err)
	require.Len(t, result.GrantedPermissions, 1)
	require.Equal(t, "tabs", result.GrantedPermissions[0])
	require.True(t, m.IsEnabled("subset-perm"))
}

func TestDescribePermission(t *testing.T) {
	require.Equal(t, "Read and modify tab titles, URLs, and tab state", DescribePermission("tabs"))
	require.Equal(t, "Read and modify cookies", DescribePermission("cookies"))
	require.Contains(t, DescribePermission("unknown_permission"), "unknown_permission")
}

func TestDescribePermissions(t *testing.T) {
	descs := DescribePermissions([]string{"tabs", "cookies"})
	require.Len(t, descs, 2)
	require.Contains(t, descs[0], "tab")
	require.Contains(t, descs[1], "cookie")
}
