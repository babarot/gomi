package ui

import (
	"fmt"
	"log/slog"
	"strings"

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
		return m.list.View()

	case DETAIL_VIEW:
		detailView := m.detailView()
		helpView := lipgloss.NewStyle().
			Margin(1, 2).
			Render(m.help.View(m.detailKeys))
		return detailView + "\n" + helpView

	case CONFIRM_VIEW:
		return m.confirmView()

	case QUITTING:
		return ""

	default:
		return ""
	}
}

// renderHeader renders the header section with the current file title
func (m Model) renderHeader() string {
	return m.styles.RenderDetailTitle(
		m.detailFile.Title(),
		defaultWidth,
		m.detailFile.isSelected(),
	)
}

// renderFooter renders the footer section
func (m Model) renderFooter() string {
	return m.styles.Dialog.Separator.Render(
		strings.Repeat("─", defaultWidth),
	)
}

// previewHeader renders the header of the preview section
func (m Model) previewHeader() string {
	return m.styles.RenderPreviewFrame(
		m.detailFile.Size(),
		true,
		defaultWidth,
	)
}

// previewFooter renders the footer of the preview section
func (m Model) previewFooter() string {
	if m.state.preview.available {
		return m.styles.RenderPreviewFrame(
			"",
			false,
			defaultWidth,
		)
	}
	return m.styles.RenderPreviewFrame(
		fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100),
		false,
		defaultWidth,
	)
}

// newViewportModel creates a new viewport model for file preview
func (m *Model) newViewportModel(file File) viewport.Model {
	viewportModel := viewport.New(
		defaultWidth,
		15-lipgloss.Height(m.previewHeader()),
	)
	viewportModel.KeyMap = keys.PreviewKeys

	content, err := file.Browse()
	if err != nil {
		slog.Warn("file.Browse returned error", "content", err)
		m.state.preview.available = true
	}
	viewportModel.SetContent(content)
	return viewportModel
}
