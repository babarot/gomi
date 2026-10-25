package ui

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

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
		switch m.state.current {
		case ListView:
			return m.updateListView(msg)
		case DetailView:
			return m.updateDetailView(msg)
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
		return m, tea.Batch(cmds...)

	case tea.CursorPositionMsg:
		// it may come after the wait; asked before anything was drawn, it
		// still tells where the prompt left the cursor
		m.cursorY = msg.Y
		m.settled = true
		if m.termHeight > 0 {
			m.height = uiHeight(m.config.Height, m.termHeight, m.cursorY)
			m.resizePreview()
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
		// the result may come after the view moved to another file
		if m.state.current == DetailView && m.detailFile.TrashPath == msg.path {
			m.viewport.SetContent(msg.content)
		}
		return m, nil

	case sizeLoadedMsg:
		delete(m.reading, sizeKey(msg.path))
		m.sizes[msg.path] = msg.size
		if msg.err != nil {
			m.sizes[msg.path] = -1
		}
		if msg.queued {
			m.sizing--
			return m, tea.Batch(m.nextSizes()...)
		}
		return m, nil

	case FileListLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.list.setFiles(msg.files, now())
		if msg.deleted > 0 {
			m.status = fmt.Sprintf("Deleted %d %s permanently", msg.deleted, plural(msg.deleted, "file"))
		}
		cmds = append(cmds, m.queueSizes(m.list.files)...)
		return m, tea.Batch(cmds...)

	case errorMsg:
		m.state.SetView(Quitting)
		m.err = msg
		return m, tea.Quit
	}

	// the filter takes what else comes, such as the cursor's blink
	if m.list.filtering {
		var cmd tea.Cmd
		m.list.filter, cmd = m.list.filter.Update(msg)
		cmds = append(cmds, cmd)
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

	switch {
	case key.Matches(msg, m.keyMap.Common.Quit):
		m.state.SetView(Quitting)
		return m, tea.Quit

	case m.keyMap.List.Delete != nil && key.Matches(msg, *m.keyMap.List.Delete):
		if !onFile && len(m.selection.items) == 0 {
			slog.Warn("cannot get file on cursor")
			return m, nil
		}
		m.openDeleteConfirm(file)
		return m, nil

	case key.Matches(msg, m.keyMap.List.Select):
		if onFile {
			if m.selection.Contains(file) {
				m.selection.Remove(file)
			} else {
				m.selection.Add(file)
			}
			m.list.move(1, t)
		}
		return m, nil

	case key.Matches(msg, m.keyMap.List.DeSelect):
		if onFile {
			m.selection.Remove(file)
			m.list.move(-1, t)
		}
		return m, nil

	case key.Matches(msg, m.keyMap.List.Space):
		if onFile {
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
		files := m.selection.items
		if len(files) == 0 && onFile {
			files = []File{file}
		}
		m.choices = files
		slog.Debug("key input: enter",
			"files",
			strings.Join(lo.Map(m.choices, func(file File, _ int) string {
				return file.OriginalPath
			}), "\n"),
		)
		return m, tea.Quit

	case key.Matches(msg, m.keyMap.Common.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil

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
// what was typed, esc drops it, and the arrows still move the cursor.
func (m Model) updateFilter(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	t := now()
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
		m.openDeleteConfirm(m.detailFile)
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
		m.list.move(n, now())
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
		m.help.ShowAll = !m.help.ShowAll
		// the full help takes rows from the pane
		m.resizePreview()
		return m, nil

	}

	return m, nil
}

// openDeleteConfirm opens the confirmation for deleting the selected files,
// or the file on the cursor when none is selected. More than one file asks
// to type YES.
func (m *Model) openDeleteConfirm(cursor File) {
	files := slices.Clone(m.selection.items)
	if len(files) == 0 {
		files = []File{cursor}
	}
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
