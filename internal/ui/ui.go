package ui

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/ui/confirm"
	"github.com/babarot/gomi/internal/ui/input"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jimschubert/answer/validate"
)

var ErrInputCanceled = errors.New("tet")

// RenderList displays an interactive list of files and returns the selected files.
// Returns nil and no error if the user cancels the operation.
func RenderList(files []*trash.File, cfg config.UI) ([]*trash.File, error) {
	if len(files) == 0 {
		return nil, errors.New("no files to display")
	}

	// Initialize and run the TUI
	m := New(files, cfg)
	p := tea.NewProgram(&m)

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run UI: %w", err)
	}

	// Check if the user canceled
	model := finalModel.(Model)
	if model.Canceled() {
		if msg := cfg.ExitMessage; msg != "" {
			fmt.Println(msg)
		}
		return nil, nil
	}

	// Return selected files
	return model.SelectedFiles(), nil
}

// Confirm displays a confirmation prompt and returns the user's decision.
func Confirm(prompt string) bool {
	m := confirm.New()
	m.Prompt = prompt
	m.DefaultValue = confirm.Denied
	m.Immediately = true

	p := tea.NewProgram(&m)
	if _, err := p.Run(); err != nil {
		return false
	}

	return m.Selected().IsAccepted()
}

// InputFilename prompts the user to input a new filename.
func InputFilename(file *trash.File) (string, error) {
	m := input.New()
	m.Prompt = "New name to avoid to overwrite:"
	m.Placeholder = file.Name
	m.Validate = validate.NewValidation().
		MinLength(1, "min: 1 characters").
		And(func(input string) error {
			if strings.ToLower(input) == file.Name {
				return errors.New("name should be changed")
			}
			return nil
		}).
		And(func(input string) error {
			if input == "" {
				return nil
			}
			matched, err := regexp.MatchString(`^[a-zA-Z0-9._-]*$`, input)
			if err != nil {
				return fmt.Errorf("regexp failed: %w", err)
			}
			if !matched {
				return errors.New("not valid chars are included")
			}
			if onlySpecialChars(input) {
				return errors.New("using only special characters is not allowed")
			}
			return nil
		}).
		Build()

	p := tea.NewProgram(&m)
	if _, err := p.Run(); err != nil {
		return "", err
	}

	if m.Canceled() {
		return m.Value(), ErrInputCanceled
	}
	return m.Value(), nil
}

// onlySpecialChars checks if the input string contains only special characters.
func onlySpecialChars(input string) bool {
	for _, char := range input {
		if char != '.' && char != '_' && char != '-' {
			return false
		}
	}
	return true
}
