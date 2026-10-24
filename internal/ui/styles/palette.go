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

// NewPalette returns the palette for the colors in the config's ui.style.
//
// A config color is used as it is, even when empty: a config file is
// written with every color when it is created, and a color later left out of
// it is no color, as it always has been. Colors the config has no setting for
// are fixed here.
func NewPalette(cfg config.StyleConfig) Palette {
	return Palette{
		Cursor:            same(cfg.ListView.Cursor),
		Selected:          same(cfg.ListView.Selected),
		FilterMatch:       same(cfg.ListView.FilterMatch),
		FilterPrompt:      same(cfg.ListView.FilterPrompt),
		Description:       Color{Light: "#626262", Dark: "#777777"},
		Dimmed:            Color{Light: "#808080", Dark: "#777777"},
		DimmedDescription: Color{Light: "#999999", Dark: "#4D4D4D"},

		Border:                  same(cfg.DetailView.Border),
		SelectedTitle:           same("#000000"),
		DeletedFrom:             same(cfg.DetailView.InfoPane.DeletedFrom.Foreground),
		DeletedFromBackground:   same(cfg.DetailView.InfoPane.DeletedFrom.Background),
		DeletedAt:               same(cfg.DetailView.InfoPane.DeletedAt.Foreground),
		DeletedAtBackground:     same(cfg.DetailView.InfoPane.DeletedAt.Background),
		PreviewBorder:           same(cfg.DetailView.PreviewPane.Border),
		PreviewSize:             same(cfg.DetailView.PreviewPane.Size.Foreground),
		PreviewSizeBackground:   same(cfg.DetailView.PreviewPane.Size.Background),
		PreviewScroll:           same(cfg.DetailView.PreviewPane.Scroll.Foreground),
		PreviewScrollBackground: same(cfg.DetailView.PreviewPane.Scroll.Background),
		PreviewError:            same("8"),

		Dialog:            same(cfg.DeletionDialog),
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
