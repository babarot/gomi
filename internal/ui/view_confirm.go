package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
)

// confirmView renders the delete confirmation dialog
func (m Model) confirmView() string {
	var baseView string
	switch m.state.previous {
	case LIST_VIEW:
		baseView = m.list.View()
	case DETAIL_VIEW:
		baseView = m.detailView()
	}

	_, displayText, isSingleTarget := m.prepareDeleteTarget()
	dialogContent := m.formatDeleteConfirmation(displayText, isSingleTarget)

	return m.renderDialogOverBase(baseView, dialogContent)
}

// prepareDeleteTarget prepares target files information for confirmation
func (m Model) prepareDeleteTarget() ([]File, string, bool) {
	files := selectionManager.items
	if len(files) == 0 {
		// Single target on cursor line
		file := m.list.SelectedItem().(File)
		return []File{file}, "'" + file.Title() + "'", true
	}

	// Multiple files from selection manager
	quotedNames := strings.Join(
		lo.Map(files, func(f File, index int) string {
			return "'" + f.Title() + "'"
		}),
		", ",
	)

	isSingleTarget := len(files) == 1
	dialogMaxWidth := defaultWidth - 6 // border (2) + padding (2) + buffer (2)
	if len(files) > 1 && len(quotedNames) > dialogMaxWidth {
		return files, fmt.Sprintf("%d files", len(files)), true
	}

	return files, quotedNames, isSingleTarget
}

// formatDeleteConfirmation formats the confirmation dialog content
func (m Model) formatDeleteConfirmation(target string, isSingleTarget bool) string {
	var contents []string
	if isSingleTarget {
		contents = []string{
			"Are you sure you want to",
			"completely delete " + target + "?",
			"",
			"(y/n)",
		}
	} else {
		contents = []string{
			"Are you sure you want to completely delete",
			target + "?",
			"",
			"(y/n)",
		}
	}

	return m.styles.RenderDialog(
		lipgloss.JoinVertical(lipgloss.Center, contents...),
	)
}

// renderDialogOverBase renders dialog box centered over base view
func (m Model) renderDialogOverBase(baseView, dialogContent string) string {
	listLines := strings.Split(baseView, "\n")
	dialogLines := strings.Split(dialogContent, "\n")

	dialogStartLine := (len(listLines) - len(dialogLines)) / 2

	for i, line := range dialogLines {
		centeredLine := lipgloss.NewStyle().
			Width(defaultWidth).
			Align(lipgloss.Center).
			Render(line)
		listLines[dialogStartLine+i] = centeredLine
	}

	return strings.Join(listLines, "\n")
}
