package confirm

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// rendering is how the confirmation is drawn and takes keys. Update reports
// only the command; the Model it belongs to is what the program runs.
type rendering interface {
	Init() tea.Cmd
	Update(msg tea.Msg) tea.Cmd
	View() string
}

// inputStyles styles a text input like bubbles v1 did: the given prompt,
// placeholder and text styles in either focus state, and a cursor drawn
// reversed with no color of its own.
func inputStyles(prompt, placeholder, text lipgloss.Style) textinput.Styles {
	s := textinput.DefaultStyles(true)
	for _, st := range []*textinput.StyleState{&s.Focused, &s.Blurred} {
		st.Prompt = prompt
		st.Placeholder = placeholder
		st.Text = text
	}
	s.Cursor.Color = lipgloss.NoColor{}
	return s
}

// fitPlaceholder works around bubbles v2 drawing only the first character of
// the placeholder when the input has no width: while the input is empty, it
// gets the narrowest width that shows the whole placeholder with no padding,
// and no width once something is typed.
func fitPlaceholder(t *textinput.Model) {
	w := 0
	if t.Value() == "" {
		w = max(utf8.RuneCountInString(t.Placeholder)-1, 0)
	}
	t.SetWidth(w)
}

// KeyMap defines the key bindings for selectable renderings
type KeyMap struct {
	Toggle key.Binding
	Help   key.Binding
	Enter  key.Binding
}

var DefaultVerticalKeyMap = KeyMap{
	Toggle: key.NewBinding(
		key.WithKeys("up", "down"),
		key.WithHelp("↑/↓", "toggle"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Enter: key.NewBinding(key.WithKeys("enter")),
}
var DefaultHorizontalKeyMap = KeyMap{
	Toggle: key.NewBinding(
		key.WithKeys("left", "right"),
		key.WithHelp("←/→", "toggle"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Enter: key.NewBinding(key.WithKeys("enter")),
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Toggle}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Toggle},
	}
}

// selectionRenderer renders in a horizontal or vertical selection list
type selectionRenderer struct {
	m          *Model
	KeyMap     KeyMap
	isVertical bool
	help       help.Model
	hideHelp   bool
}

// Init satisfies the tea.Model interface
func (s *selectionRenderer) Init() tea.Cmd {
	s.isVertical = s.m.Rendering == VerticalSelection
	if s.isVertical {
		s.KeyMap = DefaultVerticalKeyMap
	} else {
		s.KeyMap = DefaultHorizontalKeyMap
	}
	s.help = help.New()
	s.hideHelp = !s.m.ShowHelp
	return nil
}

// Update satisfies the tea.Model interface
func (s *selectionRenderer) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// If we set a width on the help menu it can gracefully truncate its view as needed.
		s.help.SetWidth(msg.Width)
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, s.KeyMap.Enter):
			s.hideHelp = true
			return tea.Quit
		case key.Matches(msg, s.KeyMap.Toggle):
			switch s.m.selected {
			case Undecided, Denied:
				s.m.SetDecision(Accepted)
			default:
				s.m.SetDecision(Denied)
			}
		case key.Matches(msg, s.KeyMap.Help):
			s.help.ShowAll = !s.help.ShowAll
		}
	}
	return nil
}

// View satisfies the tea.Model interface
func (s *selectionRenderer) View() string {
	styleText := s.m.Styles.Text.Inline(true).Render

	var b strings.Builder
	promptPrefixRender := s.m.Styles.PromptPrefix.Inline(true).Render
	b.WriteString(promptPrefixRender(s.m.PromptPrefix))
	if !strings.HasSuffix(s.m.PromptPrefix, " ") {
		b.WriteString(promptPrefixRender(" "))
	}

	promptRender := s.m.Styles.Prompt.Render
	b.WriteString(promptRender(s.m.Prompt))
	chooser := s.m.Styles.ChooserIndicator.Inline(true).Render(string(s.m.ChooserIndicator))
	if !s.isVertical {
		b.WriteString(promptRender(" "))
		if s.m.selected == Accepted {
			b.WriteString(chooser)
		} else {
			b.WriteString(styleText(" "))
		}
		b.WriteString(styleText(s.m.AcceptedDecisionText))

		b.WriteString(styleText(" "))
		if s.m.selected == Denied {
			b.WriteString(chooser)
		} else {
			b.WriteString(styleText(" "))
		}
		b.WriteString(styleText(s.m.DeniedDecisionText))
	} else {
		b.WriteString("\n")
		if s.m.selected == Accepted {
			b.WriteString(chooser)
		} else {
			b.WriteString(styleText(" "))
		}
		b.WriteString(styleText(" "))
		b.WriteString(styleText(s.m.AcceptedDecisionText))
		b.WriteString(" \n")
		if s.m.selected == Denied {
			b.WriteString(chooser)
		} else {
			b.WriteString(styleText(" "))
		}
		b.WriteString(styleText(" "))
		b.WriteString(styleText(s.m.DeniedDecisionText))
	}

	if !s.hideHelp {
		helpView := s.help.View(s.KeyMap)
		b.WriteString("\n")
		b.WriteString(helpView)
	}
	b.WriteString("\n")
	return b.String()
}

