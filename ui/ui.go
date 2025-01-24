package ui

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"

	"github.com/babarot/gomi/config"
	"github.com/babarot/gomi/inventory"

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

type NavState uint8

const (
	INVENTORY_LIST NavState = iota
	INVENTORY_DETAILS
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
	defaultHeight = 20
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
	navState      NavState
	detailFile    File
	datefmt       string
	cannotPreview bool

	files   []File
	config  config.UI
	choices []File

	list     list.Model
	viewport viewport.Model

	err error
}

var _ list.DefaultItem = (*File)(nil)

type inventoryLoadedMsg struct {
	files []list.Item
	err   error
}

func (m Model) loadInventory() tea.Msg {
	files := m.files
	slog.Debug("loadInventory starts")
	if len(files) == 0 {
		return errorMsg{errors.New("no deleted files found")}
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].File.Timestamp.After(files[j].File.Timestamp)
	})
	files = lo.Reject(files, func(f File, index int) bool {
		_, err := os.Stat(f.File.To)
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

type (
	keyMap struct {
		Quit     key.Binding
		Select   key.Binding
		DeSelect key.Binding
	}
	listAdditionalKeyMap struct {
		Enter key.Binding
		Space key.Binding
	}
	detailKeyMap struct {
		Up           key.Binding
		Down         key.Binding
		PreviewUp    key.Binding
		PreviewDown  key.Binding
		Esc          key.Binding
		AtSign       key.Binding
		GotoTop      key.Binding
		GotoBottom   key.Binding
		HalfPageUp   key.Binding
		HalfPageDown key.Binding
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
		Space: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "info"),
		),
	}
	detailKeys = detailKeyMap{
		PreviewUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "preview up"),
		),
		PreviewDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "preview down"),
		),
		Up: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "prev"),
		),
		Down: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next"),
		),
		Esc: key.NewBinding(
			key.WithKeys("esc"), // space itself should be already defined another part
			key.WithHelp("space/esc", "back"),
		),
		AtSign: key.NewBinding(
			key.WithKeys("@"),
			key.WithHelp("@", "datefmt"),
		),
		GotoTop:      key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "go to start")),
		GotoBottom:   key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "go to end")),
		HalfPageUp:   key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "½ page up")),
		HalfPageDown: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "½ page down")),
	}
)

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		detailKeys.Up, detailKeys.Down,
		detailKeys.PreviewUp, detailKeys.PreviewDown,
		detailKeys.AtSign,
		detailKeys.Esc,
	}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp(), {}}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

		case key.Matches(msg, detailKeys.AtSign):
			switch m.navState {
			case INVENTORY_DETAILS:
				switch m.datefmt {
				case datefmtRel:
					m.datefmt = datefmtAbs
				case datefmtAbs:
					m.datefmt = datefmtRel
				}
			}

		case key.Matches(
			msg, detailKeys.PreviewUp, detailKeys.PreviewDown,
			detailKeys.HalfPageUp, detailKeys.HalfPageDown,
		):
			switch m.navState {
			case INVENTORY_DETAILS:
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)

			}

		case key.Matches(msg, detailKeys.GotoTop):
			switch m.navState {
			case INVENTORY_DETAILS:
				m.viewport.GotoTop()
			}

		case key.Matches(msg, detailKeys.GotoBottom):
			switch m.navState {
			case INVENTORY_DETAILS:
				m.viewport.GotoBottom()
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

		case key.Matches(msg, detailKeys.Esc):
			switch m.navState {
			case INVENTORY_DETAILS:
				m.navState = INVENTORY_LIST
			}

		case key.Matches(msg, listAdditionalKeys.Space):
			switch m.navState {
			case INVENTORY_LIST:
				if m.list.FilterState() != list.Filtering {
					file, ok := m.list.SelectedItem().(File)
					if ok {
						cmds = append(cmds, getInventoryDetails(file))
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
					slog.Debug("key input: enter", slog.Any("selected_files", files))
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
		m.navState = INVENTORY_DETAILS
		m.detailFile = msg.file
		m.cannotPreview = false
		m.viewport = m.newViewportModel(msg.file)

	case errorMsg:
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

func (m Model) View() string {
	defer color.Unset()
	s := ""

	if m.err != nil {
		s += fmt.Sprintf("error happen %s", m.err)
		return s
	}

	switch m.navState {
	case INVENTORY_LIST:
		s += m.list.View()
	case INVENTORY_DETAILS:
		s += renderDetailed(m)
		s += "\n" + lipgloss.NewStyle().Margin(1, 2).Render(help.New().View(keys))
	case QUITTING:
		return s
	}

	if len(m.choices) > 0 {
		// do not render when nothing is selected
		return ""
	}

	return s
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
	viewportModel.KeyMap = viewport.KeyMap{
		Up:           key.NewBinding(key.WithKeys("k", "up")),
		Down:         key.NewBinding(key.WithKeys("j", "down")),
		HalfPageUp:   key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "½ page up")),
		HalfPageDown: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "½ page down")),
	}
	content, err := file.Browse()
	if err != nil {
		m.cannotPreview = true
	}
	viewportModel.SetContent(content)
	return viewportModel
}

func Run(filteredFiles []inventory.File, cfg config.UI) ([]inventory.File, error) {
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

	// TODO: configable?
	// l := list.New(items, ClassicDelegate{}, defaultWidth, defaultHeight)
	l := list.New(items, NewRestoreDelegate(cfg, files), defaultWidth, defaultHeight)

	switch cfg.Paginator {
	case "arabic":
		l.Paginator.Type = paginator.Arabic
	case "dots":
		l.Paginator.Type = paginator.Dots
	default:
		l.Paginator.Type = paginator.Dots
	}

	l.Title = ""
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{listAdditionalKeys.Enter, listAdditionalKeys.Space}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{listAdditionalKeys.Enter, listAdditionalKeys.Space, keys.Quit, keys.Select, keys.DeSelect}
	}
	l.DisableQuitKeybindings()
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)

	m := Model{
		navState: INVENTORY_LIST,
		datefmt:  datefmtRel,
		files:    files,
		config:   cfg,
		list:     l,
		viewport: viewport.Model{},
	}

	returnModel, err := tea.NewProgram(m).Run()
	if err != nil {
		return []inventory.File{}, err
	}

	choices := returnModel.(Model).choices
	if returnModel.(Model).navState == QUITTING {
		if msg := cfg.ByeMessage; msg != "" {
			fmt.Println(msg)
		}
		return []inventory.File{}, nil
	}

	invFiles := make([]inventory.File, len(choices))
	for i, file := range choices {
		invFiles[i] = file.File
	}
	return invFiles, nil
}
