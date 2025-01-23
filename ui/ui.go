package ui

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/babarot/gomi/config"
	"github.com/babarot/gomi/inventory"
	"github.com/babarot/gomi/utils"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/gabriel-vasile/mimetype"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/termenv"
	"github.com/samber/lo"
)

const (
	datefmtRel = "relative"
	datefmtAbs = "absolute"
)

const (
	defaultWidth  = 56
	defaultHeight = 20
)

type NavState int

const (
	INVENTORY_LIST NavState = iota
	INVENTORY_DETAILS
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

type Model struct {
	selector      *SelectionManager
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

const (
	bullet   = "•"
	ellipsis = "…"
)

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
		headerHeight := lipgloss.Height(m.headerView())
		verticalMarginHeight := headerHeight
		viewportModel, err := m.newViewportModel(msg.file, defaultWidth, 15-verticalMarginHeight)
		if err != nil {
			fmt.Println("Error reading file:", err)
			return m, tea.Quit
		}
		m.viewport = viewportModel

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

var (
	AccentColor             = lipgloss.ANSIColor(termenv.ANSIBlack)
	stylesSectionStyle      = lipgloss.NewStyle().BorderStyle(lipgloss.HiddenBorder()).BorderForeground(AccentColor).Padding(0, 1)
	stylesSectionTitleStyle = lipgloss.NewStyle().Padding(0, 1).Background(AccentColor).Foreground(lipgloss.Color("15")).Bold(true).Transform(strings.ToUpper)
)

func renderInventoryDetails(m Model) string {
	header := m.renderHeader(m.detailFile)

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		renderDeletedWhere(m.detailFile),
		renderDeletedAt(m.detailFile, m.datefmt),
		fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView()),
	)

	return content
}

func (m Model) renderHeader(file File) string {
	name := file.Name
	if file.isSelected() {
		name = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#000000"}).
			Background(lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"}).
			Render(file.Name)
	}
	title := lipgloss.NewStyle().
		BorderStyle(func() lipgloss.Border {
			b := lipgloss.RoundedBorder()
			b.Right = "├"
			return b
		}()).
		Padding(0, 1).
		Bold(true).
		Render(name)

	line := strings.Repeat("─", max(0, defaultWidth-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func renderDeletedAt(f File, datefmt string) string {
	var ts string
	switch datefmt {
	case "absolute":
		ts = f.Timestamp.Format(time.DateTime)
	default:
		ts = humanize.Time(f.Timestamp)
	}
	return stylesSectionStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			stylesSectionTitleStyle.MarginRight(3).Render("Deleted At"),
			lipgloss.NewStyle().Render(ts)),
	)
}

func renderDeletedWhere(f File) string {
	s := filepath.Dir(f.From)
	w := wordwrap.NewWriter(46)
	w.Breakpoints = []rune{'/', '.'}
	w.KeepNewlines = false
	_, _ = w.Write([]byte(s))
	_ = w.Close()
	return stylesSectionStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			stylesSectionTitleStyle.MarginBottom(1).Render("Where it was"),
			lipgloss.NewStyle().Render(w.String())),
	)
}

func renderMetadata(f File) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		stylesSectionStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left, stylesSectionTitleStyle.MarginBottom(1).Render("Size"), renderFileSize(f))),
		stylesSectionStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left, stylesSectionTitleStyle.MarginBottom(1).Render("Type"), renderFileType(f))),
	)
}

func renderFileSize(f File) string {
	var sizeStr string
	size, err := utils.DirSize(f.To)
	if err != nil {
		sizeStr = "(Cannot be calculated)"
	} else {
		sizeStr = humanize.Bytes(uint64(size))
	}
	return sizeStr
}

func renderFileType(f File) string {
	var result string
	fi, err := os.Stat(f.To)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			result = "file has been totally removed"
		default:
			result = err.Error()
		}
	} else {
		if fi.IsDir() {
			result = "(directory)"
		}
	}

	if result == "" {
		mtype, err := mimetype.DetectFile(f.File.To)
		if err != nil {
			result = err.Error()
		} else {
			result = mtype.String()
		}
	}

	return result
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
		s += renderInventoryDetails(m)
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
var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	currentItemStyle  = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170")).Width(150)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("#00ff00"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

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

var headerStyle = func(cfg config.UI) lipgloss.Style {
	fgAsBg := cfg.Color.PreviewBoarder.Foreground
	bgAsFg := cfg.Color.PreviewBoarder.Background
	return lipgloss.NewStyle().Padding(0, 1, 0, 1).
		Foreground(lipgloss.Color(bgAsFg)).
		Background(lipgloss.Color(fgAsBg))
}

func (m Model) footerView() string {
	fg := m.config.Color.PreviewBoarder.Foreground
	if m.cannotPreview {
		header := m.renderHeader(m.detailFile)
		return lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Render(strings.Repeat("─", lipgloss.Width(header)))
	}
	info := headerStyle(m.config).Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Render(lipgloss.JoinHorizontal(lipgloss.Center, line, info))
}

