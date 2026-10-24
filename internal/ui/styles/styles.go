package styles

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Styles holds all UI styles for consistent theming
type Styles struct {
	Detail  DetailStyles
	Dialog  DialogStyles
	Confirm ConfirmStyles
}

// DetailStyles contains styles for detail view
type DetailStyles struct {
	View     func(sections ...string) string
	Title    lipgloss.Style
	Selected lipgloss.Style
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
	Filler  lipgloss.Style
}

// DialogStyles contains styles for dialogs
type DialogStyles struct {
	Box       lipgloss.Style
	Text      lipgloss.Style
	Separator lipgloss.Style
}

// ConfirmStyles contains styles for typing YES in the delete confirmation
type ConfirmStyles struct {
	Text        lipgloss.Style
	Success     lipgloss.Style
	Error       lipgloss.Style
	Placeholder lipgloss.Style
}

// New creates the styles drawn with p, for a dark or light terminal
// background
func New(p Palette, isDark bool) *Styles {
	s := &Styles{}
	fg := func(c Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c.Of(isDark)) }

	s.Detail = DetailStyles{
		View: func(sections ...string) string {
			return lipgloss.JoinVertical(lipgloss.Left, sections...)
		},
		Title: lipgloss.NewStyle().
			BorderForeground(p.Border.Of(isDark)).
			Padding(0, 1).
			Bold(true),
		Selected: fg(p.SelectedTitle).
			Background(p.Selected.Of(isDark)),
		Info: InfoStyles{
			DeletedFrom: DeletedInfoStyle{
				Title: lipgloss.NewStyle().
					Padding(0, 1).
					Background(p.DeletedFromBackground.Of(isDark)).
					Foreground(p.DeletedFrom.Of(isDark)).
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
					Background(p.DeletedAtBackground.Of(isDark)).
					Foreground(p.DeletedAt.Of(isDark)).
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
			Border: fg(p.PreviewBorder),
			Size: lipgloss.NewStyle().
				Padding(0, 1, 0, 1).
				Foreground(p.PreviewSize.Of(isDark)).
				Background(p.PreviewSizeBackground.Of(isDark)),
			Scroll: lipgloss.NewStyle().
				Padding(0, 1, 0, 1).
				Foreground(p.PreviewScroll.Of(isDark)).
				Background(p.PreviewScrollBackground.Of(isDark)),
			Error: PreviewErrorStyles{
				Title: fg(p.PreviewError),
				Content: lipgloss.NewStyle().
					Bold(true).
					Transform(strings.ToUpper),
				Filler: fg(p.PreviewError),
			},
		},
	}

	s.Dialog = DialogStyles{
		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.Dialog.Of(isDark)).
			Foreground(p.Dialog.Of(isDark)).
			Bold(true).
			Padding(1, 1).
			BorderTop(true).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true).
			Align(lipgloss.Center),
		Text: lipgloss.NewStyle().
			Align(lipgloss.Center),
		Separator: fg(p.Border),
	}

	s.Confirm = ConfirmStyles{
		Text:        fg(p.DialogInput),
		Success:     fg(p.DialogValid),
		Error:       fg(p.DialogInvalid),
		Placeholder: fg(p.DialogPlaceholder),
	}

	return s
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
		lipgloss.WithWhitespaceStyle(s.Detail.Preview.Error.Filler),
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
