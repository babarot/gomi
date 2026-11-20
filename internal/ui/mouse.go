package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// The mouse only points and moves: a click puts the cursor on a row or
// goes to the next view from the header, and the wheel moves the cursor in
// the list and scrolls a preview. Restoring
// and deleting stay on keys, and the mouse does nothing while a dialog,
// the sort menu, the key list or the filter has the focus.

// wheelLines is how many lines a turn of the wheel scrolls a preview by
const wheelLines = 3

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
// from the list and the preview
func (m Model) mouseTakesNothing() bool {
	return m.state.Confirming() || m.sortMenu || m.helpOpen || m.list.filtering
}

// updateMouse handles a click or a turn of the wheel
func (m Model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if !m.mouseOn() || m.mouseTakesNothing() {
		return m, nil
	}
	mouse := msg.Mouse()
	x, y := mouse.X, mouse.Y-m.frameTop()

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

// listMouse handles the mouse in the list view, at column x of the UI's
// row y
func (m Model) listMouse(msg tea.MouseMsg, x, y int) (tea.Model, tea.Cmd) {
	t := now()
	l := m.listLayout()
	first := 1 // the header
	if m.list.filtering || m.list.filter.Value() != "" {
		first++ // the filter's line
	}
	inRows := y >= first && y < first+l.rows && (l.side != sideRight || x < l.listW)
	inPane := l.side == sideRight && x > l.listW && y >= first && y < first+l.rows ||
		l.side == sideBelow && y >= first+l.rows+1 && y < first+l.rows+1+l.paneH

	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			return m, nil
		}
		if y == 0 && m.onViewLabel(x) {
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
