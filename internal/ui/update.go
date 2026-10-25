package ui

import (
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/samber/lo"

	"github.com/babarot/gomi/internal/ui/keys"
)

// Update handles all UI state updates based on incoming messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		slog.Debug("Key pressed", "key", msg.String())
		// Handle view-specific key updates
		if m.state.Confirming() {
			return m.updateConfirmView(msg)
		}
		if m.sortMenu {
			return m.updateSortMenu(msg)
		}
		if m.helpOpen {
			// the key list takes the keys until it is closed
			switch msg.String() {
			case "?", "esc", "q":
				m.helpOpen = false
			}
			return m, nil
		}
		switch m.state.current {
		case ListView:
			next, cmd := m.updateListView(msg)
			if nm, ok := next.(Model); ok {
				return nm.followCursor(cmd)
			}
			return next, cmd
		case DetailView:
			next, cmd := m.updateDetailView(msg)
			if nm, ok := next.(Model); ok && nm.state.current == ListView {
				// back in the list, the preview beside it takes its size again
				return nm.followCursor(cmd)
			}
			return next, cmd
		}

	case tea.BackgroundColorMsg:
		m.setBackground(msg.IsDark())
		return m, nil

	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
			// the help is indented by two columns
			m.help.SetWidth(max(msg.Width-2, 0))
		}
		if msg.Height > 0 {
			m.termHeight = msg.Height
			m.height = uiHeight(m.config.Height, m.termHeight, m.cursorY)
		}
		m.resizePreview()
		return m.followCursor(tea.Batch(cmds...))

	case tea.CursorPositionMsg:
		// it may come after the wait; asked before anything was drawn, it
		// still tells where the prompt left the cursor
		m.cursorY = msg.Y
		m.settled = true
		if m.termHeight > 0 {
			m.height = uiHeight(m.config.Height, m.termHeight, m.cursorY)
			m.resizePreview()
		}
		return m.followCursor(nil)

	case previewDueMsg:
		// the cursor rested on the file the preview shows
		if msg.seq == m.previewSeq && m.state.current == ListView && m.detailFile.File != nil {
			return m, tea.Batch(m.loadDetail(m.detailFile)...)
		}
		return m, nil

	case cursorWaitedMsg:
		m.settled = true
		return m, nil

	case ShowDetailMsg:
		m.state.SetView(DetailView)
		m.detailFile = msg.file
		m.viewport = m.newViewportModel(msg.file)
		m.resizePreview()
		cmds = append(cmds, m.loadDetail(msg.file)...)
		return m, tea.Batch(cmds...)

	case previewLoadedMsg:
		delete(m.reading, previewKey(msg.path))
		m.previews[msg.path] = msg.previewResult
		// the result may come after the preview moved to another file
		if m.detailFile.File != nil && m.detailFile.TrashPath == msg.path {
			m.viewport.SetContent(msg.content)
		}
		return m, nil

	case statusClearMsg:
		if msg.seq == m.statusSeq {
			m.status = ""
		}
		return m, nil

	case originsCheckedMsg:
		maps.Copy(m.origins, msg.origins)
		return m, nil

	case sizeLoadedMsg:
		// a size may reorder the picker's rows by size; the cursor stays on
		// the file it was on
		on := m.list.cursorKey(now())
		delete(m.reading, sizeKey(msg.path))
		m.sizes[msg.path] = msg.size
		if msg.err != nil {
			m.sizes[msg.path] = -1
		}
		m.list.find(on, now())
		if msg.queued {
			m.sizing--
			return m.followCursor(tea.Batch(m.nextSizes()...))
		}
		return m.followCursor(nil)

	case FileListLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.list.setFiles(msg.files, now())
		if msg.deleted > 0 {
			cmds = append(cmds, m.setStatus(fmt.Sprintf("Deleted %d %s permanently", msg.deleted, plural(msg.deleted, "file"))))
		}
		cmds = append(cmds, m.queueSizes(m.list.files)...)
		cmds = append(cmds, checkOriginsCmd(m.uncheckedOrigins(msg.files)))
		return m.followCursor(tea.Batch(cmds...))

	case errorMsg:
		m.state.SetView(Quitting)
		m.err = msg
		return m, tea.Quit
	}

	// the filter takes what else comes, such as the cursor's blink or a
	// paste; a paste narrows the list as typing does
	if m.list.filtering {
		before := m.list.filter.Value()
		var cmd tea.Cmd
		m.list.filter, cmd = m.list.filter.Update(msg)
		cmds = append(cmds, cmd)
		if m.list.filter.Value() != before {
			m.list.top = 0
			m.list.toStart(now())
			return m.followCursor(tea.Batch(cmds...))
		}
	}
	return m, tea.Batch(cmds...)
}

