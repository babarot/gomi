package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/babarot/gomi/internal/core/atomic"
	"github.com/babarot/gomi/internal/core/types"
)

func (c *CLI) Put(args []string) error {
	slog.Debug("cli.put started")
	defer slog.Debug("cli.put finished")

	if len(args) == 0 {
		return errors.New("too few arguments")
	}

	// Process each file or directory atomically
	for _, arg := range args {
		if err := c.putPath(arg); err != nil {
			return fmt.Errorf("failed to process %s: %w", arg, err)
		}
	}

	return nil
}

func (c *CLI) putPath(path string) error {
	// 1. File existence check and stats
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if !c.option.Rm.Force {
				return fmt.Errorf("%s: no such file or directory", path)
			}
			return nil
		}
		return err
	}

	// 2. Validate path
	if err := c.validatePath(path); err != nil {
		return err
	}

	// 3. Prepare file info
	file, err := c.history.FileInfo(c.runID, path)
	if err != nil {
		return fmt.Errorf("prepare file info: %w", err)
	}
	file.IsDir = info.IsDir()

	// 4. Prepare metadata transaction
	tx, err := c.history.PrepareMove(types.TrashFile{
		Name:      file.Name,
		ID:        file.ID,
		RunID:     file.RunID,
		From:      file.From,
		To:        file.To,
		Timestamp: file.Timestamp,
		IsDir:     file.IsDir,
	})
	if err != nil {
		return fmt.Errorf("prepare move: %w", err)
	}

	// 5. Move file or directory atomically
	if err := atomic.Move(file.From, file.To, atomic.MoveOptions{
		AllowCrossDev: true,
		Force:         c.option.Rm.Force,
	}); err != nil {
		// Rollback metadata if move fails
		c.history.RollbackMove(tx)
		return fmt.Errorf("move %s: %w", path, err)
	}

	// time.Sleep(5 * time.Minute)

	// 6. Commit metadata transaction
	if err := c.history.CommitMove(tx); err != nil {
		slog.Error("failed to commit metadata",
			"file", file.Name,
			"error", err,
		)
		return fmt.Errorf("commit metadata: %w", err)
	}

	// 7. Log success if verbose
	if c.option.Rm.Verbose {
		if info.IsDir() {
			fmt.Printf("removed directory '%s'\n", path)
		} else {
			fmt.Printf("removed '%s'\n", path)
		}
	}

	return nil
}

// validatePath checks if path is valid for trashing
func (c *CLI) validatePath(path string) error {
	// Common paths that should not be trashed
	protected := []string{
		"/",
		"/home",
		"/usr",
		"/etc",
		"/var",
		"/tmp",
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	for _, p := range protected {
		if absPath == p {
			return fmt.Errorf("cannot trash protected path: %s", path)
		}
	}

	return nil
}
