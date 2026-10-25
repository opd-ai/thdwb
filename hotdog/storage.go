package hotdog

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// originKey returns a normalized origin key for storage partitioning.
// Format: scheme://host:port
func originKey(u *url.URL) string {
	if u == nil {
		return "null"
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if port == "" {
		switch scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		}
	}
	if port != "" {
		return scheme + "://" + host + ":" + port
	}
	return scheme + "://" + host
}

// StorageMap represents the underlying values for localStorage or sessionStorage.
type StorageMap map[string]string

type storageOperations struct {
	length  func() int
	keyAt   func(int) (string, bool)
	getItem func(string) (string, bool)
	setItem func(string, string) error
	remove  func(string) error
	clear   func() error
}

// OriginStorageManager manages localStorage per origin and persists it when path is non-empty.
type OriginStorageManager struct {
	mu       sync.Mutex
	storages map[string]StorageMap
	path     string
}

// NewOriginStorageManager loads persistent localStorage from path.
// An empty path creates an in-memory manager.
func NewOriginStorageManager(path string) (*OriginStorageManager, error) {
	manager := &OriginStorageManager{storages: make(map[string]StorageMap), path: path}
	if path != "" {
		if err := manager.load(); err != nil {
			return nil, err
		}
	}
	return manager, nil
}

var globalLocalStorageState struct {
	once    sync.Once
	manager *OriginStorageManager
	err     error
}

func defaultOriginStorageManager() (*OriginStorageManager, error) {
	globalLocalStorageState.once.Do(func() {
		configDir, err := os.UserConfigDir()
		if err != nil {
			globalLocalStorageState.err = fmt.Errorf("get user config directory for localStorage: %w", err)
			return
		}
		path := filepath.Join(configDir, "thdwb", "local-storage.json")
		globalLocalStorageState.manager, globalLocalStorageState.err = NewOriginStorageManager(path)
	})
	return globalLocalStorageState.manager, globalLocalStorageState.err
}

// GetLocalStorage returns the shared underlying map for legacy callers.
// Mutations made directly to the map bypass persistent storage.
func GetLocalStorage(origin *url.URL) StorageMap {
	manager, err := defaultOriginStorageManager()
	if err != nil {
		log.Printf("initialize localStorage for %s: %v", originKey(origin), err)
		return nil
	}
	return manager.getLocalStorage(origin)
}

func getLocalStorageManager() (*OriginStorageManager, error) {
	return defaultOriginStorageManager()
}

func (manager *OriginStorageManager) getLocalStorage(origin *url.URL) StorageMap {
	key := originKey(origin)
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if manager.storages[key] == nil {
		manager.storages[key] = make(StorageMap)
	}
	return manager.storages[key]
}

// GetItem reads one value from localStorage for the given origin.
func (manager *OriginStorageManager) GetItem(origin *url.URL, itemKey string) (string, bool) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	value, ok := manager.storageLocked(origin)[itemKey]
	return value, ok
}

// SetItem stores a value in localStorage and persists it.
func (manager *OriginStorageManager) SetItem(origin *url.URL, itemKey, value string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	key := originKey(origin)
	storage := manager.storageLocked(origin)
	previous, existed := storage[itemKey]
	storage[itemKey] = value
	if err := manager.persistLocked(); err != nil {
		if existed {
			storage[itemKey] = previous
		} else {
			delete(storage, itemKey)
		}
		if len(storage) == 0 {
			delete(manager.storages, key)
		}
		return err
	}
	return nil
}

// RemoveItem deletes one key from localStorage and persists it.
func (manager *OriginStorageManager) RemoveItem(origin *url.URL, itemKey string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	key := originKey(origin)
	storage := manager.storageLocked(origin)
	previous, existed := storage[itemKey]
	if !existed {
		return nil
	}
	delete(storage, itemKey)
	if len(storage) == 0 {
		delete(manager.storages, key)
	}
	if err := manager.persistLocked(); err != nil {
		storage[itemKey] = previous
		manager.storages[key] = storage
		return err
	}
	return nil
}

// ClearOrigin clears localStorage for an origin and persists the change.
func (manager *OriginStorageManager) ClearOrigin(origin *url.URL) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	key := originKey(origin)
	previous, existed := manager.storages[key]
	delete(manager.storages, key)
	if err := manager.persistLocked(); err != nil {
		if existed {
			manager.storages[key] = previous
		}
		return err
	}
	return nil
}

func (manager *OriginStorageManager) storageLocked(origin *url.URL) StorageMap {
	key := originKey(origin)
	if manager.storages[key] == nil {
		manager.storages[key] = make(StorageMap)
	}
	return manager.storages[key]
}

func (manager *OriginStorageManager) length(origin *url.URL) int {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	return len(manager.storages[originKey(origin)])
}

func (manager *OriginStorageManager) keyAt(origin *url.URL, index int) (string, bool) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	storage := manager.storages[originKey(origin)]
	keys := make([]string, 0, len(storage))
	for key := range storage {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if index < 0 || index >= len(keys) {
		return "", false
	}
	return keys[index], true
}

func localStorageOperations(manager *OriginStorageManager, origin *url.URL) storageOperations {
	return storageOperations{
		length:  func() int { return manager.length(origin) },
		keyAt:   func(index int) (string, bool) { return manager.keyAt(origin, index) },
		getItem: func(key string) (string, bool) { return manager.GetItem(origin, key) },
		setItem: func(key, value string) error { return manager.SetItem(origin, key, value) },
		remove:  func(key string) error { return manager.RemoveItem(origin, key) },
		clear:   func() error { return manager.ClearOrigin(origin) },
	}
}

