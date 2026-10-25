package ui

import (
	"log/slog"

	tea "charm.land/bubbletea/v2"
	"github.com/dustin/go-humanize"
	"github.com/gabriel-vasile/mimetype"

	"github.com/babarot/gomi/internal/utils/fs"
)

// previewResult is a file's preview, read in the background
type previewResult struct {
	content string
	// available is false when the file cannot be previewed; mimeType then
	// says what it is
	available bool
	mimeType  string
}

// previewLoadedMsg carries the preview of the file at path
type previewLoadedMsg struct {
	path string
	previewResult
}

// sizeLoadedMsg carries the total size of the contents at path
type sizeLoadedMsg struct {
	path string
	size int64
	err  error
}

// loadSlots bounds how many previews and sizes are read at once, since
// each may walk a large directory or run a command
var loadSlots = make(chan struct{}, 4)

// slot waits for a free slot and returns its release
func slot() func() {
	loadSlots <- struct{}{}
	return func() { <-loadSlots }
}

// loadPreviewCmd reads the preview of f, which may run a command, decode
// an image or highlight a whole file
func loadPreviewCmd(f File) tea.Cmd {
	return func() tea.Msg {
		defer slot()()
		content, err := f.Browse()
		r := previewResult{content: content, available: err == nil}
		if err != nil {
			slog.Warn("file.Browse returned error", "content", err)
			if mtype, _ := mimetype.DetectFile(f.TrashPath); mtype != nil {
				r.mimeType = mtype.String()
			}
		}
		return previewLoadedMsg{path: f.TrashPath, previewResult: r}
	}
}

// loadSizeCmd walks f for the total size of its contents
func loadSizeCmd(f File) tea.Cmd {
	return func() tea.Msg {
		defer slot()()
		size, err := fs.DirSize(f.TrashPath)
		return sizeLoadedMsg{path: f.TrashPath, size: size, err: err}
	}
}

// loadDetail returns the commands that read what the detail view of f
// shows and has been neither read nor started; moving back and forth
// between files does not start the same read twice
func (m Model) loadDetail(f File) []tea.Cmd {
	var cmds []tea.Cmd
	path := f.TrashPath
	if _, ok := m.previews[path]; !ok && !m.reading[previewKey(path)] {
		m.reading[previewKey(path)] = true
		cmds = append(cmds, loadPreviewCmd(f))
	}
	if _, ok := m.sizes[path]; !ok && !f.SizeKnown && !m.reading[sizeKey(path)] {
		m.reading[sizeKey(path)] = true
		cmds = append(cmds, loadSizeCmd(f))
	}
	return cmds
}

// previewKey and sizeKey name the reads in Model.reading
func previewKey(path string) string { return "preview:" + path }
func sizeKey(path string) string    { return "size:" + path }

// sizeText is the size of f as the detail view shows it
func (m Model) sizeText(f File) string {
	size, ok := m.sizes[f.TrashPath]
	switch {
	case f.SizeKnown:
		return humanize.Bytes(uint64(f.Size))
	case !ok:
		return ellipsis
	case size < 0:
		return "(cannot be calculated)"
	}
	return humanize.Bytes(uint64(size))
}
