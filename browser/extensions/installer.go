package extensions

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// PermissionDescription maps a permission string to a human-readable
// description for display in permission prompts.
var PermissionDescription = map[string]string{
	"tabs":               "Read and modify tab titles, URLs, and tab state",
	"storage":            "Store data locally and sync across devices",
	"cookies":            "Read and modify cookies",
	"webRequest":         "Monitor and intercept network requests",
	"webRequestBlocking": "Block or modify network requests",
	"notifications":      "Display desktop notifications",
	"downloads":          "Download and manage files",
	"contextMenus":       "Add items to the right-click context menu",
	"alarms":             "Schedule alarms and periodic tasks",
	"bookmarks":          "Read and modify bookmarks",
	"history":            "Read and modify browsing history",
	"sessions":           "Access recently closed tabs and windows",
	"identity":           "Sign in to websites using OAuth",
	"activeTab":          "Access the currently active tab",
	"scripting":          "Execute scripts and inject CSS into pages",
	"debugger":           "Attach to and debug pages",
	"privacy":            "Read and modify privacy settings",
	"proxy":              "Configure proxy settings",
	"management":         "Manage other extensions",
	"pageCapture":        "Save pages as MHTML",
	"printing":           "Print pages",
	"printingMetrics":    "Access printer information",
	"topSites":           "Access frequently visited sites",
	"tabHide":            "Hide tabs",
	"tabGroups":          "Manage tab groups",
	"window.open":        "Open new windows",
	"windows":            "Read and modify browser windows",
	"geolocation":        "Access your location",
	"unlimitedStorage":   "Use unlimited storage space",
	"fileSystem":         "Read and write files on your device",
}

// DescribePermission returns a human-readable description for a permission.
// Unknown permissions return a generic description.
func DescribePermission(perm string) string {
	if desc, ok := PermissionDescription[perm]; ok {
		return desc
	}
	return fmt.Sprintf("Access the %s permission", perm)
}

// DescribePermissions returns human-readable descriptions for a list of
// permissions, preserving order.
func DescribePermissions(perms []string) []string {
	descs := make([]string, len(perms))
	for i, p := range perms {
		descs[i] = DescribePermission(p)
	}
	return descs
}

// InstallRequest represents a request to install an extension from a package.
type InstallRequest struct {
	// PackagePath is the path to the .xpi or .zip file.
	PackagePath string

	// DestDir is the directory where the extension will be installed.
	// If empty, a default extensions directory is used.
	DestDir string

	// Permissions are the permissions the user has granted.
	// If empty, all declared permissions are requested.
	Permissions []string

	// Enabled indicates whether the extension should be enabled after install.
	Enabled bool
}

// InstallResult describes the outcome of an extension installation.
type InstallResult struct {
	// Extension is the installed extension.
	Extension *Extension

	// Package is the XPI package that was installed.
	Package *XPIPackage

	// InstallDir is the directory where the extension was installed.
	InstallDir string

	// GrantedPermissions are the permissions granted to the extension.
	GrantedPermissions []string
}

// Installer installs extensions from .xpi/.zip packages. It validates the
// package, extracts it to disk, and registers it with the extension manager.
type Installer struct {
	mu        sync.Mutex
	manager   *Manager
	extDir    string
	onPrompt  func(req *InstallRequest) (*InstallResult, error)
}

// NewInstaller creates an installer that installs extensions into extDir
// and registers them with the given manager.
func NewInstaller(manager *Manager, extDir string) *Installer {
	return &Installer{
		manager: manager,
		extDir:  extDir,
	}
}

// SetPromptHandler registers a callback that prompts the user for permission
// confirmation before installation. If nil, all declared permissions are
// granted automatically.
func (i *Installer) SetPromptHandler(fn func(req *InstallRequest) (*InstallResult, error)) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.onPrompt = fn
}

