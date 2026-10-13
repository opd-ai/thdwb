package mustard

import "testing"

func TestAppActivationSwitchesTabsWithoutForcingReflow(t *testing.T) {
	app := CreateNewApp("tabs")
	first := &Window{active: true}
	second := &Window{active: true}
	app.AddWindow(first)
	app.AddWindow(second)

	firstChanges, secondChanges := []bool{}, []bool{}
	first.AttachActiveChangeListener(func(active bool) { firstChanges = append(firstChanges, active) })
	second.AttachActiveChangeListener(func(active bool) { secondChanges = append(secondChanges, active) })
	if err := app.ActivateWindow(first); err != nil {
		t.Fatal(err)
	}
	first.needsReflow = true
	if err := app.ActivateWindow(second); err != nil {
		t.Fatal(err)
	}
	if !second.active || first.active || !first.needsReflow {
		t.Fatal("tab switch did not isolate activity while preserving deferred frame work")
	}
	if len(firstChanges) != 2 || !firstChanges[0] || firstChanges[1] {
		t.Fatalf("first window active transitions = %v, want [true false]", firstChanges)
	}
	if len(secondChanges) != 3 || !secondChanges[0] || secondChanges[1] || !secondChanges[2] {
		t.Fatalf("second window active transitions = %v, want [true false true]", secondChanges)
	}
}

func TestAppTabModeAddsInactiveWindowsAndRejectsUnknownActivation(t *testing.T) {
	app := CreateNewApp("tabs")
	first := &Window{active: true}
	app.AddWindow(first)
	if err := app.ActivateWindow(first); err != nil {
		t.Fatal(err)
	}
	added := &Window{active: true}
	app.AddWindow(added)
	if added.active {
		t.Fatal("new window was active before being selected")
	}
	if err := app.ActivateWindow(&Window{}); err == nil {
		t.Fatal("activation accepted a window not managed by the app")
	}
}
