package ui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/paginator"
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

	// File data
	files   []File
	choices []File

	// Selection tracking
	selection *SelectionManager

	// UI components and config
	config   config.UI
	palette  styles.Palette
	help     help.Model
	list     list.Model
	viewport viewport.Model
	styles   *styles.Styles

	// Error state if any
	err error
}

// NewModel creates a new UI model instance
func NewModel(t trash.Trash, files []*trash.File, opts RenderOptions) Model {
	var items []list.Item
	var fileList []File

	uiCfg := opts.Config

	// Convert trash files to UI files
	for _, file := range files {
		items = append(items, File{File: file})
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

	// Initialize selection manager
	selection := &SelectionManager{items: []File{}}

	// Create and configure list; setBackground replaces its delegate and
	// styles
	palette := styles.NewPalette(uiCfg)
	l := list.New(items, NewRestoreDelegate(uiCfg, palette, fileList, selection, true), defaultWidth, defaultHeight)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	l.SetShowHelp(false) // do not use default help of list model
	l.DisableQuitKeybindings()

	// Set paginator type based on config
	switch uiCfg.Paginator {
	case PaginatorArabic:
		l.Paginator.Type = paginator.Arabic
	default:
		l.Paginator.Type = paginator.Dots
	}

	m := Model{
		trash:     t,
		state:     NewViewState(),
		keyMap:    keyMap,
		selection: selection,
		files:     fileList,
		config:    uiCfg,
		palette:   palette,
		list:      l,
		viewport:  viewport.Model{},
		previews:  map[string]previewResult{},
		sizes:     map[string]int64{},
		reading:   map[string]bool{},
		help:      help.New(),
	}
	// Until the terminal reports its background, assume a dark one
	m.setBackground(true)
	return m
}

// setBackground styles the model for a dark or light terminal background
func (m *Model) setBackground(isDark bool) {
	m.styles = styles.New(m.palette, isDark)
	m.help.Styles = helpStyles(m.palette, isDark)

	delegate := NewRestoreDelegate(m.config, m.palette, m.files, m.selection, isDark)
	delegate.ShortHelpFunc = m.keyMap.AsListKeyMap().ShortHelp
	delegate.FullHelpFunc = m.keyMap.AsListKeyMap().FullHelp
	m.list.SetDelegate(delegate)

	ls := list.DefaultStyles(isDark)
	m.list.Styles = ls
	m.list.Paginator.ActiveDot = ls.ActivePaginationDot.String()
	m.list.Paginator.InactiveDot = ls.InactivePaginationDot.String()

	// bubbles does not apply ls.Filter to the filter input itself
	filter := ls.Filter
	filter.Focused.Prompt = lipgloss.NewStyle().
		Foreground(m.palette.FilterPrompt.Of(isDark)).
		Bold(true)
	filter.Blurred.Prompt = filter.Focused.Prompt
	m.list.FilterInput.SetStyles(filter)
}
