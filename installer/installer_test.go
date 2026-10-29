package installer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	assert.Equal(t, "thdwb", c.AppName)
	assert.Equal(t, "0.1.0", c.Version)
	assert.Equal(t, "./bin/thdwb", c.BinaryPath)
	assert.Equal(t, "./dist", c.OutputDir)
	assert.Equal(t, Platform(runtime.GOOS), c.TargetPlatform)
	assert.Equal(t, Arch(runtime.GOARCH), c.TargetArch)
	assert.Equal(t, "./LICENSE", c.LicensePath)
}

func TestConfigValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		// Create a temp binary file.
		dir := t.TempDir()
		binaryPath := filepath.Join(dir, "thdwb")
		require.NoError(t, os.WriteFile(binaryPath, []byte("fake binary"), 0o755))

		c := &Config{
			AppName:       "thdwb",
			Version:       "0.1.0",
			BinaryPath:    binaryPath,
			OutputDir:     filepath.Join(dir, "out"),
			TargetPlatform: PlatformLinux,
			TargetArch:    ArchAMD64,
		}
		assert.NoError(t, c.Validate())
	})

	t.Run("empty app name", func(t *testing.T) {
		c := &Config{AppName: ""}
		assert.Error(t, c.Validate())
	})

	t.Run("empty version", func(t *testing.T) {
		c := &Config{AppName: "thdwb", Version: ""}
		assert.Error(t, c.Validate())
	})

	t.Run("missing binary", func(t *testing.T) {
		c := &Config{
			AppName:       "thdwb",
			Version:       "0.1.0",
			BinaryPath:    "/nonexistent/thdwb",
			TargetPlatform: PlatformLinux,
		}
		assert.Error(t, c.Validate())
	})

	t.Run("unsupported platform", func(t *testing.T) {
		dir := t.TempDir()
		binaryPath := filepath.Join(dir, "thdwb")
		require.NoError(t, os.WriteFile(binaryPath, []byte("fake"), 0o755))

		c := &Config{
			AppName:         "thdwb",
			Version:         "0.1.0",
			BinaryPath:      binaryPath,
			TargetPlatform:  "plan9",
			TargetArch:      ArchAMD64,
		}
		assert.Error(t, c.Validate())
	})
}

func TestBuildUnsupportedPlatform(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "thdwb")
	require.NoError(t, os.WriteFile(binaryPath, []byte("fake"), 0o755))

	c := &Config{
		AppName:         "thdwb",
		Version:         "0.1.0",
		BinaryPath:      binaryPath,
		OutputDir:       filepath.Join(dir, "out"),
		TargetPlatform:  "plan9",
		TargetArch:      ArchAMD64,
	}
	b := NewBuilder(c)
	_, err := b.Build()
	assert.Error(t, err)
}

func TestBuildLinuxCreatesTarGz(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "thdwb")
	require.NoError(t, os.WriteFile(binaryPath, []byte("fake binary"), 0o755))

	outDir := filepath.Join(dir, "out")
	c := &Config{
		AppName:         "thdwb",
		Version:         "0.1.0",
		BinaryPath:      binaryPath,
		OutputDir:       outDir,
		TargetPlatform:  PlatformLinux,
		TargetArch:      ArchAMD64,
		LicensePath:     "",
	}
	b := NewBuilder(c)
	outPath, err := b.Build()
	require.NoError(t, err)
	assert.Contains(t, outPath, "linux-amd64-installer.tar.gz")

	// Verify the output file exists and is non-empty.
	info, err := os.Stat(outPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestBuildWindowsCreatesZip(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "thdwb.exe")
	require.NoError(t, os.WriteFile(binaryPath, []byte("fake binary"), 0o755))

	outDir := filepath.Join(dir, "out")
	c := &Config{
		AppName:         "thdwb",
		Version:         "0.1.0",
		BinaryPath:      binaryPath,
		OutputDir:       outDir,
		TargetPlatform:  PlatformWindows,
		TargetArch:      ArchAMD64,
	}
	b := NewBuilder(c)
	outPath, err := b.Build()
	require.NoError(t, err)
	assert.Contains(t, outPath, "windows-amd64-installer.zip")

	info, err := os.Stat(outPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestBuildDarwinCreatesZip(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "thdwb")
	require.NoError(t, os.WriteFile(binaryPath, []byte("fake binary"), 0o755))

	outDir := filepath.Join(dir, "out")
	c := &Config{
		AppName:         "thdwb",
		Version:         "0.1.0",
		BinaryPath:      binaryPath,
		OutputDir:       outDir,
		TargetPlatform:  PlatformDarwin,
		TargetArch:      ArchAMD64,
	}
	b := NewBuilder(c)
	outPath, err := b.Build()
	require.NoError(t, err)
	assert.Contains(t, outPath, "macos-amd64-installer.zip")

	info, err := os.Stat(outPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	require.NoError(t, os.WriteFile(src, []byte("hello world"), 0o644))
	require.NoError(t, copyFile(src, dst))

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}

func TestPlatformConstants(t *testing.T) {
	assert.Equal(t, "windows", string(PlatformWindows))
	assert.Equal(t, "darwin", string(PlatformDarwin))
	assert.Equal(t, "linux", string(PlatformLinux))
	assert.Equal(t, "amd64", string(ArchAMD64))
	assert.Equal(t, "arm64", string(ArchARM64))
}
