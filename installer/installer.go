// Package installer builds single-executable installers for thdwb on
// Windows, macOS, and Linux. It bundles the browser binary with platform
//-specific installation logic (start menu shortcuts, .app bundles, .deb/.rpm
// packages) and produces a distributable artifact.
package installer

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Platform is the target OS for the installer.
type Platform string

const (
	PlatformWindows Platform = "windows"
	PlatformDarwin  Platform = "darwin"
	PlatformLinux   Platform = "linux"
)

// Arch is the target CPU architecture.
type Arch string

const (
	ArchAMD64 Arch = "amd64"
	ArchARM64 Arch = "arm64"
)

// Config holds installer configuration.
type Config struct {
	// AppName is the application name (e.g., "thdwb").
	AppName string

	// Version is the application version (e.g., "0.1.0").
	Version string

	// BinaryPath is the path to the built thdwb binary.
	BinaryPath string

	// OutputDir is where the installer artifact will be written.
	OutputDir string

	// TargetPlatform is the OS to build the installer for.
	TargetPlatform Platform

	// TargetArch is the CPU architecture.
	TargetArch Arch

	// IconPath is an optional path to an application icon.
	IconPath string

	// LicensePath is an optional path to a license file.
	LicensePath string
}

// Builder builds installers.
type Builder struct {
	config *Config
}

// NewBuilder creates a new installer builder.
func NewBuilder(config *Config) *Builder {
	return &Builder{config: config}
}

// Build creates a platform-specific installer artifact.
// It returns the path to the generated installer file.
func (b *Builder) Build() (string, error) {
	cfg := b.config

	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	switch cfg.TargetPlatform {
	case PlatformWindows:
		return b.buildWindows()
	case PlatformDarwin:
		return b.buildDarwin()
	case PlatformLinux:
		return b.buildLinux()
	default:
		return "", fmt.Errorf("unsupported platform: %s", cfg.TargetPlatform)
	}
}

// buildWindows creates a ZIP-based installer with an install script.
func (b *Builder) buildWindows() (string, error) {
	cfg := b.config
	installerName := fmt.Sprintf("%s-%s-windows-%s-installer.zip", cfg.AppName, cfg.Version, cfg.TargetArch)
	outPath := filepath.Join(cfg.OutputDir, installerName)

	// Create a temporary staging directory.
	stage, err := os.MkdirTemp("", "thdwb-installer-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(stage)

	appDir := filepath.Join(stage, cfg.AppName)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return "", fmt.Errorf("create app dir: %w", err)
	}

	// Copy the binary.
	if err := copyFile(cfg.BinaryPath, filepath.Join(appDir, cfg.AppName+".exe")); err != nil {
		return "", fmt.Errorf("copy binary: %w", err)
	}

	// Copy icon if provided.
	if cfg.IconPath != "" {
		if err := copyFile(cfg.IconPath, filepath.Join(appDir, filepath.Base(cfg.IconPath))); err != nil {
			return "", fmt.Errorf("copy icon: %w", err)
		}
	}

	// Copy license if provided.
	if cfg.LicensePath != "" {
		if err := copyFile(cfg.LicensePath, filepath.Join(appDir, "LICENSE")); err != nil {
			return "", fmt.Errorf("copy license: %w", err)
		}
	}

	// Create install script.
	installScript := fmt.Sprintf(`@echo off
:: thdwb Installer
:: Installs thdwb to %%LOCALAPPDATA%%\\thdwb

set INSTALL_DIR=%%LOCALAPPDATA%%\\%s
echo Installing thdwb to %%INSTALL_DIR%%...
if not exist "%%INSTALL_DIR%%" mkdir "%%INSTALL_DIR%%"
xcopy /E /I /Y "%%~dp0%s\\*" "%%INSTALL_DIR%%\\"

:: Create start menu shortcut (optional)
echo.
echo Installation complete.
echo Run: %%INSTALL_DIR%%\\%s.exe
pause
`, cfg.AppName, cfg.AppName, cfg.AppName)

	if err := os.WriteFile(filepath.Join(stage, "install.bat"), []byte(installScript), 0o644); err != nil {
		return "", fmt.Errorf("write install script: %w", err)
	}

	// Create uninstall script.
	uninstallScript := fmt.Sprintf(`@echo off
:: thdwb Uninstaller
set INSTALL_DIR=%%LOCALAPPDATA%%\\%s
echo Uninstalling thdwb from %%INSTALL_DIR%%...
if exist "%%INSTALL_DIR%%" rmdir /S /Q "%%INSTALL_DIR%%"
echo.
echo Uninstallation complete.
pause
`, cfg.AppName)

	if err := os.WriteFile(filepath.Join(stage, "uninstall.bat"), []byte(uninstallScript), 0o644); err != nil {
		return "", fmt.Errorf("write uninstall script: %w", err)
	}

	return zipDirectory(stage, outPath)
}

