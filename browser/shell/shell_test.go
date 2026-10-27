package shell

import (
	"testing"
)

func TestNewBrowserShellDefaults(t *testing.T) {
	bs := NewBrowserShell(nil, nil)
	if bs == nil {
		t.Fatal("NewBrowserShell returned nil")
	}
	if bs.settings == nil {
		t.Fatal("settings should not be nil")
	}
	if bs.tabManager == nil {
		t.Fatal("tabManager should not be nil")
	}
	if bs.bookmarkManager == nil {
		t.Fatal("bookmarkManager should not be nil")
	}
	if bs.historyManager == nil {
		t.Fatal("historyManager should not be nil")
	}
	if bs.extensionManager == nil {
		t.Fatal("extensionManager should not be nil")
	}
}

func TestBrowserShellTabManager(t *testing.T) {
	bs := NewBrowserShell(nil, nil)
	if bs.TabManager() != bs.tabManager {
		t.Fatal("TabManager() should return the internal manager")
	}
}

func TestBrowserShellBookmarkManager(t *testing.T) {
	bs := NewBrowserShell(nil, nil)
	if bs.BookmarkManager() != bs.bookmarkManager {
		t.Fatal("BookmarkManager() should return the internal manager")
	}
}

func TestBrowserShellHistoryManager(t *testing.T) {
	bs := NewBrowserShell(nil, nil)
	if bs.HistoryManager() != bs.historyManager {
		t.Fatal("HistoryManager() should return the internal manager")
	}
}

func TestBrowserShellExtensionManager(t *testing.T) {
	bs := NewBrowserShell(nil, nil)
	if bs.ExtensionManager() != bs.extensionManager {
		t.Fatal("ExtensionManager() should return the internal manager")
	}
}

func TestBrowserShellSettings(t *testing.T) {
	bs := NewBrowserShell(nil, nil)
	if bs.Settings() != bs.settings {
		t.Fatal("Settings() should return the internal settings")
	}
}
