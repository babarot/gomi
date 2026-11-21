package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// fakeLayout is a Layout kept in memory
type fakeLayout struct{ w, h, saves int }

func (f *fakeLayout) PaneWidth() int  { return f.w }
func (f *fakeLayout) PaneHeight() int { return f.h }
func (f *fakeLayout) SavePaneWidth(w int) error {
	f.w = w
	f.saves++
	return nil
}
func (f *fakeLayout) SavePaneHeight(h int) error {
	f.h = h
	return nil
}

func motion(x, y int) tea.MouseMotionMsg {
	return tea.MouseMotionMsg{X: x, Y: y, Button: tea.MouseLeft}
}

func release(x, y int) tea.MouseReleaseMsg {
	return tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft}
}

// dragModel is mouseModel with the preview right of the list on a
// terminal 140 columns wide, or under it
func dragModel(t *testing.T, position string) (Model, *fakeLayout) {
	t.Helper()
	m := mouseModel(t)
	m.config.Preview.Position = position
	m.width = 140
	store := &fakeLayout{}
	m.layoutStore = store
	m, _ = m.followCursor(nil) // the preview shows the file on the cursor
	return m, store
}

func send(t *testing.T, m Model, msgs ...tea.Msg) Model {
	t.Helper()
	for _, msg := range msgs {
		updated, _ := m.Update(msg)
		m = asModel(t, updated)
	}
	return m
}

func TestDrag_LeftEdgeSetsWidthAndKeepsIt(t *testing.T) {
	withNow(t)
	m, store := dragModel(t, "right")
	l := m.listLayout()
	y := m.frameTop() + 5
	if l.paneW != 58 { // 42% of 140
		t.Fatalf("pane starts %d wide, want 58", l.paneW)
	}

	m = send(t, m, click(l.listW+1, y), motion(80, y))
	if got := m.listLayout().paneW; got != 60 {
		t.Errorf("dragged to column 80: pane %d wide, want 60", got)
	}
	if store.saves != 0 {
		t.Errorf("the width was kept before the mouse was let go")
	}
	m = send(t, m, release(80, y))
	if store.w != 60 || m.drag != dragNone {
		t.Errorf("kept %d, drag %v; want 60, none", store.w, m.drag)
	}

	// what comes after the drag is the list's again
	m = send(t, m, motion(30, y))
	if got := m.listLayout().paneW; got != 60 {
		t.Errorf("a move after letting go resized the pane to %d", got)
	}
}

func TestDrag_ListKeepsRoom(t *testing.T) {
	withNow(t)
	m, _ := dragModel(t, "right")
	l := m.listLayout()
	y := m.frameTop() + 5
	m = send(t, m, click(l.listW+1, y), motion(0, y))
	if got := m.listLayout().listW; got != minListWidth {
		t.Errorf("dragged to the left edge: list %d wide, want %d", got, minListWidth)
	}
	m = send(t, m, motion(139, y))
	if got := m.listLayout().paneW; got != minDraggedPane {
		t.Errorf("dragged to the right edge: pane %d wide, want %d", got, minDraggedPane)
	}
}

func TestDrag_TopEdgeSetsHeight(t *testing.T) {
	withNow(t)
	m, store := dragModel(t, "below")
	l := m.listLayout()
	top := m.frameTop() + m.firstRow() + l.rows + 1 // the pane's top frame
	if m.edgeAt(l, 10, top-m.frameTop()) != dragHeight {
		t.Fatalf("row %d is not the pane's top edge", top)
	}
	m = send(t, m, click(10, top), motion(10, top-3), release(10, top-3))
	if got := m.listLayout().paneH; got != l.paneH+3 {
		t.Errorf("pane %d high, want %d", got, l.paneH+3)
	}
	if store.h != l.paneH+3 {
		t.Errorf("kept %d, want %d", store.h, l.paneH+3)
	}
}

func TestDrag_SecondClickPutsSizeBack(t *testing.T) {
	withNow(t)
	m, store := dragModel(t, "right")
	m.paneW, store.w = 70, 70
	l := m.listLayout()
	y := m.frameTop() + 5
	m = send(t, m, click(l.listW+1, y), release(l.listW+1, y), click(l.listW+1, y))
	if m.paneW != 0 || store.w != 0 || m.listLayout().paneW != 58 {
		t.Errorf("width %d, kept %d, drawn %d; want 0, 0, 58", m.paneW, store.w, m.listLayout().paneW)
	}

	// two clicks far apart are two drags
	m.paneW = 70
	m = send(t, m, click(m.listLayout().listW+1, y))
	m.edgeClick = m.edgeClick.Add(-time.Second)
	m = send(t, m, release(0, y), click(m.listLayout().listW+1, y))
	if m.paneW != 70 {
		t.Errorf("a slow second click put the width back")
	}
}

func TestDrag_SizeFromLastLaunch(t *testing.T) {
	m := NewModel(nil, nil, RenderOptions{Layout: &fakeLayout{w: 50, h: 9}})
	if m.paneW != 50 || m.paneH != 9 {
		t.Errorf("width %d, height %d; want 50, 9", m.paneW, m.paneH)
	}

	// a width kept on a wider terminal gives way to the list
	var l layout
	l.rightOf(100, 30, 80)
	if l.side != sideRight || l.listW != minListWidth {
		t.Errorf("side %v, list %d wide; want right, %d", l.side, l.listW, minListWidth)
	}
}

func TestDrag_LostReleaseEndsAtNextClick(t *testing.T) {
	withNow(t)
	m, _ := dragModel(t, "right")
	l := m.listLayout()
	y := m.frameTop() + 5
	m = send(t, m, click(l.listW+1, y), motion(80, y)) // let go outside
	m = send(t, m, click(20, rowOf(t, m, "c")), motion(100, y))
	if got := m.listLayout().paneW; got != 60 {
		t.Errorf("a move after a click on a row resized the pane to %d", got)
	}
}
