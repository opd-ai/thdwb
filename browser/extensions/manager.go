package extensions

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	goja "github.com/dop251/goja"
)

// Manager is the top-level extension manager. It combines discovery,
// lifecycle, storage, and content script registration into a single
// interface used by the browser shell.
type Manager struct {
	mu               sync.Mutex
	discovery        *Discovery
	lifecycle        *LifecycleManager
	extensions       map[string]*Extension
	storageDir       string
	contentReg       *ContentScriptRegistry
	contentLoader    *ContentScriptLoader
	sandboxes        map[string]*ContentScriptSandbox
	messageRouter    *MessageRouter
	chromeAPIs       map[string]*ChromeAPI
	tabOps           TabOperations
	winOps           WindowOperations
	containerOps     ContainerOperations
	cookieOps        CookieOperations
	scriptingOps     ScriptingOperations
	contextMenuOps   ContextMenuOperations
	permissionOps    PermissionOperations
	alarmOps         AlarmOperations
	i18nOps          I18nOperations
}

// NewManager creates a new extension manager.
func NewManager(storageDir string) *Manager {
	m := &Manager{
		discovery:     NewDiscovery(),
		lifecycle:     NewLifecycleManager(),
		extensions:    make(map[string]*Extension),
		storageDir:    storageDir,
		contentReg:    NewContentScriptRegistry(),
		contentLoader: NewContentScriptLoader(),
		sandboxes:     make(map[string]*ContentScriptSandbox),
		messageRouter: NewMessageRouter(),
		chromeAPIs:    make(map[string]*ChromeAPI),
	}

	// Wire lifecycle hooks to initialize Chrome API bindings on start and
	// clean up on stop.
	m.lifecycle.SetHooks(
		m.onExtensionStart,
		m.onExtensionStop,
		m.onExtensionReload,
	)

	return m
}

// AddWatchDir adds a directory to scan for extensions.
func (m *Manager) AddWatchDir(dir string) {
	m.discovery.AddWatchDir(dir)
}

// Scan discovers and loads extensions from all watch directories.
func (m *Manager) Scan() ([]*Extension, error) {
	found, err := m.discovery.Scan()
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, ext := range found {
		if _, exists := m.extensions[ext.ID]; exists {
			continue
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

		// Register content scripts from manifest.
		for _, cs := range ext.Manifest.ContentScripts {
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

		// Initialize lifecycle state (extensions start enabled).
		m.lifecycle.SetInitialState(ext.ID, true)

		m.extensions[ext.ID] = ext
	}

	return found, nil
}

// Get returns an extension by ID, or nil if not loaded.
func (m *Manager) Get(id string) *Extension {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.extensions[id]
}

// GetAll returns all loaded extensions.
func (m *Manager) GetAll() []*Extension {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Extension, 0, len(m.extensions))
	for _, ext := range m.extensions {
		out = append(out, ext)
	}
	return out
}

// GetEnabled returns all enabled extensions.
func (m *Manager) GetEnabled() []*Extension {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*Extension
	for _, ext := range m.extensions {
		if ext.IsEnabled() {
			out = append(out, ext)
		}
	}
	return out
}

// Enable enables an extension by ID.
func (m *Manager) Enable(id string) error {
	m.mu.Lock()
	ext := m.extensions[id]
	m.mu.Unlock()
	if ext == nil {
		return fmt.Errorf("extension not found: %s", id)
	}
	return m.lifecycle.Enable(ext)
}

// Disable disables an extension by ID.
func (m *Manager) Disable(id string) error {
	m.mu.Lock()
	ext := m.extensions[id]
	m.mu.Unlock()
	if ext == nil {
		return fmt.Errorf("extension not found: %s", id)
	}
	return m.lifecycle.Disable(ext)
}

// Reload reloads an extension by ID.
func (m *Manager) Reload(id string) error {
	m.mu.Lock()
	ext := m.extensions[id]
	m.mu.Unlock()
	if ext == nil {
		return fmt.Errorf("extension not found: %s", id)
	}
	return m.lifecycle.Reload(ext)
}

// Uninstall removes an extension by ID.
func (m *Manager) Uninstall(id string) error {
	m.mu.Lock()
	ext := m.extensions[id]
	delete(m.extensions, id)
	m.contentLoader.RemoveForExtension(id)
	if s, ok := m.sandboxes[id]; ok {
		delete(m.sandboxes, id)
		_ = s.Close()
	}
	m.mu.Unlock()

	if ext == nil {
		return fmt.Errorf("extension not found: %s", id)
	}
	return m.lifecycle.Uninstall(ext)
}

// IsEnabled reports whether an extension is enabled.
func (m *Manager) IsEnabled(id string) bool {
	return m.lifecycle.IsEnabled(id)
}

// Count returns the number of loaded extensions.
func (m *Manager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.extensions)
}

// EnabledCount returns the number of enabled extensions.
func (m *Manager) EnabledCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, ext := range m.extensions {
		if ext.IsEnabled() {
			count++
		}
	}
	return count
}

// StorageFor returns the storage partition for the given extension ID.
func (m *Manager) StorageFor(id string) *StoragePartition {
	m.mu.Lock()
	defer m.mu.Unlock()
	ext := m.extensions[id]
	if ext == nil {
		return nil
	}
	return ext.Storage
}

// ContentRegistry returns the content script registry.
func (m *Manager) ContentRegistry() *ContentScriptRegistry {
	return m.contentReg
}

// ContentLoader returns the content script loader.
func (m *Manager) ContentLoader() *ContentScriptLoader {
	return m.contentLoader
}

