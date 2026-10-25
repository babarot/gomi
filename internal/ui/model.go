package ui

import (
	"os"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/ui/keys"
	"github.com/babarot/gomi/internal/ui/styles"
)

// Model represents the main UI model following the Bubble Tea pattern
type Model struct {
	// Trash handles trash operations (put, list, restore, remove)
	trash trash.Trash

	// State management
	state *ViewState

	// Key mappings
	keyMap *keys.KeyMap

	// Current detail view file if any
	detailFile File

	// previews and sizes keep what was read in the background, by trash
	// path; a size that could not be computed is -1
	previews map[string]previewResult
	sizes    map[string]int64
	// reading has the reads started and not yet returned
	reading map[string]bool
	// sizeQueue is the directories of the list waiting to be sized, and
	// sizing how many of the list's are being sized
	sizeQueue []File
	sizing    int

	// File data
	files   []File
	choices []File

	// Selection tracking
	selection *SelectionManager

	// UI components and config
	config   config.UI
	palette  styles.Palette
	help     help.Model
	list     *fileList
	viewport viewport.Model
	styles   *styles.Styles
	isDark   bool

	// width and height are the columns and rows the UI takes
	width, height int
	// home is the home directory, written as ~ in paths
	home string
	// status tells what was just done, such as deleting files
	status string

	// Error state if any
	err error
}

// NewModel creates a new UI model instance
func NewModel(t trash.Trash, files []*trash.File, opts RenderOptions) Model {
	uiCfg := opts.Config

	// Convert trash files to UI files
	var fileList []File
	for _, file := range files {
		fileList = append(fileList, File{
			File:            file,
			dirListCommand:  uiCfg.Preview.DirectoryCommand,
			syntaxHighlight: uiCfg.Preview.SyntaxHighlight,
			colorscheme:     uiCfg.Preview.Colorscheme,
		})
	}

	// Initialize key map
	keyMap := keys.NewKeyMap(keys.KeyMapConfig{
		DeleteEnabled: opts.DeleteEnabled,
	})

	home, _ := os.UserHomeDir()
	list := newFileList()
	m := Model{
		trash:     t,
		state:     NewViewState(),
		keyMap:    keyMap,
		selection: &SelectionManager{items: []File{}},
		files:     fileList,
		config:    uiCfg,
		palette:   styles.NewPalette(uiCfg),
		list:      &list,
		viewport:  viewport.Model{},
		previews:  map[string]previewResult{},
		sizes:     map[string]int64{},
		reading:   map[string]bool{},
		help:      help.New(),
		width:     defaultWidth,
		height:    defaultHeight,
		home:      home,
	}
	// Until the terminal reports its background, assume a dark one
	m.setBackground(true)
	return m
}

// setBackground styles the model for a dark or light terminal background
func (m *Model) setBackground(isDark bool) {
	m.isDark = isDark
	m.styles = styles.New(m.palette, isDark)
	m.help.Styles = helpStyles(m.palette, isDark)

	prompt := lipgloss.NewStyle().Foreground(m.palette.FilterPrompt.Of(isDark)).Bold(true)
	st := textinput.DefaultStyles(isDark)
	st.Focused.Prompt, st.Blurred.Prompt = prompt, prompt
	m.list.filter.SetStyles(st)
}

// uiHeight is how many rows the UI takes in a terminal of termHeight
// rows, as ui.height says: a number of rows or a share of the terminal,
// never more than the terminal.
func uiHeight(spec string, termHeight int) int {
	n, percent, ok := config.ParseHeight(spec)
	if !ok {
		n, percent, _ = config.ParseHeight(config.DefaultHeight)
	}
	switch {
	case percent:
		n = termHeight * n / 100
	}
	// the header, a row and the three lines at the bottom
	return min(max(n, minHeight), termHeight)
}
