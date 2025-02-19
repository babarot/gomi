package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/gabriel-vasile/mimetype"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/termenv"
)

// renderDetailed renders the detail view of a file
func renderDetailed(m Model) string {
	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderHeader(),
		m.renderDeletedFrom(),
		m.renderDeletedAt(),
		m.renderPreview(),
		m.renderFooter(),
	)
}

// renderHeader renders the header section of the detail view
func (m Model) renderHeader() string {
	return m.styles.RenderDetailTitle(
		m.detailFile.Title(),
		defaultWidth,
		m.detailFile.isSelected(),
	)
}

func (m Model) renderFooter() string {
	return m.styles.Dialog.Separator.Render(strings.Repeat("─", defaultWidth))
}

// renderDeletedFrom renders the section showing where the file was deleted from
func (m Model) renderDeletedFrom() string {
	text := filepath.Dir(m.detailFile.OriginalPath)
	title := "Deleted From"
	if !m.state.detail.showOrigin {
		title = "Trash Path"
		text = filepath.Dir(m.detailFile.TrashPath)
	}

	w := wordwrap.NewWriter(46)
	w.Breakpoints = []rune{'/', '.'}
	w.KeepNewlines = false
	_, _ = w.Write([]byte(text))
	_ = w.Close()

	return m.styles.RenderDeletedFrom(title, w.String())
}

// renderDeletedAt renders the section showing when the file was deleted
func (m Model) renderDeletedAt() string {
	var ts string
	switch m.state.detail.dateFormat {
	case DateFormatAbsolute:
		ts = m.detailFile.DeletedAt.Format(time.DateTime)
	default:
		ts = humanize.Time(m.detailFile.DeletedAt)
	}
	return m.styles.RenderDeletedAt("Deleted At", ts)
}

func (m Model) previewHeader() string {
	return m.styles.RenderPreviewFrame(
		"",
		// m.viewport.Width,
		defaultWidth,
		true,
		m.detailFile.Size(),
	)
}

func (m Model) previewFooter() string {
	if m.state.preview.available {
		return m.styles.RenderPreviewFrame(
			"",
			defaultWidth,
			false,
			"",
		)
	}
	return m.styles.RenderPreviewFrame(
		"",
		// m.viewport.Width,
		defaultWidth,
		false,
		fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100),
	)
}

// renderPreview renders the preview section
func (m Model) renderPreview() string {
	content := m.viewport.View()
	if m.state.preview.available {
		mtype, _ := mimetype.DetectFile(m.detailFile.TrashPath)
		verticalMarginHeight := lipgloss.Height(m.previewHeader())

		errorContent := lipgloss.NewStyle().Bold(true).Transform(strings.ToUpper).Render(errCannotPreview.Error())
		mimeInfo := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(termenv.ANSIBrightBlack)).Render("(" + mtype.String() + ")")

		content = lipgloss.Place(defaultWidth, 15-verticalMarginHeight,
			lipgloss.Center, lipgloss.Center,
			errorContent+"\n\n\n"+mimeInfo,
			lipgloss.WithWhitespaceChars("`"),
			lipgloss.WithWhitespaceForeground(lipgloss.ANSIColor(termenv.ANSIBrightBlack)),
		)
	}

	return fmt.Sprintf("%s\n%s\n%s",
		m.previewHeader(),
		content,
		m.previewFooter(),
	)
}
