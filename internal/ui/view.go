package ui

import (
	"log/slog"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/fatih/color"

	"github.com/babarot/gomi/internal/ui/keys"
)

// helpStyles returns the help styles for a dark or light terminal background.
func helpStyles(isDark bool) help.Styles {
	lightDark := lipgloss.LightDark(isDark)
	s := help.DefaultStyles(isDark)
	s.ShortKey = lipgloss.NewStyle().
		Foreground(lightDark(lipgloss.Color("#626262"), lipgloss.Color("#626262")))
	s.ShortDesc = lipgloss.NewStyle().
		Foreground(lightDark(lipgloss.Color("#808080"), lipgloss.Color("#4A4A4A")))
	s.ShortSeparator = lipgloss.NewStyle().
		Foreground(lightDark(lipgloss.Color("#999999"), lipgloss.Color("#3C3C3C")))
	// bubbles v2 made the default a little lighter; keep v1's
	s.Ellipsis = lipgloss.NewStyle().
		Foreground(lightDark(lipgloss.Color("#DDDADA"), lipgloss.Color("#3C3C3C")))
	s.FullKey = s.ShortKey
	s.FullDesc = s.ShortDesc
	s.FullSeparator = s.ShortSeparator
	return s
}

// View returns the current UI state
func (m Model) View() tea.View {
	return tea.NewView(m.view())
}

func (m Model) view() string {
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

	case Quitting:
		return ""

	default:
		return ""
	}

	if m.state.Confirming() {
		view = m.renderDialogOverBase(view, m.confirmDialog())
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
		viewport.WithWidth(defaultWidth),
		viewport.WithHeight(defaultHeight-11-1), // info pane height (11) + preview border (1)
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
