package log

import "github.com/charmbracelet/lipgloss"

var importantStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FF5F87")).
	Background(lipgloss.Color("#3A3A3A")).
	Padding(0, 1)

func Underline(text string) string {
	return lipgloss.NewStyle().
		Underline(true).
		Render(text)
}

// Highlight makes the given text stand out (yello fg and dark bg)
func Highlight(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F0F080")).
		Background(lipgloss.Color("#3A3A3A")).
		Bold(true).
		Render(" " + text + " ")
}

// UnderBold makes the given text with an underline and bold
func UnderBold(text string) string {
	return lipgloss.NewStyle().
		Underline(true).
		Bold(true).
		Render(" " + text + " ")
}

// ColorizeInfo wraps the given text with INFO level color (blue)
func ColorizeInfo(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#0000FF")).
		Render(text)
}

// ColorizeWarn wraps the given text with WARN level color (yellow)
func ColorizeWarn(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFF00")).
		Render(text)
}

// ColorizeError wraps the given text with ERROR level color (red)
func ColorizeError(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF0000")).
		Render(text)
}

// ColorizeDebug wraps the given text with DEBUG level color (gray)
func ColorizeDebug(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Render(text)
}
