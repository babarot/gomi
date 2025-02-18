package ui

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/ui/keys"
	"github.com/fatih/color"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
)

type ListDensityType uint8

const (
	Compact ListDensityType = iota
	Spacious
)

const (
	CompactDensityVal  = "compact"
	SpaciousDensityVal = "spacious"
)

const (
	bullet   = "•"
	ellipsis = "…"

	defaultWidth  = 66 // 56
	defaultHeight = 26
)

var (
	errCannotPreview = errors.New("cannot preview")

	ErrInputCanceled = errors.New("input is canceled")
)

type DetailsMsg struct {
	file File
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

type Model struct {
	trashManager *trash.Manager
	state        *ViewState

	listKeys   *keys.ListKeyMap
	detailKeys *keys.DetailKeyMap

	detailFile File
	files      []File
	config     config.UI
	choices    []File

	styles dialogStyles

	help     help.Model
	list     list.Model
	viewport viewport.Model

	err error
}

type dialogStyles struct {
	dialog lipgloss.Style
}

func initStyles(c config.StyleConfig) dialogStyles {
	s := dialogStyles{}
	s.dialog = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(c.DeletionDialog)).
		Foreground(lipgloss.Color(c.DeletionDialog)).
		Bold(true).
		Padding(1, 1).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true).
		Align(lipgloss.Center).
		Width(defaultWidth - 4)

	return s
}

var _ list.DefaultItem = (*File)(nil)

type inventoryLoadedMsg struct {
	files []list.Item
	err   error
}

type refreshFilesMsg struct {
	files []list.Item
	err   error
}

func (m Model) loadInventory() tea.Msg {
	files := m.files
	slog.Info("loadInventory starts", "len(files)", len(files))

	if len(files) == 0 {
		return errorMsg{errors.New("no deleted files found")}
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].File.DeletedAt.After(files[j].File.DeletedAt)
	})
	files = lo.Reject(files, func(f File, index int) bool {
		_, err := os.Stat(f.File.TrashPath)
		return os.IsNotExist(err)
	})
	items := make([]list.Item, len(files))
	for i, file := range files {
		items[i] = file
	}
	return inventoryLoadedMsg{files: items}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadInventory,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state.current {
		case CONFIRM_VIEW:
			return m.handleConfirmViewKeyPress(msg)
		default:
			return m.handleDefaultKeyPress(msg)
		}

	case refreshFilesMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.list.SetItems(msg.files)
		return m, nil

	case DetailsMsg:
		m.state.SetView(DETAIL_VIEW)
		m.detailFile = msg.file
		m.state.preview.available = false // reset everytime
		m.viewport = m.newViewportModel(msg.file)
		return m, nil

	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)

	case inventoryLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.list.SetItems(msg.files)

	case errorMsg:
		m.state.SetView(QUITTING)
		m.err = msg
		return m, tea.Quit
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	defer color.Unset()

	if m.err != nil {
		slog.Error("rendering of the view has stopped", "error", m.err)
		return m.err.Error()
	}

	if len(m.choices) > 0 {
		return ""
	}

	switch m.state.current {
	case LIST_VIEW:
		return m.list.View()

	case DETAIL_VIEW:
		detailView := renderDetailed(m)
		helpView := lipgloss.NewStyle().Margin(1, 2).Render(m.help.View(m.detailKeys))
		return detailView + "\n" + helpView

	case CONFIRM_VIEW:
		return m.renderDeleteConfirmation()

	case QUITTING:
		return ""

	default:
		return ""
	}
}

func getInventoryDetails(file File) tea.Cmd {
	return func() tea.Msg {
		return DetailsMsg{file: file}
	}
}

func (m *Model) newViewportModel(file File) viewport.Model {
	viewportModel := viewport.New(defaultWidth, 15-lipgloss.Height(m.previewHeader()))
	viewportModel.KeyMap = keys.PreviewKeys
	content, err := file.Browse()
	if err != nil {
		slog.Warn("file.Browse returned error", "content", err)
		m.state.preview.available = true
	}
	viewportModel.SetContent(content)
	return viewportModel
}

func RenderList(manager *trash.Manager, filteredFiles []*trash.File, c *config.Config) ([]*trash.File, error) {
	cfg := c.UI
	var items []list.Item
	var files []File
	for _, file := range filteredFiles {
		items = append(items, File{File: file})
		files = append(files, File{
			File:            file,
			dirListCommand:  cfg.Preview.DirectoryCommand,
			syntaxHighlight: cfg.Preview.SyntaxHighlight,
			colorscheme:     cfg.Preview.Colorscheme,
		})
	}

	d := NewRestoreDelegate(cfg, files)

	if !c.Core.Delete.Disable {
		keys.ListKeys.AddDeleteKey()
		keys.DetailKeys.AddDeleteKey()
	}

	d.ShortHelpFunc = keys.ListKeys.ShortHelp
	d.FullHelpFunc = keys.ListKeys.FullHelp
	l := list.New(items, d, defaultWidth, defaultHeight)
	switch cfg.Paginator {
	case "arabic":
		l.Paginator.Type = paginator.Arabic
	case "dots":
		l.Paginator.Type = paginator.Dots
	default:
		l.Paginator.Type = paginator.Dots
	}

	l.Title = ""
	l.DisableQuitKeybindings()
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)

	m := Model{
		state:        NewViewState(),
		trashManager: manager,
		listKeys:     keys.ListKeys,
		detailKeys:   keys.DetailKeys,
		files:        files,
		config:       cfg,
		list:         l,
		viewport:     viewport.Model{},
		styles:       initStyles(cfg.Style),
		help:         help.New(),
	}

	returnModel, err := tea.NewProgram(m).Run()
	if err != nil {
		return []*trash.File{}, err
	}

	choices := returnModel.(Model).choices
	if returnModel.(Model).state.current == QUITTING {
		if msg := cfg.ExitMessage; msg != "" {
			fmt.Println(msg)
		}
		return []*trash.File{}, nil
	}

	trashFiles := make([]*trash.File, len(choices))
	for i, file := range choices {
		trashFiles[i] = file.File
	}
	return trashFiles, nil
}

