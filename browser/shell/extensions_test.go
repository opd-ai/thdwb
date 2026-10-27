package shell

import (
	"testing"
)

func TestExtensionManagerAddAndGet(t *testing.T) {
	em := NewExtensionManager()

	ext := &Extension{ID: "ext1", Name: "Test Ext", Version: "1.0", Enabled: true}
	em.AddExtension(ext)

	got := em.GetExtension("ext1")
	if got == nil || got.ID != "ext1" {
		t.Fatalf("GetExtension(ext1) = %v, want ext1", got)
	}
	if em.GetExtension("unknown") != nil {
		t.Fatal("GetExtension(unknown) should return nil")
	}
}

func TestExtensionManagerGetAllAndEnabled(t *testing.T) {
	em := NewExtensionManager()
	em.AddExtension(&Extension{ID: "a", Name: "A", Enabled: true})
	em.AddExtension(&Extension{ID: "b", Name: "B", Enabled: false})
	em.AddExtension(&Extension{ID: "c", Name: "C", Enabled: true})

	all := em.GetAllExtensions()
	if len(all) != 3 {
		t.Fatalf("GetAllExtensions = %d, want 3", len(all))
	}

	enabled := em.GetEnabledExtensions()
	if len(enabled) != 2 {
		t.Fatalf("GetEnabledExtensions = %d, want 2", len(enabled))
	}
}

func TestExtensionManagerEnableDisable(t *testing.T) {
	em := NewExtensionManager()
	em.AddExtension(&Extension{ID: "a", Name: "A", Enabled: true})

	em.DisableExtension("a")
	if em.GetExtension("a").Enabled {
		t.Fatal("extension should be disabled")
	}

	em.EnableExtension("a")
	if !em.GetExtension("a").Enabled {
		t.Fatal("extension should be enabled")
	}

	em.ToggleExtension("a")
	if em.GetExtension("a").Enabled {
		t.Fatal("extension should be disabled after toggle")
	}
}

func TestExtensionManagerRemove(t *testing.T) {
	em := NewExtensionManager()
	em.AddExtension(&Extension{ID: "a", Name: "A"})

	em.RemoveExtension("a")
	if em.GetExtension("a") != nil {
		t.Fatal("extension should be removed")
	}
	if got := em.ExtensionCount(); got != 0 {
		t.Fatalf("ExtensionCount = %d, want 0", got)
	}
}
