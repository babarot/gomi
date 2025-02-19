package styles

import (
	"strings"

	"github.com/babarot/gomi/internal/config"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Package styles provides unified styling for the UI components

// Styles holds all UI styles for consistent theming
type Styles struct {
	List    ListStyles
	Detail  DetailStyles
	Dialog  DialogStyles
	Confirm ConfirmStyles
}

// ListStyles contains styles for list view
type ListStyles struct {
	Normal      lipgloss.Style
	Selected    lipgloss.Style
	Cursor      lipgloss.Style
	Description lipgloss.Style
	Title       lipgloss.Style
	FilterMatch lipgloss.Style
}

// DetailStyles contains styles for detail view
type DetailStyles struct {
	View     func(sections ...string) string
	Title    lipgloss.Style
	Selected lipgloss.Style
	Border   lipgloss.Style
	Info     InfoStyles
	Preview  PreviewStyles
}

// InfoStyles contains styles for information display
type InfoStyles struct {
	DeletedFrom DeletedInfoStyle
	DeletedAt   DeletedInfoStyle
}

type DeletedInfoStyle struct {
	Title   lipgloss.Style
	Content lipgloss.Style
	Section lipgloss.Style
}

// PreviewStyles contains styles for preview pane
type PreviewStyles struct {
	Border lipgloss.Style
	Size   lipgloss.Style
	Scroll lipgloss.Style
	Error  PreviewErrorStyles
}

type PreviewErrorStyles struct {
	Title   lipgloss.Style
	Content lipgloss.Style
}

// DialogStyles contains styles for dialogs
type DialogStyles struct {
	Box       lipgloss.Style
	Text      lipgloss.Style
	Separator lipgloss.Style
}

// ConfirmStyles contains styles for confirmation prompts
type ConfirmStyles struct {
	Prompt     lipgloss.Style
	Text       lipgloss.Style
	Indicator  lipgloss.Style
	Prefix     lipgloss.Style
	Error      lipgloss.Style
	Suggestion lipgloss.Style
}

// New creates a new Styles instance with the provided configuration
func New(cfg config.UI) *Styles {
	s := &Styles{}

	s.List = ListStyles{
		Normal: lipgloss.NewStyle().
			Padding(0, 0, 0, 2),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.Style.ListView.Selected)).
			Padding(0, 0, 0, 2),
		Cursor: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color(cfg.Style.ListView.Cursor)).
			Foreground(lipgloss.Color(cfg.Style.ListView.Cursor)).
			Padding(0, 0, 0, 1),
		Description: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}),
		Title: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(cfg.Style.DetailView.Border)).
			Padding(0, 1).
			Bold(true),
		FilterMatch: lipgloss.NewStyle().
			Underline(true),
	}

	s.Detail = DetailStyles{
		View: func(sections ...string) string {
			return lipgloss.JoinVertical(lipgloss.Left, sections...)
		},
		Border: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(cfg.Style.DetailView.Border)),
		Title: lipgloss.NewStyle().
			BorderForeground(lipgloss.Color(cfg.Style.DetailView.Border)).
			Padding(0, 1).
			Bold(true),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#000000"}).
			Background(lipgloss.AdaptiveColor{Light: cfg.Style.ListView.Selected, Dark: cfg.Style.ListView.Selected}),
		Info: InfoStyles{
			DeletedFrom: DeletedInfoStyle{
				Title: lipgloss.NewStyle().
					Padding(0, 1).
					Background(lipgloss.Color(cfg.Style.DetailView.InfoPane.DeletedFrom.Background)).
					Foreground(lipgloss.Color(cfg.Style.DetailView.InfoPane.DeletedFrom.Foreground)).
					Bold(true).
					Transform(strings.ToUpper),
				Content: lipgloss.NewStyle().
					Padding(0, 1),
				Section: lipgloss.NewStyle().
					BorderStyle(lipgloss.HiddenBorder()).
					Padding(0, 1),
			},
			DeletedAt: DeletedInfoStyle{
				Title: lipgloss.NewStyle().
					Padding(0, 1).
					Background(lipgloss.Color(cfg.Style.DetailView.InfoPane.DeletedAt.Background)).
					Foreground(lipgloss.Color(cfg.Style.DetailView.InfoPane.DeletedAt.Foreground)).
					Bold(true).
					Transform(strings.ToUpper),
				Content: lipgloss.NewStyle().
					Padding(0, 1),
				Section: lipgloss.NewStyle().
					BorderStyle(lipgloss.HiddenBorder()).
					Padding(0, 1),
			},
		},
		Preview: PreviewStyles{
			Border: lipgloss.NewStyle().
				Foreground(lipgloss.Color(cfg.Style.DetailView.PreviewPane.Border)),
			Size: lipgloss.NewStyle().
				Padding(0, 1, 0, 1).
				Foreground(lipgloss.Color(cfg.Style.DetailView.PreviewPane.Size.Foreground)).
				Background(lipgloss.Color(cfg.Style.DetailView.PreviewPane.Size.Background)),
			Scroll: lipgloss.NewStyle().
				Padding(0, 1, 0, 1).
				Foreground(lipgloss.Color(cfg.Style.DetailView.PreviewPane.Scroll.Foreground)).
				Background(lipgloss.Color(cfg.Style.DetailView.PreviewPane.Scroll.Background)),
			Error: PreviewErrorStyles{
				Title: lipgloss.NewStyle().
					Foreground(lipgloss.ANSIColor(termenv.ANSIBrightBlack)),
				Content: lipgloss.NewStyle().
					Bold(true).
					Transform(strings.ToUpper),
			},
		},
	}

	s.Dialog = DialogStyles{
		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(cfg.Style.DeletionDialog)).
			Foreground(lipgloss.Color(cfg.Style.DeletionDialog)).
			Bold(true).
			Padding(1, 1).
			BorderTop(true).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true).
			Align(lipgloss.Center),
		Text: lipgloss.NewStyle().
			Align(lipgloss.Center),
		Separator: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.Style.DetailView.Border)),
	}

	s.Confirm = ConfirmStyles{
		Prompt: lipgloss.NewStyle().
			Bold(true),
		Text: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#1F1F1F", Dark: "#FFFFFF"}),
		Indicator: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.Style.ListView.Selected)),
		Prefix: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.Style.ListView.Cursor)),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")),
		Suggestion: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}).
			Italic(true),
	}

	return s
}

