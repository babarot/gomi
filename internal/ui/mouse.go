package ui

import (
	"log/slog"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// The mouse only points and moves: a click puts the cursor on a row or
// goes to the next view from the header, the wheel moves the cursor in the
// list and scrolls a preview, and dragging the preview's edge resizes it.
// Restoring and deleting stay on keys, and the mouse does nothing while a
// dialog, the sort menu or the key list is open.

// wheelLines is how many lines a turn of the wheel scrolls a preview by
const wheelLines = 3

// doubleClick is how close two clicks on the preview's edge are to put it
// back at the size it starts at
const doubleClick = 400 * time.Millisecond

// drag is the edge of the preview being dragged
type drag int

const (
	dragNone   drag = iota
	dragWidth       // the left edge of the preview right of the list
	dragHeight      // the top edge of the preview under the list
)

// Layout keeps the sizes the preview was dragged to for the next launch;
// 0 is the size it starts at
type Layout interface {
	PaneWidth() int
	PaneHeight() int
	SavePaneWidth(w int) error
	SavePaneHeight(h int) error
}

// mouseOn reports whether the UI takes the mouse: in the list and the
// preview, once the terminal told where the UI is drawn
func (m Model) mouseOn() bool {
	if !m.settled || m.cursorY < 0 {
		return false
	}
	return m.state.current == ListView || m.state.current == DetailView
}

// frameTop is the terminal's row the UI's first line is drawn on. The UI
// starts on the row the prompt left the cursor on, and the terminal
// scrolls up what does not fit below it.
func (m Model) frameTop() int {
	top := m.cursorY
	if m.termHeight > 0 {
		top = min(top, m.termHeight-m.height)
	}
	return max(0, top)
}

// mouseTakesNothing reports whether what has the focus keeps the mouse
// from the list and the preview. The filter does not: nothing the mouse
// does throws away what was typed, and it keeps the focus while the mouse
// moves the cursor or scrolls, as the arrows do while typing.
func (m Model) mouseTakesNothing() bool {
	return m.state.Confirming() || m.sortMenu || m.helpOpen
}

// updateMouse handles a click, a turn of the wheel, and a drag
func (m Model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if !m.mouseOn() || m.mouseTakesNothing() {
		return m, nil
	}
	mouse := msg.Mouse()
	x, y := mouse.X, mouse.Y-m.frameTop()

	switch msg.(type) {
	case tea.MouseMotionMsg:
		if m.drag != dragNone && m.state.current == ListView {
			m.dragTo(x, y)
		}
		return m, nil
	case tea.MouseReleaseMsg:
		if m.drag == dragNone {
			return m, nil
		}
		m.drag = dragNone
		m.keepPaneSize()
		// an image is drawn again for the size the pane rests at
		return m.followCursor(nil)
	}

	switch m.state.current {
	case DetailView:
		if _, ok := msg.(tea.MouseWheelMsg); ok {
			m.scrollPreview(mouse.Button)
		}
		return m, nil
	case ListView:
		return m.listMouse(msg, x, y)
	}
	return m, nil
}

// firstRow is the UI's row the list's first row is drawn on, under the
// header and the filter's line
func (m Model) firstRow() int {
	if m.list.filtering || m.list.filter.Value() != "" {
		return 2
	}
	return 1
}

// edgeAt is the edge of the preview at column x of the UI's row y: the
// left frame of the one right of the list or the gap before it, or the top
// frame of the one under the list
func (m Model) edgeAt(l layout, x, y int) drag {
	first := m.firstRow()
	switch {
	case l.side == sideRight && y >= first && y < first+l.rows && (x == l.listW || x == l.listW+1):
		return dragWidth
	case l.side == sideBelow && y == first+l.rows+1:
		return dragHeight
	}
	return dragNone
}

// dragTo resizes the preview from where its edge was dragged to, as far as
// the list keeps room, and keeps the size it got
func (m *Model) dragTo(x, y int) {
	l := m.listLayout()
	switch m.drag {
	case dragWidth:
		m.paneW = max(1, m.width-x)
		if nl := m.listLayout(); nl.side == sideRight {
			m.paneW = nl.paneW
		}
	case dragHeight:
		// the list's rows and the pane share what the count line leaves
		m.paneH = max(1, l.rows+l.paneH-(y-m.firstRow()-1))
		if nl := m.listLayout(); nl.side == sideBelow {
			m.paneH = nl.paneH
		}
	}
	m.resizePreview()
}

// keepPaneSize keeps the sizes of the preview for the next launch
func (m Model) keepPaneSize() {
	if m.layoutStore == nil {
		return
	}
	err := m.layoutStore.SavePaneWidth(m.paneW)
	if err == nil {
		err = m.layoutStore.SavePaneHeight(m.paneH)
	}
	if err != nil {
		slog.Warn("the size of the preview cannot be kept", "error", err)
	}
}

// listMouse handles the mouse in the list view, at column x of the UI's
// row y
func (m Model) listMouse(msg tea.MouseMsg, x, y int) (tea.Model, tea.Cmd) {
	t := now()
	l := m.listLayout()
	first := m.firstRow()
	inRows := y >= first && y < first+l.rows && (l.side != sideRight || x < l.listW)
	inPane := l.side == sideRight && x > l.listW && y >= first && y < first+l.rows ||
		l.side == sideBelow && y >= first+l.rows+1 && y < first+l.rows+1+l.paneH

	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		// a drag whose release was lost, let go outside the terminal, ends
		// at the next click
		m.drag = dragNone
		if msg.Button != tea.MouseLeft {
			return m, nil
		}
		if edge := m.edgeAt(l, x, y); edge != dragNone {
			// a second click on the edge puts the preview back at the size
			// it starts at
			if now().Sub(m.edgeClick) < doubleClick {
				m.edgeClick = time.Time{}
				switch edge {
				case dragWidth:
					m.paneW = 0
				case dragHeight:
					m.paneH = 0
				}
				m.keepPaneSize()
				m.resizePreview()
				return m.followCursor(nil)
			}
			m.edgeClick = now()
			m.drag = edge
			return m, nil
		}
		if y == 0 && m.onViewLabel(x) {
			// what is typed is kept, as esc in the picker and enter in the
			// filter keep it, and the next view narrows by it
			m.list.filtering = false
			m.list.filter.Blur()
			m, cmd := m.switchView()
			return m.followCursor(cmd)
		}
		if !inRows {
			return m, nil
		}
		rows := m.list.rows(t)
		j := m.list.top + y - first
		if j >= len(rows) || !rows[j].selectable() {
			return m, nil
		}
		m.list.cursor = j
		return m.followCursor(nil)

	case tea.MouseWheelMsg:
		switch {
		case inPane:
			m.scrollPreview(msg.Button)
			return m, nil
		case inRows:
			switch msg.Button {
			case tea.MouseWheelUp:
				m.list.move(-1, t)
			case tea.MouseWheelDown:
				m.list.move(1, t)
			}
			return m.followCursor(nil)
		}
	}
	return m, nil
}

// onViewLabel reports whether column x of the header is on the view's
// name, drawn last on the header and ending a column before its edge
func (m Model) onViewLabel(x int) bool {
	header := ansi.Strip(m.headerLine())
	if !strings.HasSuffix(header, m.viewLabel()) {
		return false // cut off on a narrow terminal
	}
	end := ansi.StringWidth(header)
	return x >= end-ansi.StringWidth(m.viewLabel()) && x < end
}

// scrollPreview scrolls the preview's contents by a turn of the wheel
func (m *Model) scrollPreview(button tea.MouseButton) {
	switch button {
	case tea.MouseWheelUp:
		m.viewport.ScrollUp(wheelLines)
	case tea.MouseWheelDown:
		m.viewport.ScrollDown(wheelLines)
	}
}
