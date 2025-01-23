package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/babarot/gomi/ui/styles"
	"github.com/babarot/gomi/utils"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/muesli/reflow/wordwrap"
)

func renderDetailed(m Model) string {
	color := m.config.Style.Window.Border

	header := renderHeader(m.detailFile, color)
	footer := renderFooter(m.detailFile, color)

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		m.renderFilepath(),
		m.renderTimestamp(),
		m.renderPreview(),
		footer,
	)

	return content
}

func renderHeader(f File, color string) string {
	name := f.Title()

	if f.isSelected() {
		name = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#000000"}).
			Background(lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"}).
			Render(name)
	}

	title := lipgloss.NewStyle().
		BorderStyle(func() lipgloss.Border {
			b := lipgloss.RoundedBorder()
			b.Right = "├"
			return b
		}()).
		BorderForeground(lipgloss.Color(color)).
		Padding(0, 1).
		Bold(true).
		Render(name)

	line := lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Render(strings.Repeat("─", max(0, defaultWidth-lipgloss.Width(title))))

	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func renderFooter(f File, color string) string {
	line := lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Render(strings.Repeat("─", defaultWidth))
	return lipgloss.JoinHorizontal(lipgloss.Center, line)
}

func (m Model) renderFilepath() string {
	file := m.detailFile
	text := filepath.Dir(file.From)
	w := wordwrap.NewWriter(46)
	w.Breakpoints = []rune{'/', '.'}
	w.KeepNewlines = false
	_, _ = w.Write([]byte(text))
	_ = w.Close()
	return styles.Section(m.config).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				styles.SectionTitle(m.config).MarginBottom(1).Render("Where it was"),
				lipgloss.NewStyle().Render(w.String())),
		)
}

func (m Model) renderTimestamp() string {
	file := m.detailFile
	var ts string
	switch m.datefmt {
	case datefmtAbs:
		ts = file.Timestamp.Format(time.DateTime)
	default:
		ts = humanize.Time(file.Timestamp)
	}
	return styles.Section(m.config).
		Render(
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				styles.SectionTitle(m.config).MarginRight(3).Render("Deleted At"),
				lipgloss.NewStyle().Render(ts)),
		)
}

func calcSize(f File) string {
	var sizeStr string
	size, err := utils.DirSize(f.To)
	if err != nil {
		sizeStr = "(Cannot be calculated)"
	} else {
		sizeStr = humanize.Bytes(uint64(size))
	}
	return sizeStr
}

func (m Model) previewHeader() string {
	color := m.config.Style.PreviewPane.Border
	size := styles.Size(m.config).Render(calcSize(m.detailFile))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(size)))
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(lipgloss.JoinHorizontal(lipgloss.Center, line, size))
}

func (m Model) previewFooter() string {
	color := m.config.Style.PreviewPane.Border
	if m.cannotPreview {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(strings.Repeat("─", defaultWidth))
	}
	info := styles.Scroll(m.config).Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(lipgloss.JoinHorizontal(lipgloss.Center, line, info))
}

func (m Model) renderPreview() string {
	return fmt.Sprintf("%s\n%s\n%s",
		m.previewHeader(),
		m.viewport.View(),
		m.previewFooter(),
	)
}
