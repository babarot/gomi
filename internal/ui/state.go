package ui

// ViewType represents the current view state
type ViewType uint8

const (
	ListView ViewType = iota
	DetailView
	Quitting
	// Loading is while the files are read, before the list is shown
	Loading
)

func (v ViewType) String() string {
	switch v {
	case ListView:
		return "list view"
	case DetailView:
		return "detail view"
	case Quitting:
		return "quit"
	case Loading:
		return "loading"
	}
	return "unknown"
}

type ViewState struct {
	current      ViewType
	confirmation confirmation
}

// confirmation is the delete confirmation dialog, drawn over the current
// view while open
type confirmation struct {
	open     bool
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
		current: ListView,
		confirmation: confirmation{
			state: ConfirmStateYesNo,
		},
	}
}

// SetView changes the current view
func (v *ViewState) SetView(newView ViewType) {
	v.current = newView
}

// OpenConfirm opens the confirmation dialog for files over the current view
func (v *ViewState) OpenConfirm(state ConfirmState, files []File) {
	v.confirmation.open = true
	v.confirmation.state = state
	v.confirmation.files = files
	v.confirmation.yesInput = "" // reset input
}

// CloseConfirm closes the confirmation dialog
func (v *ViewState) CloseConfirm() {
	v.confirmation.open = false
}

// Confirming reports whether the confirmation dialog is open
func (v *ViewState) Confirming() bool {
	return v.confirmation.open
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
