package ui

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/ui/components/confirm"
	"github.com/babarot/gomi/internal/ui/components/input"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jimschubert/answer/validate"
)

func Confirm(prompt string) bool {
	m := confirm.New()
	m.Prompt = prompt
	m.DefaultValue = confirm.Denied
	m.Immediately = true

	p := tea.NewProgram(&m)
	if _, err := p.Run(); err != nil {
		slog.Error("confirm failed", "error", err)
		return false
	}

	return m.Selected().IsAccepted()
}

func ConfirmYes(prompt string) bool {
	m := confirm.NewYesValidation()
	m.Prompt = prompt

	p, err := tea.NewProgram(&m).Run()
	if err != nil {
		slog.Error("confirmYes failed", "error", err)
		return false
	}

	return p.(*confirm.YesValidationModel).IsAccepted()
}

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
		// m.Value() returns value even if canceled
		return m.Value(), ErrInputCanceled
	}
	return m.Value(), nil
}

func onlySpecialChars(input string) bool {
	for _, char := range input {
		if char != '.' && char != '_' && char != '-' {
			return false
		}
	}
	return true
}
