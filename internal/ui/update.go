package ui

import (
	"log/slog"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/samber/lo"
)

// Update handles all UI state updates based on incoming messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		slog.Debug("Key pressed", "key", msg.String())
		// Handle view-specific key updates
		switch m.state.current {
		case ListView:
			return m.updateListView(msg)
		case DetailView:
			return m.updateDetailView(msg)
		case ConfirmView:
			return m.updateConfirmView(msg)
		}

	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, tea.Batch(cmds...)

	case FileListUpdatedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.list.SetItems(msg.files)
		return m, tea.Batch(cmds...)

	case ShowDetailMsg:
		m.state.SetView(DetailView)
		m.detailFile = msg.file
		m.state.preview.available = true
		m.viewport = m.newViewportModel(msg.file)
		return m, tea.Batch(cmds...)

	case FileListLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.list.SetItems(msg.files)
		return m, tea.Batch(cmds...)

	case errorMsg:
		m.state.SetView(Quitting)
		m.err = msg
		return m, tea.Quit
	}

	// update default list always
	slog.Debug("update default list")
	var listCmd tea.Cmd
	m.list, listCmd = m.list.Update(msg)
	if listCmd != nil {
		cmds = append(cmds, listCmd)
	}

	return m, tea.Batch(cmds...)
}

// updateListView handles updates specific to the list view
func (m Model) updateListView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Common.Quit):
		m.state.SetView(Quitting)
		return m, tea.Quit

	case m.keyMap.List.Delete != nil && key.Matches(msg, *m.keyMap.List.Delete):
		if m.list.FilterState() != list.Filtering {
			files := selectionManager.items
			switch len(files) {
			case 0:
				file, ok := m.list.SelectedItem().(File)
				if !ok {
					slog.Warn("cannot get file on cursor")
					return m, nil
				}
				m.state.SetConfirmState(ConfirmStateYesNo, []File{file})
			case 1:
				m.state.SetConfirmState(ConfirmStateYesNo, files)
			default:
				m.state.SetConfirmState(ConfirmStateTypeYES, files)
			}
			m.state.SetView(ConfirmView)
		}
		return m, nil

	case key.Matches(msg, m.keyMap.List.Select):
		if m.list.FilterState() != list.Filtering {
			item, ok := m.list.SelectedItem().(File)
			if !ok {
				return m, nil
			}
			if item.isSelected() {
				selectionManager.Remove(item)
			} else {
				selectionManager.Add(item)
			}
			m.list.CursorDown()
		}
		return m, nil

	case key.Matches(msg, m.keyMap.List.DeSelect):
		if m.list.FilterState() != list.Filtering {
			item, ok := m.list.SelectedItem().(File)
			if !ok {
				return m, nil
			}
			if item.isSelected() {
				selectionManager.Remove(item)
			}
			m.list.CursorUp()
		}
		return m, nil

	case key.Matches(msg, m.keyMap.List.Space):
		if m.list.FilterState() != list.Filtering {
			file, ok := m.list.SelectedItem().(File)
			if ok {
				return m, func() tea.Msg { return newShowDetailMsg(file) }
			}
		}
		return m, nil

	case key.Matches(msg, m.keyMap.List.Esc):
		if m.list.FilterState() != list.Filtering {
			if len(selectionManager.items) > 0 {
				selectionManager = &SelectionManager{items: []File{}}
				return m, nil
			}
		}
		// DO NOT RETURN HERE
		// to allow to update default list navigation

	case key.Matches(msg, m.keyMap.List.Enter):
		if m.list.FilterState() != list.Filtering {
			files := selectionManager.items
			if len(files) == 0 {
				file, ok := m.list.SelectedItem().(File)
				if ok {
					m.choices = append(m.choices, file)
				}
			} else {
				m.choices = files
			}
			slog.Debug("key input: enter",
				"files",
				strings.Join(lo.Map(m.choices, func(file File, _ int) string {
					return file.OriginalPath
				}), "\n"),
			)
			return m, tea.Quit
		}
		// DO NOT RETURN HERE
		// to allow to update default list navigation

	case key.Matches(msg, m.keyMap.Common.Help):
		// do not use default help of list model
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	}

	// Handle default list navigation
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// updateDetailView handles updates specific to the detail view
func (m Model) updateDetailView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Common.Quit):
		m.state.SetView(Quitting)
		return m, tea.Quit

	case m.keyMap.Detail.Delete != nil && key.Matches(msg, *m.keyMap.Detail.Delete):
		if m.list.FilterState() != list.Filtering {
			m.state.SetView(ConfirmView)
			slog.Debug("pressed delete key", "state", ConfirmView)
		}
		return m, nil

	case key.Matches(msg, m.keyMap.Detail.AtSign):
		m.state.ToggleDateFormat()
		m.state.ToggleOriginPath()
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

	case key.Matches(msg, m.keyMap.Detail.Prev):
		m.list.CursorUp()
		file, ok := m.list.SelectedItem().(File)
		if ok {
			return m, func() tea.Msg { return newShowDetailMsg(file) }
		}
		return m, nil

	case key.Matches(msg, m.keyMap.Detail.Next):
		m.list.CursorDown()
		file, ok := m.list.SelectedItem().(File)
		if ok {
			return m, func() tea.Msg { return newShowDetailMsg(file) }
		}
		return m, nil

	case key.Matches(msg, m.keyMap.Detail.Esc):
		m.state.SetView(ListView)
		return m, nil

	case key.Matches(msg, m.keyMap.Detail.Space):
		m.state.detail.showOrigin = true
		m.state.detail.dateFormat = DateFormatRelative
		m.state.SetView(ListView)
		return m, nil

	case key.Matches(msg, m.keyMap.Common.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil

	}

	return m, nil
}

