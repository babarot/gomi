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

	// Get list of files in trash
	files, err := c.manager.List()
	if err != nil {
		return fmt.Errorf("failed to list trash contents: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("The trash is empty. Let's try deleting a file first")
		return nil
	}

	// Filter files based on configuration
	filtered := c.filterFiles(files)
	if len(filtered) == 0 {
		fmt.Println("Could not find any files to display. The trash may be empty, or all files may be filtered out")
		return nil
	}

	// Show UI for file selection
	selected, err := ui.RenderList(filtered, c.config.UI)
	if err != nil {
		return fmt.Errorf("failed to show file selection UI: %w", err)
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

	// Check if the file exists at the original location
	if _, err := os.Stat(originalPath); err == nil {
		// File exists at original location, ask for new name if necessary
		newName, err := ui.InputFilename(file)
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
	if c.config.Core.Restore.Confirm && !ui.Confirm(fmt.Sprintf("OK to restore? %s", filepath.Base(originalPath))) {
		c.printVerbose("Replied no, canceled!\n")
		return nil
	}

	// Check again if destination exists (might have been created while confirming)
	if _, err := os.Stat(originalPath); err == nil {
		msg := fmt.Sprintf("Caution! The same name already exists. Even so okay to restore? %s", filepath.Base(originalPath))
		if !ui.Confirm(msg) {
			c.printVerbose("Replied no, canceled!\n")
			return nil
		}
	}

	// Perform the restore
	if err := c.manager.Restore(file, originalPath); err != nil {
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
