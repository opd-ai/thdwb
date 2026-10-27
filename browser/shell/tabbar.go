package shell

import (
	assets "github.com/danfragoso/thdwb/assets"
	mustard "github.com/danfragoso/thdwb/mustard"
)

// tabButtonHeight is the fixed height of the tab bar and its buttons.
const tabButtonHeight = 32.0

// TabBarUI is the widget representation of the browser's tab bar. It mirrors
// the TabManager's state: one button per open tab (with a container indicator
// for container tabs) plus a trailing "new tab" button.
type TabBarUI struct {
	Frame   *mustard.Frame
	window  *mustard.WindowBase
	manager *TabManager
	buttons map[int]*mustard.ButtonWidget
	newBtn  *mustard.ButtonWidget
}

// BuildTabBar constructs a horizontal tab bar bound to manager. The bar
// rebuilds its tab buttons automatically whenever the manager changes, and the
// new-tab button creates a fresh tab.
func BuildTabBar(window *mustard.WindowBase, manager *TabManager) *TabBarUI {
	bar := &TabBarUI{
		Frame:   mustard.CreateFrame(mustard.HorizontalFrame),
		window:  window,
		manager: manager,
		buttons: make(map[int]*mustard.ButtonWidget),
	}
	bar.Frame.SetHeight(tabButtonHeight)
	bar.Frame.SetBackgroundColor("#e8e8e8")

	bar.newBtn = mustard.CreateButtonWidget("+", assets.ArrowRight())
	bar.newBtn.SetHeight(tabButtonHeight)
	bar.newBtn.SetWidth(36)
	bar.newBtn.SetFontSize(20)
	window.RegisterButton(bar.newBtn, func() {
		manager.Add("New Tab", "", "", "", "")
	})

	manager.OnChanged(func() { bar.Rebuild() })
	bar.Rebuild()
	return bar
}

// Rebuild recreates the tab buttons so they match the manager's current tabs.
// Existing tab buttons are unregistered from the window before being detached
// so that stale hit targets cannot fire old callbacks.
func (b *TabBarUI) Rebuild() {
	for _, btn := range b.buttons {
		b.window.UnregisterButton(btn)
		b.Frame.DetachWidget(btn)
	}
	b.buttons = make(map[int]*mustard.ButtonWidget)

	for _, tab := range b.manager.Tabs() {
		btn := b.makeTabButton(tab)
		b.Frame.AttachWidget(btn)
		b.buttons[tab.ID] = btn
	}
	b.Frame.AttachWidget(b.newBtn)
}

func (b *TabBarUI) makeTabButton(tab *Tab) *mustard.ButtonWidget {
	label := tab.Title
	if label == "" {
		label = "New Tab"
	}
	if indicator := tab.ContainerIndicator(); indicator != "" {
		label = "⬢ " + label
	}

	btn := mustard.CreateButtonWidget(label, nil)
	btn.SetHeight(tabButtonHeight)
	btn.SetFontSize(14)
	if tab.IsActive {
		btn.SetBackgroundColor("#ffffff")
	} else {
		btn.SetBackgroundColor("#dcdcdc")
	}
	// Tint the container tab with its container color for a quick visual cue.
	if tab.IsContainerTab() && tab.ContainerColor != "" {
		btn.SetBackgroundColor(tab.ContainerColor)
	}
	b.window.RegisterButton(btn, func() { b.manager.Activate(tab.ID) })
	return btn
}

// ButtonFor returns the tab button for the given tab ID, if present. Exposed
// for testing and for the shell to apply focus styling.
func (b *TabBarUI) ButtonFor(tabID int) *mustard.ButtonWidget {
	return b.buttons[tabID]
}

// NewTabButton returns the trailing new-tab button.
func (b *TabBarUI) NewTabButton() *mustard.ButtonWidget {
	return b.newBtn
}
