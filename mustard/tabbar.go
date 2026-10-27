package mustard

import (
	"strings"

	assets "github.com/danfragoso/thdwb/assets"
	gg "github.com/danfragoso/thdwb/gg"
	"golang.org/x/image/font"
)

// Tab represents a single tab in the tab bar.
type Tab struct {
	ID        int
	Title     string
	URL       string
	Closed    bool
}

// TabBar is a horizontal bar of closable tabs with a new-tab button.
type TabBar struct {
	baseWidget
	tabs       []*Tab
	activeTab  *Tab
	font       font.Face
	fontSize   float64
	tabHeight  float64
	closeSize  float64
	newTabSize float64
	tabWidth   float64
}

// CreateTabBar creates a new tab bar with the specified height.
func CreateTabBar(height float64) *TabBar {
	face, _ := loadFontFace(assets.OpenSans(400), 14)
	return &TabBar{
		baseWidget: baseWidget{
			widgetType:    tabbarWidget,
			needsRepaint:  true,
			visible:       true,
			backgroundColor: "#f0f0f0",
			font:          face,
		},
		fontSize:   14,
		tabHeight:  height,
		closeSize:  12,
		newTabSize: 16,
		tabWidth:   140,
	}
}

// AddTab adds a new tab with the given title and URL.
func (tb *TabBar) AddTab(title, url string) *Tab {
	id := 1
	for _, t := range tb.tabs {
		if !t.Closed && t.ID >= id {
			id = t.ID + 1
		}
	}
	tab := &Tab{ID: id, Title: title, URL: url}
	tb.tabs = append(tb.tabs, tab)
	if tb.activeTab == nil || tb.activeTab.Closed {
		tb.activeTab = tab
	}
	tb.needsRepaint = true
	return tab
}

// CloseTab closes the tab with the given ID.
func (tb *TabBar) CloseTab(id int) {
	for _, tab := range tb.tabs {
		if tab.ID == id {
			tab.Closed = true
			if tb.activeTab == tab {
				tb.activateNext(tab)
			}
			tb.needsRepaint = true
			return
		}
	}
}

// SetActiveTab sets the active tab by ID.
func (tb *TabBar) SetActiveTab(id int) {
	for _, tab := range tb.tabs {
		if tab.ID == id {
			tb.activeTab = tab
			tb.needsRepaint = true
			return
		}
	}
}

// GetActiveTab returns the active tab.
func (tb *TabBar) GetActiveTab() *Tab {
	return tb.activeTab
}

// Tabs returns the list of open tabs.
func (tb *TabBar) Tabs() []*Tab {
	open := make([]*Tab, 0)
	for _, tab := range tb.tabs {
		if !tab.Closed {
			open = append(open, tab)
		}
	}
	return open
}

// SetTabTitle sets the title for a tab.
func (tb *TabBar) SetTabTitle(id int, title string) {
	for _, tab := range tb.tabs {
		if tab.ID == id {
			tab.Title = title
			tb.needsRepaint = true
			return
		}
	}
}

func (tb *TabBar) SetWidth(width float64) {
	tb.box.width = width
	tb.fixedWidth = true
	tb.RequestReflow()
}

func (tb *TabBar) SetHeight(height float64) {
	tb.box.height = height
	tb.fixedHeight = true
	tb.RequestReflow()
}

func (tb *TabBar) activateNext(closed *Tab) {
	for _, tab := range tb.tabs {
		if !tab.Closed && tab != closed {
			tb.activeTab = tab
			return
		}
	}
	tb.activeTab = nil
}

func (tb *TabBar) tabAt(x float64) (*Tab, bool) {
	var offsetX float64 = 0
	for _, tab := range tb.tabs {
		if tab.Closed {
			continue
		}
		tabBoxLeft := offsetX
		tabBoxRight := offsetX + tb.tabWidth
		if x >= tabBoxLeft && x <= tabBoxRight {
			return tab, true
		}
		offsetX = tabBoxRight
	}
	return nil, false
}

func (tb *TabBar) closeButtonAt(x, y float64, tab *Tab) bool {
	var offsetX float64 = 0
	for _, t := range tb.tabs {
		if t.Closed {
			continue
		}
		closeX := offsetX + tb.tabWidth - tb.closeSize - 4
		closeY := tb.tabHeight - tb.closeSize - 4
		if t == tab && x >= closeX && x <= closeX+tb.closeSize && y >= closeY && y <= closeY+tb.closeSize {
			return true
		}
		offsetX += tb.tabWidth
	}
	return false
}

func (tb *TabBar) draw() {
	if !tb.GetVisible() {
		return
	}

	top, left, width, height := tb.computedBox.GetCoords()
	window := tb.window
	if window == nil {
		return
	}

	context := window.context.(*gg.Context)
	context.SetHexColor(tb.backgroundColor)
	context.DrawRectangle(float64(left), float64(top), float64(width), float64(height))
	context.Fill()

	var offsetX float64 = float64(left)
	for _, tab := range tb.tabs {
		if tab.Closed {
			continue
		}

		tabLeft := offsetX
		tabWidth := tb.tabWidth

		if tab == tb.activeTab {
			context.SetHexColor("#ffffff")
		} else {
			context.SetHexColor("#e0e0e0")
		}
		context.DrawRectangle(float64(tabLeft), float64(top), float64(tabWidth), float64(height)-2)
		context.Fill()

		context.SetHexColor("#999999")
		context.DrawLine(float64(tabLeft+tabWidth-1), float64(top+2), float64(tabLeft+tabWidth-1), float64(top+height-2))

		context.SetHexColor("#333333")
		context.SetFont(tb.font, tb.fontSize)
		title := tab.Title
		if len(title) > 18 {
			title = title[:18] + "..."
		}
		if strings.TrimSpace(title) == "" {
			title = "New Tab"
		}
		context.DrawString(title, float64(tabLeft+8), float64(top+20))

		closeX := tabLeft + tabWidth - tb.closeSize - 4
		closeY := top + (height - tb.closeSize) / 2
		context.SetHexColor("#666666")
		context.DrawLine(float64(closeX), float64(closeY), float64(closeX+tb.closeSize), float64(closeY+tb.closeSize))
		context.DrawLine(float64(closeX+tb.closeSize), float64(closeY), float64(closeX), float64(closeY+tb.closeSize))

		offsetX = tabLeft + tabWidth
	}

	if offsetX < left+width-30 {
		context.SetHexColor("#666666")
		context.SetFont(tb.font, tb.fontSize)
		context.DrawString("+", float64(offsetX+4), float64(top+20))
	}

	copyWidgetToBuffer(tb, context.Image())
}

// OnTabClick handles a click at the given coordinates. Returns the clicked tab
// and whether it was a close action.
func (tb *TabBar) OnTabClick(x, y float64) (*Tab, bool) {
	x = x - tb.computedBox.left
	y = y - tb.computedBox.top

	tab, found := tb.tabAt(x)
	if found {
		if tb.closeButtonAt(x, y, tab) {
			return tab, true
		}
		if tab != tb.activeTab {
			tb.activeTab = tab
			tb.needsRepaint = true
		}
		return tab, false
	}

	openCount := 0
	for _, t := range tb.tabs {
		if !t.Closed {
			openCount++
		}
	}
	if openCount > 0 {
		tb.activeTab = tb.tabs[openCount-1]
	}
	tb.needsRepaint = true
	return nil, false
}