package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"

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
	files []File
	cli   *CLI

	quitting bool
	list     list.Model
	err      error
}

type scrapboxPage struct {
	Title_ string `json:"title"`
	ID     string `json:"id"`
}

// Description implements list.DefaultItem
func (p scrapboxPage) Description() string {
	return p.ID
}

// Title implements list.DefaultItem
func (p scrapboxPage) Title() string {
	return p.Title_
}

// FilterValue implements list.Item
func (p scrapboxPage) FilterValue() string {
	return p.Title_
}

var _ list.DefaultItem = (*scrapboxPage)(nil)

type scrapboxPagesLoadedMsg struct {
	pages []list.Item
	err   error
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
	// if err := m.cli.inventory.open(); err != nil {
	// 	return errorMsg{errors.New("no deleted files found")}
	// }
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

func (m model) loadScrapboxPages() tea.Msg {
	resp, err := http.Get(
		"https://scrapbox.io/api/pages/help-jp",
	)
	if err != nil {
		return errorMsg{err}
	}

	defer resp.Body.Close()
	var pages []scrapboxPage
	result := struct{ Pages []scrapboxPage }{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return errorMsg{err}
	}

	pages = result.Pages
	if len(pages) == 0 {
		return scrapboxPagesLoadedMsg{pages: []list.Item{}, err: errors.New("page not found")}
		// return errorMsg{errors.New("page not found")}
	}

	items := make([]list.Item, len(pages))
	for i, page := range pages {
		items[i] = page
	}
	return scrapboxPagesLoadedMsg{pages: items}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		// m.loadScrapboxPages,
		m.loadInventory,
		m.list.StartSpinner(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// m.list.SetSize(msg.Width, msg.Height)
		m.list.SetWidth(msg.Width)

	case inventoryLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.list.StopSpinner()
		m.list.SetItems(msg.files)

	case scrapboxPagesLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.list.StopSpinner()
		m.list.SetItems(msg.pages)

	case errorMsg:
		m.quitting = true
		m.err = msg
		return m, tea.Quit
	}

	return m, cmd
}

func (m model) View() string {
	s := ""
	if m.err != nil {
		s += fmt.Sprintf("error happen %s", m.err)
	} else {
		s += m.list.View()
	}
	return s + "\n"
}
