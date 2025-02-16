package ui

import (
	"github.com/babarot/gomi/internal/trash"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// updateList handles key events and updates for the list view.
func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Skip key handling if we're filtering
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.listKeys.Select):
			// Handle item selection
			item := m.currentListItem()
			if item == nil {
				break
			}
			m.toggleSelection(item)
			cmds = append(cmds, m.list.NewStatusMessage("Selected: "+item.Title()))
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.listKeys.DeSelect):
			// Handle item deselection
			item := m.currentListItem()
			if item == nil {
				break
			}
			if m.isSelected(item) {
				m.toggleSelection(item)
				cmds = append(cmds, m.list.NewStatusMessage("Deselected: "+item.Title()))
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.listKeys.Space):
			// Switch to detail view
			item := m.currentListItem()
			if item == nil {
				break
			}
			m.state = DetailView
			m.currentItem = item
			cmds = append(cmds, previewCmd(item))
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.listKeys.Enter):
			// Confirm selection
			var selectedFiles []*trash.File
			if len(m.selectedItems) == 0 {
				// If no items are explicitly selected, use the current item
				if item := m.currentListItem(); item != nil {
					selectedFiles = []*trash.File{item.File()}
				}
			} else {
				// Otherwise use all selected items
				for _, item := range m.items {
					if m.isSelected(item) {
						selectedFiles = append(selectedFiles, item.File())
					}
				}
			}
			if len(selectedFiles) > 0 {
				return m, tea.Quit
			}
		}
	}

	// Handle other list updates
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}
