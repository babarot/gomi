package keys

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
)

type DetailKeyMap struct {
	Space  key.Binding
	Next   key.Binding
	Prev   key.Binding
	Esc    key.Binding
	AtSign key.Binding

	// Preview
	GotoTop      key.Binding
	GotoBottom   key.Binding
	PreviewUp    key.Binding
	PreviewDown  key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding

	Help      key.Binding
	HelpClose key.Binding
}

func (k DetailKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Next,
		k.Prev,
		k.Space,
		k.Help,
	}
}

func (k DetailKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			k.Next,
			k.Prev,
			k.Space, k.Esc,
			k.AtSign,
		},
		{k.PreviewUp, k.PreviewDown, k.HalfPageUp, k.HalfPageDown, k.GotoTop, k.GotoBottom},
		{k.HelpClose},
	}
}

var DetailKeys = &DetailKeyMap{
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "more"),
	),
	HelpClose: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "close help"),
	),
	Space: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "back"),
	),
	PreviewUp: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "preview up"),
	),
	PreviewDown: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "preview down"),
	),
	Next: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next"),
	),
	Prev: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "prev"),
	),
	Esc: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	AtSign: key.NewBinding(
		key.WithKeys("@"),
		key.WithHelp("@", "datefmt"),
	),
	GotoTop:      key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "go to start")),
	GotoBottom:   key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "go to end")),
	HalfPageUp:   key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "½ page up")),
	HalfPageDown: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "½ page down")),
}

var PreviewKeys = viewport.KeyMap{
	Up:           key.NewBinding(key.WithKeys("k", "up")),
	Down:         key.NewBinding(key.WithKeys("j", "down")),
	HalfPageUp:   key.NewBinding(key.WithKeys("u")),
	HalfPageDown: key.NewBinding(key.WithKeys("d")),
}
