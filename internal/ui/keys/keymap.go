package keys

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
)

// KeyMapConfig holds configuration for key bindings
type KeyMapConfig struct {
	// DeleteEnabled controls whether delete functionality is available
	DeleteEnabled bool
}

// Common keys shared across views
type Common struct {
	Quit key.Binding
	Help key.Binding
}

// List view specific keys
type List struct {
	Space    key.Binding
	Esc      key.Binding
	Select   key.Binding
	DeSelect key.Binding
	Enter    key.Binding
	Delete   *key.Binding // Optional key based on configuration
}

// Detail view specific keys
type Detail struct {
	Space        key.Binding
	Esc          key.Binding
	PreviewUp    key.Binding
	PreviewDown  key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	Next         key.Binding
	Prev         key.Binding
	GotoTop      key.Binding
	GotoBottom   key.Binding
	AtSign       key.Binding
	Delete       *key.Binding // Optional key based on configuration
}

// Confirm view specific keys
type Confirm struct {
	Yes key.Binding
	No  key.Binding
}

// KeyMap holds all key bindings and help functions
type KeyMap struct {
	Common  Common
	List    List
	Detail  Detail
	Confirm Confirm

	// Function to generate help view
	shortHelp func() []key.Binding
	fullHelp  func() [][]key.Binding
}

var (
	DefaultKeyMapListCursorUp      = list.DefaultKeyMap().CursorUp
	DefaultKeyMapListCursorDown    = list.DefaultKeyMap().CursorDown
	DefaultKeyMapListNextPage      = list.DefaultKeyMap().NextPage
	DefaultKeyMapListPrevPage      = list.DefaultKeyMap().PrevPage
	DefaultKeyMapListGoToStart     = list.DefaultKeyMap().GoToStart
	DefaultKeyMapListGoToEnd       = list.DefaultKeyMap().GoToEnd
	DefaultKeyMapListFilter        = list.DefaultKeyMap().Filter
	DefaultKeyMapListClearFilter   = list.DefaultKeyMap().ClearFilter
	DefaultKeyMapListShowFullHelp  = list.DefaultKeyMap().ShowFullHelp
	DefaultKeyMapListCloseFullHelp = list.DefaultKeyMap().CloseFullHelp
)

// PreviewKeys holds key bindings for preview viewport
var PreviewKeys = viewport.KeyMap{
	Up:           key.NewBinding(key.WithKeys("k", "up")),
	Down:         key.NewBinding(key.WithKeys("j", "down")),
	HalfPageUp:   key.NewBinding(key.WithKeys("u")),
	HalfPageDown: key.NewBinding(key.WithKeys("d")),
}

// NewKeyMap creates a new key map with the given configuration
func NewKeyMap(cfg KeyMapConfig) *KeyMap {
	km := &KeyMap{}

	// Initialize common keys
	km.Common = Common{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
		),
	}

	// Initialize list view keys
	km.List = List{
		Space: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "detail"),
		),
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "reset"),
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
			key.WithHelp("enter", "restore"),
		),
	}

	// Initialize detail view keys
	km.Detail = Detail{
		Space: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "back"),
		),
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		PreviewUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "preview up"),
		),
		PreviewDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "preview down"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "½ page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "½ page down"),
		),
		Next: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next"),
		),
		Prev: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "prev"),
		),
		GotoTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "go to start"),
		),
		GotoBottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "go to end"),
		),
		AtSign: key.NewBinding(
			key.WithKeys("@"),
			key.WithHelp("@", "info"),
		),
	}

	// Add delete key if enabled
	if cfg.DeleteEnabled {
		deleteKey := key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "delete"),
		)
		km.List.Delete = &deleteKey
		km.Detail.Delete = &deleteKey
	}

	km.Confirm = Confirm{
		Yes: key.NewBinding(
			key.WithKeys("y", "Y"),
			key.WithHelp("y", "yes"),
		),
		No: key.NewBinding(
			key.WithKeys("n", "N"),
			key.WithHelp("n", "no"),
		),
	}

	// Set default full help function
	km.shortHelp = km.defaultShortHelp
	km.fullHelp = km.defaultFullHelp

	return km
}

// Help interface implementations

// ShortHelp returns condensed help view
func (k KeyMap) ShortHelp() []key.Binding {
	return k.shortHelp()
}

// FullHelp returns complete help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return k.fullHelp()
}

// Default help function
func (k KeyMap) defaultShortHelp() []key.Binding {
	return []key.Binding{
		k.Common.Quit, k.Common.Help,
	}
}

// Default help function
func (k KeyMap) defaultFullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Common.Quit, k.Common.Help},
	}
}

// AsListKeyMap returns a KeyMap that only shows list view related help
func (k KeyMap) AsListKeyMap() KeyMap {
	newMap := k
	newMap.shortHelp = func() []key.Binding {
		return []key.Binding{
			DefaultKeyMapListCursorUp,
			DefaultKeyMapListCursorDown,
			k.List.Enter,
			k.List.Space,
			k.List.Select,
			DefaultKeyMapListFilter,
			DefaultKeyMapListShowFullHelp,
		}
	}
	newMap.fullHelp = func() [][]key.Binding {
		bindings := [][]key.Binding{
			{
				DefaultKeyMapListCursorUp,
				DefaultKeyMapListCursorDown,
				DefaultKeyMapListNextPage,
				DefaultKeyMapListPrevPage,
				DefaultKeyMapListGoToStart,
				DefaultKeyMapListGoToEnd,
			},
			{k.List.Enter, k.List.Space, k.List.Esc, k.List.Select, k.List.DeSelect},
			{k.Common.Quit, DefaultKeyMapListCloseFullHelp},
		}
		if k.List.Delete != nil {
			bindings[1] = append(bindings[1], *k.List.Delete)
		}
		return bindings
	}
	return newMap
}

// AsDetailKeyMap returns a KeyMap that only shows detail view related help
func (k KeyMap) AsDetailKeyMap() KeyMap {
	newMap := k
	newMap.shortHelp = func() []key.Binding {
		return []key.Binding{
			k.Detail.Space, k.Detail.Next, k.Detail.Prev, DefaultKeyMapListShowFullHelp,
		}
	}
	newMap.fullHelp = func() [][]key.Binding {
		bindings := [][]key.Binding{
			{
				k.Detail.Next, k.Detail.Prev,
				k.Detail.Space, k.Detail.Esc,
				k.List.Select, k.List.DeSelect,
			},
			{
				k.Detail.PreviewUp, k.Detail.PreviewDown,
				k.Detail.HalfPageUp, k.Detail.HalfPageDown,
				k.Detail.GotoTop, k.Detail.GotoBottom,
			},
			{k.Detail.AtSign, k.Common.Quit, DefaultKeyMapListCloseFullHelp},
		}
		if k.Detail.Delete != nil {
			bindings[0] = append(bindings[0], *k.Detail.Delete)
		}
		return bindings
	}
	return newMap
}
