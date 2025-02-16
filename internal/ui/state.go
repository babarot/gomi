package ui

// ViewState represents the current view state of the UI.
type ViewState int

const (
	// ListView shows a list of deleted files.
	ListView ViewState = iota

	// DetailView shows detailed information about a selected file.
	DetailView

	// Quitting indicates the application is about to quit.
	Quitting
)

// String returns a string representation of the ViewState.
func (v ViewState) String() string {
	return [...]string{
		"list",
		"detail",
		"quitting",
	}[v]
}

// DateFormat represents the format used for displaying dates.
type DateFormat string

const (
	// DateFormatRelative shows dates in relative format (e.g., "2 hours ago").
	DateFormatRelative DateFormat = "relative"

	// DateFormatAbsolute shows dates in absolute format (e.g., "2024-02-16 15:04:05").
	DateFormatAbsolute DateFormat = "absolute"
)
