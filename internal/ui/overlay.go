package ui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/babarot/gomi/internal/ui/keys"
)

// overlay draws box over the middle of base, the lines of base showing on
// both sides of it. A box taller than base is drawn alone, so that all of
// it can be read.
func overlay(base, box string, width int) string {
	lines := strings.Split(base, "\n")
	boxLines := strings.Split(box, "\n")
	if len(boxLines) >= len(lines) {
		center := lipgloss.NewStyle().Width(width).Align(lipgloss.Center)
		for i, l := range boxLines {
			boxLines[i] = center.Render(l)
		}
		return strings.Join(boxLines, "\n")
	}
	boxW := lipgloss.Width(box)
	x := max(0, (width-boxW)/2)
	top := (len(lines) - len(boxLines)) / 2
	for i, l := range boxLines {
		under := lines[top+i]
		left := ansi.Truncate(under, x, "")
		left += strings.Repeat(" ", max(0, x-ansi.StringWidth(left)))
		right := ansi.TruncateLeft(under, x+ansi.StringWidth(l), "")
		lines[top+i] = left + l + right
	}
	return strings.Join(lines, "\n")
}

// keyGroup is a group of keys in the key list ? shows
type keyGroup struct {
	name  string
	binds []key.Binding
}

// keyGroups are the keys that work, grouped; the group of the view open
// now comes first
func (m Model) keyGroups() []keyGroup {
	move := keyGroup{"Move", []key.Binding{
		keys.DefaultKeyMapListCursorUp, keys.DefaultKeyMapListCursorDown,
		keys.DefaultKeyMapListGoToStart, keys.DefaultKeyMapListGoToEnd,
	}}
	if m.list.folds() {
		move.binds = append(move.binds, m.keyMap.List.Open, m.keyMap.List.Close)
	} else {
		move.binds = append(move.binds, keys.DefaultKeyMapListNextPage, keys.DefaultKeyMapListPrevPage)
	}
	files := keyGroup{"Files", []key.Binding{m.keyMap.List.Enter, m.keyMap.List.Space, m.keyMap.List.Select, m.keyMap.List.DeSelect}}
	if m.keyMap.List.Delete != nil {
		files.binds = append(files.binds, *m.keyMap.List.Delete)
	}
	views := keyGroup{"Views", []key.Binding{m.keyMap.List.View, keys.DefaultKeyMapListFilter, m.keyMap.List.Scope}}
	if m.list.view == viewPicker {
		views.binds = append(views.binds, m.keyMap.List.Sort)
	}
	preview := keyGroup{"Preview", []key.Binding{
		m.keyMap.Detail.PreviewUp, m.keyMap.Detail.PreviewDown,
		m.keyMap.Detail.HalfPageUp, m.keyMap.Detail.HalfPageDown,
		m.keyMap.Detail.Next, m.keyMap.Detail.Prev, m.keyMap.Detail.Space,
	}}
	other := keyGroup{"Other", []key.Binding{m.keyMap.List.Esc, m.keyMap.Common.Help, m.keyMap.Common.Quit}}
	if m.state.current == DetailView {
		return []keyGroup{preview, files, move, views, other}
	}
	return []keyGroup{move, files, views, preview, other}
}

// helpBox is the list of keys ? shows over the view, the group of the view
// open now first and marked, the groups in two columns
func (m Model) helpBox() string {
	keyStyle := m.fg(m.palette.Cursor).Bold(true)
	desc := m.fg(m.palette.Name)
	var blocks []string
	for i, g := range m.keyGroups() {
		head := m.fg(m.palette.Heading).Bold(true).Render(g.name)
		if i == 0 {
			head += " " + lipgloss.NewStyle().
				Foreground(m.palette.CursorRow.Of(m.isDark)).
				Background(m.palette.Cursor.Of(m.isDark)).
				Bold(true).Render(" here ")
		}
		lines := []string{head}
		for _, b := range g.binds {
			h := b.Help()
			if h.Key == "" {
				continue
			}
			lines = append(lines, "  "+keyStyle.Render(padRight(h.Key, 11))+desc.Render(h.Desc))
		}
		blocks = append(blocks, strings.Join(lines, "\n"))
	}
	// each group goes to the shorter column, in order
	var cols [2][]string
	var heights [2]int
	for _, b := range blocks {
		c := 0
		if heights[1] < heights[0] {
			c = 1
		}
		if len(cols[c]) > 0 {
			heights[c]++ // the blank line between groups
		}
		cols[c] = append(cols[c], b)
		heights[c] += lipgloss.Height(b)
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().PaddingRight(4).Render(strings.Join(cols[0], "\n\n")),
		strings.Join(cols[1], "\n\n"))
	body += "\n\n" + m.fg(m.palette.Description).Render("? esc q close")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.palette.Cursor.Of(m.isDark)).
		Padding(0, 1).
		Render(body)
}

func padRight(s string, w int) string {
	return s + strings.Repeat(" ", max(0, w-ansi.StringWidth(s)))
}
