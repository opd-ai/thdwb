package shell

import (
	mustard "github.com/danfragoso/thdwb/mustard"
)

// ExtensionToolbar displays enabled extension icons in the browser toolbar.
// Each enabled extension gets a clickable button that can open its popup.
type ExtensionToolbar struct {
	Frame     *mustard.Frame
	window    *mustard.WindowBase
	manager   *ExtensionManager
	buttons   map[string]*mustard.ButtonWidget
	onClick   func(ext *Extension) // callback when an extension icon is clicked
}

// NewExtensionToolbar constructs a toolbar bound to manager. Enabled
// extensions appear as buttons; clicking one invokes onClick.
func NewExtensionToolbar(window *mustard.WindowBase, manager *ExtensionManager) *ExtensionToolbar {
	tb := &ExtensionToolbar{
		window:  window,
		manager: manager,
		buttons: make(map[string]*mustard.ButtonWidget),
	}

	tb.Frame = mustard.CreateFrame(mustard.HorizontalFrame)
	tb.Frame.SetHeight(32)
	tb.Frame.SetBackgroundColor("#f5f5f5")

	managerChanged := func() { tb.Rebuild() }
	// Rebuild whenever extensions change. Since ExtensionManager has no
	// listener mechanism, callers should invoke Rebuild after mutations.
	_ = managerChanged

	tb.Rebuild()
	return tb
}

// Rebuild recreates the extension buttons to match the manager's enabled
// extensions.
func (tb *ExtensionToolbar) Rebuild() {
	for _, w := range tb.Frame.Widgets() {
		tb.Frame.DetachWidget(w)
	}
	tb.buttons = make(map[string]*mustard.ButtonWidget)

	for _, ext := range tb.manager.GetEnabledExtensions() {
		btn := mustard.CreateButtonWidget(ext.Name, nil)
		btn.SetWidth(32)
		btn.SetHeight(32)
		btn.SetFontSize(14)
		tb.Frame.AttachWidget(btn)
		tb.buttons[ext.ID] = btn

		id := ext.ID
		tb.window.RegisterButton(btn, func() {
			if tb.onClick != nil {
				if e := tb.manager.GetExtension(id); e != nil {
					tb.onClick(e)
				}
			}
		})
	}
}

// SetClickCallback sets the function invoked when an extension icon is clicked.
func (tb *ExtensionToolbar) SetClickCallback(fn func(ext *Extension)) {
	tb.onClick = fn
}

// ButtonFor returns the button for the given extension ID, if present.
func (tb *ExtensionToolbar) ButtonFor(extID string) *mustard.ButtonWidget {
	return tb.buttons[extID]
}
