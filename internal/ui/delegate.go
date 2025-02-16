package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/ui/keys"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	ellipsis = "…"
)

// ListDelegate manages the rendering and behavior of list items.
type ListDelegate struct {
	showDescription bool
	height          int
	spacing         int
	styles          *DelegateStyles
	parentModel     *Model
}

// DelegateStyles holds all the styles used for list item rendering.
type DelegateStyles struct {
	NormalTitle         lipgloss.Style
	NormalDesc          lipgloss.Style
	SelectedTitle       lipgloss.Style
	SelectedDesc        lipgloss.Style
	DimmedTitle         lipgloss.Style
	DimmedDesc          lipgloss.Style
	CursorTitle         lipgloss.Style
	CursorDesc          lipgloss.Style
	SelectedCursorTitle lipgloss.Style
	SelectedCursorDesc  lipgloss.Style
	FilterMatch         lipgloss.Style
}

// NewListDelegate creates a new delegate with the provided configuration.
func NewListDelegate(cfg config.UI) *ListDelegate {
	var height = 2
	var spacing = 1

	// Adjust layout based on density configuration
	showDescription := true
	switch cfg.Density {
	case "compact":
		showDescription = false
		height = 1
		spacing = 0
	case "spacious":
		showDescription = true
	}

	// Initialize styles
	styles := &DelegateStyles{
		NormalTitle: lipgloss.NewStyle().
			Padding(0, 0, 0, 2),

		NormalDesc: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}).
			Padding(0, 0, 0, 2),

		SelectedTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.Style.ListView.Selected)).
			Padding(0, 0, 0, 2),

		SelectedDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.Style.ListView.Selected)).
			Padding(0, 0, 0, 2),

		DimmedTitle: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}).
			Padding(0, 0, 0, 2),

		DimmedDesc: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#C2B8C2", Dark: "#4D4D4D"}).
			Padding(0, 0, 0, 2),

		CursorTitle: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color(cfg.Style.ListView.Cursor)).
			Foreground(lipgloss.Color(cfg.Style.ListView.Cursor)).
			Padding(0, 0, 0, 1),

		CursorDesc: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color(cfg.Style.ListView.Cursor)).
			Foreground(lipgloss.Color(cfg.Style.ListView.Cursor)).
			Padding(0, 0, 0, 1),

		SelectedCursorTitle: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color(cfg.Style.ListView.Cursor)).
			Foreground(lipgloss.Color(cfg.Style.ListView.Selected)).
			Padding(0, 0, 0, 1),

		SelectedCursorDesc: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color(cfg.Style.ListView.Cursor)).
			Foreground(lipgloss.Color(cfg.Style.ListView.Selected)).
			Padding(0, 0, 0, 1),

		FilterMatch: lipgloss.NewStyle().
			Underline(true),
	}

	return &ListDelegate{
		showDescription: showDescription,
		height:          height,
		spacing:         spacing,
		styles:          styles,
	}
}

// Height returns the height of the delegate.
func (d *ListDelegate) Height() int {
	return d.height
}

// Spacing returns the spacing of the delegate.
func (d *ListDelegate) Spacing() int {
	return d.spacing
}

// Update handles any updates for the delegate.
func (d *ListDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// ItemDelegate is an interface that extends list.ItemDelegate with selection state.
type ItemDelegate interface {
	list.ItemDelegate
	IsSelected(item *Item) bool
}

// Render renders a list item.
func (d *ListDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(*Item)
	if !ok {
		return
	}

	var (
		title, desc  string
		matchedRunes []int
		styles       = d.styles
	)

	// Get item details
	title = item.Title()
	desc = item.Description()

	if m.Width() <= 0 {
		return
	}

	// Handle text truncation
	textWidth := m.Width() - styles.NormalTitle.GetPaddingLeft() - styles.NormalTitle.GetPaddingRight()
	title = ansi.Truncate(title, textWidth, ellipsis)
	if d.showDescription {
		var lines []string
		for i, line := range strings.Split(desc, "\n") {
			if i >= d.height-1 {
				break
			}
			lines = append(lines, ansi.Truncate(line, textWidth, ellipsis))
		}
		desc = strings.Join(lines, "\n")
	}

	// Determine render state
	var (
		isSelected  = false
		onCursor    = index == m.Index()
		emptyFilter = m.FilterState() == list.Filtering && m.FilterValue() == ""
		isFiltered  = m.FilterState() == list.Filtering || m.FilterState() == list.FilterApplied
	)

	// Check if the delegate implements ItemDelegate
	// Get selection state directly from parent model via delegate
	if d.parentModel != nil {
		isSelected = d.parentModel.isSelected(item)
	}

	if isFiltered {
		matchedRunes = m.MatchesForItem(index)
	}

	// Apply appropriate styles based on item state
	switch {
	case emptyFilter:
		title = styles.DimmedTitle.Render(title)
		desc = styles.DimmedDesc.Render(desc)

	case onCursor:
		if isSelected {
			title = styles.SelectedCursorTitle.Render(title)
			desc = styles.SelectedCursorDesc.Render(desc)
		} else {
			title = styles.CursorTitle.Render(title)
			desc = styles.CursorDesc.Render(desc)
		}

	case isSelected:
		title = styles.SelectedTitle.Render(title)
		desc = styles.SelectedDesc.Render(desc)

	default:
		if isFiltered {
			unmatched := styles.NormalTitle.Inline(true)
			matched := unmatched.Copy().Inherit(styles.FilterMatch)
			title = lipgloss.StyleRunes(title, matchedRunes, matched, unmatched)
		}
		title = styles.NormalTitle.Render(title)
		desc = styles.NormalDesc.Render(desc)
	}

	// Write the final output
	if d.showDescription {
		fmt.Fprintf(w, "%s\n%s", title, desc)
		return
	}
	fmt.Fprintf(w, "%s", title)
}

// ShortHelp returns keybindings to be shown in the mini help view.
func (d *ListDelegate) ShortHelp() []key.Binding {
	return []key.Binding{
		keys.ListKeys.Enter,
		keys.ListKeys.Space,
		keys.ListKeys.Select,
	}
}

// FullHelp returns keybindings for the expanded help view.
func (d *ListDelegate) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			keys.ListKeys.Enter,
			keys.ListKeys.Space,
			keys.ListKeys.Select,
			keys.ListKeys.DeSelect,
			keys.ListKeys.Quit,
		},
	}
}

// IsSelected implements ItemDelegate interface.
func (d *ListDelegate) IsSelected(item *Item) bool {
	if d.parentModel == nil {
		return false
	}
	return d.parentModel.isSelected(item)
}
