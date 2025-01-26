package keys

import "github.com/charmbracelet/bubbles/key"

type ListKeyMap struct {
	Quit     key.Binding
	Enter    key.Binding
	Space    key.Binding
	Select   key.Binding
	DeSelect key.Binding
	Help     key.Binding
}

func (k ListKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Enter,
		k.Space,
		k.Select,
		k.Help,
	}
}

func (k ListKeyMap) FullHelp() []key.Binding {
	return append(
		k.ShortHelp(),
		k.DeSelect,
		k.Quit,
	)
}

var ListKeys = &ListKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Select: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "select"),
	),
	DeSelect: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("s+tab", "de-select"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "ok"),
	),
	Space: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "info"),
	),
}
