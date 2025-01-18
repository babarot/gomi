package main

import (
	"errors"
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
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
	files    []File
	cli      *CLI
	choice   File
	selected bool

	quitting bool
	list     list.Model
	err      error
}

// Description implements list.DefaultItem
func (p File) Description() string {
	return humanize.Time(p.Timestamp)
}

// Title implements list.DefaultItem
func (p File) Title() string {
	return p.Name
}

// FilterValue implements list.Item
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
		Quit key.Binding
	}
	listAdditionalKeyMap struct {
		Enter key.Binding
	}
	detailKeyMap struct {
		Back key.Binding
		Tab  key.Binding
	}
)

var (
	keys = keyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
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
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "units"),
		),
	}
)

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{detailKeys.Tab, detailKeys.Back, k.Quit}
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

		case key.Matches(msg, listAdditionalKeys.Enter):
			if m.list.FilterState() != list.Filtering {
				file, ok := m.list.SelectedItem().(File)
				if ok {
					m.choice = file
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
		if m.selected {
			return s
		}
		s += m.list.View()
	}
	return s + "\n"
}
