package hotdog

import (
	"net/url"
	"sync"
)

// originKey returns a normalized origin key for storage partitioning.
// Format: scheme://host:port
func originKey(u *url.URL) string {
	if u == nil {
		return "null"
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else if u.Scheme == "http" {
			port = "80"
		}
	}
	if port != "" {
		return u.Scheme + "://" + u.Hostname() + ":" + port
	}
	return u.Scheme + "://" + u.Hostname()
}

// StorageMap represents a key-value storage map (localStorage or sessionStorage)
type StorageMap map[string]string

// OriginStorageManager manages localStorage per origin (shared across windows).
type OriginStorageManager struct {
	mu       sync.RWMutex
	storages map[string]StorageMap // key: originKey
}

var globalLocalStorage = &OriginStorageManager{
	storages: make(map[string]StorageMap),
}

// GetLocalStorage returns the localStorage map for the given origin.
// Creates a new one if it doesn't exist.
func GetLocalStorage(origin *url.URL) StorageMap {
	key := originKey(origin)
	globalLocalStorage.mu.Lock()
	defer globalLocalStorage.mu.Unlock()

	if _, ok := globalLocalStorage.storages[key]; !ok {
		globalLocalStorage.storages[key] = make(StorageMap)
	}
	return globalLocalStorage.storages[key]
}

// ClearLocalStorage clears the localStorage for the given origin.
// This is typically not needed unless explicitly requested (e.g., clear-site-data).
func ClearLocalStorage(origin *url.URL) {
	key := originKey(origin)
	globalLocalStorage.mu.Lock()
	defer globalLocalStorage.mu.Unlock()
	delete(globalLocalStorage.storages, key)
}

// GetAllLocalStorageOrigins returns all origins that have localStorage data.
func GetAllLocalStorageOrigins() []string {
	globalLocalStorage.mu.RLock()
	defer globalLocalStorage.mu.RUnlock()
	origins := make([]string, 0, len(globalLocalStorage.storages))
	for k := range globalLocalStorage.storages {
		origins = append(origins, k)
	}
	return origins
}

// SessionStorageManager manages sessionStorage per window per origin.
// Unlike localStorage, sessionStorage is NOT shared across windows.
// However, it is still partitioned by origin within a window.
type SessionStorageManager struct {
	mu       sync.RWMutex
	storages map[string]StorageMap // key: originKey
}

func NewSessionStorageManager() *SessionStorageManager {
	return &SessionStorageManager{
		storages: make(map[string]StorageMap),
	}
}

// GetSessionStorage returns the sessionStorage map for the given origin within this window.
// Creates a new one if it doesn't exist.
func (s *SessionStorageManager) GetSessionStorage(origin *url.URL) StorageMap {
	key := originKey(origin)
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.storages[key]; !ok {
		s.storages[key] = make(StorageMap)
	}
	return s.storages[key]
}

// ClearSessionStorage clears the sessionStorage for the given origin in this window.
func (s *SessionStorageManager) ClearSessionStorage(origin *url.URL) {
	key := originKey(origin)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.storages, key)
}

// ClearAllSessionStorage clears all sessionStorage for this window (called on window close).
func (s *SessionStorageManager) ClearAllSessionStorage() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.storages = make(map[string]StorageMap)
}
