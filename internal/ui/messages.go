package ui

// ShowDetailMsg represents a request to switch to detail view
type ShowDetailMsg struct {
	file File
}

// newShowDetailMsg creates a message to switch to detail view
func newShowDetailMsg(file File) ShowDetailMsg {
	return ShowDetailMsg{file: file}
}

// FileListLoadedMsg carries the file list, loaded at start or reloaded
// after files are deleted
type FileListLoadedMsg struct {
	files []File
	// deleted is how many files were deleted before the list was reloaded
	deleted int
	err     error
}

// errorMsg represents any error that occurred during UI operations
type errorMsg struct {
	err error
}

func (e errorMsg) Error() string { return e.err.Error() }
