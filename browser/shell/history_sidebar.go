package shell

import (
	mustard "github.com/danfragoso/thdwb/mustard"
)

// HistorySidebar displays browsing history in a searchable list. It is bound
// to a HistoryManager so it reflects the current history state.
type HistorySidebar struct {
	Frame   *mustard.Frame
	window  *mustard.WindowBase
	manager *HistoryManager

	list     *mustard.Frame
	visible  bool
	onSelect func(entry *HistoryEntry) // callback when a history entry is selected
}

// NewHistorySidebar constructs a sidebar bound to manager. History entries
// are displayed as a list with title and URL.
func NewHistorySidebar(window *mustard.WindowBase, manager *HistoryManager) *HistorySidebar {
	hs := &HistorySidebar{
		window:  window,
		manager: manager,
		visible: false,
	}

	hs.Frame = mustard.CreateFrame(mustard.VerticalFrame)
	hs.Frame.SetBackgroundColor("#fafafa")
	hs.Frame.SetWidth(300)

	// Header.
	header := mustard.CreateLabelWidget("History")
	header.SetFontSize(16)
	header.SetFontColor("#333333")
	hs.Frame.AttachWidget(header)

	// History list container.
	hs.list = mustard.CreateFrame(mustard.VerticalFrame)
	hs.list.SetBackgroundColor("#fafafa")
	hs.Frame.AttachWidget(hs.list)

	hs.Rebuild()
	return hs
}

// Rebuild recreates the history list to match the current manager state.
func (hs *HistorySidebar) Rebuild() {
	for _, w := range hs.list.Widgets() {
		hs.list.DetachWidget(w)
	}

	entries := hs.manager.GetAllEntries()
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		hs.addHistoryEntry(entry)
	}
}

// addHistoryEntry adds a single history entry row to the list.
func (hs *HistorySidebar) addHistoryEntry(entry *HistoryEntry) {
	row := mustard.CreateFrame(mustard.HorizontalFrame)
	row.SetBackgroundColor("#ffffff")

	title := mustard.CreateLabelWidget(entry.Title)
	title.SetFontSize(12)
	title.SetFontColor("#333333")
	row.AttachWidget(title)

	url := mustard.CreateLabelWidget(entry.URL)
	url.SetFontSize(11)
	url.SetFontColor("#666666")
	row.AttachWidget(url)

	hs.list.AttachWidget(row)
}

// Search filters the history list by query (title or URL, case-insensitive).
func (hs *HistorySidebar) Search(query string) {
	q := foldCase(query)
	for _, w := range hs.list.Widgets() {
		// Each row is a frame with two labels: title and URL.
		row, ok := w.(*mustard.Frame)
		if !ok {
			continue
		}
		widgets := row.Widgets()
		if len(widgets) < 2 {
			continue
		}
		titleLbl, ok1 := widgets[0].(*mustard.LabelWidget)
		urlLbl, ok2 := widgets[1].(*mustard.LabelWidget)
		if !ok1 || !ok2 {
			continue
		}
		match := q == "" ||
			containsFold(titleLbl.GetContent(), q) ||
			containsFold(urlLbl.GetContent(), q)
		row.SetVisible(match)
	}
}

// SetSelectCallback sets the function invoked when a history entry is
// selected. The callback receives the selected history entry.
func (hs *HistorySidebar) SetSelectCallback(fn func(entry *HistoryEntry)) {
	hs.onSelect = fn
}

// Toggle shows or hides the sidebar.
func (hs *HistorySidebar) Toggle() {
	hs.visible = !hs.visible
	hs.Frame.SetVisible(hs.visible)
}

// IsVisible reports whether the sidebar is currently visible.
func (hs *HistorySidebar) IsVisible() bool {
	return hs.visible
}

// SetVisible sets the sidebar visibility.
func (hs *HistorySidebar) SetVisible(visible bool) {
	hs.visible = visible
	hs.Frame.SetVisible(visible)
}