func (s *Styles) RenderListItem(title, description string, selected, cursor bool) string {
	var style lipgloss.Style
	if cursor {
		if selected {
			style = s.List.Cursor
			style = style.Inherit(s.List.Selected)
		} else {
			style = s.List.Cursor
		}
	} else if selected {
		style = s.List.Selected
	} else {
		style = s.List.Normal
	}

	return style.Render(title + "\n" + s.List.Description.Render(description))
}

func (s *Styles) RenderDetailTitle(title string, width int, selected bool) string {
	filename := title
	if selected {
		title = s.Detail.Selected.Render(filename)
	}

	renderedTitle := s.Detail.Title.
		BorderStyle(func() lipgloss.Border {
			b := lipgloss.RoundedBorder()
			if len(filename) < width {
				b.Right = "├"
			}
			return b
		}()).
		Render(title)

	line := s.Dialog.Separator.Render(strings.Repeat("─", width-lipgloss.Width(renderedTitle)))
	return lipgloss.JoinHorizontal(lipgloss.Center, renderedTitle, line)
}

func (s *Styles) RenderErrorPreview(errorMsg, mime string, width, height int) string {
	title := s.Detail.Preview.Error.Content.Render(errorMsg)
	content := s.Detail.Preview.Error.Title.Render("(" + mime + ")")

	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		title+"\n\n\n"+content,
		lipgloss.WithWhitespaceChars("`"),
		lipgloss.WithWhitespaceForeground(lipgloss.ANSIColor(termenv.ANSIBrightBlack)),
	)
}

func (s *Styles) RenderDeletedFrom(title, content string) string {
	return s.Detail.Info.DeletedFrom.Section.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			s.Detail.Info.DeletedFrom.Title.MarginBottom(1).Render(title),
			s.Detail.Info.DeletedFrom.Content.Render(content),
		),
	)
}

func (s *Styles) RenderDeletedAt(title, content string) string {
	return s.Detail.Info.DeletedAt.Section.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			s.Detail.Info.DeletedAt.Title.MarginRight(3).Render(title),
			s.Detail.Info.DeletedAt.Content.Render(content),
		),
	)
}

// RenderPreviewFrame renders the preview frame with size or scroll info
func (s *Styles) RenderPreviewFrame(content string, isSize bool, width int) string {
	var infoStyle lipgloss.Style
	if isSize {
		infoStyle = s.Detail.Preview.Size
	} else {
		infoStyle = s.Detail.Preview.Scroll
	}

	if content == "" {
		line := s.Detail.Preview.Border.Render(strings.Repeat("─", width))
		return lipgloss.JoinHorizontal(lipgloss.Center, line)
	}

	content = infoStyle.Render(content)
	line := s.Detail.Preview.Border.Render(strings.Repeat("─", width-lipgloss.Width(content)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, content)
}

func (s *Styles) RenderDialog(content string) string {
	return s.Dialog.Box.Render(
		s.Dialog.Text.Render(content),
	)
}

func (s *Styles) RenderDimmed(title, desc string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		s.List.Normal.Inherit(s.List.Description).Render(title),
		s.List.Description.Render(desc),
	)
}

func (s *Styles) RenderFilterMatch(title string, matchedRunes []int) string {
	unmatched := s.List.Normal.Inline(true)
	matched := unmatched.Inherit(s.List.FilterMatch)
	return lipgloss.StyleRunes(title, matchedRunes, matched, unmatched)
}

func (s *Styles) RenderSelectedCursor(title, desc string) string {
	style := s.List.Cursor
	style = style.Inherit(s.List.Selected)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		style.Render(title),
		s.List.Description.Inherit(s.List.Selected).Render(desc),
	)
}

func (s *Styles) RenderCursor(title, desc string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		s.List.Cursor.Render(title),
		s.List.Description.Render(desc),
	)
}

func (s *Styles) RenderSelected(title, desc string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		s.List.Selected.Render(title),
		s.List.Description.Inherit(s.List.Selected).Render(desc),
	)
}

func (s *Styles) RenderNormal(title, desc string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		s.List.Normal.Render(title),
		s.List.Description.Render(desc),
	)
}
