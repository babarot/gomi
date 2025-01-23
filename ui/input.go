package ui

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type (
	errMsg error
)

type model struct {
	canceled  bool
	quitting  bool
	prompt    string
	textInput textinput.Model
	value     string
	err       error
}

type InputMsg struct{}

func Input(prompt, placeholder string) (string, error) {
	ti := textinput.New()
	ti.Placeholder = placeholder
	// ti.Validate = ValidateFunc(validate.NewValidation())
	ti.Validate = func(s string) error {
		match, _ := regexp.MatchString(`^[a-zA-Z0-9]+$`, s)
		if !match {
			return fmt.Errorf("%s: invalid input", s)
		}
		return nil
	}
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	m := model{
		prompt:    prompt,
		textInput: ti,
		err:       nil,
	}

	p := tea.NewProgram(m)
	returned, err := p.Run()
	if err != nil {
		return "", err
	}

	return returned.(model).value, nil
}

// func (m model) Init() tea.Cmd {
// 	return textinput.Blink
// }

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.input,
	)
}

func (m model) input() tea.Msg {
	return tea.Quit
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.canceled = true
			return m, tea.Quit

		case tea.KeyEnter:
			m.value = m.textInput.Value()
			// should implement input validater like survey
			// instead of this condition
			if m.value == "" {
				m.canceled = true
				return m, tea.Quit
			}
			err := m.textInput.Validate(m.value)
			if err != nil {
				m.err = err
			} else {
				m.quitting = true
				return m, tea.Quit
			}
		}
	case InputMsg:

	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.err != nil {
		return m.err.Error() + "\n"
	}

	if m.quitting {
		return ""
	}

	s := fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		m.prompt,
		m.textInput.View(),
		"(esc to quit)",
	) + "\n"

	if m.canceled {
		s += "canceled.\n"
	}
	return s
}

// Only allow letter inputs
func validateInput() textinput.ValidateFunc {
	return func(s string) error {
		if s == "" {
			return errors.New("not valid input")
		}
		return nil
	}
}
