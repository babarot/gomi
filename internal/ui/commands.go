package ui

import (
	"errors"
	"log/slog"
	"os"
	"sort"

	tea "charm.land/bubbletea/v2"
	"github.com/samber/lo"
)

// loadFileListCmd creates a command to load the initial file list
func loadFileListCmd(files []File) tea.Cmd {
	return func() tea.Msg {
		slog.Info("loading file list", "len(files)", len(files))

		if len(files) == 0 {
			return errorMsg{errors.New("no deleted files found")}
		}

		// Sort files by deletion time, newest first
		sort.Slice(files, func(i, j int) bool {
			return files[i].DeletedAt.After(files[j].DeletedAt)
		})

		// existence already checked by cli.filterFiles
		return FileListLoadedMsg{files: files}
	}
}

// deletePermanentlyCmd creates a command to permanently delete files
func deletePermanentlyCmd(m *Model, files ...File) tea.Cmd {
	return func() tea.Msg {
		for _, file := range files {
			slog.Debug("permanently delete", "file", file.TrashPath)
			if err := m.trash.Remove(file.File); err != nil {
				return FileListLoadedMsg{err: err}
			}
		}

		// Refresh file list after deletion
		origin := m.files
		origin = lo.Reject(origin, func(f File, index int) bool {
			_, err := os.Stat(f.TrashPath)
			return os.IsNotExist(err)
		})

		return FileListLoadedMsg{files: origin, deleted: len(files)}
	}
}
