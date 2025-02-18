package keys

import "github.com/charmbracelet/bubbles/key"

type ListKeyMap struct {
	Quit     key.Binding
	Enter    key.Binding
	Space    key.Binding
	Select   key.Binding
	DeSelect key.Binding
	Delete   key.Binding
	Esc      key.Binding

	showDelete bool
}

func (k ListKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Enter,
		k.Space,
		k.Select,
	}
}

func (k ListKeyMap) FullHelp() [][]key.Binding {
	keys := append(
		k.ShortHelp(),
		k.DeSelect,
		k.Esc,
	)
	if k.showDelete {
		keys = append(keys, k.Delete)
	}
	keys = append(keys, k.Quit)
	return [][]key.Binding{
		keys,
	}
}

var ListKeys = &ListKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
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
		key.WithHelp("space", "detail"),
	),
	Esc: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "reset"),
	),
}

func (k *ListKeyMap) AddDeleteKey() {
	k.showDelete = true
	k.Delete = key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "delete"),
	)
}
