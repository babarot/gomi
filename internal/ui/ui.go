package ui

import (
	"errors"
	"fmt"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
	tea "github.com/charmbracelet/bubbletea"
)

// ListDensityType represents the density of the list view
type ListDensityType uint8

const (
	// Compact shows items without descriptions
	Compact ListDensityType = iota
	// Spacious shows items with descriptions
	Spacious
)

// Density configuration values
const (
	CompactDensityVal  = "compact"
	SpaciousDensityVal = "spacious"
)

// UI constants
const (
	bullet   = "•"
	ellipsis = "…"

	defaultWidth  = 66
	defaultHeight = 30
)

const (
	PaginatorDots   = "dots"
	PaginatorArabic = "arabic"
)

// Common errors
var (
	ErrCannotPreview = errors.New("cannot preview")
	ErrInputCanceled = errors.New("input is canceled")
)

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		// load files
		loadFileListCmd(m.files),
		// other cmds...
	)
}

// Render displays the file selection interface and returns the selected files
func Render(manager *trash.Manager, files []*trash.File, cfg *config.Config) ([]*trash.File, error) {
	// Create and initialize the model
	m := NewModel(manager, files, cfg)

	// Initialize UI program
	p := tea.NewProgram(m)

	// Run the UI
	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	// Process results
	finalModel := result.(Model)
	if finalModel.state.current == Quitting {
		if msg := cfg.UI.ExitMessage; msg != "" {
			fmt.Println(msg)
		}
		return nil, nil
	}

	// Convert UI files back to trash files
	choices := finalModel.choices
	trashFiles := make([]*trash.File, len(choices))
	for i, file := range choices {
		trashFiles[i] = file.File
	}

	return trashFiles, nil
}
