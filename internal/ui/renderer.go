package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/babarot/gomi/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/dustin/go-humanize"
	"github.com/gabriel-vasile/mimetype"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/termenv"
)

func renderDetailed(m Model) string {
	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderHeader(),
		m.renderDeletedFrom(),
		m.renderDeletedAt(),
		m.renderPreview(),
		m.renderFooter(),
	)
}

func (m Model) renderHeader() string {
	borderForeground := m.config.Style.DetailView.Border
	file := m.detailFile
	name := ansi.Truncate(file.Title(), defaultWidth-len(ellipsis), ellipsis)

	if file.isSelected() {
		selected := m.config.Style.ListView.Selected
		name = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#000000"}).
			Background(lipgloss.AdaptiveColor{Light: selected, Dark: selected}). // green
			Render(name)
	}

	title := lipgloss.NewStyle().
		BorderStyle(func() lipgloss.Border {
			b := lipgloss.RoundedBorder()
			if len(file.Title()) < defaultWidth {
				b.Right = "├"
			}
			return b
		}()).
		BorderForeground(lipgloss.Color(borderForeground)).
		Padding(0, 1).
		Bold(true).
		Render(name)

	line := lipgloss.NewStyle().
		Foreground(lipgloss.Color(borderForeground)).
		Render(strings.Repeat("─", max(0, defaultWidth-lipgloss.Width(title))))

	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m Model) renderFooter() string {
	foreground := m.config.Style.DetailView.Border
	line := lipgloss.NewStyle().
		Foreground(lipgloss.Color(foreground)).
		Render(strings.Repeat("─", defaultWidth))
	return lipgloss.JoinHorizontal(lipgloss.Center, line)
}

func (m Model) renderDeletedFrom() string {
	file := m.detailFile
	text := filepath.Dir(file.OriginalPath)
	title := "Deleted From"
	if !m.locationOrigin {
		title = "Trash Path"
		text = filepath.Dir(file.TrashPath)
	}
	w := wordwrap.NewWriter(46)
	w.Breakpoints = []rune{'/', '.'}
	w.KeepNewlines = false
	_, _ = w.Write([]byte(text))
	_ = w.Close()
	return styles.DeletedFromSection(m.config).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			styles.DeletedFromTitle(m.config).MarginBottom(1).Render(title),
			lipgloss.NewStyle().Render(w.String())),
	)
}

func (m Model) renderDeletedAt() string {
	file := m.detailFile
	var ts string
	switch m.datefmt {
	case datefmtAbs:
		ts = file.DeletedAt.Format(time.DateTime)
	default:
		ts = humanize.Time(file.DeletedAt)
	}
	return styles.DeletedAtSection(m.config).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			styles.DeletedAtTitle(m.config).MarginRight(3).Render("Deleted At"),
			lipgloss.NewStyle().Render(ts)),
	)
}

func (m Model) previewHeader() string {
	color := m.config.Style.DetailView.PreviewPane.Border
	size := styles.Size(m.config).Render(m.detailFile.Size())
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(size)))
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(lipgloss.JoinHorizontal(lipgloss.Center, line, size))
}

func (m Model) previewFooter() string {
	color := m.config.Style.DetailView.PreviewPane.Border
	if m.cannotPreview {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(strings.Repeat("─", defaultWidth))
	}
	info := styles.Scroll(m.config).Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(lipgloss.JoinHorizontal(lipgloss.Center, line, info))
}

func (m Model) renderPreview() string {
	content := m.viewport.View()
	if m.cannotPreview {
		mtype, _ := mimetype.DetectFile(m.detailFile.TrashPath)
		verticalMarginHeight := lipgloss.Height(m.previewHeader())
		content = lipgloss.Place(defaultWidth, 15-verticalMarginHeight,
			lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Bold(true).Transform(strings.ToUpper).Render(errCannotPreview.Error())+"\n\n\n"+
				lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(termenv.ANSIBrightBlack)).Render("("+mtype.String()+")"),
			lipgloss.WithWhitespaceChars("`"),
			lipgloss.WithWhitespaceForeground(lipgloss.ANSIColor(termenv.ANSIBrightBlack)))
	}
	return fmt.Sprintf("%s\n%s\n%s",
		m.previewHeader(),
		content,
		m.previewFooter(),
	)
}