func (m Model) deletePermanentlyCmd(files ...File) tea.Cmd {
	return func() tea.Msg {
		for _, file := range files {
			slog.Debug("permanently delete", "file", file.TrashPath)
			if err := m.trashManager.Remove(file.File); err != nil {
				return refreshFilesMsg{err: err}
			}
		}
		origin := m.files
		origin = lo.Reject(origin, func(f File, index int) bool {
			_, err := os.Stat(f.File.TrashPath)
			return os.IsNotExist(err)
		})
		items := make([]list.Item, len(origin))
		for i, file := range origin {
			items[i] = file
		}
		return refreshFilesMsg{files: items}
	}
}

func (m Model) handleConfirmViewKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.state.SetView(m.state.previous)
		files := selectionManager.items
		if len(files) > 0 {
			return m, m.deletePermanentlyCmd(files...)
		}
		if file, ok := m.list.SelectedItem().(File); ok {
			return m, m.deletePermanentlyCmd(file)
		}
		return m, nil

	case "n", "N":
		m.state.SetView(m.state.previous)
		return m, nil

	case "ctrl+c", "q":
		m.state.SetView(QUITTING)
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleDefaultKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	slog.Debug("Key pressed", "key", msg.String())
	switch {
	case key.Matches(msg, m.listKeys.Quit):
		m.state.SetView(QUITTING)
		return m, tea.Quit

	case key.Matches(msg, m.listKeys.Select):
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
			if m.state.current == DETAIL_VIEW {
				cmds = append(cmds, getInventoryDetails(item))
			}
		}

	case key.Matches(msg, m.listKeys.DeSelect):
		if m.list.FilterState() != list.Filtering {
			item, ok := m.list.SelectedItem().(File)
			if !ok {
				break
			}
			if item.isSelected() {
				selectionManager.Remove(item)
			}
			m.list.CursorUp()
			if m.state.current == DETAIL_VIEW {
				cmds = append(cmds, getInventoryDetails(item))
			}
		}

	case key.Matches(msg, m.detailKeys.AtSign):
		if m.state.current == DETAIL_VIEW {
			m.state.ToggleDateFormat()
			m.state.ToggleOriginPath()
		}

	case key.Matches(msg, m.detailKeys.PreviewUp, m.detailKeys.PreviewDown,
		m.detailKeys.HalfPageUp, m.detailKeys.HalfPageDown):
		if m.state.current == DETAIL_VIEW {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}

	case key.Matches(msg, m.detailKeys.GotoTop):
		if m.state.current == DETAIL_VIEW {
			m.viewport.GotoTop()
		}

	case key.Matches(msg, m.detailKeys.GotoBottom):
		if m.state.current == DETAIL_VIEW {
			m.viewport.GotoBottom()
		}

	case key.Matches(msg, m.detailKeys.Prev):
		if m.state.current == DETAIL_VIEW {
			m.list.CursorUp()
			file, ok := m.list.SelectedItem().(File)
			if ok {
				cmds = append(cmds, getInventoryDetails(file))
			}
		}

	case key.Matches(msg, m.detailKeys.Next):
		if m.state.current == DETAIL_VIEW {
			m.list.CursorDown()
			file, ok := m.list.SelectedItem().(File)
			if ok {
				cmds = append(cmds, getInventoryDetails(file))
			}
		}

	case key.Matches(msg, m.detailKeys.Esc):
		switch m.state.current {
		case LIST_VIEW:
			selectionManager = &SelectionManager{items: []File{}}
		case DETAIL_VIEW:
			m.state.SetView(LIST_VIEW)
		}

	case key.Matches(msg, m.listKeys.Delete):
		if m.state.current == LIST_VIEW || m.state.current == DETAIL_VIEW {
			if m.list.FilterState() != list.Filtering {
				m.state.SetView(CONFIRM_VIEW)
				slog.Debug("pressed delete key", "state", CONFIRM_VIEW)
			}
		}

	case key.Matches(msg, m.listKeys.Space):
		switch m.state.current {
		case LIST_VIEW:
			if m.list.FilterState() != list.Filtering {
				file, ok := m.list.SelectedItem().(File)
				if ok {
					cmds = append(cmds, getInventoryDetails(file))
				}
			}
		case DETAIL_VIEW:
			m.state.detail.showOrigin = true
			m.state.detail.dateFormat = DateFormatRelative
			m.state.SetView(LIST_VIEW)
		}

	case key.Matches(msg, m.listKeys.Enter):
		if m.state.current == LIST_VIEW {
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
				slog.Debug("key input: enter", slog.Any("selected_files", m.choices))
				return m, tea.Quit
			}
		}

	case key.Matches(msg, m.detailKeys.Help):
		m.help.ShowAll = !m.help.ShowAll
	}

	var cmd tea.Cmd
	if m.state.current == LIST_VIEW {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}
