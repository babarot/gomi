package ui

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
)

// The views, the ways the list lays out the files
const (
	viewRecent  = "recent"  // under a heading for each day they were deleted on
	viewPlaces  = "places"  // folded by the directory they were deleted from
	viewBatches = "batches" // folded by the run of gomi that deleted them
)

// fileList is the list of files the TUI shows, newest first, laid out by
// the view and narrowed by the filter.
type fileList struct {
	files []File // newest first
	view  string
	// open has the folds opened in places and batches, by their key
	open map[string]bool

	filter    textinput.Model
	filtering bool // the filter has the focus

	cursor int // the row the cursor is on; never a day heading
	top    int // the first row shown
	shownN int // how many rows the view showed last, a page
}

// rowKind tells what a row of the list is
type rowKind int

const (
	rowDay  rowKind = iota // a day heading in recent
	rowFile                // a file
	rowFold                // a directory in places or a run in batches, with its files
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

	// a fold: its key, the files of it the filter keeps and how many it has
	// in all, and whether it is open. A file in an open fold has depth 1,
	// and last set when it is the fold's last.
	key   string
	total int
	open  bool
	depth int
}

// selectable reports whether the cursor can be on r
func (r row) selectable() bool { return r.kind != rowDay }

// cursorKey names what the row is about, to find it again after the rows
// change
func (r row) cursorKey() string {
	switch r.kind {
	case rowFile:
		return r.file.TrashPath
	case rowFold:
		return "fold:" + r.key
	}
	return ""
}

func newFileList() fileList {
	ti := textinput.New()
	ti.Prompt = "/ "
	return fileList{filter: ti, view: viewRecent, open: map[string]bool{}}
}

// setFiles replaces the files, keeping the cursor on the file it was on
// when that file is still there.
func (l *fileList) setFiles(files []File, now time.Time) {
	on := l.cursorKey(now)
	l.files = files
	sort.SliceStable(l.files, func(i, j int) bool {
		return l.files[i].DeletedAt.After(l.files[j].DeletedAt)
	})
	if l.view != viewRecent && !l.opened(l.view) {
		l.setView(l.view, now)
		return
	}
	l.find(on, now)
}

// setView lays the files out by view, keeping the cursor on the file it
// was on, or on the fold the file is in when the fold is closed. A view
// shown for the first time opens its first fold.
func (l *fileList) setView(view string, now time.Time) {
	f, onFile := l.current(now)
	l.view = view
	rows := l.rows(now)
	if view != viewRecent && len(rows) > 0 && !l.opened(view) {
		l.open[foldKey(view, rows[0].key)] = true
		l.open[view] = true // the view was shown
	}
	if !onFile {
		l.cursor = 0
		l.fixCursor(l.rows(now))
		return
	}
	l.find(f.TrashPath, now)
	if c, ok := l.current(now); !ok || c.TrashPath != f.TrashPath {
		l.find("fold:"+l.foldOf(f), now)
	}
}

// opened reports whether view has been shown already
func (l fileList) opened(view string) bool { return l.open[view] }

// find puts the cursor on the row of key, or on the first row
func (l *fileList) find(key string, now time.Time) {
	rows := l.rows(now)
	l.cursor = 0
	for i, r := range rows {
		if key != "" && r.cursorKey() == key {
			l.cursor = i
			break
		}
	}
	l.fixCursor(rows)
}

// cursorKey is the key of the row the cursor is on
func (l fileList) cursorKey(now time.Time) string {
	if r, ok := l.currentRow(now); ok {
		return r.cursorKey()
	}
	return ""
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

// rows lays out the files the filter keeps, by the view
func (l fileList) rows(now time.Time) []row {
	switch l.view {
	case viewPlaces, viewBatches:
		return l.foldRows()
	}
	return l.dayRows(now)
}

// foldOf is the key of the fold f is in, in the current view
func (l fileList) foldOf(f File) string {
	if l.view == viewPlaces {
		return filepath.Dir(f.OriginalPath)
	}
	// a file trashed without a run ID is a run of its own
	if f.RunID == "" {
		return f.TrashPath
	}
	return f.RunID
}

// foldKey keeps the folds of the two views apart in fileList.open
func foldKey(view, key string) string { return view + ":" + key }

// foldRows lays out places and batches: a fold for each directory or
// run, newest first, with its files under it when it is open. While the
// filter has words, the folds with files it keeps are open.
func (l fileList) foldRows() []row {
	type fold struct {
		key   string
		total int
		files []File
	}
	var folds []*fold
	byKey := map[string]*fold{}
	for _, f := range l.files {
		k := l.foldOf(f)
		g := byKey[k]
		if g == nil {
			g = &fold{key: k}
			byKey[k] = g
			folds = append(folds, g)
		}
		g.total++
		if l.matches(f) {
			g.files = append(g.files, f)
		}
	}
	var rows []row
	for _, g := range folds {
		if len(g.files) == 0 {
			continue
		}
		open := l.open[foldKey(l.view, g.key)] || l.query() != ""
		rows = append(rows, row{kind: rowFold, key: g.key, files: g.files, total: g.total, open: open})
		if !open {
			continue
		}
		for i, f := range g.files {
			rows = append(rows, row{kind: rowFile, file: f, depth: 1, last: i == len(g.files)-1})
		}
	}
	return rows
}

// toggle opens or closes the fold the cursor is on, or closes the one the
// file it is on is in and moves to the fold; open says which.
func (l *fileList) toggle(open bool, now time.Time) {
	r, ok := l.currentRow(now)
	if !ok || l.view == viewRecent {
		return
	}
	switch {
	case r.kind == rowFold:
		l.open[foldKey(l.view, r.key)] = open
	case r.kind == rowFile && !open:
		k := l.foldOf(r.file)
		l.open[foldKey(l.view, k)] = false
		l.find("fold:"+k, now)
	}
}

// dayRows lays out recent: the files under their days.
func (l fileList) dayRows(now time.Time) []row {
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
	r, ok := l.currentRow(now)
	if !ok || r.kind != rowFile {
		return File{}, false
	}
	return r.file, true
}

// currentRow returns the row the cursor is on
func (l fileList) currentRow(now time.Time) (row, bool) {
	rows := l.rows(now)
	if l.cursor < 0 || l.cursor >= len(rows) || !rows[l.cursor].selectable() {
		return row{}, false
	}
	return rows[l.cursor], true
}

// fixCursor puts the cursor on a row it can be on, the next one down if
// it is on a heading, and keeps it within the rows.
func (l *fileList) fixCursor(rows []row) {
	if len(rows) == 0 {
		l.cursor, l.top = 0, 0
		return
	}
	l.cursor = min(max(l.cursor, 0), len(rows)-1)
	for i := l.cursor; i < len(rows); i++ {
		if rows[i].selectable() {
			l.cursor = i
			return
		}
	}
	for i := l.cursor; i >= 0; i-- {
		if rows[i].selectable() {
			l.cursor = i
			return
		}
	}
}

// move moves the cursor by n rows, down when n is positive, past headings.
func (l *fileList) move(n int, now time.Time) {
	rows := l.rows(now)
	step := 1
	if n < 0 {
		step, n = -1, -n
	}
	i := l.cursor
	for ; n > 0; n-- {
		j := i + step
		for j >= 0 && j < len(rows) && !rows[j].selectable() {
			j += step
		}
		if j < 0 || j >= len(rows) {
			break
		}
		i = j
	}
	l.cursor = i
}

// moveFile moves the cursor by n files, past headings and folds, and
// stays where it is when there is no file that way
func (l *fileList) moveFile(n int, now time.Time) {
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
