package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/babarot/gomi/config"
	"github.com/babarot/gomi/ui/styles"
	"github.com/babarot/gomi/utils"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/muesli/reflow/wordwrap"
)

func renderDetailed(m Model) string {
	fg := m.config.Color.PreviewBoarder.Foreground
	bg := m.config.Color.PreviewBoarder.Background

	header := renderHeader(m.detailFile, fg, bg)

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		m.renderFilepath(),
		m.renderTimestamp(),
		m.renderPreview(),
	)

	return content
}

func renderHeader(f File, fg, bg string) string {
	name := f.Name

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
		BorderForeground(lipgloss.Color(fg)).
		Padding(0, 1).
		Bold(true).
		Render(name)

	line := lipgloss.NewStyle().
		Foreground(lipgloss.Color(fg)).
		Render(strings.Repeat("─", max(0, defaultWidth-lipgloss.Width(title))))

	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
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
	return styles.SectionStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			styles.SectionTitleStyle.MarginRight(3).Render("Deleted At"),
			lipgloss.NewStyle().Render(ts)),
	)
}

func (m Model) renderFilepath() string {
	file := m.detailFile
	text := filepath.Dir(file.From)
	w := wordwrap.NewWriter(46)
	w.Breakpoints = []rune{'/', '.'}
	w.KeepNewlines = false
	_, _ = w.Write([]byte(text))
	_ = w.Close()
	return styles.SectionStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			styles.SectionTitleStyle.MarginBottom(1).Render("Where it was"),
			lipgloss.NewStyle().Render(w.String())),
	)
}

func renderFileSize(f File) string {
	var sizeStr string
	size, err := utils.DirSize(f.To)
	if err != nil {
		sizeStr = "(Cannot be calculated)"
	} else {
		sizeStr = humanize.Bytes(uint64(size))
	}
	return sizeStr
}

func (m Model) previewFooter() string {
	fg := m.config.Color.PreviewBoarder.Foreground
	bg := m.config.Color.PreviewBoarder.Background
	if m.cannotPreview {
		header := renderHeader(m.detailFile, fg, bg)
		return lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Render(strings.Repeat("─", lipgloss.Width(header)))
	}
	info := headerStyle(m.config).Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Render(lipgloss.JoinHorizontal(lipgloss.Center, line, info))
}

func (m Model) previewHeader() string {
	fg := m.config.Color.PreviewBoarder.Foreground
	size := headerStyle(m.config).Render(renderFileSize(m.detailFile))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(size)))
	return lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Render(lipgloss.JoinHorizontal(lipgloss.Center, line, size))
}

var headerStyle = func(cfg config.UI) lipgloss.Style {
	fgAsBg := cfg.Color.PreviewBoarder.Foreground
	bgAsFg := cfg.Color.PreviewBoarder.Background
	return lipgloss.NewStyle().Padding(0, 1, 0, 1).
		Foreground(lipgloss.Color(bgAsFg)).
		Background(lipgloss.Color(fgAsBg))
}

func (m Model) renderPreview() string {
	return fmt.Sprintf("%s\n%s\n%s",
		m.previewHeader(),
		m.viewport.View(),
		m.previewFooter(),
	)
}