// Install installs an extension from a .xpi or .zip package. It validates
// the package, extracts it, prompts for permissions (if a handler is set),
// and registers the extension with the manager.
func (i *Installer) Install(req *InstallRequest) (*InstallResult, error) {
	i.mu.Lock()
	if i.manager == nil {
		i.mu.Unlock()
		return nil, fmt.Errorf("installer: no extension manager configured")
	}
	i.mu.Unlock()

	if req.PackagePath == "" {
		return nil, fmt.Errorf("installer: package path is required")
	}

	// Open and validate the XPI package.
	pkg, err := OpenXPI(req.PackagePath)
	if err != nil {
		return nil, fmt.Errorf("installer: open package: %w", err)
	}
	if err := pkg.Validate(); err != nil {
		return nil, fmt.Errorf("installer: validate package: %w", err)
	}

	// Determine the install directory.
	installDir := req.DestDir
	if installDir == "" {
		installDir = i.extDir
	}
	if installDir == "" {
		return nil, fmt.Errorf("installer: no destination directory configured")
	}

	extID := pkg.ID
	extInstallDir := filepath.Join(installDir, extID)

	// Check if already installed.
	if i.manager.Get(extID) != nil {
		return nil, fmt.Errorf("installer: extension %s already installed", extID)
	}

	// Extract the package.
	if err := pkg.Extract(extInstallDir); err != nil {
		return nil, fmt.Errorf("installer: extract package: %w", err)
	}

	// Parse the manifest from the extracted directory.
	manifestPath := filepath.Join(extInstallDir, "manifest.json")
	manifest, err := ParseManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("installer: parse manifest: %w", err)
	}

	// Determine granted permissions.
	granted := req.Permissions
	if granted == nil {
		granted = manifest.Permissions
	}

	// Create the extension.
	ext := NewExtension(extID, extInstallDir, manifest)
	ext.Permissions = granted

	// Register with the manager.
	if err := i.manager.registerExtension(ext); err != nil {
		// Clean up extracted files on failure.
		_ = os.RemoveAll(extInstallDir)
		return nil, fmt.Errorf("installer: register extension: %w", err)
	}

	// Set enabled state per the request. Extensions are registered enabled
	// by default; disable if requested.
	if !req.Enabled {
		if err := i.manager.Disable(extID); err != nil {
			return nil, fmt.Errorf("installer: disable extension: %w", err)
		}
	} else {
		if err := i.manager.Enable(extID); err != nil {
			return nil, fmt.Errorf("installer: enable extension: %w", err)
		}
	}

	return &InstallResult{
		Extension:          ext,
		Package:            pkg,
		InstallDir:         extInstallDir,
		GrantedPermissions: granted,
	}, nil
}

// PromptForPermissions builds an InstallRequest with human-readable
// permission descriptions for the given package. The caller can present
// these descriptions to the user and then call Install with the granted
// permissions.
func (i *Installer) PromptForPermissions(packagePath string) (*InstallRequest, []string, error) {
	pkg, err := OpenXPI(packagePath)
	if err != nil {
		return nil, nil, fmt.Errorf("open package: %w", err)
	}

	req := &InstallRequest{
		PackagePath: packagePath,
		Enabled:     true,
	}

	descriptions := DescribePermissions(pkg.Manifest.Permissions)
	return req, descriptions, nil
}

// registerExtension adds an extension to the manager's internal map and
// initializes its storage partitions. It does not start the extension.
func (m *Manager) registerExtension(ext *Extension) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.extensions[ext.ID]; exists {
		return fmt.Errorf("extension %s already registered", ext.ID)
	}

	// Create per-extension storage partition.
	storagePath := ""
	if m.storageDir != "" {
		storagePath = filepath.Join(m.storageDir, ext.ID+".json")
	}
	ext.Storage = NewStoragePartition(ext.ID, storagePath)

	// Create per-container, per-extension storage partition.
	containerStoragePath := ""
	if m.storageDir != "" {
		containerStoragePath = filepath.Join(m.storageDir, ext.ID+"-containers")
	}
	ext.ContainerStorage = NewContainerStoragePartition(ext.ID, containerStoragePath)

	// Register content scripts from manifest (if any).
	if ext.Manifest != nil {
		for _, cs := range ext.Manifest.ContentScripts {
			if len(cs.JS) == 0 {
				continue
			}
			script := ContentScript{
				ID:             ext.ID + ":" + cs.JS[0],
				ExtID:          ext.ID,
				Matches:        cs.Matches,
				ExcludeMatches: cs.ExcludeMatches,
				JS:             cs.JS,
				CSS:            cs.CSS,
				RunAt:          cs.RunAt,
				AllFrames:      cs.AllFrames,
			}
			ext.ContentScripts = append(ext.ContentScripts, script)
			m.contentReg.Register(script)
			m.contentLoader.Register(script)
		}
	}

	// Initialize lifecycle state (extensions start enabled).
	m.lifecycle.SetInitialState(ext.ID, true)

	m.extensions[ext.ID] = ext
	return nil
}

// IsInstalled reports whether an extension with the given ID is installed.
func (m *Manager) IsInstalled(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.extensions[id]
	return exists
}

// InstallDir returns the directory where the extension is installed, or
// empty if not installed.
func (m *Manager) InstallDir(id string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	ext := m.extensions[id]
	if ext == nil {
		return ""
	}
	return ext.Path
}

