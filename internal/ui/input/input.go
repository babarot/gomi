package input

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jimschubert/answer/colors"
	"github.com/jimschubert/answer/validate"
)

var (
	_ tea.Model = (*Model)(nil)
)

type suggestions []string

type keyMap struct {
	Enter key.Binding
	Quit  key.Binding
}

type exitType int8

const (
	none exitType = iota
	userQuit
	userEnter
)

// ValidateFunc determines if the input string is valid, returning nil if valid or an error if invalid
type ValidateFunc validate.Func

// Styles holds relevant styles used for rendering
// For an introduction to styling with Lip Gloss see:
// https://github.com/charmbracelet/lipgloss
type Styles struct {
	PromptPrefix lipgloss.Style
	Prompt       lipgloss.Style
	ErrorPrefix  lipgloss.Style
	Text         lipgloss.Style
	Placeholder  lipgloss.Style
	Suggestions  lipgloss.Style
}

// Model represents the bubble tea model for the input
type Model struct {
	PromptPrefix     string
	Prompt           string
	Placeholder      string
	CharLimit        int
	MaxWidth         int
	EchoMode         textinput.EchoMode
	Validate         ValidateFunc
	Styles           Styles
	Suggest          func(input string) []string
	SuggestionPrefix string
	err              error
	done             exitType
	input            textinput.Model
	initialized      bool
	suggestions      []string
	keyMap           keyMap
}

// New creates a new model with default settings.
func New() Model {
	return Model{
		PromptPrefix:     "? ",
		SuggestionPrefix: "Suggestions:",
		CharLimit:        0,
		Styles: Styles{
			PromptPrefix: lipgloss.NewStyle().Foreground(lipgloss.Color(colors.PromptPrefix)),
			ErrorPrefix:  lipgloss.NewStyle().Foreground(lipgloss.Color(colors.ErrorPrefix)),
			Placeholder:  lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Placeholder)),
			Suggestions:  lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color(colors.Placeholder)),
		},
		keyMap: keyMap{
			Quit: key.NewBinding(
				key.WithKeys(tea.KeyEsc.String(), tea.KeyCtrlC.String()),
			),
			Enter: key.NewBinding(key.WithKeys(tea.KeyEnter.String())),
		},
	}
}

func (m *Model) setup() {
	if m.Validate == nil {
		m.Validate = ValidateFunc(validate.NewValidation())
	}
	if m.Prompt == "" {
		m.Prompt = "Please enter:"
	}
	input := textinput.New()
	input.CharLimit = m.CharLimit
	input.Width = m.MaxWidth
	if !strings.HasSuffix(m.Prompt, " ") {
		input.Prompt = m.Prompt + " "
	} else {
		input.Prompt = m.Prompt
	}
	input.Placeholder = m.Placeholder
	input.PromptStyle = m.Styles.Prompt
	input.PlaceholderStyle = m.Styles.Placeholder
	input.TextStyle = m.Styles.Text
	input.EchoMode = m.EchoMode
	input.Focus()
	m.input = input
	m.initialized = true
}

func (m *Model) Init() tea.Cmd {
	m.setup()
	return nil
}

func (m *Model) SetValue(value string) {
	m.input.SetValue(value)
}

func (m *Model) Value() string {
	return m.input.Value()
}

func (m Model) Canceled() bool {
	return m.done == userQuit
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.initialized {
		m.setup()
	}

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Enter):
			if m.err == nil {
				m.done = userEnter
				return m, tea.Quit
			}
		case key.Matches(msg, m.keyMap.Quit):
			m.done = userQuit
			return m, tea.Quit
		}
	case error:
		m.err = msg
	case suggestions:
		m.suggestions = msg
	}

	var cmds []tea.Cmd
	var before, after string
	before = m.input.Value()
	m.input, cmd = m.input.Update(msg)
	after = m.input.Value()

	changed := before != m.input.Value()
	if changed {
		m.err = m.Validate(m.input.Value())
	}

	// Originally, validation runs after the text is entered,
	// but with this change,
	// validation will be executed even when no text has been entered.
	// (added by @babarot)
	if m.initialized {
		m.err = m.Validate(m.input.Value())
	}

	if (changed && m.err != nil) || after == "" {
		m.suggestions = m.suggestions[:0]
	} else if changed && m.err == nil && m.Suggest != nil {
		// asynchronously update the suggestions
		cmds = append(cmds, func() tea.Msg {
			search := after
			result := m.Suggest(search)
			return suggestions(result)
		})
	}

	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) writeError(err error, b *strings.Builder) {
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		// inline removes newlines, so we need to use false when the error is wrapped
		xRender := m.Styles.ErrorPrefix.Inline(false).Render("✘ ")
		placeholderRender := m.Styles.Placeholder.Inline(false).Render
		errs := joined.Unwrap()
		for _, err := range errs {
			if _, ok := err.(interface{ Unwrap() []error }); ok { //nolint:govet
				m.writeError(err, b)
			} else {
				b.WriteString(xRender)
				b.WriteString(placeholderRender(err.Error()))
				b.WriteRune('\n')
			}
		}
	} else {
		b.WriteString(m.Styles.ErrorPrefix.Inline(true).Render("✘ "))
		b.WriteString(m.Styles.Placeholder.Inline(true).Render(err.Error()))
		b.WriteRune('\n')
	}
}

func (m *Model) View() string {
	var b strings.Builder
	if m.PromptPrefix != "" {
		b.WriteString(m.Styles.PromptPrefix.Inline(true).Render(m.PromptPrefix))
		if m.Prompt != "" && !strings.HasSuffix(m.PromptPrefix, " ") {
			b.WriteRune(' ')
		}
	}

	if m.done == userQuit {
		return ""
	} else if m.done == userEnter {
		// rather than clearing the program output, we want to show the question + answer just as AlecAivazis/survey did
		if m.Prompt != "" {
			b.WriteString(m.Styles.Prompt.Inline(true).Render(m.Prompt))
			b.WriteRune(' ')
		}
		b.WriteString(m.input.Value())
		b.WriteRune('\n')
		return b.String()
	}

	b.WriteString(m.input.View())
	if m.err != nil {
		b.WriteRune('\n')
		m.writeError(m.err, &b)
	} else if len(m.suggestions) > 0 {
		sRender := m.Styles.Suggestions.Render
		if m.SuggestionPrefix != "" {
			b.WriteRune('\n')
			b.WriteString(sRender(m.SuggestionPrefix))
		}
		b.WriteRune('\n')
		for _, suggestion := range m.suggestions {
			b.WriteString(sRender(suggestion))
			b.WriteRune('\n')
		}
	}
	return b.String()
}
