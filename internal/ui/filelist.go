package ui

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
)

// fileList is the list of files the TUI shows: newest first, under a
// heading for each day they were deleted on, narrowed by the filter.
type fileList struct {
	files []File // newest first

	filter    textinput.Model
	filtering bool // the filter has the focus

	cursor int // the row the cursor is on; always a file row
	top    int // the first row shown
	shownN int // how many rows the view showed last, a page
}

// rowKind tells a day heading from a file
type rowKind int

const (
	rowDay rowKind = iota
	rowFile
)

// row is one line of the list
type row struct {
	kind rowKind

	// a day heading: the day and the files deleted on it
	day   string
	files []File

	// a file, and where it stands among the files of its run: first is the
	// first of them, last the last, and batch is set when the run removed
	// more than this file
	file               File
	first, last, batch bool
}

func newFileList() fileList {
	ti := textinput.New()
	ti.Prompt = "/ "
	return fileList{filter: ti}
}

// setFiles replaces the files, keeping the cursor on the file it was on
// when that file is still there.
func (l *fileList) setFiles(files []File, now time.Time) {
	var on string
	if f, ok := l.current(now); ok {
		on = f.TrashPath
	}
	l.files = files
	sort.SliceStable(l.files, func(i, j int) bool {
		return l.files[i].DeletedAt.After(l.files[j].DeletedAt)
	})
	l.cursor = 0
	rows := l.rows(now)
	for i, r := range rows {
		if r.kind == rowFile && r.file.TrashPath == on {
			l.cursor = i
			break
		}
	}
	l.fixCursor(rows)
}

// query is the filter, lowercased
func (l fileList) query() string {
	return strings.ToLower(strings.TrimSpace(l.filter.Value()))
}

// matches reports whether f is shown under the filter: its name or the
// directory it was deleted from has the filter in it.
func (l fileList) matches(f File) bool {
	q := l.query()
	if q == "" {
		return true
	}
	return strings.Contains(strings.ToLower(f.Name), q) ||
		strings.Contains(strings.ToLower(filepath.Dir(f.OriginalPath)), q)
}

// shown returns the files the filter keeps, newest first.
func (l fileList) shown() []File {
	var out []File
	for _, f := range l.files {
		if l.matches(f) {
			out = append(out, f)
		}
	}
	return out
}

// rows lays out the files the filter keeps under their days.
func (l fileList) rows(now time.Time) []row {
	files := l.shown()
	var rows []row
	day := ""
	for i, f := range files {
		d := dayLabel(f.DeletedAt, now)
		if d != day {
			rows = append(rows, row{kind: rowDay, day: d})
			day = d
		}
		h := &rows[len(rows)-1-countFileRowsSince(rows)]
		h.files = append(h.files, f)

		first := i == 0 || !sameRun(files[i-1], f) || dayLabel(files[i-1].DeletedAt, now) != d
		last := i == len(files)-1 || !sameRun(files[i+1], f) || dayLabel(files[i+1].DeletedAt, now) != d
		rows = append(rows, row{kind: rowFile, file: f, first: first, last: last, batch: !first || !last})
	}
	return rows
}

// countFileRowsSince counts the file rows after the last day heading
func countFileRowsSince(rows []row) int {
	n := 0
	for i := len(rows) - 1; i >= 0 && rows[i].kind == rowFile; i-- {
		n++
	}
	return n
}

// sameRun reports whether a and b were removed by one run of gomi. Files
// trashed without a run ID are each a run of their own.
func sameRun(a, b File) bool {
	return a.RunID != "" && a.RunID == b.RunID
}

// dayLabel names the day t falls on, seen from now: Today, Yesterday, the
// weekday within a week, else the date.
func dayLabel(t, now time.Time) string {
	t = t.In(now.Location())
	y, m, d := now.Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	switch days := int(today.Sub(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, now.Location())).Hours() / 24); {
	case days <= 0:
		return "Today"
	case days == 1:
		return "Yesterday"
	case days < 7:
		return t.Format("Monday")
	}
	return t.Format("Mon, Jan 2")
}

// current returns the file the cursor is on
func (l fileList) current(now time.Time) (File, bool) {
	rows := l.rows(now)
	if l.cursor < 0 || l.cursor >= len(rows) || rows[l.cursor].kind != rowFile {
		return File{}, false
	}
	return rows[l.cursor].file, true
}

// fixCursor puts the cursor on a file row, the next one down if it is on
// a heading, and keeps it within the rows.
func (l *fileList) fixCursor(rows []row) {
	if len(rows) == 0 {
		l.cursor, l.top = 0, 0
		return
	}
	l.cursor = min(max(l.cursor, 0), len(rows)-1)
	for i := l.cursor; i < len(rows); i++ {
		if rows[i].kind == rowFile {
			l.cursor = i
			return
		}
	}
	for i := l.cursor; i >= 0; i-- {
		if rows[i].kind == rowFile {
			l.cursor = i
			return
		}
	}
}

// move moves the cursor by n files, down when n is positive.
func (l *fileList) move(n int, now time.Time) {
	rows := l.rows(now)
	step := 1
	if n < 0 {
		step, n = -1, -n
	}
	i := l.cursor
	for ; n > 0; n-- {
		j := i + step
		for j >= 0 && j < len(rows) && rows[j].kind != rowFile {
			j += step
		}
		if j < 0 || j >= len(rows) {
			break
		}
		i = j
	}
	l.cursor = i
}

// toStart and toEnd move the cursor to the first and the last file.
func (l *fileList) toStart(now time.Time) {
	l.cursor = 0
	l.fixCursor(l.rows(now))
}

func (l *fileList) toEnd(now time.Time) {
	rows := l.rows(now)
	l.cursor = len(rows) - 1
	l.fixCursor(rows)
}

// scroll keeps the cursor within the n rows shown, and the heading of its
// day above it when there is room.
func (l *fileList) scroll(rows []row, n int) {
	if n < 1 {
		return
	}
	if l.cursor < l.top {
		l.top = l.cursor
		if l.top > 0 && rows[l.top-1].kind == rowDay {
			l.top--
		}
	}
	if l.cursor >= l.top+n {
		l.top = l.cursor - n + 1
	}
	l.top = min(l.top, max(0, len(rows)-n))
	l.top = max(l.top, 0)
}
