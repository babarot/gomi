package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/babarot/gomi/internal/ui/styles"
)

// The columns of a file row, from the left: the cursor bar, the selection
// mark, the time, the line joining the files of one run, then the name,
// where it was deleted from and its size.
const (
	colMark    = 2
	colTime    = 4
	colJoin    = 10
	colName    = 12
	sizeWidth  = 6
	nameShare  = 45 // percent of the room left for the name; the rest is the place
	minNameCol = 10
)

// fg returns a style drawing in c
func (m Model) fg(c styles.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(c.Of(m.isDark))
}

// listView draws the list: a header, the rows, and the count, the status
// and the key help at the bottom, in the rows the UI has.
func (m Model) listView() string {
	lines := []string{m.headerLine()}
	if m.list.filtering || m.list.filter.Value() != "" {
		lines = append(lines, " "+m.list.filter.View())
	}
	l := m.listLayout()
	n := l.rows

	rows := m.list.rows(now())
	m.list.scroll(rows, n)
	m.list.shownN = n
	var shown []string
	for i := range n {
		j := m.list.top + i
		if j >= len(rows) {
			shown = append(shown, "")
			continue
		}
		shown = append(shown, m.rowLine(rows[j], j == m.list.cursor, l.listW))
	}
	if len(rows) == 0 {
		shown[0] = "  " + m.fg(m.palette.Description).Render(m.emptyText())
	}

	switch l.side {
	case sideRight:
		pane := strings.Split(m.sidePane(l), "\n")
		for i, s := range shown {
			s += strings.Repeat(" ", max(0, l.listW-ansi.StringWidth(s)))
			if i < len(pane) {
				s += " " + pane[i]
			}
			shown[i] = s
		}
		lines = append(lines, shown...)
		lines = append(lines, m.countLine(rows, n))
	case sideBelow:
		lines = append(lines, shown...)
		lines = append(lines, m.countLine(rows, n), m.sidePane(l))
	default:
		lines = append(lines, shown...)
		lines = append(lines, m.countLine(rows, n))
	}
	lines = append(lines, m.statusLine(), m.helpView())
	return strings.Join(lines, "\n")
}

// sidePane is the preview beside the list, of the file or the fold on
// the cursor
func (m Model) sidePane(l layout) string {
	r, ok := m.list.currentRow(now())
	switch {
	case !ok:
		return ""
	case r.kind == rowFold:
		return m.foldPane(r, l.paneW, l.paneH)
	}
	return m.pane(m.detailFile, l.paneW, l.paneH)
}

func (m Model) emptyText() string {
	if m.list.filter.Value() != "" {
		return fmt.Sprintf("No files match %q", m.list.filter.Value())
	}
	return "The trash is empty"
}

// headerLine is what is in the trash on the left, and what is selected on
// the right.
func (m Model) headerLine() string {
	left := m.fg(m.palette.Cursor).Bold(true).Render("gomi") +
		m.fg(m.palette.Description).Render(fmt.Sprintf("  %d in the trash · %s", len(m.list.files), m.totalSize()))
	right := m.fg(m.palette.Description).Render("view: " + m.list.view)
	if n := len(m.selection.items); n > 0 {
		right = m.fg(m.palette.Selected).Render(fmt.Sprintf("%d selected", n)) + m.fg(m.palette.Description).Render(" · ") + right
	}
	gap := m.width - 2 - ansi.StringWidth(left) - ansi.StringWidth(right)
	if gap < 1 {
		return ansi.Truncate(" "+left, m.width, "")
	}
	return " " + left + strings.Repeat(" ", gap) + right
}

// totalSize is the size of everything in the trash, with … while a
// directory is still being sized.
func (m Model) totalSize() string {
	return m.sizeOfFiles(m.list.files)
}

// knownSize is the size of f's contents, if known by now
func (m Model) knownSize(f File) (int64, bool) {
	if f.SizeKnown {
		return f.Size, true
	}
	size, ok := m.sizes[f.TrashPath]
	return size, ok && size >= 0
}

// rowLine draws a day heading, a fold or a file, w columns wide
func (m Model) rowLine(r row, onCursor bool, w int) string {
	switch {
	case r.kind == rowDay:
		return m.dayLine(r, w)
	case r.kind == rowFold:
		return m.foldLine(r, onCursor, w)
	case r.depth > 0:
		return m.foldFileLine(r, onCursor, w)
	case m.list.view == viewPicker:
		return m.pickLine(r, onCursor, w)
	}
	return m.fileLine(r, onCursor, w)
}

