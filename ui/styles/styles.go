package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	AccentColor = lipgloss.ANSIColor(termenv.ANSIBlack)

	SectionStyle      = lipgloss.NewStyle().BorderStyle(lipgloss.HiddenBorder()).BorderForeground(AccentColor).Padding(0, 1)
	SectionTitleStyle = lipgloss.NewStyle().Padding(0, 1).Background(AccentColor).Foreground(lipgloss.Color("15")).Bold(true).Transform(strings.ToUpper)
)
