package styles

import (
	"strings"

	"github.com/babarot/gomi/internal/config"
	"github.com/charmbracelet/lipgloss"
)

// Color chart: https://github.com/muesli/termenv

var Section = func(cfg config.UI) lipgloss.Style {
	fg := cfg.Style.InfoPane.Section.Foreground
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.HiddenBorder()).
		BorderForeground(lipgloss.Color(fg)).Padding(0, 1).
		Padding(0, 1)
}

var SectionTitle = func(cfg config.UI) lipgloss.Style {
	bg := cfg.Style.InfoPane.Section.Background
	fg := cfg.Style.InfoPane.Section.Foreground
	return lipgloss.NewStyle().Padding(0, 1).
		Background(lipgloss.Color(bg)).
		Foreground(lipgloss.Color(fg)).
		Bold(true).
		Transform(strings.ToUpper)
}

var Scroll = func(cfg config.UI) lipgloss.Style {
	bg := cfg.Style.PreviewPane.Scroll.Background
	fg := cfg.Style.PreviewPane.Scroll.Foreground
	return lipgloss.NewStyle().Padding(0, 1, 0, 1).
		Foreground(lipgloss.Color(fg)).
		Background(lipgloss.Color(bg))
}

var Size = func(cfg config.UI) lipgloss.Style {
	bg := cfg.Style.PreviewPane.Size.Background
	fg := cfg.Style.PreviewPane.Size.Foreground
	return lipgloss.NewStyle().Padding(0, 1, 0, 1).
		Foreground(lipgloss.Color(fg)).
		Background(lipgloss.Color(bg))
}
