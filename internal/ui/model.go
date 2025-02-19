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
)

// Model represents the main UI model following the Bubble Tea pattern
type Model struct {
	// Manager handles trash operations
	trashManager *trash.Manager

	// State management
	state *ViewState

	// Key mappings
	listKeys   *keys.ListKeyMap
	detailKeys *keys.DetailKeyMap

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

	// UI styles
	styles *styles.Styles

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

	// Initialize list delegate
	delegate := NewRestoreDelegate(cfg.UI, fileList)
	delegate.ShortHelpFunc = keys.ListKeys.ShortHelp
	delegate.FullHelpFunc = keys.ListKeys.FullHelp

	// Create and configure list
	l := list.New(items, delegate, defaultWidth, defaultHeight)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	l.DisableQuitKeybindings()

	// Set paginator type based on config
	switch cfg.UI.Paginator {
	case "arabic":
		l.Paginator.Type = paginator.Arabic
	default:
		l.Paginator.Type = paginator.Dots
	}

	if !cfg.Core.Delete.Disable {
		keys.ListKeys.AddDeleteKey()
		keys.DetailKeys.AddDeleteKey()
	}

	// Return fully initialized model
	return Model{
		trashManager: manager,
		state:        NewViewState(),
		listKeys:     keys.ListKeys,
		detailKeys:   keys.DetailKeys,
		files:        fileList,
		config:       cfg.UI,
		list:         l,
		viewport:     viewport.Model{},
		styles:       styles.New(cfg.UI),
		help:         help.New(),
	}
}
