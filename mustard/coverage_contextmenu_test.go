package mustard

import (
	"image"
	"testing"

	gg "github.com/danfragoso/thdwb/gg"
)

func setupContextMenu(t *testing.T) *WindowBase {
	t.Helper()
	window := newTestWindow(t, 300, 300)
	window.EnableContextMenus()
	window.AddContextMenuEntry("First item", func() {})
	window.AddContextMenuEntry("Second item", func() {})
	// DrawContextMenu builds the overlay (with a buffer) and positions entries.
	window.cursorX, window.cursorY = 20, 20
	window.DrawContextMenu()
	return window
}

func TestDrawContextMenuPositionsAndClampsOverlay(t *testing.T) {
	window := setupContextMenu(t)
	overlay := window.contextMenu.overlay
	if overlay == nil || overlay.buffer == nil {
		t.Fatal("DrawContextMenu did not install an overlay with a buffer")
	}
	if !window.hasActiveOverlay {
		t.Fatal("expected the context menu overlay to be active")
	}

	// Reposition: cursor near the bottom-right should clamp the menu.
	window.cursorX, window.cursorY = 290, 290
	window.DrawContextMenu()
	clamped := window.contextMenu.overlay
	if clamped.left+clamped.width > float64(window.width) {
		t.Fatalf("menu left %v width %v overflows window width %d", clamped.left, clamped.width, window.width)
	}
}

func TestSelectAndDeselectContextEntry(t *testing.T) {
	window := setupContextMenu(t)
	if window.contextMenu.selectedEntry != nil {
		t.Fatal("no entry should be selected before selection")
	}
	window.SelectEntry(window.contextMenu.entries[0])
	if window.contextMenu.selectedEntry != window.contextMenu.entries[0] || window.cursor != PointerCursor {
		t.Fatal("expected the first entry selected with a pointer cursor")
	}
	window.DeselectEntries()
	if window.contextMenu.selectedEntry != nil || window.cursor != DefaultCursor {
		t.Fatal("expected the entry deselected with a default cursor")
	}
}

func TestProcessContextMenuDeselectsOutsideOverlay(t *testing.T) {
	window := setupContextMenu(t)
	window.SelectEntry(window.contextMenu.entries[0])
	// Move the cursor far outside the overlay bounds.
	window.cursorX, window.cursorY = 0.5, 0.5
	window.ProcessContextMenu()
	if window.contextMenu.selectedEntry != nil {
		t.Fatal("expected the selection to clear when the cursor leaves the menu")
	}
}

func TestMenuEntryPointIntersectsCoverage(t *testing.T) {
	entry := &menuEntry{top: 0, left: 0, width: 100, height: 20}
	if !entry.PointIntersects(50, 10) {
		t.Fatal("expected a point inside the entry to intersect")
	}
	if entry.PointIntersects(150, 10) {
		t.Fatal("expected a point outside the entry not to intersect")
	}
}

func TestPrepEntryTruncatesLongText(t *testing.T) {
	ctx := gg.NewContext(50, 20)
	// A very long string must be shortened with an ellipsis to fit the width.
	long := "this is a fairly long context menu entry"
	got := prepEntry(ctx, long, 40)
	if got == long {
		t.Fatalf("prepEntry did not truncate a long entry: %q", got)
	}
	// A short string fits and is returned unchanged.
	if got := prepEntry(ctx, "ok", 100); got != "ok" {
		t.Fatalf("prepEntry altered a short entry: %q", got)
	}
}

func TestExtractOverlayCopiesBufferAndPosition(t *testing.T) {
	buf := image.NewRGBA(image.Rect(0, 0, 30, 40))
	pos := image.Point{X: 5, Y: 6}
	ov := extractOverlay(buf, pos)
	if ov.width != 30 || ov.height != 40 || ov.left != 5 || ov.top != 6 || ov.position != pos {
		t.Fatalf("extractOverlay = %+v, want 30x40 at (5,6)", ov)
	}
}

// ---- Scroll bar ----

func TestCreateScrollBarWidgetAndSetters(t *testing.T) {
	sb := CreateScrollBarWidget(HorizontalScrollBar)
	if sb == nil || sb.widgetType != scrollbarWidget {
		t.Fatal("CreateScrollBarWidget returned an unexpected widget")
	}
	sb.SetWidth(30)
	sb.SetHeight(12)
	sb.SetTrackColor("#123456")
	sb.SetScrollerSize(200)
	sb.SetThumbSize(10)
	sb.SetThumbColor("#654321")
	sb.SetScrollerOffset(5)
	if !sb.fixedWidth || !sb.fixedHeight {
		t.Fatal("expected width/height to be marked fixed")
	}
	if sb.scrollerSize != 200 || sb.scrollerOffset != 5 || sb.thumbSize != 10 {
		t.Fatalf("scrollbar state = %+v, want size 200 offset 5 thumb 10", sb)
	}
}

func TestScrollBarDrawNoThumbWhenContentFits(t *testing.T) {
	window := newTestWindow(t, 120, 20)
	sb := &ScrollBarWidget{
		baseWidget: baseWidget{window: window, needsRepaint: true, backgroundColor: "#fff"},
		thumbColor: "#000",
	}
	sb.computedBox.SetCoords(0, 0, 100, 12)
	sb.scrollerSize = 100 // equals height, so no thumb
	sb.draw()
	if sb.Buffer() == nil {
		t.Fatal("scrollbar draw did not produce a buffer")
	}
}
