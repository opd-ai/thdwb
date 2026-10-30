package hotdog

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStorageOperations(t *testing.T) {
	manager, err := NewOriginStorageManager("")
	if err != nil {
		t.Fatal(err)
	}
	origin, _ := url.Parse("https://local-operations.example")
	ops := localStorageOperations(manager, DefaultContainerID, origin)
	if err := ops.setItem("b", "two"); err != nil {
		t.Fatal(err)
	}
	if err := ops.setItem("a", "one"); err != nil {
		t.Fatal(err)
	}
	key, ok := ops.keyAt(0)
	if ops.length() != 2 || !ok || key != "a" {
		t.Fatalf("unexpected localStorage operations: length=%d key=%q ok=%t", ops.length(), key, ok)
	}
	if _, ok := ops.keyAt(-1); ok {
		t.Fatal("keyAt() accepted a negative index")
	}
	if value, ok := ops.getItem("a"); !ok || value != "one" {
		t.Fatalf("getItem() = %q, %t", value, ok)
	}
	if err := ops.remove("a"); err != nil || ops.length() != 1 {
		t.Fatalf("remove() = %v, length=%d", err, ops.length())
	}
	if err := ops.clear(); err != nil || ops.length() != 0 {
		t.Fatalf("clear() = %v, length=%d", err, ops.length())
	}
}

func TestSessionStorageOperations(t *testing.T) {
	manager := NewSessionStorageManager()
	origin, _ := url.Parse("https://session-operations.example")
	ops := sessionStorageOperations(manager, DefaultContainerID, origin)
	if err := ops.setItem("key", "value"); err != nil {
		t.Fatal(err)
	}
	if value, ok := ops.getItem("key"); !ok || value != "value" || ops.length() != 1 {
		t.Fatalf("sessionStorage read = %q, %t, length=%d", value, ok, ops.length())
	}
	if key, ok := ops.keyAt(0); !ok || key != "key" {
		t.Fatalf("sessionStorage keyAt() = %q, %t", key, ok)
	}
	if err := ops.remove("key"); err != nil || ops.length() != 0 {
		t.Fatalf("sessionStorage remove() = %v, length=%d", err, ops.length())
	}
	if err := ops.clear(); err != nil {
		t.Fatalf("sessionStorage clear() = %v", err)
	}
}

func TestStorageRemovalAndClearRollbackOnPersistenceFailure(t *testing.T) {
	manager, err := NewOriginStorageManager("")
	if err != nil {
		t.Fatal(err)
	}
	origin, _ := url.Parse("https://rollback.example")
	if err := manager.SetItem(DefaultContainerID, origin, "key", "value"); err != nil {
		t.Fatal(err)
	}
	blockedPath := filepath.Join(t.TempDir(), "not-a-file")
	if err := os.Mkdir(blockedPath, 0o700); err != nil {
		t.Fatal(err)
	}
	manager.path = blockedPath
	if err := manager.RemoveItem(DefaultContainerID, origin, "key"); err == nil {
		t.Fatal("RemoveItem() unexpectedly persisted through a directory path")
	}
	if value, ok := manager.GetItem(DefaultContainerID, origin, "key"); !ok || value != "value" {
		t.Fatalf("failed RemoveItem() did not restore value: %q, %t", value, ok)
	}
	if err := manager.ClearOrigin(DefaultContainerID, origin); err == nil {
		t.Fatal("ClearOrigin() unexpectedly persisted through a directory path")
	}
	if value, ok := manager.GetItem(DefaultContainerID, origin, "key"); !ok || value != "value" {
		t.Fatalf("failed ClearOrigin() did not restore value: %q, %t", value, ok)
	}
}

func TestLocalStorageCompatibilityHelpers(t *testing.T) {
	origin, _ := url.Parse("https://compatibility.example")
	storage := GetLocalStorage(origin)
	if storage == nil {
		t.Fatal("GetLocalStorage() returned nil")
	}
	storage["key"] = "value"
	if len(GetAllLocalStorageOrigins()) == 0 {
		t.Fatal("GetAllLocalStorageOrigins() omitted stored origins")
	}
	if err := ClearLocalStorageWithError(DefaultContainerID, origin); err != nil {
		t.Fatal(err)
	}
	ClearLocalStorage(origin)
}
