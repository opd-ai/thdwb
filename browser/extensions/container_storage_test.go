package extensions

import (
	"testing"
)

func TestContainerStoragePartitionSetAndGet(t *testing.T) {
	csp := NewContainerStoragePartition("ext1", "")

	err := csp.Set("c1", StorageLocal, "key1", "value1")
	if err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	val, ok := csp.Get("c1", StorageLocal, "key1")
	if !ok || val != "value1" {
		t.Fatalf("Get = %q, %v, want %q, true", val, ok, "value1")
	}
}

func TestContainerStoragePartitionIsolation(t *testing.T) {
	csp := NewContainerStoragePartition("ext1", "")

	csp.Set("c1", StorageLocal, "key", "value-c1")
	csp.Set("c2", StorageLocal, "key", "value-c2")

	v1, _ := csp.Get("c1", StorageLocal, "key")
	if v1 != "value-c1" {
		t.Fatalf("c1 value = %q, want %q", v1, "value-c1")
	}
	v2, _ := csp.Get("c2", StorageLocal, "key")
	if v2 != "value-c2" {
		t.Fatalf("c2 value = %q, want %q", v2, "value-c2")
	}

	// Default container (empty ID) should be isolated too.
	csp.Set("", StorageLocal, "key", "value-default")
	vd, _ := csp.Get("", StorageLocal, "key")
	if vd != "value-default" {
		t.Fatalf("default value = %q, want %q", vd, "value-default")
	}
}

func TestContainerStoragePartitionRemove(t *testing.T) {
	csp := NewContainerStoragePartition("ext1", "")

	csp.Set("c1", StorageLocal, "key1", "value1")
	if err := csp.Remove("c1", StorageLocal, "key1"); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}

	if _, ok := csp.Get("c1", StorageLocal, "key1"); ok {
		t.Fatal("key should be removed")
	}
}

func TestContainerStoragePartitionClear(t *testing.T) {
	csp := NewContainerStoragePartition("ext1", "")

	csp.Set("c1", StorageLocal, "key1", "value1")
	csp.Set("c1", StorageLocal, "key2", "value2")
	csp.Set("c2", StorageLocal, "key1", "value1-c2")

	if err := csp.Clear("c1", StorageLocal); err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}

	if csp.Length("c1", StorageLocal) != 0 {
		t.Fatal("c1 should be empty after clear")
	}
	if csp.Length("c2", StorageLocal) != 1 {
		t.Fatal("c2 should still have 1 item")
	}
}

func TestContainerStoragePartitionClearContainer(t *testing.T) {
	csp := NewContainerStoragePartition("ext1", "")

	csp.Set("c1", StorageLocal, "key1", "value1")
	csp.Set("c1", StorageSync, "key2", "value2")
	csp.Set("c2", StorageLocal, "key1", "value1-c2")

	if err := csp.ClearContainer("c1"); err != nil {
		t.Fatalf("ClearContainer returned error: %v", err)
	}

	if csp.Length("c1", StorageLocal) != 0 {
		t.Fatal("c1 local should be empty after ClearContainer")
	}
	if csp.Length("c1", StorageSync) != 0 {
		t.Fatal("c1 sync should be empty after ClearContainer")
	}
	if csp.Length("c2", StorageLocal) != 1 {
		t.Fatal("c2 should still have 1 item")
	}
}

func TestContainerStoragePartitionContainerCount(t *testing.T) {
	csp := NewContainerStoragePartition("ext1", "")

	if csp.ContainerCount() != 0 {
		t.Fatalf("initial count = %d, want 0", csp.ContainerCount())
	}

	csp.Set("c1", StorageLocal, "key1", "value1")
	csp.Set("c2", StorageLocal, "key1", "value1")

	if csp.ContainerCount() != 2 {
		t.Fatalf("count = %d, want 2", csp.ContainerCount())
	}
}

func TestContainerStoragePartitionSyncToDisk(t *testing.T) {
	csp := NewContainerStoragePartition("ext1", "")

	err := csp.Set("c1", StorageLocal, "key1", "value1")
	if err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	if err := csp.SyncToDisk(); err != nil {
		t.Fatalf("SyncToDisk returned error: %v", err)
	}
}
