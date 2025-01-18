package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
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
	bullet = "•"
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
		Quit   key.Binding
		Select key.Binding
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
			key.WithKeys("tab", " "),
			key.WithHelp("tab", "select"),
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
	return []key.Binding{detailKeys.Back, k.Quit, k.Select}
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
			switch m.files[m.list.Index()].selected {
			case true:
				m.files[m.list.Index()].selected = false
			case false:
				m.files[m.list.Index()].selected = true
			}
			// switch c.options[c.currentIndex].selected {
			// case true:
			// 	c.options[c.currentIndex].selected = false
			// case false:
			// 	c.options[c.currentIndex].selected = true
			// }
			// file, ok := m.list.SelectedItem().(File)
			// if !ok {
			// 	break
			// }
			// if file.isSelected() {
			// 	selectionManager.Remove(file)
			// } else {
			// 	selectionManager.Add(file)
			// }

		case key.Matches(msg, listAdditionalKeys.Enter):
			if m.list.FilterState() != list.Filtering {
				files := lo.Filter(m.files, func(file File, index int) bool {
					return file.selected
				})
				if len(files) == 0 {
					// selectedItem := m.list.SelectedItem()
					// if selectedItem == nil {
					// 	return nil
					// }
					// w, ok := selectedItem.(ItemWrapper)
					// if !ok {
					// 	// this should never happen
					// 	return nil
					// }
					// return &w
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
