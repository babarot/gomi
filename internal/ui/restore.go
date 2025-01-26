package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/babarot/gomi/internal/config"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type RestoreItemStyles struct {
	NormalTitle lipgloss.Style
	NormalDesc  lipgloss.Style

	CursorTitle lipgloss.Style
	CursorDesc  lipgloss.Style

	SelectedTitle lipgloss.Style
	SelectedDesc  lipgloss.Style

	SelectedCursorTitle lipgloss.Style
	SelectedCursorDesc  lipgloss.Style

	DimmedTitle lipgloss.Style
	DimmedDesc  lipgloss.Style

	FilterMatch lipgloss.Style
}

func NewRestoreItemStyles(cfg config.UI) (s RestoreItemStyles) {
	indentOnSelect := 0
	if cfg.Style.ListView.IndentOnSelect {
		indentOnSelect = 1
	}

	// TODO: support adaptive?
	cursor := cfg.Style.ListView.Cursor
	selected := cfg.Style.ListView.Selected

	s.NormalTitle = lipgloss.NewStyle().
		Padding(0, 0, 0, 2)

	s.NormalDesc = s.NormalTitle.
		Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"})

	s.CursorTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.AdaptiveColor{Light: cursor, Dark: cursor}).
		Foreground(lipgloss.AdaptiveColor{Light: currentItemStyle.String(), Dark: cursor}).
		Padding(0, 0, 0, 1)

	s.CursorDesc = s.CursorTitle.
		Foreground(lipgloss.AdaptiveColor{Light: cursor, Dark: cursor})

	s.DimmedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}).
		Padding(0, 0, 0, 2)

	s.DimmedDesc = s.DimmedTitle.
		Foreground(lipgloss.AdaptiveColor{Light: "#C2B8C2", Dark: "#4D4D4D"})

	s.FilterMatch = lipgloss.NewStyle().Underline(true)

	s.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: selected, Dark: selected}).
		Padding(0, 0, 0, 2+indentOnSelect)

	s.SelectedDesc = s.SelectedTitle.
		Foreground(lipgloss.AdaptiveColor{Light: selected, Dark: selected})

	s.SelectedCursorTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.AdaptiveColor{Light: cursor, Dark: cursor}).
		Foreground(lipgloss.AdaptiveColor{Light: selected, Dark: selected}).
		Padding(0, 0, 0, 1)

	s.SelectedCursorDesc = s.SelectedCursorTitle.
		Foreground(lipgloss.AdaptiveColor{Light: selected, Dark: selected})

	return s
}

// RestoreItem describes an item designed to work with RestoreDelegate.
type RestoreItem interface {
	list.Item
	Title() string
	Description() string
}

type RestoreDelegate struct {
	Styles        RestoreItemStyles
	UpdateFunc    func(tea.Msg, *list.Model) tea.Cmd
	ShortHelpFunc func() []key.Binding
	FullHelpFunc  func() [][]key.Binding

	showDescription bool
	height          int
	spacing         int
}

// NewRestoreDelegate creates a new delegate with Restore styles.
func NewRestoreDelegate(cfg config.UI, files []File) RestoreDelegate {
	var height = 2
	var spacing = 1

	showDescription := true
	switch cfg.Density {
	case CompactDensityVal:
		showDescription = false
	case SpaciousDensityVal:
		showDescription = true
	}

	if !showDescription {
		height = 1
		spacing = 0
	}

	return RestoreDelegate{
		showDescription: showDescription,
		Styles:          NewRestoreItemStyles(cfg),
		height:          height,
		spacing:         spacing,
	}
}

func (d RestoreDelegate) Height() int {
	if d.showDescription {
		return d.height
	}
	return 1
}

func (d RestoreDelegate) Spacing() int {
	return d.spacing
}

func (d RestoreDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	if d.UpdateFunc == nil {
		return nil
	}
	return d.UpdateFunc(msg, m)
}

func (d RestoreDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var (
		title, desc  string
		matchedRunes []int
		s            = &d.Styles
	)

	file, ok := item.(File)
	if !ok {
		return
	}
	title = file.Title()
	desc = file.Description()

	if m.Width() <= 0 {
		// short-circuit
		return
	}

	// Prevent text from exceeding list width
	textwidth := m.Width() - s.NormalTitle.GetPaddingLeft() - s.NormalTitle.GetPaddingRight()
	title = ansi.Truncate(title, textwidth, ellipsis)
	if d.showDescription {
		var lines []string
		for i, line := range strings.Split(desc, "\n") {
			if i >= d.height-1 {
				break
			}
			lines = append(lines, ansi.Truncate(line, textwidth, ellipsis))
		}
		desc = strings.Join(lines, "\n")
	}

	var (
		onCursor    = index == m.Index()
		emptyFilter = m.FilterState() == list.Filtering && m.FilterValue() == ""
		isFiltered  = m.FilterState() == list.Filtering || m.FilterState() == list.FilterApplied
	)

	if isFiltered {
		// Get indices of matched characters
		matchedRunes = m.MatchesForItem(index)
	}

	if emptyFilter {
		title = s.DimmedTitle.Render(title)
		desc = s.DimmedDesc.Render(desc)
	} else if onCursor && m.FilterState() != list.Filtering {
		if isFiltered {
			// Highlight matches
			unmatched := s.CursorTitle.Inline(true)
			matched := unmatched.Inherit(s.FilterMatch)
			title = lipgloss.StyleRunes(title, matchedRunes, matched, unmatched)
		}
		if file.isSelected() {
			title = s.SelectedCursorTitle.Render(title)
			desc = s.SelectedCursorDesc.Render(desc)
		} else {
			title = s.CursorTitle.Render(title)
			desc = s.CursorDesc.Render(desc)
		}
	} else if file.isSelected() {
		title = s.SelectedTitle.Render(title)
		desc = s.SelectedDesc.Render(desc)
	} else {
		if isFiltered {
			// Highlight matches
			unmatched := s.NormalTitle.Inline(true)
			matched := unmatched.Inherit(s.FilterMatch)
			title = lipgloss.StyleRunes(title, matchedRunes, matched, unmatched)
		}
		title = s.NormalTitle.Render(title)
		desc = s.NormalDesc.Render(desc)
	}

	if d.showDescription {
		fmt.Fprintf(w, "%s\n%s", title, desc)
		return
	}
	fmt.Fprintf(w, "%s", title)
}

func (d RestoreDelegate) ShortHelp() []key.Binding {
	if d.ShortHelpFunc != nil {
		return d.ShortHelpFunc()
	}
	return nil
}

func (d RestoreDelegate) FullHelp() [][]key.Binding {
	if d.FullHelpFunc != nil {
		return d.FullHelpFunc()
	}
	return nil
}
