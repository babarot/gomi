package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/samber/lo"
)

type NavState int

const (
	INVENTORY_LIST NavState = iota
	// LOADING_INVENTORY_LIST

	INVENTORY_DETAILS
	// LOADING_INVENTORY_DETAILS

	QUITTING
)

type DetailsMsg struct {
	file File
}
type GotInventorysMsg struct {
	files []list.Item
}

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
	navState   NavState
	detailFile File

	files        []File
	cli          *CLI
	choices      []File
	currentIndex int

	// quitting bool
	list list.Model
	err  error
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
		Info  key.Binding
		Esc   key.Binding
	}
	detailKeyMap struct {
		Up   key.Binding
		Down key.Binding
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
		Info: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "info"),
		),
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
	}
	detailKeys = detailKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Back: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "back"),
		),
	}
)

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{detailKeys.Up, detailKeys.Down, detailKeys.Back}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp(), {listAdditionalKeys.Esc}}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// var cmd tea.Cmd
	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.navState = QUITTING
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
				if m.navState == INVENTORY_DETAILS {
					cmds = append(cmds, getInventoryDetails(item))
				}
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
				if m.navState == INVENTORY_DETAILS {
					cmds = append(cmds, getInventoryDetails(item))
				}
			}

		case key.Matches(msg, detailKeys.Up):
			switch m.navState {
			case INVENTORY_DETAILS:
				m.list.CursorUp()
				file, ok := m.list.SelectedItem().(File)
				if ok {
					cmds = append(cmds, getInventoryDetails(file))
				}
			}
		case key.Matches(msg, detailKeys.Down):
			switch m.navState {
			case INVENTORY_DETAILS:
				m.list.CursorDown()
				file, ok := m.list.SelectedItem().(File)
				if ok {
					cmds = append(cmds, getInventoryDetails(file))
				}
			}

		case key.Matches(msg, listAdditionalKeys.Esc):
			switch m.navState {
			case INVENTORY_DETAILS:
				m.navState = INVENTORY_LIST
			}

		case key.Matches(msg, listAdditionalKeys.Info):
			switch m.navState {
			case INVENTORY_LIST:
				if m.list.FilterState() != list.Filtering {
					file, ok := m.list.SelectedItem().(File)
					if ok {
						cmds = append(cmds, getInventoryDetails(file))
						// m.navState = INVENTORY_DETAILS
					}
				}
			case INVENTORY_DETAILS:
				m.navState = INVENTORY_LIST
			}

		case key.Matches(msg, listAdditionalKeys.Enter):
			switch m.navState {
			case INVENTORY_LIST:
				if m.list.FilterState() != list.Filtering {
					files := selectionManager.items
					if len(files) == 0 {
						file, ok := m.list.SelectedItem().(File)
						if ok {
							m.choices = append(m.choices, file)
						}
					} else {
						m.choices = files
					}
					return m, tea.Quit
				}
			}
		}

	case tea.WindowSizeMsg:
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

	case DetailsMsg:
		m.detailFile = msg.file
		m.navState = INVENTORY_DETAILS

	case errorMsg:
		// m.quitting = true
		m.navState = QUITTING
		m.err = msg
		return m, tea.Quit
	}

	var cmd tea.Cmd
	switch m.navState {
	case INVENTORY_LIST:
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func renderInventoryDetails(m model) string {
	size, _ := DirSize(m.detailFile.To)
	s := fmt.Sprintf("name: %s\nfrom: %s\nto: %s\nsize: %s\n",
		m.detailFile.Name,
		m.detailFile.From,
		m.detailFile.To,
		humanize.Bytes(uint64(size)),
	)
	if m.detailFile.isSelected() {
		return lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"}).
			Render(s)
	}
	return s
}

func (m model) View() string {
	s := ""

	if m.err != nil {
		s += fmt.Sprintf("error happen %s", m.err)
		return s
	}

	switch m.navState {
	case INVENTORY_LIST:
		s += m.list.View()
	case INVENTORY_DETAILS:
		s += renderInventoryDetails(m)
		s += "\n" + lipgloss.NewStyle().Margin(1, 2).Render(help.New().View(keys))
	case QUITTING:
		return s
	}

	if len(m.choices) > 0 {
		// do not render when selected one or more
		return ""
	}

	return s
}

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	currentItemStyle  = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170")).Width(150)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("#00ff00"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

func getAllInventoryItems(files []File) tea.Msg {
	result := []list.Item{}
	for _, file := range files {
		result = append(result, file)
	}
	return GotInventorysMsg{files: result}
}

func getInventoryDetails(file File) tea.Cmd {
	return func() tea.Msg {
		return DetailsMsg{file: file}
	}
}

func DirSize(path string) (int64, error) {
	var size int64
	var mu sync.Mutex

	// Function to calculate size for a given path
	var calculateSize func(string) error
	calculateSize = func(p string) error {
		fileInfo, err := os.Lstat(p)
		if err != nil {
			return err
		}

		// Skip symbolic links to avoid counting them multiple times
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		if fileInfo.IsDir() {
			entries, err := os.ReadDir(p)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if err := calculateSize(filepath.Join(p, entry.Name())); err != nil {
					return err
				}
			}
		} else {
			mu.Lock()
			size += fileInfo.Size()
			mu.Unlock()
		}
		return nil
	}

	// Start calculation from the root path
	if err := calculateSize(path); err != nil {
		return 0, err
	}

	return size, nil
}
