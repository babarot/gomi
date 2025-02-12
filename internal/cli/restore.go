package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/babarot/gomi/internal/core/atomic"
	"github.com/babarot/gomi/internal/core/types"
	"github.com/babarot/gomi/internal/ui"
)

func (c *CLI) Restore() error {
	slog.Debug("cli.restore started")
	defer slog.Debug("cli.restore finished")

	// Check if history exists
	files := c.history.List()
	if len(files) == 0 {
		return errors.New("no files in trash")
	}

	// Get filtered files
	filtered := c.history.Filter()
	if len(filtered) == 0 {
		return errors.New("no files match the filter criteria")
	}

	// Let user select files
	selectedFiles, err := ui.RenderList(filtered, c.config.UI)
	if err != nil {
		return fmt.Errorf("file selection: %w", err)
	}

	// Process each selected file
	var (
		restored []types.TrashFile
		errs     []error
	)

	for _, file := range selectedFiles {
		if err := c.restoreFile(file); err != nil {
			errs = append(errs, fmt.Errorf("restore %s: %w", file.Name, err))
			continue
		}
		restored = append(restored, file)
	}

	// Report errors if any
	if len(errs) > 0 {
		return formatErrors(errs)
	}

	return nil
}

func (c *CLI) restoreFile(file types.TrashFile) error {
	// 1. Check destination path
	if err := c.checkRestorePath(file); err != nil {
		return err
	}

	// 2. Prepare metadata transaction
	tx, err := c.history.PrepareRestore(file)
	if err != nil {
		return fmt.Errorf("prepare restore: %w", err)
	}

	// 3. Move file atomically
	if err := atomic.Move(file.To, file.From, atomic.MoveOptions{
		AllowCrossDev: true,
		Force:         true,
	}); err != nil {
		// Rollback metadata if move fails
		c.history.RollbackRestore(tx)
		return fmt.Errorf("move file: %w", err)
	}

	// 4. Commit metadata transaction
	if err := c.history.CommitRestore(tx); err != nil {
		slog.Error("failed to commit restore metadata",
			"file", file.Name,
			"error", err,
		)
		return fmt.Errorf("commit metadata: %w", err)
	}

	// 5. Log success if verbose
	if c.config.Core.Restore.Verbose {
		fmt.Printf("restored '%s' to %s\n", file.Name, file.From)
	}

	return nil
}

func (c *CLI) checkRestorePath(file types.TrashFile) error {
	// Check if destination exists
	if _, err := os.Stat(file.From); err == nil {
		// If destination exists, prompt for new name
		newName, err := ui.InputFilename(file)
		if err != nil {
			if errors.Is(err, ui.ErrInputCanceled) {
				return fmt.Errorf("restore canceled by user")
			}
			return err
		}

		// Update destination path
		file.From = filepath.Join(filepath.Dir(file.From), newName)
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(file.From)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	return nil
}

func formatErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	msg := fmt.Sprintf("%d errors occurred:\n", len(errs))
	for _, err := range errs {
		msg += fmt.Sprintf("  * %v\n", err)
	}
	return errors.New(msg)
}
