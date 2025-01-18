package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/dustin/go-humanize"
	"github.com/samber/lo"
)

type errorMsg struct {
	err error
}

func (e errorMsg) Error() string { return e.err.Error() }

func errorCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return errorMsg{err}
	}
}

type model struct {
	files        []File
	cli          *CLI
	choices      []File
	selected     bool
	currentIndex int

	quitting bool
	list     list.Model
	err      error
}

const (
	bullet   = "•"
	ellipsis = "…"
)

func (p File) Description() string {
	var meta []string
	meta = append(meta, humanize.Time(p.Timestamp))

	_, err := os.Stat(p.To)
	if os.IsNotExist(err) {
		return "(already might have been deleted)"
	}
	meta = append(meta, filepath.Dir(p.From))

	return strings.Join(meta, " "+bullet+" ")
}

func (p File) Title() string {
	fi, err := os.Stat(p.To)
	if err != nil {
		return p.Name + "?"
	}
	if fi.IsDir() {
		return p.Name + "/"
	}
	return p.Name
}

func (p File) FilterValue() string {
	return p.Name
}

var _ list.DefaultItem = (*File)(nil)

type inventoryLoadedMsg struct {
	files []list.Item
	err   error
}

func (m model) loadInventory() tea.Msg {
	files := m.cli.inventory.Files
	if len(files) == 0 {
		return errorMsg{errors.New("no deleted files found")}
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Timestamp.After(files[j].Timestamp)
	})
	files = lo.Filter(files, func(file File, index int) bool {
		// filter not found inventory out
		_, err := os.Stat(file.To)
		if os.IsNotExist(err) {
			return false
		}
		return true
	})
	items := make([]list.Item, len(files))
	for i, file := range files {
		items[i] = file
	}
	return inventoryLoadedMsg{files: items}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.loadInventory,
		m.list.StartSpinner(),
	)
}

type (
	keyMap struct {
		Quit     key.Binding
		Select   key.Binding
		DeSelect key.Binding
	}
	listAdditionalKeyMap struct {
		Enter key.Binding
	}
	detailKeyMap struct {
		Back key.Binding
	}
)

var (
	keys = keyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		Select: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "select"),
		),
		DeSelect: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("s+tab", "de-select"),
		),
	}
	listAdditionalKeys = listAdditionalKeyMap{
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "ok"),
		),
	}
	detailKeys = detailKeyMap{
		Back: key.NewBinding(
			key.WithKeys("backspace"),
			key.WithHelp("backspace", "list"),
		),
	}
)

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Select, k.DeSelect}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp(), {}}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, keys.Select):
			if m.list.FilterState() != list.Filtering {
				item, ok := m.list.SelectedItem().(File)
				if !ok {
					break
				}
				if item.isSelected() {
					selectionManager.Remove(item)
				} else {
					selectionManager.Add(item)
				}
				m.list.CursorDown()
			}

		case key.Matches(msg, keys.DeSelect):
			if m.list.FilterState() != list.Filtering {
				item, ok := m.list.SelectedItem().(File)
				if !ok {
					break
				}
				if item.isSelected() {
					selectionManager.Remove(item)
				}
				m.list.CursorUp()
			}

		case key.Matches(msg, listAdditionalKeys.Enter):
			if m.list.FilterState() != list.Filtering {
				files := selectionManager.items
				if len(files) == 0 {
					file, ok := m.list.SelectedItem().(File)
					if ok {
						m.choices = append(m.choices, file)
						m.selected = true
					}
				} else {
					m.choices = files
					m.selected = true
				}
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		// m.list.SetSize(msg.Width, msg.Height)
		m.list.SetWidth(msg.Width)

	case inventoryLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		for _, file := range msg.files {
			m.files = append(m.files, file.(File))
		}
		m.list.SetItems(msg.files)

	case errorMsg:
		m.quitting = true
		m.err = msg
		return m, tea.Quit
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	s := ""
	if m.err != nil {
		s += fmt.Sprintf("error happen %s", m.err)
	} else {
		if m.selected || m.quitting {
			return s
		}
		s += m.list.View()
	}
	return s + "\n"
}

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	currentItemStyle  = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170")).Width(150)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("#00ff00"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

type FileDelegate struct{}

func (h FileDelegate) Height() int                               { return 1 }
func (h FileDelegate) Spacing() int                              { return 0 }
func (h FileDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (h FileDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	file, ok := listItem.(File)
	if !ok {
		return
	}
	var str string
	if file.isSelected() {
		str = selectedItemStyle.Render(fmt.Sprintf("%d. %s (%d)", index+1, file.Name, selectionManager.IndexOf(file)+1))
	} else {
		str = fmt.Sprintf("%d. %s", index+1, file.Name)
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			var str = []string{"> "}
			return currentItemStyle.Render(append(str, s...)...)
		}
	}

	fmt.Fprint(w, fn(str))
}

