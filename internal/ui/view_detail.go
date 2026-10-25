package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// paneInfoRows are the rows of the pane above the file's contents: From,
// Deleted, Size and Restore, then a rule
const paneInfoRows = 5

// detailView is the preview of the file on the cursor, opened over the
// whole of the UI: the header, the pane, and the status and the key help
// at the bottom.
func (m Model) detailView() string {
	lines := []string{m.headerLine()}
	lines = append(lines, m.pane(m.detailFile, m.width, m.paneHeight()))
	lines = append(lines, m.statusLine(), m.helpView())
	return strings.Join(lines, "\n")
}

// paneHeight is the rows the pane takes when it is opened over the UI
func (m Model) paneHeight() int {
	return max(paneInfoRows+3, m.height-2-lipgloss.Height(m.helpView()))
}

// previewSize is the columns and rows of a pane's contents
func previewSize(w, h int) (int, int) {
	return max(0, w-4), max(0, h-2-paneInfoRows)
}

// resizePreview fits the viewport to the pane it is shown in: the one
// opened over the UI, or the one beside the list
func (m *Model) resizePreview() {
	pw, ph := m.width, m.paneHeight()
	if m.state.current == ListView {
		l := m.listLayout()
		pw, ph = l.paneW, l.paneH
	}
	w, h := previewSize(pw, ph)
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(h)
}

// pane draws f in a frame of w columns and h rows: its name on the frame,
// where and when it was deleted and its size, then its contents, and how
// far they are scrolled on the bottom of the frame.
func (m Model) pane(f File, w, h int) string {
	border := m.fg(m.palette.Border)
	inner, bodyH := previewSize(w, h)

	var lines []string
	// the top of the frame, with the name on it
	name := f.Name
	nameColor := m.palette.Name
	if f.IsDir {
		name += "/"
		nameColor = m.palette.DirName
	}
	title := m.fg(nameColor).Bold(true).Render(ansi.Truncate(name, max(0, w-8), ellipsis))
	if m.selection.Contains(f) {
		title += " " + m.fg(m.palette.Selected).Bold(true).Render("✓")
	}
	top := border.Render("╭─ ") + title + " "
	lines = append(lines, top+border.Render(strings.Repeat("─", max(0, w-1-ansi.StringWidth(top)))+"╮"))

	row := func(s string) string {
		return border.Render("│") + " " + s + strings.Repeat(" ", max(0, inner-ansi.StringWidth(s))) + " " + border.Render("│")
	}
	label := func(k string) string { return m.fg(m.palette.Description).Render(fmt.Sprintf("%-9s", k)) }
	value := m.fg(m.palette.Name)
	dim := m.fg(m.palette.Description)

	place := shortPath(m.displayPath(filepath.Dir(f.OriginalPath)), max(0, inner-9))
	lines = append(lines, row(label("From")+value.Render(place)))
	when := f.DeletedAt.In(now().Location())
	lines = append(lines, row(ansi.Truncate(label("Deleted")+value.Render(when.Format("2006-01-02 15:04"))+dim.Render("  "+relativeTime(f.DeletedAt)), inner, ellipsis)))
	kind := "file"
	if f.IsDir {
		kind = "directory"
	}
	lines = append(lines, row(label("Size")+value.Render(m.sizeText(f))+dim.Render("  "+kind)))
	lines = append(lines, row(ansi.Truncate(label("Restore")+m.restoreText(f), inner, ellipsis)))
	lines = append(lines, row(m.fg(m.palette.PreviewBorder).Render(strings.Repeat("╌", inner))))

	for _, l := range m.paneBody(f, inner, bodyH) {
		lines = append(lines, row(l))
	}

	// the bottom of the frame, with how far the contents are scrolled
	pos := ""
	if r, ok := m.previews[f.TrashPath]; ok && r.available && m.viewport.TotalLineCount() > bodyH {
		last := min(m.viewport.YOffset()+bodyH, m.viewport.TotalLineCount())
		pos = fmt.Sprintf(" %d–%d/%d ", m.viewport.YOffset()+1, last, m.viewport.TotalLineCount())
	}
	bottom := "╰" + strings.Repeat("─", max(0, w-3-ansi.StringWidth(pos))) + pos + "─╯"
	lines = append(lines, border.Render(bottom))
	return strings.Join(lines, "\n")
}

// paneBody is the h rows of f's contents, each cut to w: the preview, or
// what is there while it is read or when it cannot be shown.
func (m Model) paneBody(f File, w, h int) []string {
	dim := m.fg(m.palette.PreviewError)
	var lines []string
	r, ok := m.previews[f.TrashPath]
	switch {
	case !ok:
		lines = []string{dim.Render("Loading…")}
	case !r.available:
		lines = []string{dim.Render(ErrCannotPreview.Error() + " (" + r.mimeType + ")")}
	default:
		lines = strings.Split(m.viewport.View(), "\n")
	}
	out := make([]string, h)
	for i := range out {
		if i < len(lines) {
			out[i] = ansi.Truncate(lines[i], w, "")
		}
	}
	return out
}

