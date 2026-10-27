package shell

import (
	"testing"
)

func TestExtensionToolbarCreatesButtonsForEnabledExtensions(t *testing.T) {
	window := newTestWindow(t)
	em := NewExtensionManager()
	em.AddExtension(&Extension{ID: "a", Name: "Ext A", Enabled: true})
	em.AddExtension(&Extension{ID: "b", Name: "Ext B", Enabled: false})
	em.AddExtension(&Extension{ID: "c", Name: "Ext C", Enabled: true})

	tb := NewExtensionToolbar(window, em)

	// Only enabled extensions (a and c) get buttons.
	if tb.ButtonFor("a") == nil {
		t.Fatal("expected button for enabled extension a")
	}
	if tb.ButtonFor("b") != nil {
		t.Fatal("should not have button for disabled extension b")
	}
	if tb.ButtonFor("c") == nil {
		t.Fatal("expected button for enabled extension c")
	}
}

func TestExtensionToolbarClickInvokesCallback(t *testing.T) {
	window := newTestWindow(t)
	em := NewExtensionManager()
	em.AddExtension(&Extension{ID: "a", Name: "Ext A", Enabled: true})

	tb := NewExtensionToolbar(window, em)

	var clicked *Extension
	tb.SetClickCallback(func(ext *Extension) { clicked = ext })

	tb.ButtonFor("a").Click()
	if clicked == nil || clicked.ID != "a" {
		t.Fatalf("clicked = %v, want extension a", clicked)
	}
}

func TestExtensionToolbarRebuildAfterEnable(t *testing.T) {
	window := newTestWindow(t)
	em := NewExtensionManager()
	em.AddExtension(&Extension{ID: "a", Name: "Ext A", Enabled: false})

	tb := NewExtensionToolbar(window, em)
	if tb.ButtonFor("a") != nil {
		t.Fatal("should not have button for disabled extension")
	}

	em.EnableExtension("a")
	tb.Rebuild()
	if tb.ButtonFor("a") == nil {
		t.Fatal("expected button after enabling extension")
	}
}