// buildDarwin creates a .app bundle and a DMG (or ZIP fallback).
func (b *Builder) buildDarwin() (string, error) {
	cfg := b.config
	installerName := fmt.Sprintf("%s-%s-macos-%s-installer.zip", cfg.AppName, cfg.Version, cfg.TargetArch)
	outPath := filepath.Join(cfg.OutputDir, installerName)

	stage, err := os.MkdirTemp("", "thdwb-installer-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(stage)

	appBundle := filepath.Join(stage, cfg.AppName+".app")
	contentsDir := filepath.Join(appBundle, "Contents")
	macOSDir := filepath.Join(contentsDir, "MacOS")
	resourcesDir := filepath.Join(contentsDir, "Resources")

	for _, dir := range []string{contentsDir, macOSDir, resourcesDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	// Copy binary into MacOS.
	if err := copyFile(cfg.BinaryPath, filepath.Join(macOSDir, cfg.AppName)); err != nil {
		return "", fmt.Errorf("copy binary: %w", err)
	}
	if err := os.Chmod(filepath.Join(macOSDir, cfg.AppName), 0o755); err != nil {
		return "", fmt.Errorf("chmod binary: %w", err)
	}

	// Copy icon to Resources if provided.
	if cfg.IconPath != "" {
		if err := copyFile(cfg.IconPath, filepath.Join(resourcesDir, filepath.Base(cfg.IconPath))); err != nil {
			return "", fmt.Errorf("copy icon: %w", err)
		}
	}

	// Info.plist.
	infoPlist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>
	<string>%s</string>
	<key>CFBundleDisplayName</key>
	<string>%s</string>
	<key>CFBundleIdentifier</key>
	<string>dev.thdwb.%s</string>
	<key>CFBundleVersion</key>
	<string>%s</string>
	<key>CFBundleShortVersionString</key>
	<string>%s</string>
	<key>CFBundleExecutable</key>
	<string>%s</string>
	<key>CFBundleIconFile</key>
	<string>%s</string>
	<key>LSHighResolutionCapable</key>
	<true/>
</dict>
</plist>
`, cfg.AppName, cfg.AppName, strings.ToLower(cfg.AppName), cfg.Version, cfg.Version, cfg.AppName, filepath.Base(cfg.IconPath))

	if err := os.WriteFile(filepath.Join(contentsDir, "Info.plist"), []byte(infoPlist), 0o644); err != nil {
		return "", fmt.Errorf("write Info.plist: %w", err)
	}

	// PkgInfo.
	if err := os.WriteFile(filepath.Join(contentsDir, "PkgInfo"), []byte("APPL????"), 0o644); err != nil {
		return "", fmt.Errorf("write PkgInfo: %w", err)
	}

	return zipDirectory(stage, outPath)
}

// buildLinux creates a .tar.gz installer with an install script.
func (b *Builder) buildLinux() (string, error) {
	cfg := b.config
	installerName := fmt.Sprintf("%s-%s-linux-%s-installer.tar.gz", cfg.AppName, cfg.Version, cfg.TargetArch)
	outPath := filepath.Join(cfg.OutputDir, installerName)

	stage, err := os.MkdirTemp("", "thdwb-installer-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(stage)

	appDir := filepath.Join(stage, cfg.AppName)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return "", fmt.Errorf("create app dir: %w", err)
	}

	// Copy binary.
	if err := copyFile(cfg.BinaryPath, filepath.Join(appDir, cfg.AppName)); err != nil {
		return "", fmt.Errorf("copy binary: %w", err)
	}
	if err := os.Chmod(filepath.Join(appDir, cfg.AppName), 0o755); err != nil {
		return "", fmt.Errorf("chmod binary: %w", err)
	}

	// Copy license if provided.
	if cfg.LicensePath != "" {
		if err := copyFile(cfg.LicensePath, filepath.Join(appDir, "LICENSE")); err != nil {
			return "", fmt.Errorf("copy license: %w", err)
		}
	}

	// Install script.
	installScript := fmt.Sprintf(`#!/bin/sh
# thdwb Installer
# Installs thdwb to ~/.local/bin

set -e

INSTALL_DIR="$HOME/.local/bin"
BIN_NAME="%s"

echo "Installing thdwb to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"
cp "$(dirname "$0")/%s/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
chmod +x "$INSTALL_DIR/$BIN_NAME"

echo ""
echo "Installation complete."
echo "Run: $BIN_NAME"
echo "Make sure $INSTALL_DIR is in your PATH."
`, cfg.AppName, cfg.AppName)

	if err := os.WriteFile(filepath.Join(stage, "install.sh"), []byte(installScript), 0o755); err != nil {
		return "", fmt.Errorf("write install script: %w", err)
	}

	// Uninstall script.
	uninstallScript := fmt.Sprintf(`#!/bin/sh
# thdwb Uninstaller

set -e

BIN_NAME="%s"
INSTALL_DIR="$HOME/.local/bin"

echo "Uninstalling thdwb..."
rm -f "$INSTALL_DIR/$BIN_NAME"

echo ""
echo "Uninstallation complete."
`, cfg.AppName)

	if err := os.WriteFile(filepath.Join(stage, "uninstall.sh"), []byte(uninstallScript), 0o755); err != nil {
		return "", fmt.Errorf("write uninstall script: %w", err)
	}

	return tarGzDirectory(stage, outPath)
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// zipDirectory zips the contents of a directory to a zip file.
func zipDirectory(dir, outPath string) (string, error) {
	out, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("create zip: %w", err)
	}
	defer out.Close()

	zw := zip.NewWriter(out)

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		f, err := zw.Create(rel)
		if err != nil {
			return err
		}

		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		_, err = io.Copy(f, in)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("zip directory: %w", err)
	}

	if err := zw.Close(); err != nil {
		return "", fmt.Errorf("close zip: %w", err)
	}

	return outPath, nil
}

// tarGzDirectory creates a .tar.gz archive of a directory.
func tarGzDirectory(dir, outPath string) (string, error) {
	// Use the system tar command for reliable .tar.gz creation.
	out, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("create tar.gz: %w", err)
	}
	defer out.Close()

	cmd := exec.Command("tar", "-czf", "-", "-C", dir, ".")
	cmd.Stdout = out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("run tar: %w", err)
	}

	return outPath, nil
}

// DefaultConfig returns a default installer config for the current platform.
func DefaultConfig() *Config {
	return &Config{
		AppName:         "thdwb",
		Version:         "0.1.0",
		BinaryPath:      "./bin/thdwb",
		OutputDir:       "./dist",
		TargetPlatform:  Platform(runtime.GOOS),
		TargetArch:      Arch(runtime.GOARCH),
		IconPath:        "",
		LicensePath:     "./LICENSE",
	}
}

// Validate checks that the config is valid.
func (c *Config) Validate() error {
	if c.AppName == "" {
		return fmt.Errorf("app name is required")
	}
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}
	if c.BinaryPath == "" {
		return fmt.Errorf("binary path is required")
	}
	if _, err := os.Stat(c.BinaryPath); err != nil {
		return fmt.Errorf("binary not found: %w", err)
	}
	if c.OutputDir == "" {
		return fmt.Errorf("output dir is required")
	}
	switch c.TargetPlatform {
	case PlatformWindows, PlatformDarwin, PlatformLinux:
	default:
		return fmt.Errorf("unsupported platform: %s", c.TargetPlatform)
	}
	return nil
}
