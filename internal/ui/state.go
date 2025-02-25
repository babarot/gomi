package ui

import (
	"time"

	"github.com/dustin/go-humanize"
)

// ViewType represents the current view state
type ViewType uint8

const (
	ListView ViewType = iota
	DetailView
	ConfirmView
	Quitting
)

func (v ViewType) String() string {
	switch v {
	case ListView:
		return "list view"
	case DetailView:
		return "detail view"
	case ConfirmView:
		return "confirm view"
	case Quitting:
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
	current      ViewType
	previous     ViewType
	detail       detail
	preview      preview
	confirmation confirmation
}

type detail struct {
	showOrigin bool
	dateFormat DateFormat
}

type preview struct {
	available bool
}
type confirmation struct {
	state    ConfirmState
	files    []File
	yesInput string
}

// ConfirmState represents the confirmation dialog state
type ConfirmState string

const (
	ConfirmStateYesNo   ConfirmState = "yn"
	ConfirmStateTypeYES ConfirmState = "yes"
)

// NewViewState creates a new ViewState with default values
func NewViewState() *ViewState {
	return &ViewState{
		current:  ListView,
		previous: ListView,
		detail: detail{
			showOrigin: true,
			dateFormat: DateFormatRelative,
		},
		preview: preview{
			available: true,
		},
		confirmation: confirmation{
			state: ConfirmStateYesNo,
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
		return
	}
	v.detail.dateFormat = DateFormatRelative
}

// ToggleOriginPath switches between showing origin and trash paths
func (v *ViewState) ToggleOriginPath() {
	v.detail.showOrigin = !v.detail.showOrigin
}

// FormatDate formats the given time according to the current date format
func (v *ViewState) FormatDate(t time.Time) string {
	if v.detail.dateFormat == DateFormatAbsolute {
		return t.Format(time.DateTime)
	}
	return humanize.Time(t)
}

// SetConfirmState sets the confirmation dialog state
func (v *ViewState) SetConfirmState(state ConfirmState, files []File) {
	v.confirmation.state = state
	v.confirmation.files = files
	v.confirmation.yesInput = "" // reset input
}

// UpdateYesInput updates the YES input state
func (v *ViewState) UpdateYesInput(char string) {
	switch len(v.confirmation.yesInput) {
	case 0:
		if char == "Y" {
			v.confirmation.yesInput += char
		}
	case 1:
		if char == "E" {
			v.confirmation.yesInput += char
		}
	case 2:
		if char == "S" {
			v.confirmation.yesInput += char
		}
	}
}

// IsYesComplete checks if the complete "YES" has been entered
func (v *ViewState) IsYesComplete() bool {
	return v.confirmation.yesInput == "YES"
}

// ClearYesInput clears the YES input
func (v *ViewState) ClearYesInput() {
	v.confirmation.yesInput = ""
}

// BackspaceYesInput removes the last character from YES input
func (v *ViewState) BackspaceYesInput() {
	if len(v.confirmation.yesInput) > 0 {
		v.confirmation.yesInput = v.confirmation.yesInput[:len(v.confirmation.yesInput)-1]
	}
}
