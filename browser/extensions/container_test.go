package extensions

import (
	"testing"
)

func TestContainerManagerCreateAndGet(t *testing.T) {
	m := NewContainerManager()

	c, err := m.Create("Work", "#ff0000")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if c.ID == "" {
		t.Fatal("Create returned empty ID")
	}
	if c.Name != "Work" {
		t.Fatalf("container name = %q, want %q", c.Name, "Work")
	}
	if c.Color != "#ff0000" {
		t.Fatalf("container color = %q, want %q", c.Color, "#ff0000")
	}

	got := m.Get(c.ID)
	if got == nil {
		t.Fatal("Get returned nil for existing container")
	}
	if got.ID != c.ID || got.Name != c.Name {
		t.Fatal("Get returned wrong container")
	}
}

func TestContainerManagerCreateDuplicateName(t *testing.T) {
	m := NewContainerManager()

	if _, err := m.Create("Work", "#ff0000"); err != nil {
		t.Fatalf("first Create returned error: %v", err)
	}
	if _, err := m.Create("Work", "#00ff00"); err == nil {
		t.Fatal("expected error for duplicate container name")
	}
}

func TestContainerManagerGetByName(t *testing.T) {
	m := NewContainerManager()

	m.Create("Personal", "#00ff00")
	m.Create("Work", "#ff0000")

	c := m.GetByName("Work")
	if c == nil {
		t.Fatal("GetByName returned nil for existing container")
	}
	if c.Name != "Work" {
		t.Fatalf("GetByName name = %q, want %q", c.Name, "Work")
	}

	if m.GetByName("Nonexistent") != nil {
		t.Fatal("GetByName returned non-nil for missing container")
	}
}

func TestContainerManagerUpdate(t *testing.T) {
	m := NewContainerManager()

	c, _ := m.Create("Work", "#ff0000")
	updated, err := m.Update(c.ID, "Work2", "#00ff00")
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated == nil {
		t.Fatal("Update returned nil for existing container")
	}
	if updated.Name != "Work2" {
		t.Fatalf("updated name = %q, want %q", updated.Name, "Work2")
	}
	if updated.Color != "#00ff00" {
		t.Fatalf("updated color = %q, want %q", updated.Color, "#00ff00")
	}

	// Empty strings leave fields unchanged.
	_, err = m.Update(c.ID, "", "#0000ff")
	if err != nil {
		t.Fatalf("partial Update returned error: %v", err)
	}
	if got := m.Get(c.ID); got.Color != "#0000ff" || got.Name != "Work2" {
		t.Fatalf("partial update failed: name=%q color=%q", got.Name, got.Color)
	}

	if _, err := m.Update("nonexistent", "x", "y"); err == nil {
		t.Fatal("Update should return error for missing container")
	}
}

func TestContainerManagerRemove(t *testing.T) {
	m := NewContainerManager()

	c, _ := m.Create("Work", "#ff0000")
	if !m.Remove(c.ID) {
		t.Fatal("Remove returned false for existing container")
	}
	if m.Get(c.ID) != nil {
		t.Fatal("Get returned non-nil after Remove")
	}
	if m.Remove(c.ID) {
		t.Fatal("Remove returned true for already-removed container")
	}
}

func TestContainerManagerListAndCount(t *testing.T) {
	m := NewContainerManager()

	if m.Count() != 0 {
		t.Fatalf("initial count = %d, want 0", m.Count())
	}

	m.Create("A", "#111111")
	m.Create("B", "#222222")
	m.Create("C", "#333333")

	if m.Count() != 3 {
		t.Fatalf("count = %d, want 3", m.Count())
	}

	list := m.List()
	if len(list) != 3 {
		t.Fatalf("list length = %d, want 3", len(list))
	}

	// Verify all containers are present.
	names := map[string]bool{}
	for _, c := range list {
		names[c.Name] = true
	}
	if !names["A"] || !names["B"] || !names["C"] {
		t.Fatal("list missing containers")
	}
}

func TestContainerManagerHas(t *testing.T) {
	m := NewContainerManager()

	c, _ := m.Create("Work", "#ff0000")
	if !m.Has(c.ID) {
		t.Fatal("Has returned false for existing container")
	}
	if m.Has("nonexistent") {
		t.Fatal("Has returned true for missing container")
	}
}

func TestDefaultContainerID(t *testing.T) {
	if DefaultContainerID != "" {
		t.Fatalf("DefaultContainerID = %q, want empty string", DefaultContainerID)
	}
}
