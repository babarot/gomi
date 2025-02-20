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
		case LIST_VIEW:
			return m.updateListView(msg)
		case DETAIL_VIEW:
			return m.updateDetailView(msg)
		case CONFIRM_VIEW:
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
		m.state.SetView(DETAIL_VIEW)
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
		m.state.SetView(QUITTING)
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
		m.state.SetView(QUITTING)
		return m, tea.Quit

	case m.keyMap.List.Delete != nil && key.Matches(msg, *m.keyMap.List.Delete):
		if m.list.FilterState() != list.Filtering {
			m.state.SetView(CONFIRM_VIEW)
			slog.Debug("pressed delete key", "state", CONFIRM_VIEW)
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
			selectionManager = &SelectionManager{items: []File{}}
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
		m.state.SetView(QUITTING)
		return m, tea.Quit

	case m.keyMap.Detail.Delete != nil && key.Matches(msg, *m.keyMap.Detail.Delete):
		if m.list.FilterState() != list.Filtering {
			m.state.SetView(CONFIRM_VIEW)
			slog.Debug("pressed delete key", "state", CONFIRM_VIEW)
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
		m.state.SetView(LIST_VIEW)
		return m, nil

	case key.Matches(msg, m.keyMap.Detail.Space):
		m.state.detail.showOrigin = true
		m.state.detail.dateFormat = DateFormatRelative
		m.state.SetView(LIST_VIEW)
		return m, nil

	case key.Matches(msg, m.keyMap.Common.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil

	}

	return m, nil
}

// updateConfirmView handles updates specific to the confirmation dialog
func (m Model) updateConfirmView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Confirm.Yes):
		m.state.SetView(m.state.previous)
		files := selectionManager.items
		if len(files) > 0 {
			return m, deletePermanentlyCmd(&m, files...)
		}
		if file, ok := m.list.SelectedItem().(File); ok {
			return m, deletePermanentlyCmd(&m, file)
		}
		return m, nil

	case key.Matches(msg, m.keyMap.Confirm.No):
		m.state.SetView(m.state.previous)
		return m, nil

	case key.Matches(msg, m.keyMap.Common.Quit):
		m.state.SetView(QUITTING)
		return m, tea.Quit
	}

	return m, nil
}
