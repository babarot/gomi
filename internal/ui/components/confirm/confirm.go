package confirm

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jimschubert/answer/colors"
)

// Decision is an enumeration of decisions available in the confirmation bubble
type Decision int

const (
	// Undecided indicates the state in which a user has not made a selection, and there is no default available
	Undecided Decision = iota

	// Accepted indicates the user has provided a positive response (accepted confirmation)
	Accepted

	// Denied indicates the user has provided a negative response (denied confirmation)
	Denied
)

// TrueFalseString represents Decision in undecided/true/false format
func (d Decision) TrueFalseString() string {
	return [...]string{
		"undecided",
		"true",
		"false",
	}[d]
}

// YesNoString represents Decision in undecided/yes/no format
func (d Decision) YesNoString() string {
	return [...]string{
		"undecided",
		"yes",
		"no",
	}[d]
}

// String satisfies the fmt.Stringer interface
func (d Decision) String() string {
	return [...]string{
		"undecided",
		"accepted",
		"denied",
	}[d]
}

// IsAccepted is a helper to indicate the positive confirmation state was selected
func (d Decision) IsAccepted() bool {
	return d == Accepted
}

// IsDenied is a helper to indicate the negative confirmation state was selected
func (d Decision) IsDenied() bool {
	return d == Denied
}

// Rendering is an enumeration of available renderings, allowing modification of the bubble's view output
type Rendering int

const (
	// InputBox defines rendering as standard format: Prompt? Y/n
	// The user would then enter the desired character and hit enter.
	InputBox Rendering = iota

	// HorizontalSelection defines rendering as a left->right looping toggle: Prompt? ➤Y  N
	HorizontalSelection

	// VerticalSelection defines rendering as a up->down looping toggle:
	// Prompt?
	// ➤Y
	//  N
	VerticalSelection

	// ImmediateInput defines rendering as immediate confirmation on single key press
	ImmediateInput
)

// Styles holds relevant styles used for rendering
// For an introduction to styling with Lip Gloss see:
// https://github.com/charmbracelet/lipgloss
type Styles struct {
	PromptPrefix     lipgloss.Style
	Prompt           lipgloss.Style
	Text             lipgloss.Style
	Placeholder      lipgloss.Style
	ChooserIndicator lipgloss.Style
}

// Model represents the bubble tea model for the confirm bubble
type Model struct {
	// PromptPrefix is a character or other indicator existing before the user prompt, separately styled
	PromptPrefix string

	// Prompt is the text to display to the user, prompting them for input
	Prompt string

	// Placeholder is text used only when rendering as InputBox, providing help-like text to the user
	Placeholder string

	// AcceptedDecisionText is the text to display to a user which they will select for Accepted confirmations
	AcceptedDecisionText string

	// DeniedDecisionText is the text to display to a user which they will select for Denied confirmations
	DeniedDecisionText string

	// ChooserIndicator is a rune displayed to the user for HorizontalSelection or VerticalSelection rendering
	ChooserIndicator rune

	// DefaultValue allows the caller to define the initially selected value
	DefaultValue Decision

	// Rendering provides options for modified rendering
	Rendering Rendering

	// Styles is the group of available styles
	Styles Styles

	// ShowHelp determines whether to show help where possible (e.g. HorizontalSelection or VerticalSelection rendering)
	ShowHelp bool

	selected Decision
	renderer rendering
	done     bool
}

// New creates a new model with default settings.
func New() Model {
	m := Model{
		PromptPrefix:         "? ",
		AcceptedDecisionText: "y",
		DeniedDecisionText:   "n",
		ChooserIndicator:     '➤',
		Styles: Styles{
			PromptPrefix:     lipgloss.NewStyle().Foreground(lipgloss.Color(colors.PromptPrefix)),
			Placeholder:      lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Placeholder)),
			ChooserIndicator: lipgloss.NewStyle().Foreground(lipgloss.Color(colors.PromptPrefix)),
		},
		DefaultValue: Accepted,
		ShowHelp:     true,
	}

	return m
}

// Selected retrieves the default or user-selected Decision value
func (m *Model) Selected() Decision {
	return m.selected
}

// Value returns the Decision in human-readable form, supporting the text defined by the caller
func (m *Model) Value() string {
	switch m.selected {
	case Accepted:
		return m.AcceptedDecisionText
	case Denied:
		return m.DeniedDecisionText
	case Undecided:
		return ""
	}
	return ""
}

// SetDecision allows for externally setting the decision to a supported value
func (m *Model) SetDecision(decision Decision) {
	m.selected = decision
}

// IsAccepted is a helper to indicate the positive confirmation state was selected
func (m *Model) IsAccepted() bool {
	return m.selected == Accepted
}

// IsDenied is a helper to indicate the negative confirmation state was selected
func (m *Model) IsDenied() bool {
	return m.selected == Denied
}

// IsUndecided is a helper to indicate in cases where no default was provided, the user made no choice
func (m *Model) IsUndecided() bool {
	return m.selected == Undecided
}

// Init satisfies the tea.Model interface
func (m *Model) Init() tea.Cmd {
	m.selected = m.DefaultValue

	switch m.Rendering {
	case InputBox:
		m.renderer = &inputRenderer{m: m}
	case HorizontalSelection, VerticalSelection:
		m.renderer = &selectionRenderer{m: m}
	case ImmediateInput:
		m.renderer = &immediateRenderer{m: m}
	}
	return m.renderer.Init()
}

// Update satisfies the tea.Model interface
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.renderer.Update(msg)
}

// View satisfies the tea.Model interface
func (m *Model) View() string {
	return m.renderer.View()
}
