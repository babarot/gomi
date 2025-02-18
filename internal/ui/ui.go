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

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/samber/lo"
)

type ViewType uint8

const (
	LIST_VIEW ViewType = iota
	DETAIL_VIEW
	CONFIRM_VIEW
	QUITTING
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

	datefmtRel = "relative"
	datefmtAbs = "absolute"

	defaultWidth  = 56
	defaultHeight = 26
)

var (
	errCannotPreview = errors.New("cannot preview")

	ErrInputCanceled = errors.New("input is canceled")
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

type Model struct {
	trashManager *trash.Manager

	listKeys   *keys.ListKeyMap
	detailKeys *keys.DetailKeyMap

	viewType       ViewType
	detailFile     File
	datefmt        string
	cannotPreview  bool
	locationOrigin bool

	files   []File
	config  config.UI
	choices []File

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
		switch m.viewType {
		case CONFIRM_VIEW:
			switch msg.String() {
			case "y", "Y":
				slog.Debug("replied yes to delete permanently")
				m.viewType = LIST_VIEW
				files := selectionManager.items
				if len(files) > 0 {
					return m, m.deletePermanentlyCmd(files...)
				}
				file, ok := m.list.SelectedItem().(File)
				if ok {
					return m, m.deletePermanentlyCmd(file)
				}
			case "n", "N":
				slog.Debug("replied no to delete permanently")
				m.viewType = LIST_VIEW
			case "ctrl+c", "q":
				m.viewType = QUITTING
				return m, tea.Quit
			default:
				slog.Debug("waiting for reply to delete permanently")
				return m, nil
			}
		}

		slog.Debug("Key pressed", "key", msg.String())
		switch {
		case key.Matches(msg, m.listKeys.Quit):
			m.viewType = QUITTING
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
				if m.viewType == DETAIL_VIEW {
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
				if m.viewType == DETAIL_VIEW {
					cmds = append(cmds, getInventoryDetails(item))
				}
			}

		case key.Matches(msg, m.detailKeys.AtSign):
			switch m.viewType {
			case DETAIL_VIEW:
				switch m.datefmt {
				case datefmtRel:
					m.datefmt = datefmtAbs
				case datefmtAbs:
					m.datefmt = datefmtRel
				}
				m.locationOrigin = !m.locationOrigin
			}

		case key.Matches(
			msg, m.detailKeys.PreviewUp, m.detailKeys.PreviewDown,
			m.detailKeys.HalfPageUp, m.detailKeys.HalfPageDown,
		):
			switch m.viewType {
			case DETAIL_VIEW:
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
			}

		case key.Matches(msg, m.detailKeys.GotoTop):
			switch m.viewType {
			case DETAIL_VIEW:
				m.viewport.GotoTop()
			}

		case key.Matches(msg, m.detailKeys.GotoBottom):
			switch m.viewType {
			case DETAIL_VIEW:
				m.viewport.GotoBottom()
			}

		case key.Matches(msg, m.detailKeys.Prev):
			switch m.viewType {
			case DETAIL_VIEW:
				m.list.CursorUp()
				file, ok := m.list.SelectedItem().(File)
				if ok {
					cmds = append(cmds, getInventoryDetails(file))
				}
			}

		case key.Matches(msg, m.detailKeys.Next):
			switch m.viewType {
			case DETAIL_VIEW:
				m.list.CursorDown()
				file, ok := m.list.SelectedItem().(File)
				if ok {
					cmds = append(cmds, getInventoryDetails(file))
				}
			}

		case key.Matches(msg, m.detailKeys.Esc):
			switch m.viewType {
			case LIST_VIEW:
				selectionManager = &SelectionManager{items: []File{}}
			case DETAIL_VIEW:
				m.viewType = LIST_VIEW
			}

		case key.Matches(msg, m.listKeys.Delete):
			switch m.viewType {
			case LIST_VIEW, DETAIL_VIEW:
				if m.list.FilterState() != list.Filtering {
					m.viewType = CONFIRM_VIEW
					slog.Debug("pressed delete key", "state", CONFIRM_VIEW)
				}
			}

		case key.Matches(msg, m.listKeys.Space):
			switch m.viewType {
			case LIST_VIEW:
				if m.list.FilterState() != list.Filtering {
					file, ok := m.list.SelectedItem().(File)
					if ok {
						cmds = append(cmds, getInventoryDetails(file))
					}
				}
			case DETAIL_VIEW:
				// TODO: create another cmd (e.g. show list cmd)
				m.locationOrigin = true
				m.datefmt = datefmtRel
				m.viewType = LIST_VIEW
			}

		case key.Matches(msg, m.listKeys.Enter):
			switch m.viewType {
			case LIST_VIEW:
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

	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)

	case inventoryLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.list.SetItems(msg.files)

	case refreshFilesMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.list.SetItems(msg.files)

	case DetailsMsg:
		m.viewType = DETAIL_VIEW
		m.detailFile = msg.file
		m.cannotPreview = false
		m.viewport = m.newViewportModel(msg.file)

	case errorMsg:
		m.viewType = QUITTING
		m.err = msg
		return m, tea.Quit
	}

	var cmd tea.Cmd
	switch m.viewType {
	case LIST_VIEW:
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
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

	switch m.viewType {
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

// TODO: remove?
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

func (m *Model) newViewportModel(file File) viewport.Model {
	viewportModel := viewport.New(defaultWidth, 15-lipgloss.Height(m.previewHeader()))
	viewportModel.KeyMap = keys.PreviewKeys
	content, err := file.Browse()
	if err != nil {
		m.cannotPreview = true
	}
	viewportModel.SetContent(content)
	return viewportModel
}

func RenderList(manager *trash.Manager, filteredFiles []*trash.File, cfg config.UI) ([]*trash.File, error) {
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
		trashManager:   manager,
		listKeys:       keys.ListKeys,
		detailKeys:     keys.DetailKeys,
		viewType:       LIST_VIEW,
		datefmt:        datefmtRel,
		locationOrigin: true,
		files:          files,
		config:         cfg,
		list:           l,
		viewport:       viewport.Model{},
		styles:         initStyles(cfg.Style),
		help:           help.New(),
	}

	returnModel, err := tea.NewProgram(m).Run()
	if err != nil {
		return []*trash.File{}, err
	}

	choices := returnModel.(Model).choices
	if returnModel.(Model).viewType == QUITTING {
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