type RestoreStyles struct {
	NormalTitle   lipgloss.Style
	NormalDesc    lipgloss.Style
	CursorTitle   lipgloss.Style
	CursorDesc    lipgloss.Style
	SelectedTitle lipgloss.Style
	SelectedDesc  lipgloss.Style
}
type RestoreDelegate struct {
	Styles  RestoreStyles
	height  int
	spacing int
}

func newRestoreDelegate() RestoreDelegate {
	const defaultHeight = 2
	const defaultSpacing = 1
	return RestoreDelegate{
		Styles:  newRestoreStyles(),
		height:  defaultHeight,
		spacing: defaultSpacing,
	}
}

func (d RestoreDelegate) Height() int                               { return 1 }
func (d RestoreDelegate) Spacing() int                              { return 0 }
func (d RestoreDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d RestoreDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	// file, ok := item.(File)
	// if !ok {
	// 	return
	// }
	// var str string
	// if file.isSelected() {
	// 	str = selectedItemStyle.Render(fmt.Sprintf("%d. %s (%d)", index+1, file.Name, selectionManager.IndexOf(file)+1))
	// } else {
	// 	str = fmt.Sprintf("%d. %s", index+1, file.Name)
	// }
	//
	// fn := itemStyle.Render
	// if index == m.Index() {
	// 	fn = func(s ...string) string {
	// 		var str = []string{"> "}
	// 		return currentItemStyle.Render(append(str, s...)...)
	// 	}
	// }
	//
	// fmt.Fprint(w, fn(str))
	var (
		title, desc string
		// matchedRunes []int
		s = &d.Styles
	)

	if i, ok := item.(DefaultItem); ok {
		title = i.Title()
		desc = i.Description()
	} else {
		return
	}

	if m.Width() <= 0 {
		// short-circuit
		return
	}

	// Prevent text from exceeding list width
	textwidth := m.Width() - s.NormalTitle.GetPaddingLeft() - s.NormalTitle.GetPaddingRight()
	title = ansi.Truncate(title, textwidth, ellipsis)
	var lines []string
	for i, line := range strings.Split(desc, "\n") {
		if i >= d.height-1 {
			break
		}
		lines = append(lines, ansi.Truncate(line, textwidth, ellipsis))
	}
	desc = strings.Join(lines, "\n")

	// Conditions
	var (
		isSelected = index == m.Index()
		// emptyFilter = m.FilterState() == list.Filtering && m.FilterValue() == ""
		// isFiltered  = m.FilterState() == list.Filtering || m.FilterState() == list.FilterApplied
	)

	// // if isFiltered && index < len(m.filteredItems) {
	// if isFiltered {
	// 	// Get indices of matched characters
	// 	matchedRunes = m.MatchesForItem(index)
	// }

	file, ok := item.(File)
	if !ok {
		return
	}
	if isSelected && m.FilterState() != list.Filtering {
		// if isFiltered {
		// 	// Highlight matches
		// 	unmatched := s.SelectedTitle.Inline(true)
		// 	matched := unmatched.Inherit(s.FilterMatch)
		// 	title = lipgloss.StyleRunes(title, matchedRunes, matched, unmatched)
		// }
		title = s.CursorTitle.Render(title)
		desc = s.CursorDesc.Render(desc)
	} else if file.isSelected() && m.FilterState() != list.Filtering {
		title = s.SelectedTitle.Render(title)
		desc = s.SelectedDesc.Render(desc)
	} else {
		title = s.NormalTitle.Render(title)
		desc = s.NormalDesc.Render(desc)
	}

	fmt.Fprintf(w, "%s\n%s", title, desc) //nolint: errcheck
}

func newRestoreStyles() RestoreStyles {
	s := RestoreStyles{}
	s.NormalTitle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"}).
		Padding(0, 0, 0, 2) //nolint:mnd

	s.NormalDesc = s.NormalTitle.
		Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"})

	s.CursorTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.AdaptiveColor{Light: "#F793FF", Dark: "#AD58B4"}).
		Foreground(lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"}).
		Padding(0, 0, 0, 1)
	s.CursorDesc = s.SelectedTitle.
		Foreground(lipgloss.AdaptiveColor{Light: "#F793FF", Dark: "#AD58B4"})

	s.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.AdaptiveColor{Light: "#F793FF", Dark: "#AD58B4"}).
		Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"}).
		Padding(0, 0, 0, 1)

	return s
}
