package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/dustin/go-humanize"
	"github.com/muesli/reflow/wordwrap"
)

// updateDetail handles key events and updates for the detail view.
func (m Model) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.detailKeys.Space),
			key.Matches(msg, m.detailKeys.Esc):
			// Return to list view
			m.state = ListView
			return m, nil

		case key.Matches(msg, m.detailKeys.Prev):
			// Move to previous item
			m.list.CursorUp()
			item := m.currentListItem()
			if item != nil {
				m.currentItem = item
				cmds = append(cmds, previewCmd(item))
			}

		case key.Matches(msg, m.detailKeys.Next):
			// Move to next item
			m.list.CursorDown()
			item := m.currentListItem()
			if item != nil {
				m.currentItem = item
				cmds = append(cmds, previewCmd(item))
			}

		case key.Matches(msg, m.detailKeys.AtSign):
			// Toggle date format
			switch m.dateFormat {
			case DateFormatRelative:
				m.dateFormat = DateFormatAbsolute
			case DateFormatAbsolute:
				m.dateFormat = DateFormatRelative
			}

		case key.Matches(msg, m.detailKeys.GotoTop):
			m.viewport.GotoTop()

		case key.Matches(msg, m.detailKeys.GotoBottom):
			m.viewport.GotoBottom()

		case key.Matches(msg, m.detailKeys.Help):
			m.help.ShowAll = !m.help.ShowAll
		}

		// Handle viewport scrolling keys
		if m.currentItem != nil {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// renderDetail renders the entire detail view.
func (m Model) renderDetail() string {
	if m.currentItem == nil {
		return ""
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderDetailHeader(),
		m.renderDeletedFrom(),
		m.renderDeletedAt(),
		m.renderPreview(),
		m.renderDetailFooter(),
		lipgloss.NewStyle().Margin(1, 2).Render(m.help.View(m.detailKeys)),
	)
}

// renderDetailHeader renders the header of the detail view.
func (m Model) renderDetailHeader() string {
	borderForeground := m.config.Style.DetailView.Border
	name := ansi.Truncate(m.currentItem.Title(), defaultWidth-len(ellipsis), ellipsis)

	if m.isSelected(m.currentItem) {
		selected := m.config.Style.ListView.Selected
		name = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#000000"}).
			Background(lipgloss.AdaptiveColor{Light: selected, Dark: selected}).
			Render(name)
	}

	title := lipgloss.NewStyle().
		BorderStyle(func() lipgloss.Border {
			b := lipgloss.RoundedBorder()
			if len(m.currentItem.Title()) < defaultWidth {
				b.Right = "├"
			}
			return b
		}()).
		BorderForeground(lipgloss.Color(borderForeground)).
		Padding(0, 1).
		Bold(true).
		Render(name)

	line := lipgloss.NewStyle().
		Foreground(lipgloss.Color(borderForeground)).
		Render(strings.Repeat("─", max(0, defaultWidth-lipgloss.Width(title))))

	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

// renderDeletedFrom renders the deletion path information.
func (m Model) renderDeletedFrom() string {
	text := m.currentItem.Description()
	w := wordwrap.NewWriter(46)
	w.Breakpoints = []rune{'/', '.'}
	w.KeepNewlines = false
	_, _ = w.Write([]byte(text))
	_ = w.Close()

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.HiddenBorder()).
		Padding(0, 1).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				lipgloss.NewStyle().
					Padding(0, 1).
					Background(lipgloss.Color(m.config.Style.DetailView.InfoPane.DeletedFrom.Background)).
					Foreground(lipgloss.Color(m.config.Style.DetailView.InfoPane.DeletedFrom.Foreground)).
					Bold(true).
					Transform(strings.ToUpper).
					MarginBottom(1).
					Render("Deleted From"),
				lipgloss.NewStyle().Render(w.String()),
			),
		)
}

// renderDeletedAt renders the deletion time information.
func (m Model) renderDeletedAt() string {
	var ts string
	switch m.dateFormat {
	case DateFormatAbsolute:
		ts = m.currentItem.File().DeletedAt.Format(time.DateTime)
	default:
		ts = humanize.Time(m.currentItem.File().DeletedAt)
	}

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.HiddenBorder()).
		Padding(0, 1).
		Render(
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				lipgloss.NewStyle().
					Padding(0, 1).
					Background(lipgloss.Color(m.config.Style.DetailView.InfoPane.DeletedAt.Background)).
					Foreground(lipgloss.Color(m.config.Style.DetailView.InfoPane.DeletedAt.Foreground)).
					Bold(true).
					Transform(strings.ToUpper).
					MarginRight(3).
					Render("Deleted At"),
				lipgloss.NewStyle().Render(ts),
			),
		)
}

// renderPreview renders the file preview content.
func (m Model) renderPreview() string {
	preview, err := m.currentItem.Preview()
	if err != nil {
		content := lipgloss.Place(defaultWidth, 10,
			lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Bold(true).Transform(strings.ToUpper).Render(err.Error()),
		)
		return fmt.Sprintf("%s\n%s\n%s",
			m.renderPreviewHeader(),
			content,
			m.renderPreviewFooter(),
		)
	}

	m.viewport.SetContent(preview)
	return fmt.Sprintf("%s\n%s\n%s",
		m.renderPreviewHeader(),
		m.viewport.View(),
		m.renderPreviewFooter(),
	)
}

// renderPreviewHeader renders the header of the preview section.
func (m Model) renderPreviewHeader() string {
	color := m.config.Style.DetailView.PreviewPane.Border
	size := lipgloss.NewStyle().
		Padding(0, 1, 0, 1).
		Foreground(lipgloss.Color(m.config.Style.DetailView.PreviewPane.Size.Foreground)).
		Background(lipgloss.Color(m.config.Style.DetailView.PreviewPane.Size.Background)).
		Render(m.currentItem.Size())
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(size)))
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(lipgloss.JoinHorizontal(lipgloss.Center, line, size))
}

// renderPreviewFooter renders the footer of the preview section.
func (m Model) renderPreviewFooter() string {
	color := m.config.Style.DetailView.PreviewPane.Border
	_, err := m.currentItem.Preview()
	if err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(strings.Repeat("─", defaultWidth))
	}

	info := lipgloss.NewStyle().
		Padding(0, 1, 0, 1).
		Foreground(lipgloss.Color(m.config.Style.DetailView.PreviewPane.Scroll.Foreground)).
		Background(lipgloss.Color(m.config.Style.DetailView.PreviewPane.Scroll.Background)).
		Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Render(lipgloss.JoinHorizontal(lipgloss.Center, line, info))
}

// renderDetailFooter renders the footer of the detail view.
func (m Model) renderDetailFooter() string {
	color := m.config.Style.DetailView.Border
	line := lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Render(strings.Repeat("─", defaultWidth))
	return lipgloss.JoinHorizontal(lipgloss.Center, line)
}

// max returns the larger of x or y.
func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
