package shell

import (
	"testing"
)

func TestMenuBarCreate(t *testing.T) {
	window := newTestWindow(t)

	mb := NewMenuBar(window)
	if mb == nil {
		t.Fatal("NewMenuBar returned nil")
	}
	if mb.Frame == nil {
		t.Fatal("menu bar Frame should not be nil")
	}
	if !mb.IsVisible() {
		t.Fatal("menu bar should be visible by default")
	}
}

func TestMenuBarToggle(t *testing.T) {
	window := newTestWindow(t)

	mb := NewMenuBar(window)
	if !mb.IsVisible() {
		t.Fatal("menu bar should start visible")
	}
	mb.Toggle()
	if mb.IsVisible() {
		t.Fatal("menu bar should be hidden after Toggle")
	}
	mb.Toggle()
	if !mb.IsVisible() {
		t.Fatal("menu bar should be visible after second Toggle")
	}
}

func TestMenuBarSetVisible(t *testing.T) {
	window := newTestWindow(t)

	mb := NewMenuBar(window)
	mb.SetVisible(false)
	if mb.IsVisible() {
		t.Fatal("menu bar should be hidden after SetVisible(false)")
	}
	mb.SetVisible(true)
	if !mb.IsVisible() {
		t.Fatal("menu bar should be visible after SetVisible(true)")
	}
}

func TestMenuBarCallbacks(t *testing.T) {
	window := newTestWindow(t)

	mb := NewMenuBar(window)

	var newTabCalled bool
	mb.SetNewTabCallback(func() { newTabCalled = true })

	var historyCalled bool
	mb.SetShowHistoryCallback(func() { historyCalled = true })

	var bookmarksCalled bool
	mb.SetShowBookmarksCallback(func() { bookmarksCalled = true })

	var settingsCalled bool
	mb.SetShowSettingsCallback(func() { settingsCalled = true })

	var aboutCalled bool
	mb.SetAboutCallback(func() { aboutCalled = true })

	// Invoke the file menu button click (triggers onNewTab).
	mb.fileBtn.Click()
	if !newTabCalled {
		t.Fatal("new tab callback should be invoked")
	}

	// Invoke the history button click.
	mb.historyBtn.Click()
	if !historyCalled {
		t.Fatal("history callback should be invoked")
	}

	// Invoke the bookmarks button click.
	mb.bookmarksBtn.Click()
	if !bookmarksCalled {
		t.Fatal("bookmarks callback should be invoked")
	}

	// Invoke the tools menu button click (triggers onShowSettings).
	mb.toolsBtn.Click()
	if !settingsCalled {
		t.Fatal("settings callback should be invoked")
	}

	// Invoke the help menu button click (triggers onAbout).
	mb.helpBtn.Click()
	if !aboutCalled {
		t.Fatal("about callback should be invoked")
	}
}