// updateListView handles updates specific to the list view
func (m Model) updateListView(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.list.filtering {
		return m.updateFilter(msg)
	}
	t := now()
	file, onFile := m.list.current(t)
	r, onRow := m.list.currentRow(t)
	onFold := onRow && r.kind == rowFold

	switch {
	case key.Matches(msg, m.keyMap.Common.Quit):
		m.state.SetView(Quitting)
		return m, tea.Quit

	case m.keyMap.List.Delete != nil && key.Matches(msg, *m.keyMap.List.Delete):
		files := m.targets(t)
		if len(files) == 0 {
			slog.Warn("cannot get file on cursor")
			return m, nil
		}
		m.openDeleteConfirm(files)
		return m, nil

	case key.Matches(msg, m.keyMap.List.Select):
		switch {
		case onFold:
			// a fold selects its files, or drops them when all are selected
			all := true
			for _, f := range r.files {
				all = all && m.selection.Contains(f)
			}
			for _, f := range r.files {
				if all {
					m.selection.Remove(f)
				} else {
					m.selection.Add(f)
				}
			}
			m.list.move(1, t)
		case onFile:
			if m.selection.Contains(file) {
				m.selection.Remove(file)
			} else {
				m.selection.Add(file)
			}
			m.list.move(1, t)
		}
		return m, nil

	case key.Matches(msg, m.keyMap.List.View):
		m.list.setView(m.nextView(), t)
		if m.list.view == viewPicker {
			// the picker starts with the focus in the filter
			m.list.filtering = true
			return m, m.list.filter.Focus()
		}
		return m, nil

	case m.list.view == viewPicker && key.Matches(msg, m.keyMap.List.Sort):
		m.sortMenu = true
		m.sortSel = m.list.sort
		return m, nil

	case m.list.folds() && key.Matches(msg, m.keyMap.List.Open):
		m.list.toggle(true, t)
		return m, nil

	case m.list.folds() && key.Matches(msg, m.keyMap.List.Close):
		m.list.toggle(false, t)
		return m, nil

	case key.Matches(msg, m.keyMap.List.DeSelect):
		if onFile {
			m.selection.Remove(file)
			m.list.move(-1, t)
		}
		return m, nil

	case key.Matches(msg, m.keyMap.List.Space):
		switch {
		case onFold:
			m.list.toggle(!r.open, t)
		case onFile:
			return m, func() tea.Msg { return newShowDetailMsg(file) }
		}
		return m, nil

	case key.Matches(msg, m.keyMap.List.Esc):
		switch {
		case m.list.filter.Value() != "":
			m.list.filter.SetValue("")
			m.list.fixCursor(m.list.rows(t))
		case len(m.selection.items) > 0:
			m.selection = &SelectionManager{items: []File{}}
		}
		return m, nil

	case key.Matches(msg, m.keyMap.List.Enter):
		m.choices = m.targets(t)
		slog.Debug("key input: enter",
			"files",
			strings.Join(lo.Map(m.choices, func(file File, _ int) string {
				return file.OriginalPath
			}), "\n"),
		)
		return m, tea.Quit

	case key.Matches(msg, m.keyMap.Common.Help):
		m.helpOpen = true
		return m, nil

	case key.Matches(msg, m.keyMap.List.Scope):
		if m.list.cwd == "" {
			return m, nil
		}
		m.list.setScope(!m.list.scoped, t)
		text := "Every file in the trash"
		if m.list.scoped {
			text = "Files deleted in " + m.displayPath(m.list.cwd)
		}
		return m, m.setStatus(text)

	case key.Matches(msg, keys.DefaultKeyMapListFilter):
		m.list.filtering = true
		return m, m.list.filter.Focus()

	case key.Matches(msg, keys.DefaultKeyMapListCursorUp):
		m.list.move(-1, t)
	case key.Matches(msg, keys.DefaultKeyMapListCursorDown):
		m.list.move(1, t)
	case key.Matches(msg, keys.DefaultKeyMapListPrevPage):
		m.list.move(-m.pageSize(), t)
	case key.Matches(msg, keys.DefaultKeyMapListNextPage):
		m.list.move(m.pageSize(), t)
	case key.Matches(msg, keys.DefaultKeyMapListGoToStart):
		m.list.toStart(t)
	case key.Matches(msg, keys.DefaultKeyMapListGoToEnd):
		m.list.toEnd(t)
	}
	return m, nil
}

