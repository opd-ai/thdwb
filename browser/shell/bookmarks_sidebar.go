package shell

import (
	mustard "github.com/danfragoso/thdwb/mustard"
)

// BookmarksSidebar displays bookmarks in a collapsible sidebar with folder
// organization. It is bound to a BookmarkManager so it reflects the current
// bookmark state.
type BookmarksSidebar struct {
	Frame   *mustard.Frame
	window  *mustard.WindowBase
	manager *BookmarkManager

	tree     *mustard.TreeWidget
	visible  bool
	onSelect func(bookmark *Bookmark) // callback when a bookmark is selected
}

// NewBookmarksSidebar constructs a sidebar bound to manager. Bookmarks are
// displayed as a tree with folders as parent nodes and bookmarks as leaves.
func NewBookmarksSidebar(window *mustard.WindowBase, manager *BookmarkManager) *BookmarksSidebar {
	bs := &BookmarksSidebar{
		window:  window,
		manager: manager,
		visible: false,
	}

	bs.Frame = mustard.CreateFrame(mustard.VerticalFrame)
	bs.Frame.SetBackgroundColor("#fafafa")
	bs.Frame.SetWidth(240)

	// Header.
	header := mustard.CreateLabelWidget("Bookmarks")
	header.SetFontSize(16)
	header.SetFontColor("#333333")
	bs.Frame.AttachWidget(header)

	// Tree widget for bookmark folders and items.
	bs.tree = mustard.CreateTreeWidget()
	bs.Frame.AttachWidget(bs.tree)

	// Rebuild whenever bookmarks change.
	managerChanged := func() { bs.Rebuild() }
	_ = managerChanged

	bs.tree.SetSelectCallback(func(node *mustard.TreeWidgetNode) {
		if node.IsBookmark && bs.onSelect != nil {
			b := bs.manager.GetBookmark(node.ID)
			if b != nil {
				bs.onSelect(b)
			}
		}
	})

	bs.Rebuild()
	return bs
}

// Rebuild recreates the bookmark tree to match the current manager state.
func (bs *BookmarksSidebar) Rebuild() {
	bs.tree.RemoveNodes()

	// Root folder.
	root := mustard.CreateTreeWidgetNode("Bookmarks", "")
	bs.tree.AddNode(root)

	// Add all folders at top level.
	for _, folder := range bs.manager.GetAllFolders() {
		node := mustard.CreateTreeWidgetNode(folder.Title, "")
		node.ID = folder.ID
		node.IsFolder = true
		root.AddNode(node)

		// Add bookmarks in this folder.
		for _, b := range bs.manager.GetBookmarksByFolder(folder.ID) {
			bNode := mustard.CreateTreeWidgetNode(b.Title, b.URL)
			bNode.ID = b.ID
			bNode.IsBookmark = true
			node.AddNode(bNode)
		}
	}

	// Add top-level bookmarks (folder 0).
	for _, b := range bs.manager.GetBookmarksByFolder(0) {
		node := mustard.CreateTreeWidgetNode(b.Title, b.URL)
		node.ID = b.ID
		node.IsBookmark = true
		root.AddNode(node)
	}
}

// SetSelectCallback sets the function invoked when a bookmark node is
// selected. The callback receives the selected bookmark.
func (bs *BookmarksSidebar) SetSelectCallback(fn func(bookmark *Bookmark)) {
	bs.onSelect = fn
}

// Toggle shows or hides the sidebar.
func (bs *BookmarksSidebar) Toggle() {
	bs.visible = !bs.visible
	bs.Frame.SetVisible(bs.visible)
}

// IsVisible reports whether the sidebar is currently visible.
func (bs *BookmarksSidebar) IsVisible() bool {
	return bs.visible
}

// SetVisible sets the sidebar visibility.
func (bs *BookmarksSidebar) SetVisible(visible bool) {
	bs.visible = visible
	bs.Frame.SetVisible(visible)
}