// pickLine is a file in the picker: its name, where it was deleted from,
// how long ago, its size and its mark
func (m Model) pickLine(r row, onCursor bool, w int) string {
	on := m.rowStyle(onCursor)
	plain := on(lipgloss.NewStyle())
	f := r.file
	mark := ""
	if m.selection.Contains(f) {
		mark = "✓"
	}
	size := ellipsis
	if n, ok := m.knownSize(f); ok {
		size = compactBytes(n)
	}
	right := fmt.Sprintf("%11s  %6s", relativeTime(f.DeletedAt), size)
	line := m.rowStart(onCursor, mark)
	room := max(0, w-3-ansi.StringWidth(line)-ansi.StringWidth(right)-2)
	nameW := max(minNameCol, room*nameShare/100)
	name := m.nameCell(f, nameW, on)
	line += name + plain.Render(strings.Repeat(" ", max(2, nameW+2-ansi.StringWidth(name))))
	line += on(m.fg(m.palette.Description)).Render(shortPath(m.displayPath(filepath.Dir(f.OriginalPath)), max(0, room-nameW-2)))
	line += plain.Render(strings.Repeat(" ", max(1, w-3-ansi.StringWidth(line)-ansi.StringWidth(right))))
	line += on(m.fg(m.palette.Description)).Render(right) + plain.Render(" ") + m.originMark(f, on)
	return m.fillRow(line, onCursor, w)
}

// rowStyle gives a row's styles the cursor row's background when it is
// under the cursor
func (m Model) rowStyle(onCursor bool) func(lipgloss.Style) lipgloss.Style {
	return func(s lipgloss.Style) lipgloss.Style {
		if onCursor {
			return s.Background(m.palette.CursorRow.Of(m.isDark))
		}
		return s
	}
}

// rowStart is the cursor bar and the selection mark that every selectable
// row starts with; marked is the mark, if any
func (m Model) rowStart(onCursor bool, mark string) string {
	on := m.rowStyle(onCursor)
	plain := on(lipgloss.NewStyle())
	bar := " "
	if onCursor {
		bar = "▎"
	}
	if mark == "" {
		mark = " "
	}
	return on(m.fg(m.palette.Cursor)).Render(bar) + plain.Render(" ") +
		on(m.fg(m.palette.Selected)).Bold(true).Render(mark) + plain.Render(" ")
}

// fillRow pads a row to w columns on the row's background and cuts it to w
func (m Model) fillRow(s string, onCursor bool, w int) string {
	plain := m.rowStyle(onCursor)(lipgloss.NewStyle())
	s += plain.Render(strings.Repeat(" ", max(0, w-ansi.StringWidth(s))))
	return ansi.Truncate(s, w, "")
}

// foldLine is a directory in places or a run in batches: ▸ or ▾, what it
// is, and how many files it holds and their size on the right.
func (m Model) foldLine(r row, onCursor bool, w int) string {
	on := m.rowStyle(onCursor)
	plain := on(lipgloss.NewStyle())
	dim := on(m.fg(m.palette.Description))

	mark := ""
	selected := 0
	for _, f := range r.files {
		if m.selection.Contains(f) {
			selected++
		}
	}
	switch {
	case selected > 0 && selected == len(r.files):
		mark = "✓"
	case selected > 0:
		mark = "•"
	}
	arrow := "▸"
	if r.open {
		arrow = "▾"
	}

	count := fmt.Sprintf("%d %s", len(r.files), plural(len(r.files), "file"))
	if len(r.files) < r.total {
		// the filter leaves out some of the fold
		count = fmt.Sprintf("%d of %d files", len(r.files), r.total)
	}
	right := count + " · " + m.sizeOfFiles(r.files)
	if m.list.view == viewPlaces {
		right += "  " + fmt.Sprintf("%11s", relativeTime(r.files[0].DeletedAt))
	}
	room := max(0, w-6-ansi.StringWidth(right)-2)

	var label string
	if m.list.view == viewPlaces {
		label = on(m.fg(m.palette.Name)).Bold(true).Render(shortPath(m.displayPath(r.key), room))
	} else {
		label = m.runLabel(r, room, on)
	}
	line := m.rowStart(onCursor, mark) + on(m.fg(m.palette.Cursor)).Render(arrow) + plain.Render(" ") + label
	line += plain.Render(strings.Repeat(" ", max(1, w-1-ansi.StringWidth(line)-ansi.StringWidth(right))))
	line += dim.Render(right)
	return m.fillRow(line, onCursor, w)
}

