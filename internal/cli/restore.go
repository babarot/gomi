package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/ui"
)

// Restore handles the restoration of files from trash
func (c *CLI) Restore() error {
	slog.Debug("cli.restore started")
	defer slog.Debug("cli.restore finished")

	// Show UI for file selection; it reads the trash once it has started
	selected, err := render(c.trash, c.loadFiles, ui.RenderOptions{
		Config:        c.config.UI,
		DeleteEnabled: c.config.Core.PermanentDelete.Enable,
	})
	switch {
	case errors.Is(err, errTrashEmpty):
		fmt.Println("The trash is empty. Let's try deleting a file first")
		return nil
	case errors.Is(err, errNothingToShow):
		fmt.Println("Could not find any files to display. The trash may be empty, or all files may be filtered out")
		return nil
	case err != nil:
		return err
	}

	// If no files were selected, exit early
	if len(selected) == 0 {
		return nil
	}

	for _, file := range selected {
		if err := c.restoreFile(file); err != nil {
			return fmt.Errorf("failed to restore file '%s': %w", file.Name, err)
		}
	}

	return nil
}

// render shows the UI; tests replace it
var render = ui.Render

var (
	// errTrashEmpty and errNothingToShow end the UI when there is no file
	// to show; Restore tells which after the UI is gone
	errTrashEmpty    = errors.New("trash is empty")
	errNothingToShow = errors.New("no files to show")
)

// progressLister is a Trash that counts how far its listing has got
type progressLister interface {
	ListWithProgress(p *trash.Progress) ([]*trash.File, error)
}

// loadFiles lists the files the UI shows, counting into p when the trash
// can tell how far it has got
func (c *CLI) loadFiles(p *trash.Progress) ([]*trash.File, error) {
	var files []*trash.File
	var err error
	if l, ok := c.trash.(progressLister); ok {
		files, err = l.ListWithProgress(p)
	} else {
		files, err = c.trash.List()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list trash contents: %w", err)
	}
	if len(files) == 0 {
		return nil, errTrashEmpty
	}

	// Filter files based on configuration
	filtered := c.filterFiles(files)
	if len(filtered) == 0 {
		return nil, errNothingToShow
	}
	return filtered, nil
}

// filterFiles applies configured filters to the list of files
func (c *CLI) filterFiles(files []*trash.File) []*trash.File {
	var filtered []*trash.File

	for _, file := range files {
		// Skip files that don't exist anymore
		if !file.Exists() {
			slog.Debug("skipping non-existent file", "path", file.TrashPath)
			continue
		}

		// TODO: Add filters here based on configuration
		// For example: time-based filters, name patterns, etc.

		filtered = append(filtered, file)
	}

	return filtered
}

// restoreFile handles the restoration of a single file
func (c *CLI) restoreFile(file *trash.File) error {
	originalPath := file.OriginalPath

	// Check if anything is at the original location; Lstat, so that a
	// symlink whose target is gone counts, as the browser's marks count it
	if _, err := os.Lstat(originalPath); err == nil {
		// File exists at original location, ask for new name if necessary
		newName, err := c.prompter.InputFilename(file)
		if err != nil {
			if errors.Is(err, ui.ErrInputCanceled) {
				c.printVerbose("Canceled! No new filename input.\n")
				return nil
			}
			return fmt.Errorf("failed to get new filename: %w", err)
		}
		originalPath = filepath.Join(filepath.Dir(originalPath), newName)
		file.OriginalPath = originalPath
	}

	// If configured, ask for confirmation
	if c.config.Core.Restore.Confirm && !c.prompter.Confirm(fmt.Sprintf("OK to restore? %s", filepath.Base(originalPath))) {
		c.printVerbose("Replied no, canceled!\n")
		return nil
	}

	// Check again if destination exists (might have been created while confirming)
	if _, err := os.Lstat(originalPath); err == nil {
		msg := fmt.Sprintf("Caution! The same name already exists. Even so okay to restore? %s", filepath.Base(originalPath))
		if !c.prompter.Confirm(msg) {
			c.printVerbose("Replied no, canceled!\n")
			return nil
		}
	}

	// Perform the restore
	if err := c.trash.Restore(file, originalPath); err != nil {
		return fmt.Errorf("failed to restore '%s': %w", file.Name, err)
	}

	c.printVerbose("Restored '%s' to %s\n", file.Name, originalPath)
	return nil
}

// printVerbose logs the message if verbose is true
func (c *CLI) printVerbose(msg string, args ...any) {
	if c.config.Core.Restore.Verbose {
		fmt.Printf(msg, args...)
	}
}
