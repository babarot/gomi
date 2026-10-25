package styles

import (
	"charm.land/lipgloss/v2"
)

// Styles holds all UI styles for consistent theming
type Styles struct {
	Dialog  DialogStyles
	Confirm ConfirmStyles
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

func (s *Styles) RenderDialog(content string) string {
	return s.Dialog.Box.Render(
		s.Dialog.Text.Render(content),
	)
}
