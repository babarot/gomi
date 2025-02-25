package ui

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
)

// confirmView renders the delete confirmation dialog
func (m Model) confirmView() string {
	var baseView string
	switch m.state.previous {
	case ListView:
		baseView = m.list.View()
	case DetailView:
		baseView = m.detailView()
	}

	files := m.state.confirmation.files
	slog.Debug("selected files on confirm view", "files", files)

	var dialogContent string
	if m.state.confirmation.state == ConfirmStateTypeYES {
		dialogContent = m.formatConfirmationTypeYES(files)
	} else {
		dialogContent = m.formatConfirmation(files)
	}

	return m.renderDialogOverBase(baseView, dialogContent)
}

// formatConfirmation formats the confirmation dialog content
func (m Model) formatConfirmation(files []File) string {
	// this function expects that a length of files is always 1
	quotedNames := strings.Join(
		lo.Map(files, func(f File, index int) string {
			return "'" + f.Title() + "'"
		}),
		", ",
	)

	contents := []string{
		"Are you sure you want to",
		quotedNames + "?",
		"",
		"(y/n)",
	}

	return m.styles.RenderDialog(
		lipgloss.JoinVertical(lipgloss.Center, contents...),
	)
}

// formatConfirmationTypeYES formats the confirmation dialog content for strict YES typing verification
// It displays the current input state with visual feedback and requires the user to type "YES" exactly
func (m Model) formatConfirmationTypeYES(files []File) string {
	// Display the current input state
	const fullText = "YES"
	var inputDisplay string

	for i, char := range fullText {
		if i < len(m.state.confirmation.yesInput) {
			// Already typed characters shown in normal text
			inputDisplay += m.styles.Confirm.Text.Render(string(m.state.confirmation.yesInput[i]))
		} else {
			// Not yet typed characters shown as placeholder
			// instead of using a static underscore character for placeholders,
			// it uses the actual character from fullText that needs to be typed (Y, E, or S).
			//
			// inputDisplay += m.styles.Confirm.Placeholder.Render(string(char))
			_ = char
			inputDisplay += m.styles.Confirm.Placeholder.Render("_")
		}
	}

	// Icon indicating completion status
	var statusIcon string
	if m.state.IsYesComplete() {
		statusIcon = m.styles.Confirm.Success.Render(" ✓")
	} else {
		statusIcon = m.styles.Confirm.Error.Render(" ✗")
	}

	quotedNames := strings.Join(
		lo.Map(files, func(f File, index int) string {
			return "'" + f.Title() + "'"
		}),
		", ",
	)

	var contents []string
	dialogMaxWidth := defaultWidth - 6 // border (2) + padding (2) + buffer (2)
	if len(files) > 1 && len(quotedNames) > dialogMaxWidth {
		contents = []string{
			"Are you sure you want to",
			"completely delete " + fmt.Sprintf("%d files", len(files)) + "?",
		}
	} else {
		contents = []string{
			"Are you sure you want to completely delete",
			quotedNames + "?",
		}
	}

	contents = append(contents, []string{
		"",
		"Type YES to confirm",
		"",
		inputDisplay + statusIcon,
		"",
		"  (ESC to cancel, Enter to confirm)  ",
	}...)

	if m.state.IsYesComplete() && len(contents) > 4 {
		contents[3] = "YES confirmed! Press Enter to delete"
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