// updateFilter handles keys while the filter has the focus: enter keeps
// what was typed, esc drops it, and the arrows still move the cursor. In
// the picker, enter restores and tab selects, as they do in the list, and
// esc hands the focus to the list, keeping what was typed, so that keys
// such as v work there.
func (m Model) updateFilter(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	t := now()
	if m.list.view == viewPicker {
		switch {
		case msg.String() == "esc":
			m.list.filtering = false
			m.list.filter.Blur()
			return m, nil
		case key.Matches(msg, m.keyMap.List.Enter), key.Matches(msg, m.keyMap.List.Select), key.Matches(msg, m.keyMap.List.DeSelect):
			m.list.filtering = false
			next, cmd := m.updateListView(msg)
			if nm, ok := next.(Model); ok {
				nm.list.filtering = nm.state.current == ListView
				return nm, cmd
			}
			return next, cmd
		}
	}
	switch msg.String() {
	case "ctrl+c":
		m.state.SetView(Quitting)
		return m, tea.Quit
	case "enter":
		m.list.filtering = false
		m.list.filter.Blur()
		return m, nil
	case "esc":
		m.list.filtering = false
		m.list.filter.Blur()
		m.list.filter.SetValue("")
		m.list.fixCursor(m.list.rows(t))
		return m, nil
	case "up":
		m.list.move(-1, t)
		return m, nil
	case "down":
		m.list.move(1, t)
		return m, nil
	}
	before := m.list.filter.Value()
	var cmd tea.Cmd
	m.list.filter, cmd = m.list.filter.Update(msg)
	if m.list.filter.Value() != before {
		m.list.top = 0
		m.list.toStart(t)
	}
	return m, cmd
}

// pageSize is how many files a page moves by: the rows the list shows
func (m Model) pageSize() int {
	return max(1, m.list.shownN)
}

// updateDetailView handles updates specific to the detail view
func (m Model) updateDetailView(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Common.Quit):
		m.state.SetView(Quitting)
		return m, tea.Quit

	case m.keyMap.Detail.Delete != nil && key.Matches(msg, *m.keyMap.Detail.Delete):
		files := slices.Clone(m.selection.items)
		if len(files) == 0 {
			files = []File{m.detailFile}
		}
		m.openDeleteConfirm(files)
		return m, nil

	case key.Matches(msg,
		m.keyMap.Detail.PreviewUp,
		m.keyMap.Detail.PreviewDown,
		m.keyMap.Detail.HalfPageUp,
		m.keyMap.Detail.HalfPageDown):
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case key.Matches(msg, m.keyMap.Detail.GotoTop):
		m.viewport.GotoTop()
		return m, nil

	case key.Matches(msg, m.keyMap.Detail.GotoBottom):
		m.viewport.GotoBottom()
		return m, nil

	case key.Matches(msg, m.keyMap.Detail.Prev, m.keyMap.Detail.Next):
		n := 1
		if key.Matches(msg, m.keyMap.Detail.Prev) {
			n = -1
		}
		m.list.moveFile(n, now())
		file, ok := m.list.current(now())
		if ok {
			return m, func() tea.Msg { return newShowDetailMsg(file) }
		}
		return m, nil

	case key.Matches(msg, m.keyMap.Detail.Esc):
		m.state.SetView(ListView)
		return m, nil

	case key.Matches(msg, m.keyMap.Detail.Space):
		m.state.SetView(ListView)
		return m, nil

	case key.Matches(msg, m.keyMap.Common.Help):
		m.helpOpen = true
		return m, nil

	}

	return m, nil
}

