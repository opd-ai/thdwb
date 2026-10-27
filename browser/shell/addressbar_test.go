package shell

import (
	"testing"
)

func TestAddressBarReflectsActiveTabURL(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	ab := NewAddressBar(window, m)

	a := m.Add("A", "https://example.com", "", "", "")
	if got := ab.URL(); got != "https://example.com" {
		t.Fatalf("URL = %q, want https://example.com", got)
	}
	if ab.SecurityState() != "secure" {
		t.Fatalf("SecurityState = %q, want secure", ab.SecurityState())
	}

	m.Add("B", "http://insecure.example", "", "", "")
	m.Activate(a.ID)
	if got := ab.URL(); got != "https://example.com" {
		t.Fatalf("URL after reactivate = %q, want https://example.com", got)
	}
}

func TestAddressBarSecurityIndicator(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	ab := NewAddressBar(window, m)

	m.Add("HTTPS", "https://secure.site", "", "", "")
	if ab.SecurityState() != "secure" {
		t.Fatalf("https SecurityState = %q, want secure", ab.SecurityState())
	}

	m.Add("HTTP", "http://insecure.site", "", "", "")
	m.Activate(m.ActiveTab().ID)
	if ab.SecurityState() != "insecure" {
		t.Fatalf("http SecurityState = %q, want insecure", ab.SecurityState())
	}

	m.Add("THDWB", "thdwb://newtab/", "", "", "")
	m.Activate(m.ActiveTab().ID)
	if ab.SecurityState() != "secure" {
		t.Fatalf("thdwb SecurityState = %q, want secure", ab.SecurityState())
	}
}

func TestAddressBarContainerIndicator(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	ab := NewAddressBar(window, m)

	m.Add("Default", "https://example.com", "", "", "")
	if ab.ContainerID() != "" {
		t.Fatalf("default ContainerID = %q, want empty", ab.ContainerID())
	}

	ctr := m.Add("Work", "https://work.example", "c1", "Work", "#ff0000")
	m.Activate(ctr.ID)
	if ab.ContainerID() != "c1" {
		t.Fatalf("container ContainerID = %q, want c1", ab.ContainerID())
	}
}

func TestAddressBarNavigateUpdatesTab(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	ab := NewAddressBar(window, m)

	a := m.Add("A", "https://example.com", "", "", "")
	ab.navigate("https://newsite.com")
	if got := a.URL; got != "https://newsite.com" {
		t.Fatalf("tab URL = %q, want https://newsite.com", got)
	}
	if got := ab.URL(); got != "https://newsite.com" {
		t.Fatalf("address bar URL = %q, want https://newsite.com", got)
	}
}

func TestAddressBarNavigateAddsHTTPScheme(t *testing.T) {
	window := newTestWindow(t)
	m := NewTabManager()
	ab := NewAddressBar(window, m)

	m.Add("A", "https://example.com", "", "", "")
	ab.navigate("example.com")
	if got := m.ActiveTab().URL; got != "https://example.com" {
		t.Fatalf("tab URL = %q, want https://example.com", got)
	}
}