// runLabel is a run in batches: when it ran, and "rm" with what it
// removed, and the directory it removed them from when there is one
func (m Model) runLabel(r row, room int, on func(lipgloss.Style) lipgloss.Style) string {
	t := r.files[0].DeletedAt.In(now().Location())
	when := t.Format("15:04")
	if dayLabel(t, now()) != "Today" {
		when = t.Format("Jan 2 15:04")
	}
	names := make([]string, len(r.files))
	places := map[string]bool{}
	for i, f := range r.files {
		names[i] = f.Name
		if f.IsDir {
			names[i] += "/"
		}
		places[filepath.Dir(f.OriginalPath)] = true
	}
	where := fmt.Sprintf("%d places", len(places))
	if len(places) == 1 {
		where = m.displayPath(filepath.Dir(r.files[0].OriginalPath))
	}
	head := fmt.Sprintf("%-12s ", when)
	rest := max(0, room-ansi.StringWidth(head)-3)
	whereW := min(ansi.StringWidth(where), rest/3)
	nameW := max(0, rest-whereW-2)
	label := on(m.fg(m.palette.Description)).Render(head) +
		on(m.fg(m.palette.HelpSeparator)).Render("rm ") +
		on(m.fg(m.palette.Name)).Bold(true).Render(ansi.Truncate(strings.Join(names, " "), nameW, ellipsis))
	if whereW > 0 {
		label += on(lipgloss.NewStyle()).Render("  ") + on(m.fg(m.palette.Description)).Render(shortPath(where, whereW))
	}
	return label
}

// foldFileLine is a file in an open fold: a tree line, its name, and on
// the right when it was deleted, or in batches where from, and its size
func (m Model) foldFileLine(r row, onCursor bool, w int) string {
	on := m.rowStyle(onCursor)
	plain := on(lipgloss.NewStyle())
	f := r.file
	mark := ""
	if m.selection.Contains(f) {
		mark = "✓"
	}
	tree := "├"
	if r.last {
		tree = "└"
	}
	size := ellipsis
	if n, ok := m.knownSize(f); ok {
		size = compactBytes(n)
	}
	right := fmt.Sprintf("%11s  %6s", relativeTime(f.DeletedAt), size)
	if m.list.view == viewBatches {
		right = fmt.Sprintf("%6s", size)
	}
	line := m.rowStart(onCursor, mark) + plain.Render("  ") + on(m.fg(m.palette.HelpSeparator)).Render(tree) + plain.Render(" ")
	room := max(0, w-3-ansi.StringWidth(line)-ansi.StringWidth(right)-2)
	if m.list.view == viewBatches {
		nameW := room / 2
		line += m.nameCell(f, nameW, on)
		line += plain.Render(strings.Repeat(" ", max(2, nameW+2-ansi.StringWidth(ansi.Strip(m.nameCell(f, nameW, on))))))
		line += on(m.fg(m.palette.Description)).Render(shortPath(m.displayPath(filepath.Dir(f.OriginalPath)), max(0, room-nameW-2)))
	} else {
		line += m.nameCell(f, room, on)
	}
	line += plain.Render(strings.Repeat(" ", max(1, w-3-ansi.StringWidth(line)-ansi.StringWidth(right))))
	line += on(m.fg(m.palette.Description)).Render(right) + plain.Render(" ") + m.originMark(f, on)
	return m.fillRow(line, onCursor, w)
}

// dayLine is a rule with the day and what was deleted on it
func (m Model) dayLine(r row, w int) string {
	rule := m.fg(m.palette.HelpSeparator)
	label := m.fg(m.palette.Heading).Bold(true).Render(r.day)
	right := m.fg(m.palette.Description).Render(fmt.Sprintf("%d %s · %s", len(r.files), plural(len(r.files), "file"), m.sizeOfFiles(r.files)))
	fill := w - 3 - ansi.StringWidth(r.day) - 2 - ansi.StringWidth(right) - 3
	if fill < 1 {
		return rule.Render("── ") + label
	}
	return rule.Render("── ") + label + " " + rule.Render(strings.Repeat("─", fill)) + " " + right + " " + rule.Render("──")
}

