package ui

import (
	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/ui/keys"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	defaultWidth  = 56
	defaultHeight = 20
)

// Model represents the main application model.
type Model struct {
	// Core state
	state      ViewState
	dateFormat DateFormat
	err        error

	// Configuration
	config config.UI
	width  int
	height int

	// List view
	list          list.Model
	items         []*Item
	selectedItems map[string]struct{}
	delegate      *ListDelegate

	// Detail view
	viewport      viewport.Model
	currentItem   *Item
	previewLoaded bool

	// Common components
	help       help.Model
	listKeys   *keys.ListKeyMap
	detailKeys *keys.DetailKeyMap
}

// New creates a new Model with the provided files and configuration.
func New(files []*trash.File, cfg config.UI) Model {
	var items []*Item
	var listItems []list.Item

	// Create items
	for _, file := range files {
		item := NewItem(file, cfg)
		items = append(items, item)
		listItems = append(listItems, item)
	}

	// Initialize delegate with selection tracking
	delegate := NewListDelegate(cfg)

	// Create the model first so we can reference it
	m := Model{
		state:         ListView,
		dateFormat:    DateFormatRelative,
		config:        cfg,
		width:         defaultWidth,
		height:        defaultHeight,
		items:         items,
		selectedItems: make(map[string]struct{}),
		delegate:      delegate,
		help:          help.New(),
		listKeys:      keys.ListKeys,
		detailKeys:    keys.DetailKeys,
	}

	// Set the model as the delegate for selection management
	delegate.parentModel = &m

	// Initialize list
	l := list.New(listItems, delegate, defaultWidth, defaultHeight)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	l.DisableQuitKeybindings()
	l.SetFilteringEnabled(true)
	l.Styles.FilterCursor = l.Styles.FilterCursor.Foreground(lipgloss.Color(cfg.Style.ListView.Cursor))
	m.list = l

	return m
}

// Init initializes the model. Part of tea.Model interface.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles all the messages and events. Part of tea.Model interface.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle global keypresses
		if key.Matches(msg, m.listKeys.Quit) {
			m.state = Quitting
			return m, tea.Quit
		}

		// Delegate to the current view
		switch m.state {
		case ListView:
			return m.updateList(msg)
		case DetailView:
			return m.updateDetail(msg)
		}

	case tea.WindowSizeMsg:
		// Keep the fixed size instead of using the full window
		m.list.SetSize(defaultWidth, defaultHeight)

		if m.currentItem != nil {
			headerHeight := lipgloss.Height(m.renderDetailHeader())
			m.viewport = viewport.New(defaultWidth, defaultHeight-headerHeight)
			m.viewport.SetContent(m.currentItem.preview)
		}

	case error:
		m.err = msg
		return m, tea.Quit
	}

	return m, tea.Batch(cmds...)
}

// View renders the current view. Part of tea.Model interface.
func (m Model) View() string {
	if m.err != nil {
		return m.err.Error()
	}

	switch m.state {
	case ListView:
		return m.list.View()
	case DetailView:
		return m.renderDetail()
	default:
		return ""
	}
}

// currentListItem returns the currently selected item in the list.
func (m Model) currentListItem() *Item {
	item, ok := m.list.SelectedItem().(*Item)
	if !ok {
		return nil
	}
	return item
}

// toggleSelection toggles the selection state of the given item.
func (m *Model) toggleSelection(item *Item) {
	if _, selected := m.selectedItems[item.File().Name]; selected {
		delete(m.selectedItems, item.File().Name)
	} else {
		m.selectedItems[item.File().Name] = struct{}{}
	}
}

// isSelected returns whether the given item is selected.
func (m Model) isSelected(item *Item) bool {
	_, ok := m.selectedItems[item.File().Name]
	return ok
}

// SelectedFiles returns the list of selected files.
func (m Model) SelectedFiles() []*trash.File {
	var files []*trash.File

	// If no files are explicitly selected, return the currently highlighted file
	if len(m.selectedItems) == 0 {
		if item := m.currentListItem(); item != nil {
			return []*trash.File{item.File()}
		}
		return nil
	}

	// Return all selected files
	for _, item := range m.items {
		if _, ok := m.selectedItems[item.File().Name]; ok {
			files = append(files, item.File())
		}
	}
	return files
}

// Canceled returns whether the user canceled the operation.
func (m Model) Canceled() bool {
	return m.state == Quitting
}

// previewCmd loads the preview content for the specified item.
func previewCmd(item *Item) tea.Cmd {
	return func() tea.Msg {
		if err := item.LoadPreview(); err != nil {
			return err
		}
		return nil
	}
}
