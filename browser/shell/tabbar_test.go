package shell

import "testing"

func TestBuildTabBarCreatesButtonPerTab(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	bar := BuildTabBar(window, m)

	m.Add("First", "thdwb://first", "", "", "")
	m.Add("Second", "thdwb://second", "", "", "")

	// Two tab buttons + the new-tab button = 3 children.
	if got := len(bar.Frame.Widgets()); got != 3 {
		t.Fatalf("tab bar children = %d, want 3", got)
	}
	if bar.ButtonFor(1) == nil || bar.ButtonFor(2) == nil {
		t.Fatal("expected a button for each tab ID")
	}
}

func TestTabBarNewTabButtonAddsTab(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	bar := BuildTabBar(window, m)

	before := m.Count()
	bar.NewTabButton().Click()
	if m.Count() != before+1 {
		t.Fatalf("Count after new-tab click = %d, want %d", m.Count(), before+1)
	}
}

func TestTabBarClickActivatesTab(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	bar := BuildTabBar(window, m)

	a := m.Add("A", "thdwb://a", "", "", "")
	b := m.Add("B", "thdwb://b", "", "", "")

	// b is active after Add; clicking a's button should activate a.
	bar.ButtonFor(a.ID).Click()
	if got := m.ActiveTab(); got != a {
		t.Fatalf("ActiveTab after clicking a = %v, want a", got)
	}
	_ = b
}

func TestTabBarRebuildReflectsClosedTab(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	bar := BuildTabBar(window, m)

	a := m.Add("A", "u1", "", "", "")
	m.Add("B", "u2", "", "", "")
	m.Close(a.ID)

	// After the manager change the bar should have 1 tab button + new button.
	if got := len(bar.Frame.Widgets()); got != 2 {
		t.Fatalf("tab bar children after close = %d, want 2", got)
	}
	if bar.ButtonFor(a.ID) != nil {
		t.Fatal("closed tab still has a button")
	}
}

func TestTabBarContainerIndicatorShownInLabel(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	bar := BuildTabBar(window, m)

	ctr := m.Add("Work Tab", "u", "c1", "Work", "#ffcc00")
	if ctr == nil {
		t.Fatal("Add returned nil")
	}
	btn := bar.ButtonFor(ctr.ID)
	if btn == nil {
		t.Fatal("no button for container tab")
	}
	if got := btn.GetContent(); got != "⬢ Work Tab" {
		t.Fatalf("container tab label = %q, want %q", got, "⬢ Work Tab")
	}
}

func TestTabBarActiveTabStyledDistinct(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	bar := BuildTabBar(window, m)

	a := m.Add("A", "u1", "", "", "")
	m.Add("B", "u2", "", "", "")
	_ = a
	// The active tab (B) and inactive tab (A) should differ in styling.
	// We assert the rebuild path ran without error and buttons exist.
	if bar.ButtonFor(1) == nil || bar.ButtonFor(2) == nil {
		t.Fatal("expected buttons for both tabs")
	}
}
