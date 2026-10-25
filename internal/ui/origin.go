package ui

import (
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// origin is what restoring a file would meet at the place it was deleted
// from
type origin int

const (
	// originBack: the file goes back where it was
	originBack origin = iota + 1
	// originTaken: something has the file's name there now, so gomi asks
	// for a new name
	originTaken
	// originGone: the directory it was in is gone, so gomi makes it again
	originGone
)

// originsCheckedMsg carries what restoring each file would meet, by trash
// path
type originsCheckedMsg struct {
	origins map[string]origin
}

// checkOriginsCmd looks at the place each of files was deleted from. It
// runs in the background since the places may be many or on a slow mount.
func checkOriginsCmd(files []File) tea.Cmd {
	if len(files) == 0 {
		return nil
	}
	return func() tea.Msg {
		origins := make(map[string]origin, len(files))
		for _, f := range files {
			origins[f.TrashPath] = checkOrigin(f.OriginalPath)
		}
		return originsCheckedMsg{origins: origins}
	}
}

// checkOrigin tells what restoring to path would meet, as the restore does
// it: a file there is asked about, a missing directory is made. Tests
// replace it, since the places they show are not on the machine.
var checkOrigin = func(path string) origin {
	if _, err := os.Lstat(path); err == nil {
		return originTaken
	}
	if fi, err := os.Stat(filepath.Dir(path)); err != nil || !fi.IsDir() {
		return originGone
	}
	return originBack
}

// uncheckedOrigins are the files whose place has not been looked at
func (m Model) uncheckedOrigins(files []File) []File {
	var out []File
	for _, f := range files {
		if _, ok := m.origins[f.TrashPath]; !ok {
			out = append(out, f)
		}
	}
	return out
}

// originMark is the mark at the end of a file's row: ! when something has
// its name now, + when its directory is gone, else a space
func (m Model) originMark(f File, on func(lipgloss.Style) lipgloss.Style) string {
	switch m.origins[f.TrashPath] {
	case originTaken:
		return on(m.fg(m.palette.Dialog)).Bold(true).Render("!")
	case originGone:
		return on(m.fg(m.palette.Description)).Render("+")
	}
	return on(lipgloss.NewStyle()).Render(" ")
}

// hasMarks reports whether any of files has a mark
func (m Model) hasMarks(files []File) bool {
	for _, f := range files {
		if o := m.origins[f.TrashPath]; o == originTaken || o == originGone {
			return true
		}
	}
	return false
}
