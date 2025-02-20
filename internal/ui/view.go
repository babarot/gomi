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

	var view string
	var keyMap keys.KeyMap

	// Render different views based on current state
	switch m.state.current {
	case LIST_VIEW:
		view = m.list.View()
		keyMap = m.keyMap.AsListKeyMap()
	case DETAIL_VIEW:
		view = m.detailView()
		keyMap = m.keyMap.AsDetailKeyMap()

	case CONFIRM_VIEW:
		view = m.confirmView()
		switch m.state.previous {
		case LIST_VIEW:
			keyMap = m.keyMap.AsListKeyMap()
		case DETAIL_VIEW:
			keyMap = m.keyMap.AsDetailKeyMap()
		}

	case QUITTING:
		return ""

	default:
		return ""
	}

	if view != "" {
		helpView := lipgloss.NewStyle().
			Margin(1, 2).
			Render(m.help.View(keyMap))
		view += "\n" + helpView
	}
	return view
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
