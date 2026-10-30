package extensions

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeXPI creates a .xpi (ZIP) file at the given path containing a
// manifest.json and optional additional files.
func writeXPI(t *testing.T, path, manifest string, extraFiles map[string]string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))

	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	zw := zip.NewWriter(f)

	// Write manifest.json, unless the caller explicitly passed an empty
	// manifest (which models a package that ships no manifest at all).
	if manifest != "" {
		w, err := zw.Create("manifest.json")
		require.NoError(t, err)
		_, err = io.WriteString(w, manifest)
		require.NoError(t, err)
	}

	// Write extra files.
	for name, content := range extraFiles {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = io.WriteString(w, content)
		require.NoError(t, err)
	}

	require.NoError(t, zw.Close())
	return path
}

func TestXPIOpenAndParse(t *testing.T) {
	root := t.TempDir()
	xpiPath := filepath.Join(root, "test-ext.xpi")

	writeXPI(t, xpiPath, `{
		"manifest_version": 3,
		"name": "Test Extension",
		"version": "1.0",
		"permissions": ["tabs"]
	}`, nil)

	pkg, err := OpenXPI(xpiPath)
	require.NoError(t, err)
	require.NotNil(t, pkg)
	require.Equal(t, "Test Extension", pkg.Manifest.Name)
	require.Equal(t, "1.0", pkg.Manifest.Version)
	require.Contains(t, pkg.Manifest.Permissions, "tabs")
	require.NotEmpty(t, pkg.SHA256)
	require.Contains(t, pkg.Files, "manifest.json")
}

func TestXPIValidate(t *testing.T) {
	root := t.TempDir()
	xpiPath := filepath.Join(root, "valid.xpi")

	writeXPI(t, xpiPath, `{
		"manifest_version": 3,
		"name": "Valid Ext",
		"version": "2.0"
	}`, nil)

	pkg, err := OpenXPI(xpiPath)
	require.NoError(t, err)
	require.NoError(t, pkg.Validate())
}