func (m Model) sizeOfFiles(files []File) string {
	var total int64
	unknown := false
	for _, f := range files {
		size, ok := m.knownSize(f)
		if !ok {
			unknown = true
			continue
		}
		total += size
	}
	switch {
	case unknown && total == 0:
		return ellipsis
	case unknown:
		return compactBytes(total) + ellipsis
	}
	return compactBytes(total)
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// fileLine is one file: the time it was deleted, its name, where it was
// deleted from and its size. The row under the cursor is drawn on a
// background of its own, across the whole width.
func (m Model) fileLine(r row, onCursor bool, w int) string {
	f := r.file
	on := func(s lipgloss.Style) lipgloss.Style {
		if onCursor {
			return s.Background(m.palette.CursorRow.Of(m.isDark))
		}
		return s
	}
	plain := on(lipgloss.NewStyle())
	pad := func(s string, w int) string {
		return s + plain.Render(strings.Repeat(" ", max(0, w-ansi.StringWidth(s))))
	}

	var b strings.Builder
	bar := " "
	if onCursor {
		bar = "▎"
	}
	b.WriteString(on(m.fg(m.palette.Cursor)).Render(bar) + plain.Render(" "))
	mark := " "
	if m.selection.Contains(f) {
		mark = "✓"
	}
	b.WriteString(on(m.fg(m.palette.Selected)).Bold(true).Render(mark) + plain.Render(" "))
	clock := ""
	if r.first {
		clock = f.DeletedAt.In(now().Location()).Format("15:04")
	}
	b.WriteString(pad(on(m.fg(m.palette.Description)).Render(clock), colJoin-colTime))
	join := " "
	if r.batch {
		switch {
		case r.first:
			join = "┌"
		case r.last:
			join = "└"
		default:
			join = "│"
		}
	}
	b.WriteString(on(m.fg(m.palette.HelpSeparator)).Render(join) + plain.Render(" "))

	avail := max(0, w-colName-2-sizeWidth-3)
	nameW := max(minNameCol, avail*nameShare/100)
	placeW := max(0, avail-nameW)
	b.WriteString(pad(m.nameCell(f, nameW, on), nameW))
	b.WriteString(plain.Render("  "))
	place := shortPath(m.displayPath(filepath.Dir(f.OriginalPath)), placeW)
	b.WriteString(pad(on(m.fg(m.palette.Description)).Render(place), placeW))
	size := ellipsis
	if s, ok := m.knownSize(f); ok {
		size = compactBytes(s)
	}
	b.WriteString(plain.Render(strings.Repeat(" ", max(0, sizeWidth-ansi.StringWidth(size)))))
	b.WriteString(on(m.fg(m.palette.Description)).Render(size) + plain.Render(" ") + m.originMark(f, on) + plain.Render(" "))
	return ansi.Truncate(b.String(), w, "")
}

// nameCell is the file name cut to w, a directory marked with a slash, and
// what matches the filter highlighted.
func (m Model) nameCell(f File, w int, on func(lipgloss.Style) lipgloss.Style) string {
	name := f.Name
	color := m.palette.Name
	if f.IsDir {
		name += "/"
		color = m.palette.DirName
	}
	name = ansi.Truncate(name, w, ellipsis)
	base := on(m.fg(color)).Bold(true)
	match := on(m.fg(m.palette.FilterMatch)).Bold(true).Underline(true)
	q := m.list.query()
	if m.list.view == viewPicker {
		// the letters of the filter, wherever they are in the name
		hit := map[int]bool{}
		for _, i := range fuzzyRunes(name, q) {
			hit[i] = true
		}
		var b strings.Builder
		for i, c := range []rune(name) {
			if hit[i] {
				b.WriteString(match.Render(string(c)))
			} else {
				b.WriteString(base.Render(string(c)))
			}
		}
		return b.String()
	}
	i, n := matchRunes(name, q)
	if n == 0 {
		return base.Render(name)
	}
	r := []rune(name)
	return base.Render(string(r[:i])) + match.Render(string(r[i:i+n])) + base.Render(string(r[i+n:]))
}

// matchRunes finds q, lowercased, in s ignoring case, and returns where it
// starts and how long it is in runes of s; n is zero when it is not
// there. Runes, not bytes: lowercasing can change how many bytes a rune
// takes, so a byte offset in the lowercased string may not fit s.
func matchRunes(s, q string) (i, n int) {
	if q == "" {
		return 0, 0
	}
	rs, rq := []rune(s), []rune(q)
	for i := 0; i+len(rq) <= len(rs); i++ {
		found := true
		for j, c := range rq {
			if unicode.ToLower(rs[i+j]) != c {
				found = false
				break
			}
		}
		if found {
			return i, len(rq)
		}
	}
	return 0, 0
}

// displayPath writes the home directory as ~
func (m Model) displayPath(p string) string {
	if m.home != "" && (p == m.home || strings.HasPrefix(p, m.home+string(filepath.Separator))) {
		return "~" + p[len(m.home):]
	}
	return p
}

// shortPath cuts p to w, keeping its head and as much of its tail as fits:
// ~/…/drop/test. When only the last part fits whole, the one before it is
// cut instead, so that directories which differ only there still differ:
// /private/…/fee7ca5a…/scratchpad.
func shortPath(p string, w int) string {
	if ansi.StringWidth(p) <= w {
		return p
	}
	sep := string(filepath.Separator)
	parts := strings.Split(p, sep)
	if len(parts) < 3 {
		return ansi.Truncate(p, w, ellipsis)
	}
	head := parts[0]
	if head == "" {
		head = sep + parts[1]
	}
	tail, kept := "", 0
	for i := len(parts) - 1; i >= 2; i-- {
		next := sep + parts[i] + tail
		if ansi.StringWidth(head+sep+ellipsis+next) > w {
			break
		}
		tail, kept = next, kept+1
	}
	if kept < 2 && len(parts) > 3 {
		last, prev := sep+parts[len(parts)-1], parts[len(parts)-2]
		if room := w - ansi.StringWidth(head+sep+ellipsis+sep+last); room >= 6 {
			return head + sep + ellipsis + sep + ansi.Truncate(prev, room, ellipsis) + last
		}
	}
	if tail == "" {
		return ansi.Truncate(p, w, ellipsis)
	}
	return head + sep + ellipsis + tail
}

// compactBytes writes a size in at most six columns: 794B, 3.1K, 31M.
func compactBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	v := float64(n) / 1024
	units := "KMGTP"
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	if v < 10 {
		return fmt.Sprintf("%.1f%c", v, units[i])
	}
	return fmt.Sprintf("%.0f%c", v, units[i])
}

