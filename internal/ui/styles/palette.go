package styles

import (
	"image/color"

	"charm.land/lipgloss/v2"

	"github.com/babarot/gomi/internal/config"
)

// Color is a color for a light and a dark terminal background. Values are
// hex colors or ANSI color numbers; an empty value is no color.
type Color struct {
	Light, Dark string
}

// same is a color that does not depend on the background.
func same(c string) Color { return Color{Light: c, Dark: c} }

// Of returns the color for a dark or light background.
func (c Color) Of(isDark bool) color.Color {
	if isDark {
		return lipgloss.Color(c.Dark)
	}
	return lipgloss.Color(c.Light)
}

// Palette is every color the TUI draws with, named by what it is used for.
type Palette struct {
	// The list
	Cursor            Color // the bar and the text of the row under the cursor
	Selected          Color // selected rows
	FilterMatch       Color // characters that match the filter
	FilterPrompt      Color // "Filter:"
	Description       Color // the line under a file name
	Dimmed            Color // file names while the filter is empty
	DimmedDescription Color // the line under them

	// The detail view
	Border                  Color // the frame around the file name and the line beside it
	SelectedTitle           Color // the file name of a selected file, on Selected
	DeletedFrom             Color // the "DELETED FROM" label
	DeletedFromBackground   Color
	DeletedAt               Color // the "DELETED AT" label
	DeletedAtBackground     Color
	PreviewBorder           Color // the lines above and below the preview
	PreviewSize             Color // the size on the line above the preview
	PreviewSizeBackground   Color
	PreviewScroll           Color // the scroll position on the line below it
	PreviewScrollBackground Color
	PreviewError            Color // the file type and the filler when a file cannot be previewed

	// The delete confirmation
	Dialog            Color // the frame and the text of the dialog
	DialogInput       Color // typed characters of YES
	DialogPlaceholder Color // characters of YES not typed yet
	DialogValid       Color // the mark once YES is typed
	DialogInvalid     Color // the mark until then

	// The help below each view
	HelpKey       Color
	HelpDesc      Color
	HelpSeparator Color
	HelpEllipsis  Color // the "…" when the help is cut off
}

// NewPalette returns the palette of the config's ui.theme. The "gomi"
// theme, also used when no theme is set, takes the colors set in ui.style
// over its own; other themes ignore ui.style. A config file is written with
// every ui.style color when it is created, so those colors would otherwise
// hide any theme chosen later.
func NewPalette(cfg config.UI) Palette {
	if s, ok := schemes[cfg.Theme]; ok {
		return s.palette()
	}
	return gomiPalette(cfg.Style)
}

// gomiPalette is gomi's own colors, with the colors set in st over them.
func gomiPalette(st config.StyleConfig) Palette {
	// or is the config's color when it is set, else def
	or := func(c string, def Color) Color {
		if c == "" {
			return def
		}
		return same(c)
	}
	return Palette{
		Cursor:            or(st.ListView.Cursor, same("#AD58B4")),
		Selected:          or(st.ListView.Selected, same("#5FB458")),
		FilterMatch:       or(st.ListView.FilterMatch, same("#F39C12")),
		FilterPrompt:      or(st.ListView.FilterPrompt, same("#7AA2F7")),
		Description:       Color{Light: "#626262", Dark: "#777777"},
		Dimmed:            Color{Light: "#808080", Dark: "#777777"},
		DimmedDescription: Color{Light: "#999999", Dark: "#4D4D4D"},

		Border:                  or(st.DetailView.Border, same("#EEEEDD")),
		SelectedTitle:           same("#000000"),
		DeletedFrom:             or(st.DetailView.InfoPane.DeletedFrom.Foreground, same("#EEEEEE")),
		DeletedFromBackground:   or(st.DetailView.InfoPane.DeletedFrom.Background, same("#1C1C1C")),
		DeletedAt:               or(st.DetailView.InfoPane.DeletedAt.Foreground, same("#EEEEEE")),
		DeletedAtBackground:     or(st.DetailView.InfoPane.DeletedAt.Background, same("#1C1C1C")),
		PreviewBorder:           or(st.DetailView.PreviewPane.Border, same("#3C3C3C")),
		PreviewSize:             or(st.DetailView.PreviewPane.Size.Foreground, same("#EEEEDD")),
		PreviewSizeBackground:   or(st.DetailView.PreviewPane.Size.Background, same("#3C3C3C")),
		PreviewScroll:           or(st.DetailView.PreviewPane.Scroll.Foreground, same("#EEEEDD")),
		PreviewScrollBackground: or(st.DetailView.PreviewPane.Scroll.Background, same("#3C3C3C")),
		PreviewError:            same("8"),

		Dialog:            or(st.DeletionDialog, same("#FF007F")),
		DialogInput:       Color{Light: "#1F1F1F", Dark: "#FFFFFF"},
		DialogPlaceholder: Color{Light: "#A49FA5", Dark: "#777777"},
		DialogValid:       same("#00FF00"),
		DialogInvalid:     same("#FF0000"),

		HelpKey:       same("#626262"),
		HelpDesc:      Color{Light: "#808080", Dark: "#4A4A4A"},
		HelpSeparator: Color{Light: "#999999", Dark: "#3C3C3C"},
		// Bubbles v2 made its default a little lighter; this is v1's
		HelpEllipsis: Color{Light: "#DDDADA", Dark: "#3C3C3C"},
	}
}