func TestXPIMissingManifest(t *testing.T) {
	root := t.TempDir()
	xpiPath := filepath.Join(root, "no-manifest.xpi")

	writeXPI(t, xpiPath, "", map[string]string{
		"background.js": "console.log('hi');",
	})

	_, err := OpenXPI(xpiPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing manifest.json")
}

func TestXPIExtract(t *testing.T) {
	root := t.TempDir()
	xpiPath := filepath.Join(root, "extract-test.xpi")
	dest := filepath.Join(root, "extracted")

	writeXPI(t, xpiPath, `{
		"manifest_version": 3,
		"name": "Extract Test",
		"version": "1.0"
	}`, map[string]string{
		"background.js": "console.log('bg');",
		"content.js":    "console.log('content');",
	})

	pkg, err := OpenXPI(xpiPath)
	require.NoError(t, err)
	require.NoError(t, pkg.Extract(dest))

	// Verify extracted files.
	manifestData, err := os.ReadFile(filepath.Join(dest, "manifest.json"))
	require.NoError(t, err)
	require.Contains(t, string(manifestData), "Extract Test")

	bgData, err := os.ReadFile(filepath.Join(dest, "background.js"))
	require.NoError(t, err)
	require.Contains(t, string(bgData), "console.log('bg');")

	contentData, err := os.ReadFile(filepath.Join(dest, "content.js"))
	require.NoError(t, err)
	require.Contains(t, string(contentData), "console.log('content');")
}

func TestXPIChecksumDeterministic(t *testing.T) {
	root := t.TempDir()
	xpiPath := filepath.Join(root, "checksum.xpi")

	writeXPI(t, xpiPath, `{
		"manifest_version": 3,
		"name": "Checksum Test",
		"version": "1.0"
	}`, nil)

	pkg1, err := OpenXPI(xpiPath)
	require.NoError(t, err)

	pkg2, err := OpenXPI(xpiPath)
	require.NoError(t, err)

	require.Equal(t, pkg1.SHA256, pkg2.SHA256)
	require.NotEmpty(t, pkg1.SHA256)
}

func TestXPIPathTraversalRejected(t *testing.T) {
	root := t.TempDir()
	xpiPath := filepath.Join(root, "traversal.xpi")
	dest := filepath.Join(root, "extracted")

	// Create a ZIP with a path traversal entry.
	f, err := os.Create(xpiPath)
	require.NoError(t, err)
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create("manifest.json")
	require.NoError(t, err)
	_, err = io.WriteString(w, `{"manifest_version":3,"name":"Traversal","version":"1.0"}`)
	require.NoError(t, err)

	// Malicious path entry.
	w, err = zw.Create("../escape.txt")
	require.NoError(t, err)
	_, err = io.WriteString(w, "malicious")
	require.NoError(t, err)

	require.NoError(t, zw.Close())

	pkg, err := OpenXPI(xpiPath)
	require.NoError(t, err)
	require.Error(t, pkg.Extract(dest))
}

func TestIsXPI(t *testing.T) {
	require.True(t, IsXPI("extension.xpi"))
	require.True(t, IsXPI("EXTENSION.XPI"))
	require.False(t, IsXPI("extension.zip"))
	require.False(t, IsXPI("manifest.json"))
}

func TestIsZip(t *testing.T) {
	require.True(t, IsZip("extension.zip"))
	require.True(t, IsZip("EXTENSION.ZIP"))
	require.False(t, IsZip("extension.xpi"))
	require.False(t, IsZip("manifest.json"))
}

func TestIsPackage(t *testing.T) {
	require.True(t, IsPackage("extension.xpi"))
	require.True(t, IsPackage("extension.zip"))
	require.False(t, IsPackage("manifest.json"))
	require.False(t, IsPackage("background.js"))
}

func TestXPIUnsignedPackage(t *testing.T) {
	root := t.TempDir()
	xpiPath := filepath.Join(root, "unsigned.xpi")

	writeXPI(t, xpiPath, `{
		"manifest_version": 3,
		"name": "Unsigned Ext",
		"version": "1.0"
	}`, nil)

	pkg, err := OpenXPI(xpiPath)
	require.NoError(t, err)
	require.False(t, pkg.IsSigned())
	require.Empty(t, pkg.Signature)
	require.Empty(t, pkg.SignatureAlgorithm)
	require.NoError(t, pkg.ValidateSignature())
}

func TestXPISignedPackage(t *testing.T) {
	root := t.TempDir()
	xpiPath := filepath.Join(root, "signed.xpi")

	// Create a ZIP with a signature manifest.
	f, err := os.Create(xpiPath)
	require.NoError(t, err)
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create("manifest.json")
	require.NoError(t, err)
	_, err = io.WriteString(w, `{"manifest_version":3,"name":"Signed Ext","version":"1.0"}`)
	require.NoError(t, err)

	// Add a signature manifest.
	w, err = zw.Create("META-INF/manifest.mf")
	require.NoError(t, err)
	_, err = io.WriteString(w, "Signature-Version: 1.0\n")
	require.NoError(t, err)

	// Add a signature file.
	w, err = zw.Create("META-INF/extension.rsa")
	require.NoError(t, err)
	_, err = io.WriteString(w, "fake-signature-data")
	require.NoError(t, err)

	require.NoError(t, zw.Close())

	pkg, err := OpenXPI(xpiPath)
	require.NoError(t, err)
	require.True(t, pkg.IsSigned())
	require.NotEmpty(t, pkg.Signature)
	require.Equal(t, "rsa", pkg.SignatureAlgorithm)
	require.NoError(t, pkg.ValidateSignature())
}

func TestXPIValidateSignatureMissingData(t *testing.T) {
	root := t.TempDir()
	xpiPath := filepath.Join(root, "badsig.xpi")

	// Create a ZIP with only a manifest.mf but no actual signature file.
	// This should NOT be treated as signed.
	f, err := os.Create(xpiPath)
	require.NoError(t, err)
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create("manifest.json")
	require.NoError(t, err)
	_, err = io.WriteString(w, `{"manifest_version":3,"name":"BadSig","version":"1.0"}`)
	require.NoError(t, err)

	// Add only a manifest.mf (no .rsa/.ec/.sf signature file).
	w, err = zw.Create("META-INF/manifest.mf")
	require.NoError(t, err)
	_, err = io.WriteString(w, "Signature-Version: 1.0\n")
	require.NoError(t, err)

	require.NoError(t, zw.Close())

	pkg, err := OpenXPI(xpiPath)
	require.NoError(t, err)
	require.False(t, pkg.IsSigned())
	require.Empty(t, pkg.Signature)
	require.Empty(t, pkg.SignatureAlgorithm)
	require.NoError(t, pkg.ValidateSignature())
}