// ClearLocalStorage clears localStorage for an origin.
func ClearLocalStorage(origin *url.URL) {
	if err := ClearLocalStorageWithError(origin); err != nil {
		log.Printf("clear localStorage for %s: %v", originKey(origin), err)
	}
}

// ClearLocalStorageWithError clears localStorage and reports persistence failures.
func ClearLocalStorageWithError(origin *url.URL) error {
	manager, err := defaultOriginStorageManager()
	if err != nil {
		return err
	}
	return manager.ClearOrigin(origin)
}

// GetAllLocalStorageOrigins returns origins that currently have localStorage data.
func GetAllLocalStorageOrigins() []string {
	manager, err := defaultOriginStorageManager()
	if err != nil {
		log.Printf("list localStorage origins: %v", err)
		return nil
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	origins := make([]string, 0, len(manager.storages))
	for key := range manager.storages {
		origins = append(origins, key)
	}
	sort.Strings(origins)
	return origins
}

func (manager *OriginStorageManager) load() error {
	data, err := os.ReadFile(manager.path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(manager.path), 0o700); err != nil {
			return fmt.Errorf("create localStorage directory: %w", err)
		}
		return manager.persistLocked()
	}
	if err != nil {
		return fmt.Errorf("read localStorage: %w", err)
	}
	if err := os.Chmod(manager.path, 0o600); err != nil {
		return fmt.Errorf("secure localStorage permissions: %w", err)
	}
	if err := json.Unmarshal(data, &manager.storages); err != nil {
		return fmt.Errorf("decode localStorage: %w", err)
	}
	if manager.storages == nil {
		manager.storages = make(map[string]StorageMap)
	}
	return nil
}

func (manager *OriginStorageManager) persistLocked() error {
	if manager.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(manager.storages, "", "  ")
	if err != nil {
		return fmt.Errorf("encode localStorage: %w", err)
	}
	return writeLocalStorageFile(manager.path, data)
}

func writeLocalStorageFile(path string, data []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create localStorage temporary file: %w", err)
	}
	name := temporary.Name()
	defer func() { _ = os.Remove(name) }()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("secure localStorage temporary file: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write localStorage: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync localStorage: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close localStorage: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("replace localStorage: %w", err)
	}
	return nil
}

// SessionStorageManager manages sessionStorage per window and per origin.
type SessionStorageManager struct {
	mu       sync.Mutex
	storages map[string]StorageMap
}

// NewSessionStorageManager creates an empty per-window session storage manager.
func NewSessionStorageManager() *SessionStorageManager {
	return &SessionStorageManager{storages: make(map[string]StorageMap)}
}

// GetSessionStorage returns the sessionStorage map for this window and origin.
func (storage *SessionStorageManager) GetSessionStorage(origin *url.URL) StorageMap {
	key := originKey(origin)
	storage.mu.Lock()
	defer storage.mu.Unlock()
	if storage.storages[key] == nil {
		storage.storages[key] = make(StorageMap)
	}
	return storage.storages[key]
}

// GetItem reads one value from this window's sessionStorage.
func (storage *SessionStorageManager) GetItem(origin *url.URL, key string) (string, bool) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	value, ok := storage.storageLocked(origin)[key]
	return value, ok
}

// SetItem stores a value in this window's sessionStorage.
func (storage *SessionStorageManager) SetItem(origin *url.URL, key, value string) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	storage.storageLocked(origin)[key] = value
}

// RemoveItem deletes a key from this window's sessionStorage.
func (storage *SessionStorageManager) RemoveItem(origin *url.URL, key string) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	delete(storage.storages[originKey(origin)], key)
}

// ClearOrigin clears sessionStorage for an origin within this window.
func (storage *SessionStorageManager) ClearOrigin(origin *url.URL) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	delete(storage.storages, originKey(origin))
}

func (storage *SessionStorageManager) storageLocked(origin *url.URL) StorageMap {
	key := originKey(origin)
	if storage.storages[key] == nil {
		storage.storages[key] = make(StorageMap)
	}
	return storage.storages[key]
}

func (storage *SessionStorageManager) length(origin *url.URL) int {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	return len(storage.storages[originKey(origin)])
}

func (storage *SessionStorageManager) keyAt(origin *url.URL, index int) (string, bool) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	values := storage.storages[originKey(origin)]
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if index < 0 || index >= len(keys) {
		return "", false
	}
	return keys[index], true
}

func sessionStorageOperations(manager *SessionStorageManager, origin *url.URL) storageOperations {
	return storageOperations{
		length:  func() int { return manager.length(origin) },
		keyAt:   func(index int) (string, bool) { return manager.keyAt(origin, index) },
		getItem: func(key string) (string, bool) { return manager.GetItem(origin, key) },
		setItem: func(key, value string) error { manager.SetItem(origin, key, value); return nil },
		remove:  func(key string) error { manager.RemoveItem(origin, key); return nil },
		clear:   func() error { manager.ClearOrigin(origin); return nil },
	}
}

// ClearAllSessionStorage clears all sessionStorage for a window when it closes.
func (storage *SessionStorageManager) ClearAllSessionStorage() {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	storage.storages = make(map[string]StorageMap)
}