// countLine is how many files are shown, and how many more are below.
func (m Model) countLine(rows []row, n int) string {
	files := m.list.shown()
	s := fmt.Sprintf("%d %s", len(files), plural(len(files), "file"))
	if m.list.filter.Value() != "" {
		s = fmt.Sprintf("%d of %d files", len(files), len(m.list.files))
	}
	below := 0
	for _, r := range rows[min(len(rows), m.list.top+n):] {
		if r.kind == rowFile {
			below++
		}
	}
	if below > 0 {
		s += fmt.Sprintf(" · ↓ %d more", below)
	}
	if m.hasMarks(files) {
		s += "   ! a file has its name now   + its folder is gone"
	}
	return ansi.Truncate("  "+m.fg(m.palette.Description).Render(s), m.width, ellipsis)
}

// statusLine tells what was just done, such as deleting files
func (m Model) statusLine() string {
	if m.status == "" {
		return ""
	}
	return "  " + m.fg(m.palette.Selected).Render(m.status)
}

// helpView is the key help at the bottom
func (m Model) helpView() string {
	keyMap := m.keyMap.AsListKeyMap()
	if m.state.current == DetailView {
		keyMap = m.keyMap.AsDetailKeyMap()
	}
	return lipgloss.NewStyle().PaddingLeft(2).Render(m.help.View(keyMap))
}

// sortMenuView is the picker's sort menu: the orders, numbered, the one in
// use checked and the one under the cursor on the cursor's background
func (m Model) sortMenuView() string {
	var lines []string
	for i, o := range sortOrders {
		on := m.rowStyle(i == m.sortSel)
		check := " "
		if i == m.list.sort {
			check = "✓"
		}
		line := on(m.fg(m.palette.Cursor)).Bold(true).Render(fmt.Sprintf(" %d ", i+1)) +
			on(m.fg(m.palette.Name)).Render(fmt.Sprintf("%-24s", o.name)) +
			on(m.fg(m.palette.Selected)).Render(check+" ")
		lines = append(lines, line)
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.palette.Cursor.Of(m.isDark)).
		Render(m.fg(m.palette.Heading).Bold(true).Render(" Sort") + "\n" + strings.Join(lines, "\n"))
}
