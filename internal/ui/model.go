package ui

import (
	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/ui/keys"
	"github.com/babarot/gomi/internal/ui/styles"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the main UI model following the Bubble Tea pattern
type Model struct {
	// Manager handles trash operations
	trashManager *trash.Manager

	// State management
	state *ViewState

	// Key mappings
	keyMap *keys.KeyMap

	// Current detail view file if any
	detailFile File

	// File data
	files   []File
	choices []File

	// UI components and config
	config   config.UI
	help     help.Model
	list     list.Model
	viewport viewport.Model
	styles   *styles.Styles

	// Error state if any
	err error
}

// NewModel creates a new UI model instance
func NewModel(manager *trash.Manager, files []*trash.File, cfg *config.Config) Model {
	var items []list.Item
	var fileList []File

	// Convert trash files to UI files
	for _, file := range files {
		items = append(items, File{File: file})
		fileList = append(fileList, File{
			File:            file,
			dirListCommand:  cfg.UI.Preview.DirectoryCommand,
			syntaxHighlight: cfg.UI.Preview.SyntaxHighlight,
			colorscheme:     cfg.UI.Preview.Colorscheme,
		})
	}

	// Initialize key map
	keyMap := keys.NewKeyMap(keys.KeyMapConfig{
		DeleteEnabled: cfg.Core.PermanentDelete.Enable,
	})

	// Initialize list delegate
	delegate := NewRestoreDelegate(cfg.UI, fileList)
	delegate.ShortHelpFunc = keyMap.AsListKeyMap().ShortHelp
	delegate.FullHelpFunc = keyMap.AsListKeyMap().FullHelp

	// Create and configure list
	l := list.New(items, delegate, defaultWidth, defaultHeight)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	l.SetShowHelp(false) // do not use default help of list model
	l.DisableQuitKeybindings()
	
	// Configure filter prompt style
	l.FilterInput.PromptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(cfg.UI.Style.ListView.FilterPrompt)).
		Bold(true)

	// Set paginator type based on config
	switch cfg.UI.Paginator {
	case PaginatorArabic:
		l.Paginator.Type = paginator.Arabic
	default:
		l.Paginator.Type = paginator.Dots
	}

	// Return fully initialized model
	return Model{
		trashManager: manager,
		state:        NewViewState(),
		keyMap:       keyMap,
		files:        fileList,
		config:       cfg.UI,
		list:         l,
		viewport:     viewport.Model{},
		styles:       styles.New(cfg.UI),
		help:         help.New(),
	}
}
