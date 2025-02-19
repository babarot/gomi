package ui

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gabriel-vasile/mimetype"
	"github.com/muesli/reflow/wordwrap"
)

// detailView renders the detail view of a file
func (m Model) detailView() string {
	return m.styles.Detail.View(
		m.renderHeader(),
		m.renderDeletedFrom(),
		m.renderDeletedAt(),
		m.renderPreview(),
		m.renderFooter(),
	)
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

// renderPreview renders the preview section of the file
func (m Model) renderPreview() string {
	content := m.viewport.View()

	if m.state.preview.available {
		mtype, _ := mimetype.DetectFile(m.detailFile.TrashPath)
		content = m.styles.RenderErrorPreview(
			ErrCannotPreview.Error(),
			mtype.String(),
			defaultWidth,
			defaultHeight-11-1, // info pane height (11) + preview border (1)
		)
	}

	return fmt.Sprintf("%s\n%s\n%s",
		m.previewHeader(),
		content,
		m.previewFooter(),
	)
}
