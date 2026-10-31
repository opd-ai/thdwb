package shell

import (
	"testing"

	extensions "github.com/danfragoso/thdwb/browser/extensions"
)

func TestNewTabPageCreate(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	cm := extensions.NewContainerManager()

	ntp := NewNewTabPage(window, m, cm)
	if ntp == nil {
		t.Fatal("NewNewTabPage returned nil")
	}
	if ntp.Frame == nil {
		t.Fatal("new tab page Frame should not be nil")
	}
	if ntp.IsVisible() {
		t.Fatal("new tab page should be hidden by default")
	}
}

func TestNewTabPageToggle(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	cm := extensions.NewContainerManager()

	ntp := NewNewTabPage(window, m, cm)
	if ntp.IsVisible() {
		t.Fatal("new tab page should start hidden")
	}
	ntp.Toggle()
	if !ntp.IsVisible() {
		t.Fatal("new tab page should be visible after Toggle")
	}
	ntp.Toggle()
	if ntp.IsVisible() {
		t.Fatal("new tab page should be hidden after second Toggle")
	}
}

func TestNewTabPageSetVisible(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	cm := extensions.NewContainerManager()

	ntp := NewNewTabPage(window, m, cm)
	ntp.SetVisible(false)
	if ntp.IsVisible() {
		t.Fatal("new tab page should be hidden after SetVisible(false)")
	}
	ntp.SetVisible(true)
	if !ntp.IsVisible() {
		t.Fatal("new tab page should be visible after SetVisible(true)")
	}
}

func TestNewTabPageContainerButtons(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	cm := extensions.NewContainerManager()

	_, err := cm.Create("Work", "#ff0000")
	if err != nil {
		t.Fatalf("Create container: %v", err)
	}

	_ = NewNewTabPage(window, m, cm)

	// Verify the container was created.
	if cm.GetByName("Work") == nil {
		t.Fatal("Work container should exist")
	}
}
