package ui

import (
	"log/slog"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"

	"github.com/babarot/gomi/internal/ui/keys"
)

// newHelpModel creates a help model with adaptive colors for light/dark terminals.
func newHelpModel() help.Model {
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#626262", Dark: "#626262"})
	h.Styles.ShortDesc = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#808080", Dark: "#4A4A4A"})
	h.Styles.ShortSeparator = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#999999", Dark: "#3C3C3C"})
	h.Styles.FullKey = h.Styles.ShortKey
	h.Styles.FullDesc = h.Styles.ShortDesc
	h.Styles.FullSeparator = h.Styles.ShortSeparator
	return h
}

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
	case ListView:
		view = m.list.View()
		keyMap = m.keyMap.AsListKeyMap()
	case DetailView:
		view = m.detailView()
		keyMap = m.keyMap.AsDetailKeyMap()

	case ConfirmView:
		view = m.confirmView()
		switch m.state.previous {
		case ListView:
			keyMap = m.keyMap.AsListKeyMap()
		case DetailView:
			keyMap = m.keyMap.AsDetailKeyMap()
		}

	case Quitting:
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