// scheme is a theme of a few colors named by what they are for; every
// color of the palette is one of them. A scheme is made for one
// background, so it draws the same on either.
type scheme struct {
	Base    string // the terminal background the scheme is made for
	Surface string // labels and bars drawn on a background of their own
	Border  string // frames and lines
	Muted   string // the line under a file name, help
	Dim     string // what matters least
	Subtle  string // the frame around the file name
	Text    string // text on Surface, typed text
	Accent  string // the cursor, the filter prompt
	OK      string // selected files
	Prompt  string // filter matches
	Warn    string // the delete confirmation
	Error   string // the mark until YES is typed
}

// schemes are the themes besides "gomi", by name
var schemes = map[string]scheme{
	"catppuccin-mocha": {
		Base: "#1e1e2e", Surface: "#3b3b52", Border: "#525270", Muted: "#8888a4", Dim: "#706f87",
		Subtle: "#b0b8d1", Text: "#e0e4f0", Accent: "#96bfff", OK: "#a6e3a1", Prompt: "#f9e2af",
		Warn: "#f5a3b5", Error: "#f38ba8",
	},
	"catppuccin-latte": {
		Base: "#eff1f5", Surface: "#ccd0da", Border: "#bcc0cc", Muted: "#8c8fa1", Dim: "#9ca0b0",
		Subtle: "#6c6f85", Text: "#4c4f69", Accent: "#1e66f5", OK: "#40a02b", Prompt: "#df8e1d",
		Warn: "#d20f39", Error: "#d20f39",
	},
	"nord": {
		Base: "#2e3440", Surface: "#3b4252", Border: "#4c566a", Muted: "#7b88a1", Dim: "#616e88",
		Subtle: "#d8dee9", Text: "#eceff4", Accent: "#88c0d0", OK: "#a3be8c", Prompt: "#ebcb8b",
		Warn: "#b48ead", Error: "#bf616a",
	},
}

func (s scheme) palette() Palette {
	return Palette{
		Cursor:            same(s.Accent),
		Selected:          same(s.OK),
		FilterMatch:       same(s.Prompt),
		FilterPrompt:      same(s.Accent),
		Description:       same(s.Muted),
		Dimmed:            same(s.Muted),
		DimmedDescription: same(s.Dim),

		Border:                  same(s.Subtle),
		SelectedTitle:           same(s.Base),
		DeletedFrom:             same(s.Text),
		DeletedFromBackground:   same(s.Surface),
		DeletedAt:               same(s.Text),
		DeletedAtBackground:     same(s.Surface),
		PreviewBorder:           same(s.Border),
		PreviewSize:             same(s.Text),
		PreviewSizeBackground:   same(s.Surface),
		PreviewScroll:           same(s.Text),
		PreviewScrollBackground: same(s.Surface),
		PreviewError:            same(s.Dim),

		Dialog:            same(s.Warn),
		DialogInput:       same(s.Text),
		DialogPlaceholder: same(s.Muted),
		DialogValid:       same(s.OK),
		DialogInvalid:     same(s.Error),

		HelpKey:       same(s.Muted),
		HelpDesc:      same(s.Dim),
		HelpSeparator: same(s.Border),
		HelpEllipsis:  same(s.Border),
	}
}
