package shell

import (
	"testing"
)

func TestSettingsPageCreate(t *testing.T) {
	window := newTestWindow(t)
	settings := NewBrowserSettings()

	sp := NewSettingsPage(window, settings)
	if sp == nil {
		t.Fatal("NewSettingsPage returned nil")
	}
	if sp.Frame == nil {
		t.Fatal("settings page Frame should not be nil")
	}
	if sp.IsVisible() {
		t.Fatal("settings page should be hidden by default")
	}
}

func TestSettingsPageToggle(t *testing.T) {
	window := newTestWindow(t)
	settings := NewBrowserSettings()

	sp := NewSettingsPage(window, settings)
	if sp.IsVisible() {
		t.Fatal("settings page should start hidden")
	}
	sp.Toggle()
	if !sp.IsVisible() {
		t.Fatal("settings page should be visible after Toggle")
	}
	sp.Toggle()
	if sp.IsVisible() {
		t.Fatal("settings page should be hidden after second Toggle")
	}
}

func TestSettingsPageSetVisible(t *testing.T) {
	window := newTestWindow(t)
	settings := NewBrowserSettings()

	sp := NewSettingsPage(window, settings)
	sp.SetVisible(false)
	if sp.IsVisible() {
		t.Fatal("settings page should be hidden after SetVisible(false)")
	}
	sp.SetVisible(true)
	if !sp.IsVisible() {
		t.Fatal("settings page should be visible after SetVisible(true)")
	}
}

func TestSettingsPageHomepageInput(t *testing.T) {
	window := newTestWindow(t)
	settings := NewBrowserSettings()

	sp := NewSettingsPage(window, settings)
	if sp.homepageInput.GetValue() != settings.GetHomepage() {
		t.Fatal("homepage input should match settings homepage")
	}

	// Verify the input value can be updated.
	sp.homepageInput.SetValue("https://example.com")
	if sp.homepageInput.GetValue() != "https://example.com" {
		t.Fatalf("homepage input value = %q, want https://example.com", sp.homepageInput.GetValue())
	}
}

func TestSettingsPageDefaultSearchInput(t *testing.T) {
	window := newTestWindow(t)
	settings := NewBrowserSettings()

	sp := NewSettingsPage(window, settings)
	if sp.defaultSearchInput.GetValue() != settings.GetDefaultSearch() {
		t.Fatal("default search input should match settings default search")
	}

	// Verify the input value can be updated.
	sp.defaultSearchInput.SetValue("https://bing.com")
	if sp.defaultSearchInput.GetValue() != "https://bing.com" {
		t.Fatalf("default search input value = %q, want https://bing.com", sp.defaultSearchInput.GetValue())
	}
}