// updateConfirmView handles updates specific to the confirmation dialog
func (m Model) updateConfirmView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	files := selectionManager.items

	// Branch processing based on ConfirmState
	if m.state.confirmation.state == ConfirmStateTypeYES {
		return m.handleTypeYesConfirmation(msg, files)
	} else {
		return m.handleYesNoConfirmation(msg, files)
	}
}

// Handle YES typing confirmation mode
func (m Model) handleTypeYesConfirmation(msg tea.KeyMsg, files []File) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyBackspace:
		m.state.BackspaceYesInput()
	case tea.KeyEnter:
		if m.state.IsYesComplete() {
			// Execute deletion if complete YES has been entered
			m.state.SetView(m.state.previous)
			if len(files) > 0 {
				return m, deletePermanentlyCmd(&m, files...)
			}
			return m, nil
		}
	case tea.KeyEsc: // Cancel
		m.state.SetView(m.state.previous)
		// Forcibly restructure a current window by sending WindowSizeMsg
		return m, func() tea.Msg {
			return tea.WindowSizeMsg{
				Width: m.list.Width(),
			}
		}
	default:
		// Character input processing
		char := msg.String()
		if len(char) == 1 {
			m.state.UpdateYesInput(char)

			// You can automatically confirm when YES is completed
			if m.state.IsYesComplete() {
				// Uncomment to enable auto-confirmation
				//
				// m.state.SetView(m.state.previous)
				// return m, deletePermanentlyCmd(&m, files...)
				//
				// This branch is intentionally empty
				slog.Debug("YES completed but auto-confirmation is disabled")
			}
		}
	}
	return m, nil
}

// Handle standard Yes/No confirmation mode
func (m Model) handleYesNoConfirmation(msg tea.KeyMsg, files []File) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Confirm.Yes):
		// Immediately delete for single file
		m.state.SetView(m.state.previous)
		if len(files) == 1 {
			return m, deletePermanentlyCmd(&m, files[0])
		}
		if file, ok := m.list.SelectedItem().(File); ok {
			return m, deletePermanentlyCmd(&m, file)
		}
		return m, nil

	case key.Matches(msg, m.keyMap.Confirm.No):
		m.state.SetView(m.state.previous)
		// Forcibly restructure a current window by sending WindowSizeMsg
		return m, func() tea.Msg {
			return tea.WindowSizeMsg{
				Width: m.list.Width(),
			}
		}

	case key.Matches(msg, m.keyMap.Common.Quit):
		m.state.SetView(Quitting)
		return m, tea.Quit
	}

	return m, nil
}
