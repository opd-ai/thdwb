package shell

import (
	"path/filepath"
	"testing"
)

func TestBrowserSettingsDefaults(t *testing.T) {
	s := NewBrowserSettings()
	if s.GetHomepage() != "thdwb://newtab/" {
		t.Fatalf("Homepage = %q, want thdwb://newtab/", s.GetHomepage())
	}
	if s.GetDefaultSearch() != "https://duckduckgo.com/?q=" {
		t.Fatalf("DefaultSearch = %q, want https://duckduckgo.com/?q=", s.GetDefaultSearch())
	}
	if !s.IsShowBookmarks() {
		t.Fatal("ShowBookmarks should default to true")
	}
	if !s.IsShowHistory() {
		t.Fatal("ShowHistory should default to true")
	}
	if !s.IsShowExtensions() {
		t.Fatal("ShowExtensions should default to true")
	}
}

func TestBrowserSettingsSetters(t *testing.T) {
	s := NewBrowserSettings()
	s.SetHomepage("https://example.com")
	if s.GetHomepage() != "https://example.com" {
		t.Fatalf("Homepage = %q, want https://example.com", s.GetHomepage())
	}

	s.SetDefaultSearch("https://bing.com")
	if s.GetDefaultSearch() != "https://bing.com" {
		t.Fatalf("DefaultSearch = %q, want https://bing.com", s.GetDefaultSearch())
	}

	s.SetShowBookmarks(false)
	if s.IsShowBookmarks() {
		t.Fatal("ShowBookmarks should be false")
	}

	s.SetShowHistory(false)
	if s.IsShowHistory() {
		t.Fatal("ShowHistory should be false")
	}

	s.SetShowExtensions(false)
	if s.IsShowExtensions() {
		t.Fatal("ShowExtensions should be false")
	}
}

func TestBrowserSettingsExtensions(t *testing.T) {
	s := NewBrowserSettings()
	s.SetExtensions([]string{"ext1", "ext2"})
	exts := s.GetExtensions()
	if len(exts) != 2 || exts[0] != "ext1" || exts[1] != "ext2" {
		t.Fatalf("Extensions = %v, want [ext1 ext2]", exts)
	}
}

func TestBrowserSettingsContainers(t *testing.T) {
	s := NewBrowserSettings()
	s.SetContainers([]string{"Work", "Personal"})
	containers := s.GetContainers()
	if len(containers) != 2 || containers[0] != "Work" || containers[1] != "Personal" {
		t.Fatalf("Containers = %v, want [Work Personal]", containers)
	}
}

func TestBrowserSettingsSaveAndLoad(t *testing.T) {
	s := NewBrowserSettings()
	s.SetHomepage("https://example.com")
	s.SetExtensions([]string{"ext1"})

	tmp := filepath.Join(t.TempDir(), "settings.json")
	if err := s.SaveSettings(tmp); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	s2 := NewBrowserSettings()
	if err := s2.LoadSettings(tmp); err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}

	if s2.GetHomepage() != "https://example.com" {
		t.Fatalf("loaded Homepage = %q, want https://example.com", s2.GetHomepage())
	}
	exts := s2.GetExtensions()
	if len(exts) != 1 || exts[0] != "ext1" {
		t.Fatalf("loaded Extensions = %v, want [ext1]", exts)
	}
}
