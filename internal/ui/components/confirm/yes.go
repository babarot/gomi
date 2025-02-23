package confirm

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// YesValidationModel extends the base Model with strict YES validation functionality.
// It only accepts "YES" (in uppercase) as a valid confirmation input and provides
// real-time visual feedback on input validity.
type YesValidationModel struct {
	Model
	showValidation bool
	validStyle     lipgloss.Style // Style for valid input checkmark
	invalidStyle   lipgloss.Style // Style for invalid input cross mark
}

// NewYesValidation creates a new instance of YesValidationModel with predefined settings.
// It initializes the model with:
// - InputBox rendering mode
// - "YES" as the only valid acceptance input
// - Visual validation indicators (green checkmark for valid, red cross for invalid)
func NewYesValidation() YesValidationModel {
	base := New()
	base.Rendering = InputBox
	base.AcceptedDecisionText = "YES"
	base.DeniedDecisionText = "No"

	return YesValidationModel{
		Model:          base,
		showValidation: true,
		validStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")), // Green color
		invalidStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")), // Red color
	}
}

// yesValidationRenderer handles the rendering and input processing for the YES validation.
// It enforces strict input rules and provides real-time visual feedback.
type yesValidationRenderer struct {
	m    *YesValidationModel
	text textinput.Model
}

// isValidYesChar checks if the input character is valid for the current input position.
// It enforces the strict sequence of "YES":
// - First character must be 'Y'
// - Second character must be 'E'
// - Third character must be 'S'
// This ensures that only "YES" can be entered, character by character.
func isValidYesChar(s string, currentValue string) bool {
	switch len(currentValue) {
	case 0:
		return s == "Y"
	case 1:
		return s == "E"
	case 2:
		return s == "S"
	default:
		return false
	}
}

func (y *YesValidationModel) Init() tea.Cmd {
	y.renderer = &yesValidationRenderer{m: y}
	return y.renderer.Init()
}

func (y *YesValidationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return y.renderer.Update(msg)
}

func (y *YesValidationModel) View() string {
	return y.renderer.View()
}

// Init initializes the text input with appropriate settings for YES validation.
// It sets up the input field with:
// - 3 character limit (for "YES")
// - Custom prompt and placeholder
// - Appropriate styling
func (i *yesValidationRenderer) Init() tea.Cmd {
	input := textinput.New()
	input.Placeholder = "YES"

	if strings.HasSuffix(i.m.Prompt, " ") {
		input.Prompt = i.m.Prompt
	} else {
		input.Prompt = i.m.Prompt + " "
	}

	input.PromptStyle = i.m.Styles.Prompt
	input.PlaceholderStyle = i.m.Styles.Placeholder
	input.TextStyle = i.m.Styles.Text
	input.CharLimit = 3
	input.Focus()
	i.text = input
	return nil
}

// Update handles input processing and validation.
// It processes several types of input:
// - Ctrl+C/Esc: Cancels the operation (equivalent to "No")
// - Enter: Confirms if input is exactly "YES"
// - Backspace: Always allowed for correction
// - Characters: Only allows Y, E, S in sequence
func (i *yesValidationRenderer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			i.m.SetDecision(Denied)
			i.m.done = true
			return i.m, tea.Quit
		case tea.KeyEnter:
			if i.text.Value() == i.m.AcceptedDecisionText {
				i.m.SetDecision(Accepted)
				i.m.done = true
				return i.m, tea.Quit
			}
		case tea.KeyBackspace:
			// Always allow backspace for corrections
			i.text, cmd = i.text.Update(msg)
			return i.m, cmd
		default:
			// Only allow valid characters in sequence
			if isValidYesChar(msg.String(), i.text.Value()) {
				i.text, cmd = i.text.Update(msg)
			}
			return i.m, cmd
		}
	}

	return i.m, cmd
}

// View renders the current state of the confirmation prompt.
// It shows:
// - The prompt text
// - Current input
// - Validation indicator (✓ in green for "YES", ✗ in red otherwise)
// The view adapts based on whether input is complete or still in progress.
func (i *yesValidationRenderer) View() string {
	var b strings.Builder

	if i.m.PromptPrefix != "" {
		promptPrefixRender := i.m.Styles.PromptPrefix.Inline(true).Render
		b.WriteString(promptPrefixRender(i.m.PromptPrefix))
		if !strings.HasSuffix(i.m.PromptPrefix, " ") {
			b.WriteString(promptPrefixRender(" "))
		}
	}

	if i.m.done {
		if i.m.Prompt != "" {
			promptRender := i.m.Styles.Prompt.Inline(true).Render
			b.WriteString(promptRender(i.m.Prompt))
			b.WriteString(promptRender(" "))
		}
		b.WriteString(i.m.Value())
		b.WriteRune('\n')
		return b.String()
	}

	b.WriteString(i.text.View())

	// Show validation status with colored indicators
	if i.m.showValidation {
		b.WriteString(" ")
		if i.text.Value() == i.m.AcceptedDecisionText {
			b.WriteString(i.m.validStyle.Render("✓"))
		} else {
			b.WriteString(i.m.invalidStyle.Render("✗"))
		}
	}

	return b.String()
}
