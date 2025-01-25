package styles

import (
	"strings"

	"github.com/babarot/gomi/internal/config"
	"github.com/charmbracelet/lipgloss"
)

// Color chart: https://github.com/muesli/termenv

var DeletedFromSection = func(cfg config.UI) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.HiddenBorder()).
		Padding(0, 1)
}

var DeletedFromTitle = func(cfg config.UI) lipgloss.Style {
	bg := cfg.Style.DetailView.InfoPane.DeletedFrom.Background
	fg := cfg.Style.DetailView.InfoPane.DeletedFrom.Foreground
	return lipgloss.NewStyle().Padding(0, 1).
		Background(lipgloss.Color(bg)).
		Foreground(lipgloss.Color(fg)).
		Bold(true).
		Transform(strings.ToUpper)
}

var DeletedAtSection = func(cfg config.UI) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.HiddenBorder()).
		Padding(0, 1)
}

var DeletedAtTitle = func(cfg config.UI) lipgloss.Style {
	bg := cfg.Style.DetailView.InfoPane.DeletedAt.Background
	fg := cfg.Style.DetailView.InfoPane.DeletedAt.Foreground
	return lipgloss.NewStyle().Padding(0, 1).
		Background(lipgloss.Color(bg)).
		Foreground(lipgloss.Color(fg)).
		Bold(true).
		Transform(strings.ToUpper)
}

var Scroll = func(cfg config.UI) lipgloss.Style {
	bg := cfg.Style.DetailView.PreviewPane.Scroll.Background
	fg := cfg.Style.DetailView.PreviewPane.Scroll.Foreground
	return lipgloss.NewStyle().Padding(0, 1, 0, 1).
		Foreground(lipgloss.Color(fg)).
		Background(lipgloss.Color(bg))
}

var Size = func(cfg config.UI) lipgloss.Style {
	bg := cfg.Style.DetailView.PreviewPane.Size.Background
	fg := cfg.Style.DetailView.PreviewPane.Size.Foreground
	return lipgloss.NewStyle().Padding(0, 1, 0, 1).
		Foreground(lipgloss.Color(fg)).
		Background(lipgloss.Color(bg))
}
