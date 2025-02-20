package ui

import (
	"log/slog"

	"github.com/babarot/gomi/internal/ui/keys"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

// View returns the string representation of the current UI state
func (m Model) View() string {
	defer color.Unset()

	// Handle error state
	if m.err != nil {
		slog.Error("rendering of the view has stopped", "error", m.err)
		return m.err.Error()
	}

	// If choices are made, return empty string to exit
	if len(m.choices) > 0 {
		return ""
	}

	// Render different views based on current state
	switch m.state.current {
	case LIST_VIEW:
		listView := m.list.View()
		helpView := lipgloss.NewStyle().
			Margin(1, 2).
			Render(m.help.View(m.keyMap.AsListKeyMap()))
		return listView + "\n" + helpView

	case DETAIL_VIEW:
		detailView := m.detailView()
		helpView := lipgloss.NewStyle().
			Margin(1, 2).
			Render(m.help.View(m.keyMap.AsDetailKeyMap()))
		return detailView + "\n" + helpView

	case CONFIRM_VIEW:
		return m.confirmView()

	case QUITTING:
		return ""

	default:
		return ""
	}
}

// newViewportModel creates a new viewport model for file preview
func (m *Model) newViewportModel(file File) viewport.Model {
	viewportModel := viewport.New(
		defaultWidth,
		defaultHeight-11-1, // info pane height (11) + preview border (1)
	)
	viewportModel.KeyMap = keys.PreviewKeys

	content, err := file.Browse()
	if err != nil {
		slog.Warn("file.Browse returned error", "content", err)
		m.state.preview.available = false
	}
	viewportModel.SetContent(content)
	return viewportModel
}
