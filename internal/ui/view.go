package ui

import (
	"log/slog"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/fatih/color"

	"github.com/babarot/gomi/internal/ui/keys"
	"github.com/babarot/gomi/internal/ui/styles"
)

// helpStyles returns the help styles drawn with p, for a dark or light
// terminal background.
func helpStyles(p styles.Palette, isDark bool) help.Styles {
	s := help.DefaultStyles(isDark)
	s.ShortKey = lipgloss.NewStyle().Foreground(p.HelpKey.Of(isDark))
	s.ShortDesc = lipgloss.NewStyle().Foreground(p.HelpDesc.Of(isDark))
	s.ShortSeparator = lipgloss.NewStyle().Foreground(p.HelpSeparator.Of(isDark))
	s.Ellipsis = lipgloss.NewStyle().Foreground(p.HelpEllipsis.Of(isDark))
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

	// nothing is drawn until the terminal tells where the cursor is
	if !m.settled {
		return ""
	}

	var view string
	switch m.state.current {
	case ListView:
		view = m.listView()
	case DetailView:
		view = m.detailView() + "\n\n" + m.helpView()
	default:
		return ""
	}

	if m.state.Confirming() {
		view = m.renderDialogOverBase(view, m.confirmDialog())
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

	if r, ok := m.previews[file.TrashPath]; ok {
		viewportModel.SetContent(r.content)
	}
	return viewportModel
}
