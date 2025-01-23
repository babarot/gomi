package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/babarot/gomi/config"
	"github.com/babarot/gomi/ui/styles"
	"github.com/babarot/gomi/utils"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/gabriel-vasile/mimetype"
	"github.com/muesli/reflow/wordwrap"
)

func renderDetailed(m Model) string {
	header := m.renderHeader(m.detailFile)

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		renderFilepath(m.detailFile),
		renderTimestamp(m.detailFile, m.datefmt),
		fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView()),
	)

	return content
}

func (m Model) renderHeader(file File) string {
	name := file.Name
	if file.isSelected() {
		name = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#000000"}).
			Background(lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"}).
			Render(file.Name)
	}
	title := lipgloss.NewStyle().
		BorderStyle(func() lipgloss.Border {
			b := lipgloss.RoundedBorder()
			b.Right = "├"
			return b
		}()).
		Padding(0, 1).
		Bold(true).
		Render(name)

	line := strings.Repeat("─", max(0, defaultWidth-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func renderTimestamp(f File, datefmt string) string {
	var ts string
	switch datefmt {
	case "absolute":
		ts = f.Timestamp.Format(time.DateTime)
	default:
		ts = humanize.Time(f.Timestamp)
	}
	return styles.SectionStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			styles.SectionTitleStyle.MarginRight(3).Render("Deleted At"),
			lipgloss.NewStyle().Render(ts)),
	)
}

func renderFilepath(f File) string {
	s := filepath.Dir(f.From)
	w := wordwrap.NewWriter(46)
	w.Breakpoints = []rune{'/', '.'}
	w.KeepNewlines = false
	_, _ = w.Write([]byte(s))
	_ = w.Close()
	return styles.SectionStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			styles.SectionTitleStyle.MarginBottom(1).Render("Where it was"),
			lipgloss.NewStyle().Render(w.String())),
	)
}

func renderMetadata(f File) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		styles.SectionStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left, styles.SectionTitleStyle.MarginBottom(1).Render("Size"), renderFileSize(f))),
		styles.SectionStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left, styles.SectionTitleStyle.MarginBottom(1).Render("Type"), renderFileType(f))),
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

func renderFileType(f File) string {
	var result string
	fi, err := os.Stat(f.To)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			result = "file has been totally removed"
		default:
			result = err.Error()
		}
	} else {
		if fi.IsDir() {
			result = "(directory)"
		}
	}

	if result == "" {
		mtype, err := mimetype.DetectFile(f.File.To)
		if err != nil {
			result = err.Error()
		} else {
			result = mtype.String()
		}
	}

	return result
}

func (m Model) footerView() string {
	fg := m.config.Color.PreviewBoarder.Foreground
	if m.cannotPreview {
		header := m.renderHeader(m.detailFile)
		return lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Render(strings.Repeat("─", lipgloss.Width(header)))
	}
	info := headerStyle(m.config).Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Render(lipgloss.JoinHorizontal(lipgloss.Center, line, info))
}

func (m Model) headerView() string {
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