// MessageRouter returns the message router used for extension messaging.
func (m *Manager) MessageRouter() *MessageRouter {
	return m.messageRouter
}

// SandboxFor returns the content script sandbox for the given extension,
// creating one if it doesn't exist.
func (m *Manager) SandboxFor(extID string) *ContentScriptSandbox {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sandboxes[extID]; ok {
		return s
	}
	s := NewContentScriptSandbox(extID)
	m.sandboxes[extID] = s
	return s
}

// RemoveSandbox removes the content script sandbox for the given extension.
func (m *Manager) RemoveSandbox(extID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sandboxes[extID]; ok {
		_ = s.Close()
		delete(m.sandboxes, extID)
	}
}

// SetLifecycleHooks registers start/stop/reload hooks for background scripts.
func (m *Manager) SetLifecycleHooks(onStart, onStop, onReload func(ext *Extension) error) {
	m.lifecycle.SetHooks(onStart, onStop, onReload)
}

// SetTabOperations sets the tab operations provider for chrome.tabs API.
func (m *Manager) SetTabOperations(ops TabOperations) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tabOps = ops
}

// SetWindowOperations sets the window operations provider for chrome.windows API.
func (m *Manager) SetWindowOperations(ops WindowOperations) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.winOps = ops
}

// SetContainerOperations sets the container operations provider for chrome.contextualIdentities API.
func (m *Manager) SetContainerOperations(ops ContainerOperations) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.containerOps = ops
}

// SetCookieOperations sets the cookie operations provider for chrome.cookies API.
func (m *Manager) SetCookieOperations(ops CookieOperations) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cookieOps = ops
}

// SetScriptingOperations sets the scripting operations provider for chrome.scripting API.
func (m *Manager) SetScriptingOperations(ops ScriptingOperations) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scriptingOps = ops
}

// SetContextMenuOperations sets the context menu operations provider for chrome.contextMenus API.
func (m *Manager) SetContextMenuOperations(ops ContextMenuOperations) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.contextMenuOps = ops
}

// SetPermissionOperations sets the permission operations provider for chrome.permissions API.
func (m *Manager) SetPermissionOperations(ops PermissionOperations) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.permissionOps = ops
}

// SetAlarmOperations sets the alarm operations provider for chrome.alarms API.
func (m *Manager) SetAlarmOperations(ops AlarmOperations) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alarmOps = ops
}

// SetI18nOperations sets the i18n operations provider for chrome.i18n API.
func (m *Manager) SetI18nOperations(ops I18nOperations) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.i18nOps = ops
}

// onExtensionStart initializes the Chrome API bindings and background VM
// for an extension when it is enabled.
func (m *Manager) onExtensionStart(ext *Extension) error {
	m.mu.Lock()
	if _, exists := m.chromeAPIs[ext.ID]; exists {
		m.mu.Unlock()
		return nil // already initialized
	}

	vm := goja.New()
	ext.BackgroundVM = vm

	api := NewChromeAPI(ext, vm, m.messageRouter)
	api.SetTabOperations(m.tabOps)
	api.SetWindowOperations(m.winOps)
	api.SetContainerOperations(m.containerOps)
	api.SetCookieOperations(m.cookieOps)
	api.SetScriptingOperations(m.scriptingOps)
	api.SetContextMenuOperations(m.contextMenuOps)
	api.SetPermissionOperations(m.permissionOps)
	api.SetAlarmOperations(m.alarmOps)
	api.SetI18nOperations(m.i18nOps)

	m.chromeAPIs[ext.ID] = api
	m.mu.Unlock()

	if err := api.Bind(); err != nil {
		return fmt.Errorf("bind chrome APIs for %s: %w", ext.ID, err)
	}

	// Register the extension's sandbox with the message router.
	sandbox := m.SandboxFor(ext.ID)
	m.messageRouter.RegisterSandbox(ext.ID, sandbox)

	// Run the background script if one is declared in the manifest.
	if ext.Manifest.Background != nil {
		for _, script := range ext.Manifest.Background.Scripts {
			if err := runScriptInVM(vm, ext.ResolvePath(script)); err != nil {
				return fmt.Errorf("run background script %s: %w", script, err)
			}
		}
		if ext.Manifest.Background.ServiceWorker != "" {
			if err := runScriptInVM(vm, ext.ResolvePath(ext.Manifest.Background.ServiceWorker)); err != nil {
				return fmt.Errorf("run service worker %s: %w", ext.Manifest.Background.ServiceWorker, err)
			}
		}
	}

	return nil
}

// runScriptInVM reads a JavaScript file and executes it in the given VM.
func runScriptInVM(vm *goja.Runtime, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read script %s: %w", path, err)
	}
	if _, err := vm.RunString(string(data)); err != nil {
		return fmt.Errorf("execute script %s: %w", path, err)
	}
	return nil
}

// onExtensionStop cleans up the Chrome API bindings for an extension.
func (m *Manager) onExtensionStop(ext *Extension) error {
	m.mu.Lock()
	api, exists := m.chromeAPIs[ext.ID]
	if exists {
		delete(m.chromeAPIs, ext.ID)
	}
	m.messageRouter.UnregisterSandbox(ext.ID)
	m.mu.Unlock()

	if exists {
		_ = api.vm.Get("chrome") // release reference
		ext.BackgroundVM = nil
	}

	return nil
}

// onExtensionReload reinitializes the Chrome API bindings for an extension.
func (m *Manager) onExtensionReload(ext *Extension) error {
	if err := m.onExtensionStop(ext); err != nil {
		return err
	}
	return m.onExtensionStart(ext)
}
