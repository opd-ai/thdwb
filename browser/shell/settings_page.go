package shell

import (
	mustard "github.com/danfragoso/thdwb/mustard"
)

// SettingsPage displays browser preferences: homepage, default search engine,
// bookmarks/history/extensions visibility, and container management.
type SettingsPage struct {
	Frame   *mustard.Frame
	window  *mustard.WindowBase
	settings *BrowserSettings

	homepageInput     *mustard.InputWidget
	defaultSearchInput *mustard.InputWidget
	showBookmarksBtn  *mustard.ButtonWidget
	showHistoryBtn    *mustard.ButtonWidget
	showExtensionsBtn *mustard.ButtonWidget
	visible           bool
}

// NewSettingsPage creates a settings page bound to the given settings.
func NewSettingsPage(window *mustard.WindowBase, settings *BrowserSettings) *SettingsPage {
	sp := &SettingsPage{
		window:   window,
		settings: settings,
		visible:  false,
	}

	sp.Frame = mustard.CreateFrame(mustard.VerticalFrame)
	sp.Frame.SetBackgroundColor("#ffffff")

	// Header.
	header := mustard.CreateLabelWidget("Settings")
	header.SetFontSize(18)
	header.SetFontColor("#333333")
	sp.Frame.AttachWidget(header)

	// Homepage setting.
	homeRow := mustard.CreateFrame(mustard.HorizontalFrame)
	homeRow.SetBackgroundColor("#fafafa")
	homeLabel := mustard.CreateLabelWidget("Homepage:")
	homeLabel.SetFontSize(12)
	homeLabel.SetFontColor("#333333")
	homeRow.AttachWidget(homeLabel)

	sp.homepageInput = mustard.CreateInputWidget()
	sp.homepageInput.SetValue(settings.GetHomepage())
	sp.homepageInput.SetWidth(300)
	sp.homepageInput.SetReturnCallback(func() {
		settings.SetHomepage(sp.homepageInput.GetValue())
	})
	homeRow.AttachWidget(sp.homepageInput)
	sp.Frame.AttachWidget(homeRow)

	// Default search engine setting.
	searchRow := mustard.CreateFrame(mustard.HorizontalFrame)
	searchRow.SetBackgroundColor("#fafafa")
	searchLabel := mustard.CreateLabelWidget("Default Search:")
	searchLabel.SetFontSize(12)
	searchLabel.SetFontColor("#333333")
	searchRow.AttachWidget(searchLabel)

	sp.defaultSearchInput = mustard.CreateInputWidget()
	sp.defaultSearchInput.SetValue(settings.GetDefaultSearch())
	sp.defaultSearchInput.SetWidth(300)
	sp.defaultSearchInput.SetReturnCallback(func() {
		settings.SetDefaultSearch(sp.defaultSearchInput.GetValue())
	})
	searchRow.AttachWidget(sp.defaultSearchInput)
	sp.Frame.AttachWidget(searchRow)

	// Visibility toggles.
	sp.showBookmarksBtn = mustard.CreateButtonWidget("Bookmarks: ON", nil)
	sp.showBookmarksBtn.SetFontSize(12)
	sp.showBookmarksBtn.SetWidth(120)
	sp.window.RegisterButton(sp.showBookmarksBtn, func() {
		show := !settings.IsShowBookmarks()
		settings.SetShowBookmarks(show)
		if show {
			sp.showBookmarksBtn.SetContent("Bookmarks: ON")
		} else {
			sp.showBookmarksBtn.SetContent("Bookmarks: OFF")
		}
	})
	sp.Frame.AttachWidget(sp.showBookmarksBtn)

	sp.showHistoryBtn = mustard.CreateButtonWidget("History: ON", nil)
	sp.showHistoryBtn.SetFontSize(12)
	sp.showHistoryBtn.SetWidth(120)
	sp.window.RegisterButton(sp.showHistoryBtn, func() {
		show := !settings.IsShowHistory()
		settings.SetShowHistory(show)
		if show {
			sp.showHistoryBtn.SetContent("History: ON")
		} else {
			sp.showHistoryBtn.SetContent("History: OFF")
		}
	})
	sp.Frame.AttachWidget(sp.showHistoryBtn)

	sp.showExtensionsBtn = mustard.CreateButtonWidget("Extensions: ON", nil)
	sp.showExtensionsBtn.SetFontSize(12)
	sp.showExtensionsBtn.SetWidth(120)
	sp.window.RegisterButton(sp.showExtensionsBtn, func() {
		show := !settings.IsShowExtensions()
		settings.SetShowExtensions(show)
		if show {
			sp.showExtensionsBtn.SetContent("Extensions: ON")
		} else {
			sp.showExtensionsBtn.SetContent("Extensions: OFF")
		}
	})
	sp.Frame.AttachWidget(sp.showExtensionsBtn)

	sp.updateButtonStates()

	return sp
}

// updateButtonStates refreshes the toggle button labels from settings.
func (sp *SettingsPage) updateButtonStates() {
	if sp.settings.IsShowBookmarks() {
		sp.showBookmarksBtn.SetContent("Bookmarks: ON")
	} else {
		sp.showBookmarksBtn.SetContent("Bookmarks: OFF")
	}
	if sp.settings.IsShowHistory() {
		sp.showHistoryBtn.SetContent("History: ON")
	} else {
		sp.showHistoryBtn.SetContent("History: OFF")
	}
	if sp.settings.IsShowExtensions() {
		sp.showExtensionsBtn.SetContent("Extensions: ON")
	} else {
		sp.showExtensionsBtn.SetContent("Extensions: OFF")
	}
}

// Toggle shows or hides the settings page.
func (sp *SettingsPage) Toggle() {
	sp.visible = !sp.visible
	sp.Frame.SetVisible(sp.visible)
}

// IsVisible reports whether the settings page is currently visible.
func (sp *SettingsPage) IsVisible() bool {
	return sp.visible
}

// SetVisible sets the settings page visibility.
func (sp *SettingsPage) SetVisible(visible bool) {
	sp.visible = visible
	sp.Frame.SetVisible(visible)
}
