package log

import "github.com/charmbracelet/lipgloss"

var (
	debugStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#808080")) // Gray
	infoStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#0000FF")) // Blue
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")) // Yellow
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")) // Red

	fatalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Background(lipgloss.Color("#000000")).
			Bold(true) // Red on Black, Bold

	importantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")).
			Background(lipgloss.Color("#3A3A3A")).
			Bold(true) // Pink on Gray, Bold

	// predefined styles for each level
	levelStyles = []struct {
		level    Level
		maxWidth int
		style    lipgloss.Style
	}{
		{
			level:    DebugLevel,
			maxWidth: 5,
			style:    debugStyle,
		},
		{
			level:    InfoLevel,
			maxWidth: 5,
			style:    infoStyle,
		},
		{
			level:    WarnLevel,
			maxWidth: 5,
			style:    warnStyle,
		},
		{
			level:    ErrorLevel,
			maxWidth: 5,
			style:    errorStyle,
		},
		{
			level:    FatalLevel,
			maxWidth: 5,
			style:    fatalStyle,
		},
		{
			level:    ImportantLevel,
			maxWidth: 9,
			style:    importantStyle,
		},
	}
)

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