// foldPane is the preview of a fold: what it is, how many files it holds,
// their size and when they were deleted, then the files.
func (m Model) foldPane(r row, w, h int) string {
	border := m.fg(m.palette.Border)
	inner := max(0, w-4)
	var title string
	if m.list.view == viewPlaces {
		title = shortPath(m.displayPath(r.key), max(0, w-8))
	} else {
		title = "rm at " + r.files[0].DeletedAt.In(now().Location()).Format("2006-01-02 15:04")
	}
	top := border.Render("╭─ ") + m.fg(m.palette.Name).Bold(true).Render(title) + " "
	lines := []string{top + border.Render(strings.Repeat("─", max(0, w-1-ansi.StringWidth(top)))+"╮")}
	row := func(s string) string {
		s = ansi.Truncate(s, inner, ellipsis)
		return border.Render("│") + " " + s + strings.Repeat(" ", max(0, inner-ansi.StringWidth(s))) + " " + border.Render("│")
	}
	label := func(k string) string { return m.fg(m.palette.Description).Render(fmt.Sprintf("%-9s", k)) }
	value := m.fg(m.palette.Name)
	dim := m.fg(m.palette.Description)

	count := fmt.Sprintf("%d", len(r.files))
	if len(r.files) < r.total {
		count = fmt.Sprintf("%d of %d", len(r.files), r.total)
	}
	lines = append(lines, row(label("Files")+value.Render(count)+dim.Render("  "+m.sizeOfFiles(r.files))))
	newest := r.files[0].DeletedAt.In(now().Location())
	oldest := r.files[len(r.files)-1].DeletedAt.In(now().Location())
	when := newest.Format("2006-01-02 15:04")
	if oldest.Format("2006-01-02 15:04") != when {
		when = oldest.Format("2006-01-02 15:04") + " – " + newest.Format("15:04")
		if oldest.YearDay() != newest.YearDay() || oldest.Year() != newest.Year() {
			when = oldest.Format("2006-01-02") + " – " + newest.Format("2006-01-02")
		}
	}
	lines = append(lines, row(label("Deleted")+value.Render(when)))
	what := "enter restores them all"
	if m.list.view == viewBatches {
		what = "enter puts this rm back"
	}
	lines = append(lines, row(label("Restore")+m.fg(m.palette.Selected).Render(what)))
	lines = append(lines, row(label("Marks")+m.foldMarks(r.files)))
	lines = append(lines, row(m.fg(m.palette.PreviewBorder).Render(strings.Repeat("╌", inner))))

	body := max(0, h-2-paneInfoRows)
	for i := range body {
		if i >= len(r.files) {
			lines = append(lines, row(""))
			continue
		}
		f := r.files[i]
		name := f.Name
		color := m.palette.Name
		if f.IsDir {
			name += "/"
			color = m.palette.DirName
		}
		size := ellipsis
		if n, ok := m.knownSize(f); ok {
			size = compactBytes(n)
		}
		s := m.fg(color).Render(ansi.Truncate(name, max(0, inner-8), ellipsis))
		s += strings.Repeat(" ", max(1, inner-ansi.StringWidth(s)-len(size))) + dim.Render(size)
		lines = append(lines, row(s))
	}
	lines = append(lines, border.Render("╰"+strings.Repeat("─", max(0, w-2))+"╯"))
	return strings.Join(lines, "\n")
}

// restoreText says what restoring f would meet, once its place is looked
// at
func (m Model) restoreText(f File) string {
	switch m.origins[f.TrashPath] {
	case originTaken:
		return m.fg(m.palette.Dialog).Render("! a file has its name there now; gomi asks for a new name")
	case originGone:
		return m.fg(m.palette.Description).Render("+ its folder is gone; gomi makes it again")
	case originBack:
		return m.fg(m.palette.Selected).Render("back to where it was")
	}
	return m.fg(m.palette.Description).Render(ellipsis)
}

// foldMarks counts the marks among files
func (m Model) foldMarks(files []File) string {
	taken, gone := 0, 0
	for _, f := range files {
		switch m.origins[f.TrashPath] {
		case originTaken:
			taken++
		case originGone:
			gone++
		}
	}
	if taken == 0 && gone == 0 {
		return m.fg(m.palette.Description).Render("none")
	}
	var parts []string
	if taken > 0 {
		parts = append(parts, m.fg(m.palette.Dialog).Render(fmt.Sprintf("! %d have a file in their place", taken)))
	}
	if gone > 0 {
		parts = append(parts, m.fg(m.palette.Description).Render(fmt.Sprintf("+ %d have no folder", gone)))
	}
	return strings.Join(parts, m.fg(m.palette.Description).Render(" · "))
}