// targets are the files enter restores and D deletes: the selected files
// wherever the cursor is, else the files of the fold the cursor is on that
// the filter keeps, else the file the cursor is on.
func (m Model) targets(t time.Time) []File {
	if len(m.selection.items) > 0 {
		return slices.Clone(m.selection.items)
	}
	r, ok := m.list.currentRow(t)
	switch {
	case !ok:
		return nil
	case r.kind == rowFold:
		return slices.Clone(r.files)
	}
	return []File{r.file}
}

// nextView is the view v goes to: the one after the current in ui.views
func (m Model) nextView() string {
	views := m.views()
	i := slices.Index(views, m.list.view)
	return views[(i+1)%len(views)]
}

// views are the views v goes through, as ui.views says
func (m Model) views() []string {
	if len(m.config.Views) == 0 {
		return []string{viewRecent, viewPlaces, viewBatches, viewPicker}
	}
	return m.config.Views
}

// openDeleteConfirm opens the confirmation for deleting files. More than
// one file asks to type YES.
func (m *Model) openDeleteConfirm(files []File) {
	state := ConfirmStateYesNo
	if len(files) > 1 {
		state = ConfirmStateTypeYES
	}
	m.state.OpenConfirm(state, files)
}

// deleteConfirmed deletes the files the confirmation was opened for and
// drops them from the selection.
func (m *Model) deleteConfirmed() tea.Cmd {
	files := m.state.confirmation.files
	for _, file := range files {
		m.selection.Remove(file)
	}
	if len(files) == 0 {
		return nil
	}
	return deletePermanentlyCmd(m, files...)
}

// updateConfirmView handles updates specific to the confirmation dialog
func (m Model) updateConfirmView(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Branch processing based on ConfirmState
	if m.state.confirmation.state == ConfirmStateTypeYES {
		return m.handleTypeYesConfirmation(msg)
	} else {
		return m.handleYesNoConfirmation(msg)
	}
}

// Handle YES typing confirmation mode
func (m Model) handleTypeYesConfirmation(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "backspace":
		m.state.BackspaceYesInput()
	case "enter":
		if m.state.IsYesComplete() {
			// Execute deletion if complete YES has been entered
			m.state.CloseConfirm()
			return m, m.deleteConfirmed()
		}
	case "esc": // Cancel
		m.state.CloseConfirm()
		return m, nil
	default:
		// Character input processing
		char := msg.String()
		if len(char) == 1 {
			m.state.UpdateYesInput(char)
		}
	}
	return m, nil
}

// Handle standard Yes/No confirmation mode
func (m Model) handleYesNoConfirmation(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Confirm.Yes):
		m.state.CloseConfirm()
		return m, m.deleteConfirmed()

	case key.Matches(msg, m.keyMap.Confirm.No):
		m.state.CloseConfirm()
		return m, nil

	case key.Matches(msg, m.keyMap.Common.Quit):
		m.state.SetView(Quitting)
		return m, tea.Quit
	}

	return m, nil
}

// updateSortMenu handles keys while the sort menu is open: the arrows or
// j k move, a number or enter chooses, esc, s or q closes it
func (m Model) updateSortMenu(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	t := now()
	choose := func(i int) {
		m.list.sort = i
		m.sortMenu = false
		m.list.toStart(t)
	}
	switch s := msg.String(); {
	case s == "up" || s == "k":
		m.sortSel = (m.sortSel + len(sortOrders) - 1) % len(sortOrders)
	case s == "down" || s == "j":
		m.sortSel = (m.sortSel + 1) % len(sortOrders)
	case s == "enter":
		choose(m.sortSel)
	case len(s) == 1 && s[0] >= '1' && int(s[0]-'1') < len(sortOrders):
		choose(int(s[0] - '1'))
	case s == "esc" || s == "s" || s == "q":
		m.sortMenu = false
	}
	return m.followCursor(nil)
}

// statusFor is how long the status line shows what was just done
var statusFor = 3 * time.Second

// statusClearMsg clears the status line, when no newer status came
type statusClearMsg struct{ seq int }

// setStatus shows text on the status line for a while
func (m *Model) setStatus(text string) tea.Cmd {
	m.status = text
	m.statusSeq++
	seq := m.statusSeq
	return tea.Tick(statusFor, func(time.Time) tea.Msg { return statusClearMsg{seq: seq} })
}
