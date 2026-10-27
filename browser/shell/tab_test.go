package shell

import "testing"

func TestTabManagerAddAssignsUniqueIDsAndActivates(t *testing.T) {
	m := NewTabManager()
	a := m.Add("A", "thdwb://a", "", "", "")
	b := m.Add("B", "thdwb://b", "", "", "")

	if a.ID == b.ID {
		t.Fatalf("IDs not unique: a=%d b=%d", a.ID, b.ID)
	}
	if !b.IsActive || a.IsActive {
		t.Fatal("newest tab should be active, older inactive")
	}
	if got := m.ActiveTab(); got != b {
		t.Fatalf("ActiveTab = %v, want b", got)
	}
	if m.Count() != 2 {
		t.Fatalf("Count = %d, want 2", m.Count())
	}
}

func TestTabManagerCloseActivatesNeighborAndRecordsClosed(t *testing.T) {
	m := NewTabManager()
	m.Add("A", "u1", "", "", "")
	b := m.Add("B", "u2", "", "", "")
	c := m.Add("C", "u3", "", "", "")

	// Close the middle tab; the one after it (c) should become active.
	closed := m.Close(b.ID)
	if closed == nil || closed.ID != b.ID {
		t.Fatalf("Close returned %v, want b", closed)
	}
	if m.Count() != 2 {
		t.Fatalf("Count after close = %d, want 2", m.Count())
	}
	if got := m.ActiveTab(); got != c {
		t.Fatalf("ActiveTab after closing middle = %v, want c", got)
	}

	// Closed tab is on the recently-closed stack (oldest first).
	rc := m.RecentlyClosed()
	if len(rc) != 1 || rc[0].ID != b.ID {
		t.Fatalf("RecentlyClosed = %+v, want [b]", rc)
	}
}

func TestTabManagerCloseLastTabLeavesNoActive(t *testing.T) {
	m := NewTabManager()
	a := m.Add("A", "u", "", "", "")
	m.Close(a.ID)
	if m.ActiveTab() != nil {
		t.Fatalf("ActiveTab = %v, want nil after closing last tab", m.ActiveTab())
	}
	if m.Count() != 0 {
		t.Fatalf("Count = %d, want 0", m.Count())
	}
}

func TestTabManagerCloseUnknownIDIsNoop(t *testing.T) {
	m := NewTabManager()
	m.Add("A", "u", "", "", "")
	if got := m.Close(999); got != nil {
		t.Fatalf("Close(unknown) = %v, want nil", got)
	}
	if m.Count() != 1 {
		t.Fatalf("Count = %d, want 1", m.Count())
	}
}

func TestTabManagerActivate(t *testing.T) {
	m := NewTabManager()
	a := m.Add("A", "u1", "", "", "")
	b := m.Add("B", "u2", "", "", "")

	if got := m.Activate(a.ID); got != a {
		t.Fatalf("Activate(a) = %v, want a", got)
	}
	if !a.IsActive || b.IsActive {
		t.Fatal("activation flags wrong after Activate(a)")
	}
	// Activating the already-active tab is a no-op (no spurious change).
	if got := m.Activate(a.ID); got != a {
		t.Fatalf("re-Activate(a) = %v, want a", got)
	}
	if got := m.Activate(42); got != nil {
		t.Fatalf("Activate(unknown) = %v, want nil", got)
	}
}

func TestTabManagerMoveTab(t *testing.T) {
	m := NewTabManager()
	a := m.Add("A", "u1", "", "", "")
	b := m.Add("B", "u2", "", "", "")
	c := m.Add("C", "u3", "", "", "")

	m.MoveTab(0, 2) // move a to the end
	tabs := m.Tabs()
	want := []int{b.ID, c.ID, a.ID}
	for i, id := range want {
		if tabs[i].ID != id {
			t.Fatalf("after MoveTab(0,2) order = %v, want %v", ids(tabs), want)
		}
	}

	// Out-of-range moves are ignored.
	m.MoveTab(5, 0)
	m.MoveTab(0, 9)
	if got := len(m.Tabs()); got != 3 {
		t.Fatalf("Count after invalid moves = %d, want 3", got)
	}
}

func TestTabManagerSetTitleAndURL(t *testing.T) {
	m := NewTabManager()
	a := m.Add("old", "u0", "", "", "")
	m.SetTitle(a.ID, "new title")
	m.SetURL(a.ID, "thdwb://new")
	if a.Title != "new title" || a.URL != "thdwb://new" {
		t.Fatalf("tab = %+v, want updated title/url", a)
	}
	// Unknown IDs are ignored.
	m.SetTitle(999, "x")
	m.SetURL(999, "y")
}

func TestTabManagerRestoreClosed(t *testing.T) {
	m := NewTabManager()
	a := m.Add("A", "u1", "c1", "Work", "#ff0000")
	m.Add("B", "u2", "", "", "")
	m.Close(a.ID)

	restored := m.RestoreClosed()
	if restored == nil || restored.ID != a.ID {
		t.Fatalf("RestoreClosed = %v, want a", restored)
	}
	if m.Count() != 2 {
		t.Fatalf("Count after restore = %d, want 2", m.Count())
	}
	if !restored.IsActive {
		t.Fatal("restored tab should be active")
	}
	if len(m.RecentlyClosed()) != 0 {
		t.Fatalf("RecentlyClosed = %+v, want empty", m.RecentlyClosed())
	}
	// Restoring with an empty stack is a no-op.
	if got := m.RestoreClosed(); got != nil {
		t.Fatalf("RestoreClosed on non-empty = %v, want nil", got)
	}
}

func TestTabManagerRecentlyClosedBounded(t *testing.T) {
	m := NewTabManager()
	m.maxClosed = 3
	for i := 0; i < 5; i++ {
		tabs := m.Tabs()
		if len(tabs) == 0 {
			break
		}
		m.Close(tabs[0].ID)
	}
	if got := len(m.RecentlyClosed()); got != 3 {
		t.Fatalf("RecentlyClosed len = %d, want bounded to 3", got)
	}
}

func TestTabManagerOnChangedFires(t *testing.T) {
	m := NewTabManager()
	changes := 0
	m.OnChanged(func() { changes++ })

	m.Add("A", "u", "", "", "")
	if changes != 1 {
		t.Fatalf("changes after Add = %d, want 1", changes)
	}
	a := m.ActiveTab()
	m.Close(a.ID)
	if changes != 2 {
		t.Fatalf("changes after Close = %d, want 2", changes)
	}
	m.Add("B", "u", "", "", "")
	m.Activate(m.ActiveTab().ID) // no-op activate
	if changes != 3 {
		t.Fatalf("changes after no-op activate = %d, want 3 (unchanged)", changes)
	}
}

func TestTabContainerIndicator(t *testing.T) {
	def := &Tab{Title: "A", URL: "u"}
	ctr := &Tab{Title: "B", URL: "u", ContainerID: "c1", ContainerName: "Work", ContainerColor: "#123456"}

	if def.IsContainerTab() {
		t.Fatal("default tab reported as container tab")
	}
	if def.ContainerIndicator() != "" {
		t.Fatalf("default indicator = %q, want empty", def.ContainerIndicator())
	}
	if !ctr.IsContainerTab() {
		t.Fatal("container tab not reported as container tab")
	}
	if ctr.ContainerIndicator() != "Work" {
		t.Fatalf("container indicator = %q, want Work", ctr.ContainerIndicator())
	}
}

func ids(tabs []*Tab) []int {
	out := make([]int, len(tabs))
	for i, t := range tabs {
		out[i] = t.ID
	}
	return out
}
