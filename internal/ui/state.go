package ui

import (
	"time"

	"github.com/dustin/go-humanize"
)

// ViewType represents the current view state
type ViewType uint8

const (
	LIST_VIEW ViewType = iota
	DETAIL_VIEW
	CONFIRM_VIEW
	QUITTING
)

func (v ViewType) String() string {
	switch v {
	case LIST_VIEW:
		return "list view"
	case DETAIL_VIEW:
		return "detail view"
	case CONFIRM_VIEW:
		return "confirm view"
	case QUITTING:
		return "quit"
	}
	return "unknown"
}

// DateFormat represents the date display format
type DateFormat string

const (
	DateFormatRelative DateFormat = "relative"
	DateFormatAbsolute DateFormat = "absolute"
)

type ViewState struct {
	current  ViewType
	previous ViewType
	detail   detail
	preview  preview
}

type detail struct {
	showOrigin bool
	dateFormat DateFormat
}

type preview struct {
	available bool
}

// NewViewState creates a new ViewState with default values
func NewViewState() *ViewState {
	return &ViewState{
		current:  LIST_VIEW,
		previous: LIST_VIEW,
		detail: detail{
			showOrigin: true,
			dateFormat: DateFormatRelative,
		},
		preview: preview{
			available: true,
		},
	}
}

// SetView changes the current view and updates the previous view
func (v *ViewState) SetView(newView ViewType) {
	v.previous = v.current
	v.current = newView
}

// ToggleDateFormat switches between relative and absolute date formats
func (v *ViewState) ToggleDateFormat() {
	if v.detail.dateFormat == DateFormatRelative {
		v.detail.dateFormat = DateFormatAbsolute
	} else {
		v.detail.dateFormat = DateFormatRelative
	}
}

// ToggleOriginPath switches between showing origin and trash paths
func (v *ViewState) ToggleOriginPath() {
	v.detail.showOrigin = !v.detail.showOrigin
}

// FormatDate formats the given time according to the current date format
func (v *ViewState) FormatDate(t time.Time) string {
	switch v.detail.dateFormat {
	case DateFormatAbsolute:
		return t.Format(time.DateTime)
	default:
		return humanize.Time(t)
	}
}
