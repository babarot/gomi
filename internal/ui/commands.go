package ui

import (
	"errors"
	"log/slog"
	"os"
	"sort"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
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
			return files[i].File.DeletedAt.After(files[j].File.DeletedAt)
		})

		// Filter out files that no longer exist
		files = lo.Reject(files, func(f File, index int) bool {
			_, err := os.Stat(f.File.TrashPath)
			return os.IsNotExist(err)
		})

		// Convert to list items
		items := make([]list.Item, len(files))
		for i, file := range files {
			items[i] = file
		}

		return FileListLoadedMsg{files: items}
	}
}

// deletePermanentlyCmd creates a command to permanently delete files
func deletePermanentlyCmd(m *Model, files ...File) tea.Cmd {
	return func() tea.Msg {
		for _, file := range files {
			slog.Debug("permanently delete", "file", file.TrashPath)
			if err := m.trashManager.Remove(file.File); err != nil {
				return FileListUpdatedMsg{err: err}
			}
		}

		// Refresh file list after deletion
		origin := m.files
		origin = lo.Reject(origin, func(f File, index int) bool {
			_, err := os.Stat(f.File.TrashPath)
			return os.IsNotExist(err)
		})

		items := make([]list.Item, len(origin))
		for i, file := range origin {
			items[i] = file
		}

		return FileListUpdatedMsg{files: items}
	}
}
