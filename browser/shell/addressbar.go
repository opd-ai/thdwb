package shell

import (
	"net/url"
	"strings"

	mustard "github.com/danfragoso/thdwb/mustard"
)

// AddressBar represents the browser's URL bar with security and container
// indicators. It is a pure UI component bound to a TabManager so it always
// reflects the active tab's URL.
type AddressBar struct {
	Frame         *mustard.Frame
	window        *mustard.WindowBase
	manager       *TabManager

	input         *mustard.InputWidget
	security      *mustard.LabelWidget
	container     *mustard.LabelWidget
	goBtn         *mustard.ButtonWidget

	url           string
	securityState string // "secure", "insecure", "mixed"
	containerID   string
}

// NewAddressBar constructs an address bar bound to manager. The bar updates
// whenever the active tab changes and navigates when the user submits a URL.
func NewAddressBar(window *mustard.WindowBase, manager *TabManager) *AddressBar {
	ab := &AddressBar{
		window:         window,
		manager:        manager,
		url:            "",
		securityState:  "insecure",
		containerID:    "",
	}

	ab.Frame = mustard.CreateFrame(mustard.HorizontalFrame)
	ab.Frame.SetHeight(36)
	ab.Frame.SetBackgroundColor("#f5f5f5")

	// Security indicator (lock icon / text)
	ab.security = mustard.CreateLabelWidget("🔒")
	ab.security.SetFontSize(14)
	ab.security.SetFontColor("#4caf50")
	ab.security.SetWidth(24)
	ab.Frame.AttachWidget(ab.security)

	// Container indicator
	ab.container = mustard.CreateLabelWidget("")
	ab.container.SetFontSize(12)
	ab.container.SetFontColor("#666666")
	ab.container.SetWidth(60)
	ab.Frame.AttachWidget(ab.container)

	// URL input
	ab.input = mustard.CreateInputWidget()
	ab.input.SetFontSize(14)
	ab.input.SetFontColor("#333333")
	ab.Frame.AttachWidget(ab.input)

	// Go button
	ab.goBtn = mustard.CreateButtonWidget("Go", nil)
	ab.goBtn.SetWidth(40)
	ab.goBtn.SetHeight(30)
	ab.goBtn.SetFontSize(14)
	ab.Frame.AttachWidget(ab.goBtn)

	window.RegisterButton(ab.goBtn, func() { ab.navigate(ab.input.GetValue()) })
	ab.input.SetReturnCallback(func() { ab.navigate(ab.input.GetValue()) })

	manager.OnChanged(func() { ab.syncFromActiveTab() })
	ab.syncFromActiveTab()

	return ab
}

// navigate parses and loads the given URL string in the active tab.
func (ab *AddressBar) navigate(raw string) {
	if raw == "" {
		return
	}
	u := ab.normalizeURL(raw)
	if u == nil {
		return
	}
	tab := ab.manager.ActiveTab()
	if tab != nil {
		ab.manager.SetURL(tab.ID, u.String())
		ab.input.SetValue(u.String())
		ab.updateSecurity(u)
		ab.updateContainer(tab)
	}
}

// syncFromActiveTab refreshes the address bar from the currently active tab.
func (ab *AddressBar) syncFromActiveTab() {
	tab := ab.manager.ActiveTab()
	if tab == nil {
		ab.input.SetValue("")
		ab.security.SetContent("🔓")
		ab.security.SetFontColor("#ff9800")
		ab.container.SetContent("")
		return
	}
	ab.url = tab.URL
	ab.input.SetValue(tab.URL)
	if u, err := url.Parse(tab.URL); err == nil {
		ab.updateSecurity(u)
	} else {
		ab.security.SetContent("🔓")
		ab.security.SetFontColor("#ff9800")
	}
	ab.updateContainer(tab)
}

// normalizeURL adds a scheme if missing and parses the result.
func (ab *AddressBar) normalizeURL(raw string) *url.URL {
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil
	}
	return u
}

// updateSecurity sets the security indicator based on the URL scheme.
func (ab *AddressBar) updateSecurity(u *url.URL) {
	switch u.Scheme {
	case "https":
		ab.securityState = "secure"
		ab.security.SetContent("🔒")
		ab.security.SetFontColor("#4caf50")
	case "thdwb":
		ab.securityState = "secure"
		ab.security.SetContent("📄")
		ab.security.SetFontColor("#2196f3")
	default:
		ab.securityState = "insecure"
		ab.security.SetContent("🔓")
		ab.security.SetFontColor("#ff9800")
	}
}

// updateContainer shows the container name/color indicator for container tabs.
func (ab *AddressBar) updateContainer(tab *Tab) {
	if tab.IsContainerTab() {
		ab.containerID = tab.ContainerID
		ab.container.SetContent("⬢ " + tab.ContainerName)
		if tab.ContainerColor != "" {
			ab.container.SetFontColor(tab.ContainerColor)
		}
	} else {
		ab.containerID = ""
		ab.container.SetContent("")
	}
}

// SecurityState returns the current security state ("secure", "insecure", "mixed").
func (ab *AddressBar) SecurityState() string {
	return ab.securityState
}

// URL returns the current URL string shown in the address bar.
func (ab *AddressBar) URL() string {
	return ab.url
}

// ContainerID returns the container ID of the active tab, or empty string.
func (ab *AddressBar) ContainerID() string {
	return ab.containerID
}
