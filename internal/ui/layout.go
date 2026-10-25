package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// side is where the list shows the preview of the file on the cursor
type side int

const (
	sideNone side = iota
	sideRight
	sideBelow
)

const (
	// autoRightWidth is the narrowest terminal "auto" shows the preview
	// right of the list in
	autoRightWidth = 100
	// the preview right of the list takes 42% of the width, within these
	minRightPane = 40
	maxRightPane = 64
	// minListWidth is the narrowest the list gets beside the preview
	minListWidth = 40
	// minListRows is the fewest rows the list keeps over the preview
	minListRows = 3
)

// layout is how the list view splits its rows and columns
type layout struct {
	side side
	// listW is the list's width, rows how many rows of it are shown
	listW, rows int
	// paneW and paneH are the preview's size, when there is one
	paneW, paneH int
}

// listLayout lays out the list view: the rows of the list, and the
// preview right of them or under them, as ui.preview.position says and as
// the terminal has room for.
func (m Model) listLayout() layout {
	chrome := 1 + 2 + lipgloss.Height(m.helpView()) // header; count and status; help
	if m.list.filtering || m.list.filter.Value() != "" {
		chrome++
	}
	avail := max(1, m.height-chrome)
	l := layout{listW: m.width, rows: avail}

	switch m.config.Preview.Position {
	case "none":
	case "below":
		paneH := avail / 2
		if avail-paneH < minListRows {
			paneH = avail - minListRows
		}
		if paneH >= paneInfoRows+3 {
			l.side, l.paneW, l.paneH, l.rows = sideBelow, m.width, paneH, avail-paneH
		}
	case "right":
		l.rightOf(m.width, avail)
	default: // auto
		if m.width >= autoRightWidth {
			l.rightOf(m.width, avail)
		}
	}
	return l
}

// rightOf puts the preview right of the list, when the list keeps room
func (l *layout) rightOf(width, avail int) {
	paneW := min(max(width*42/100, minRightPane), maxRightPane)
	if width-paneW-1 < minListWidth || avail < paneInfoRows+3 {
		return
	}
	l.side, l.paneW, l.paneH, l.listW = sideRight, paneW, avail, width-paneW-1
}

// previewDelay is how long the cursor rests on a file before its preview
// is read, so that holding a key down does not read every file it passes
var previewDelay = 80 * time.Millisecond

// previewDueMsg says the cursor rested; seq tells which move it was for
type previewDueMsg struct{ seq int }

// followCursor makes the preview beside the list show the file on the
// cursor, and has it read once the cursor rests there
func (m Model) followCursor(cmd tea.Cmd) (Model, tea.Cmd) {
	if m.state.current != ListView || m.listLayout().side == sideNone {
		return m, cmd
	}
	if r, ok := m.list.currentRow(now()); ok && r.kind == rowFold {
		// a fold's preview is what is in it, drawn from the list; a file is
		// read again when the cursor comes back to one
		m.detailFile = File{}
		return m, cmd
	}
	f, ok := m.list.current(now())
	// the preview shows no file yet before the first one
	if !ok || (m.detailFile.File != nil && f.TrashPath == m.detailFile.TrashPath) {
		m.resizePreview()
		return m, cmd
	}
	m.detailFile = f
	m.viewport = m.newViewportModel(f)
	m.resizePreview()
	m.previewSeq++
	seq := m.previewSeq
	due := tea.Tick(previewDelay, func(time.Time) tea.Msg { return previewDueMsg{seq: seq} })
	return m, tea.Batch(cmd, due)
}