// inputRenderer renders in one-line as user-based textual input
type inputRenderer struct {
	m    *Model
	text textinput.Model
}

// Init satisfies the tea.Model interface
func (i *inputRenderer) Init() tea.Cmd {
	input := textinput.New()
	if i.m.Placeholder != "" {
		input.Placeholder = i.m.Placeholder
	} else {
		var s strings.Builder
		if i.m.DefaultValue == Accepted {
			s.WriteString(strings.ToUpper(i.m.AcceptedInputText))
		} else {
			s.WriteString(i.m.AcceptedInputText)
		}

		s.WriteString("/")

		if i.m.DefaultValue == Denied {
			s.WriteString(strings.ToUpper(i.m.DeniedInputText))
		} else {
			s.WriteString(i.m.DeniedInputText)
		}

		input.Placeholder = s.String()
	}
	if strings.HasSuffix(i.m.Prompt, " ") {
		input.Prompt = i.m.Prompt
	} else {
		input.Prompt = i.m.Prompt + " "
	}
	input.SetStyles(inputStyles(i.m.Styles.Prompt, i.m.Styles.Placeholder, i.m.Styles.Text))
	input.CharLimit = max(len(i.m.DeniedDecisionText), len(i.m.AcceptedDecisionText))
	input.Focus()
	fitPlaceholder(&input)
	i.text = input
	return nil
}

// Update satisfies the tea.Model interface
func (i *inputRenderer) Update(msg tea.Msg) tea.Cmd {
	var c tea.Cmd
	i.text, c = i.text.Update(msg)
	fitPlaceholder(&i.text)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			switch k := strings.ToLower(i.text.Value()); {
			case strings.HasPrefix(k, strings.ToLower(i.m.AcceptedDecisionText)):
				i.m.SetDecision(Accepted)
			case strings.HasPrefix(k, strings.ToLower(i.m.DeniedDecisionText)):
				i.m.SetDecision(Denied)
			}
			i.m.done = true
			return tea.Quit
		}
	}

	return c
}

// View satisfies the tea.Model interface
func (i *inputRenderer) View() string {
	var b strings.Builder
	if i.m.PromptPrefix != "" {
		promptPrefixRender := i.m.Styles.PromptPrefix.Inline(true).Render
		b.WriteString(promptPrefixRender(i.m.PromptPrefix))
		if i.m.Prompt != "" && !strings.HasSuffix(i.m.PromptPrefix, " ") {
			b.WriteString(promptPrefixRender(" "))
		}
	}

	if i.m.done {
		// rather than clearing the program output, we want to show the question + answer just as AlecAivazis/survey did
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

	return b.String()
}

type immediateRenderer struct {
	m    *Model
	text textinput.Model
}

func (i *immediateRenderer) Init() tea.Cmd {
	input := textinput.New()
	if i.m.Placeholder != "" {
		input.Placeholder = i.m.Placeholder
	} else {
		var s strings.Builder
		if i.m.DefaultValue == Accepted {
			s.WriteString(strings.ToUpper(i.m.AcceptedDecisionText))
		} else {
			s.WriteString(i.m.AcceptedDecisionText)
		}
		s.WriteString("/")
		if i.m.DefaultValue == Denied {
			s.WriteString(strings.ToUpper(i.m.DeniedDecisionText))
		} else {
			s.WriteString(i.m.DeniedDecisionText)
		}
		input.Placeholder = s.String()
	}

	if strings.HasSuffix(i.m.Prompt, " ") {
		input.Prompt = i.m.Prompt
	} else {
		input.Prompt = i.m.Prompt + " "
	}

	input.SetStyles(inputStyles(i.m.Styles.Prompt, i.m.Styles.Placeholder, i.m.Styles.Text))
	input.CharLimit = max(len(i.m.DeniedDecisionText), len(i.m.AcceptedDecisionText))
	input.Focus()
	fitPlaceholder(&input)
	i.text = input
	return nil
}

func (i *immediateRenderer) Update(msg tea.Msg) tea.Cmd {
	isLetter := func(s string) bool {
		return !strings.ContainsFunc(s, func(r rune) bool {
			return !unicode.IsLetter(r)
		})
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			i.m.SetDecision(Denied)
			i.m.done = true
			return tea.Quit
		default:
			if isLetter(msg.String()) {
				switch strings.ToLower(msg.String()) {
				case strings.ToLower(string(i.m.AcceptedDecisionText[0])):
					i.m.SetDecision(Accepted)
					i.m.done = true
					return tea.Quit
				case strings.ToLower(string(i.m.DeniedDecisionText[0])):
					i.m.SetDecision(Denied)
					i.m.done = true
					return tea.Quit
				}
			}
		}
	}
	return nil
}

func (i *immediateRenderer) View() string {
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
	return b.String()
}
