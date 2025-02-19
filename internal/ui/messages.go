package ui

import "github.com/charmbracelet/bubbles/list"

// ShowDetailMsg represents a request to switch to detail view
type ShowDetailMsg struct {
	file File
}

// newShowDetailMsg creates a message to switch to detail view
func newShowDetailMsg(file File) ShowDetailMsg {
	return ShowDetailMsg{file: file}
}

// FileListLoadedMsg indicates the initial file list has been loaded
type FileListLoadedMsg struct {
	files []list.Item
	err   error
}

// FileListUpdatedMsg represents an update to the existing file list
type FileListUpdatedMsg struct {
	files []list.Item
	err   error
}

// errorMsg represents any error that occurred during UI operations
type errorMsg struct {
	err error
}

func (e errorMsg) Error() string { return e.err.Error() }