func (m Model) headerView() string {
	fg := m.config.Color.PreviewBoarder.Foreground
	size := headerStyle(m.config).Render(renderFileSize(m.detailFile))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(size)))
	return lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Render(lipgloss.JoinHorizontal(lipgloss.Center, line, size))
}

func (m *Model) newViewportModel(file File, width, height int) (viewport.Model, error) {
	cfg := m.config.Preview
	getFileContent := func(path string) string {
		content := "cannot preview"
		fi, err := os.Stat(path)
		if err != nil {
			slog.Debug(fmt.Sprintf("no such file %s", path))
			return content
		}
		if fi.IsDir() {
			input := cfg.Directory
			if input == "" {
				slog.Debug("preview list_dir command is not set, fallback to builtin list_dir func")
				lines := []string{}
				dirs, _ := os.ReadDir(path)
				for _, dir := range dirs {
					info, _ := dir.Info()
					name := dir.Name()
					if info.IsDir() {
						name += "/"
					}
					lines = append(lines,
						fmt.Sprintf("%s %7s  %s",
							info.Mode().String(),
							humanize.Bytes(uint64(info.Size())),
							name,
						),
					)
				}
				return strings.Join(lines, "\n")
			}
			out, _, err := utils.RunShell(input)
			if err != nil {
				slog.Error(fmt.Sprintf("command failed: %s", input), "error", err)
			}
			return out
		}
		mtype, err := mimetype.DetectFile(path)
		if err != nil {
			return content
		}
		if !strings.Contains(mtype.String(), "text/plain") {
			slog.Debug(fmt.Sprintf("mimetype %s not supported to preview", mtype.String()))
			return content
		}
		f, err := os.Open(file.To)
		if err != nil {
			return content
		}
		defer f.Close()
		var fileContent strings.Builder
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			fileContent.WriteString(scanner.Text() + "\n")
		}
		if err := scanner.Err(); err != nil {
			return content
		}
		return fileContent.String()
	}
	content := getFileContent(file.To)
	if content == "cannot preview" {
		m.cannotPreview = true
	} else {
		m.cannotPreview = false
	}
	shouldColor := cfg.Highlight
	if shouldColor && !m.cannotPreview {
		content, _ = highlight(content, file.Name, cfg.Colorscheme)
	}
	viewportModel := viewport.New(width, height)
	viewportModel.KeyMap = viewport.KeyMap{
		Up:           key.NewBinding(key.WithKeys("k", "up")),
		Down:         key.NewBinding(key.WithKeys("j", "down")),
		HalfPageUp:   key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "½ page up")),
		HalfPageDown: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "½ page down")),
	}
	if m.cannotPreview {
		mtype, _ := mimetype.DetectFile(file.To)
		headerHeight := lipgloss.Height(m.headerView())
		verticalMarginHeight := headerHeight
		content = lipgloss.Place(defaultWidth, 15-verticalMarginHeight,
			lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Bold(true).Transform(strings.ToUpper).Render("cannot preview")+"\n\n\n"+
				lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(termenv.ANSIBrightBlack)).Render("("+mtype.String()+")"),
			lipgloss.WithWhitespaceChars("`"),
			lipgloss.WithWhitespaceForeground(lipgloss.ANSIColor(termenv.ANSIBrightBlack)))
	}
	viewportModel.SetContent(content)
	return viewportModel, nil
}

func initModel(filteredFiles []inventory.File, cfg config.UI) Model {
	slog.Debug("initModel starts")
	width := 20

	var items []list.Item
	var files []File
	var pfiles []*File
	for _, file := range filteredFiles {
		items = append(items, File{File: file})
		files = append(files, File{File: file})
		pfiles = append(pfiles, &File{File: file})
	}

	// TODO: configable?
	// l := list.New(items, ClassicDelegate{}, width, defaultHeight)
	l := list.New(items, NewRestoreDelegate(cfg, pfiles), width, defaultHeight)

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
		selector: &SelectionManager{},
		navState: INVENTORY_LIST,
		datefmt:  datefmtRel,
		// files:     files,
		files:  files,
		config: cfg,
		// models
		list:     l,
		viewport: viewport.Model{},
	}
	return m
}

func Run(filteredFiles []inventory.File, cfg config.UI) ([]File, error) {
	m := initModel(filteredFiles, cfg)
	returnModel, err := tea.NewProgram(m).Run()
	if err != nil {
		return []File{}, err
	}
	files := returnModel.(Model).choices
	if returnModel.(Model).navState == QUITTING {
		if msg := cfg.ByeMessage; msg != "" {
			fmt.Println(msg)
		}
		return []File{}, nil
	}
	return files, nil
}
