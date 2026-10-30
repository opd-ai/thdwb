package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	extensions "github.com/danfragoso/thdwb/browser/extensions"
	mustard "github.com/danfragoso/thdwb/mustard"
)

// ExtensionSettingsPage displays extension metadata, permissions, storage
// usage, and provides enable/disable and uninstall controls.
type ExtensionSettingsPage struct {
	Frame       *mustard.Frame
	window      *mustard.WindowBase
	extManager  *ExtensionManager
	extMgr      *extensions.Manager
	settings    *BrowserSettings
	onUninstall func(extID string)

	// Per-extension rows.
	rows map[string]*extensionRow
}

// extensionRow represents a single extension's UI row in the settings page.
type extensionRow struct {
	extID   string
	frame   *mustard.Frame
	name    *mustard.LabelWidget
	version *mustard.LabelWidget
	desc    *mustard.LabelWidget
	enabled *mustard.ButtonWidget
	reload  *mustard.ButtonWidget
	uninstall *mustard.ButtonWidget
	perms   *mustard.LabelWidget
	storage *mustard.LabelWidget
}

// NewExtensionSettingsPage creates an extension settings page bound to the
// given managers.
func NewExtensionSettingsPage(window *mustard.WindowBase, extManager *ExtensionManager, extMgr *extensions.Manager, settings *BrowserSettings) *ExtensionSettingsPage {
	p := &ExtensionSettingsPage{
		window:     window,
		extManager: extManager,
		extMgr:     extMgr,
		settings:   settings,
		rows:       make(map[string]*extensionRow),
	}

	p.Frame = mustard.CreateFrame(mustard.VerticalFrame)
	p.Frame.SetBackgroundColor("#ffffff")

	p.Rebuild()
	return p
}

// Rebuild recreates the extension list to match the current manager state.
func (p *ExtensionSettingsPage) Rebuild() {
	for _, w := range p.Frame.Widgets() {
		p.Frame.DetachWidget(w)
	}
	p.rows = make(map[string]*extensionRow)

	// Header.
	header := mustard.CreateLabelWidget("Extensions")
	header.SetFontSize(18)
	header.SetFontColor("#333333")
	p.Frame.AttachWidget(header)

	// Extension list.
	exts := p.extMgr.GetAll()
	for _, ext := range exts {
		p.addExtensionRow(ext)
	}

	// If no extensions, show a message.
	if len(exts) == 0 {
		empty := mustard.CreateLabelWidget("No extensions installed.")
		empty.SetFontSize(14)
		empty.SetFontColor("#666666")
		p.Frame.AttachWidget(empty)
	}
}

func (p *ExtensionSettingsPage) addExtensionRow(ext *extensions.Extension) {
	row := &extensionRow{extID: ext.ID}

	row.frame = mustard.CreateFrame(mustard.HorizontalFrame)
	row.frame.SetBackgroundColor("#fafafa")

	// Name.
	row.name = mustard.CreateLabelWidget(ext.Name)
	row.name.SetFontSize(14)
	row.name.SetFontColor("#333333")
	row.frame.AttachWidget(row.name)

	// Version.
	row.version = mustard.CreateLabelWidget(ext.Version)
	row.version.SetFontSize(12)
	row.version.SetFontColor("#666666")
	row.frame.AttachWidget(row.version)

	// Description.
	if ext.Description != "" {
		row.desc = mustard.CreateLabelWidget(ext.Description)
		row.desc.SetFontSize(12)
		row.desc.SetFontColor("#666666")
		row.frame.AttachWidget(row.desc)
	}

	// Permissions.
	perms := strings.Join(ext.Permissions, ", ")
	if perms == "" {
		perms = "None"
	}
	row.perms = mustard.CreateLabelWidget("Permissions: " + perms)
	row.perms.SetFontSize(11)
	row.perms.SetFontColor("#999999")
	row.frame.AttachWidget(row.perms)

	// Enable/disable button.
	label := "Disable"
	if !ext.IsEnabled() {
		label = "Enable"
	}
	row.enabled = mustard.CreateButtonWidget(label, nil)
	row.enabled.SetWidth(70)
	row.enabled.SetHeight(24)
	row.enabled.SetFontSize(11)
	extID := ext.ID
	if p.window != nil {
		p.window.RegisterButton(row.enabled, func() {
			if ext.IsEnabled() {
				p.extMgr.Disable(extID)
			} else {
				if err := p.extMgr.Enable(extID); err != nil {
					// Ignore errors; the UI will reflect current state on rebuild.
					_ = err
				}
			}
			p.Rebuild()
		})
	}
	row.frame.AttachWidget(row.enabled)

	// Reload button (restarts background script).
	row.reload = mustard.CreateButtonWidget("Reload", nil)
	row.reload.SetWidth(60)
	row.reload.SetHeight(24)
	row.reload.SetFontSize(11)
	extID2 := ext.ID
	if p.window != nil {
		p.window.RegisterButton(row.reload, func() {
			if err := p.extMgr.Reload(extID2); err != nil {
				// Ignore errors; the UI will reflect current state on rebuild.
				_ = err
			}
			p.Rebuild()
		})
	}
	row.frame.AttachWidget(row.reload)

	// Storage usage.
	storageSize := dirSize(ext.Path)
	row.storage = mustard.CreateLabelWidget(formatBytes(storageSize))
	row.storage.SetFontSize(11)
	row.storage.SetFontColor("#999999")
	row.frame.AttachWidget(row.storage)

	// Uninstall button.
	row.uninstall = mustard.CreateButtonWidget("Remove", nil)
	row.uninstall.SetWidth(70)
	row.uninstall.SetHeight(24)
	row.uninstall.SetFontSize(11)
	row.uninstall.SetFontColor("#d32f2f")
	extID3 := ext.ID
	if p.window != nil {
		p.window.RegisterButton(row.uninstall, func() {
			if err := p.extMgr.Uninstall(extID3); err != nil {
				_ = err
			}
			if p.onUninstall != nil {
				p.onUninstall(extID3)
			}
			p.Rebuild()
		})
	}
	row.frame.AttachWidget(row.uninstall)

	p.Frame.AttachWidget(row.frame)
	p.rows[ext.ID] = row
}

// SetUninstallCallback sets a callback invoked when an extension is
// uninstalled from the settings page.
func (p *ExtensionSettingsPage) SetUninstallCallback(fn func(extID string)) {
	p.onUninstall = fn
}

// ExtensionInfo returns summary information about an installed extension,
// including storage usage on disk.
type ExtensionInfo struct {
	ID          string
	Name        string
	Version     string
	Description string
	Enabled     bool
	Permissions []string
	InstallPath string
	StorageSize int64
}

// GetExtensionInfo returns detailed info for the given extension ID.
func GetExtensionInfo(extMgr *extensions.Manager, extID string) (*ExtensionInfo, error) {
	ext := extMgr.Get(extID)
	if ext == nil {
		return nil, fmt.Errorf("extension %q not found", extID)
	}

	info := &ExtensionInfo{
		ID:          ext.ID,
		Name:        ext.Name,
		Version:     ext.Version,
		Description: ext.Description,
		Enabled:     ext.IsEnabled(),
		Permissions: ext.Permissions,
		InstallPath: ext.Path,
	}

	// Compute storage size on disk.
	if ext.Path != "" {
		info.StorageSize = dirSize(ext.Path)
	}

	return info, nil
}

// dirSize computes the total size of files in a directory.
func dirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		size += info.Size()
		return nil
	})
	return size
}

// formatBytes formats a byte count into a human-readable string.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	prefix := "KMGTPE"[exp]
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), prefix)
}
