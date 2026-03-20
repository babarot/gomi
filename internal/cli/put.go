package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/babarot/gomi/internal/utils/fs"
	"golang.org/x/sync/errgroup"
)

// Put moves files to trash
func (c *CLI) Put(args []string) error {
	slog.Debug("cli.put started")
	defer slog.Debug("cli.put finished")

	if len(args) == 0 {
		return errors.New("too few arguments")
	}

	// Use a thread-safe slice to track failed files
	var (
		eg     errgroup.Group
		failed = &syncStringSlice{}
	)

	for _, arg := range args {
		arg := arg // Create new instance of arg for goroutine
		eg.Go(func() error {
			return c.processFile(arg, failed)
		})
	}

	// Wait for all goroutines to complete
	if err := eg.Wait(); err != nil {
		return err
	}

	if failedFiles := failed.Get(); len(failedFiles) > 0 {
		return fmt.Errorf("failed to process files %v", failedFiles)
	}

	return nil
}

// processFile handles the logic for moving a single file to trash
func (c *CLI) processFile(arg string, failed *syncStringSlice) error {
	// Expand path (replace environment variables)
	expandedPath, err := expandPath(arg)
	if err != nil {
		failed.Append(arg)
		return fmt.Errorf("failed to expand path: %w", err)
	}

	// Check for forbidden paths
	if c.isForbiddenPath(expandedPath) {
		failed.Append(arg)
		return fmt.Errorf("refusing to remove forbidden path: %q", arg)
	}

	// Check path safety
	unsafe, err := fs.IsUnsafePath(expandedPath)
	if err != nil {
		failed.Append(arg)
		return fmt.Errorf("failed to check path safety: %w", err)
	}
	if unsafe {
		failed.Append(arg)
		return fmt.Errorf("refusing to remove unsafe path: %q", arg)
	}

	// Get absolute path
	path, err := filepath.Abs(expandedPath)
	if err != nil {
		failed.Append(arg)
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if file exists (use Lstat to handle broken symlinks)
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		if !c.option.Rm.Force {
			failed.Append(arg)
			return fmt.Errorf("%s: no such file or directory", arg)
		}
		if c.option.Rm.Verbose {
			fmt.Fprintf(os.Stderr, "skipping %s: no such file or directory\n", arg)
		}
		return nil
	}

	// Move to trash
	err = c.manager.Put(path)
	if err != nil {
		if !c.option.Rm.Force {
			failed.Append(arg)
			return fmt.Errorf("failed to move to trash: %w", err)
		}
		if c.option.Rm.Verbose {
			fmt.Fprintf(os.Stderr, "failed to move %s to trash: %v\n", arg, err)
		}
		return nil
	}

	if c.option.Rm.Verbose {
		fmt.Printf("moved to trash: %s\n", path)
	}

	return nil
}

// expandPath resolves a file path to its clean form.
// It does NOT expand environment variables because file arguments
// should be treated literally — a file named "$foo" must not be
// interpreted as an environment variable reference.
// Shell-level expansion (e.g. ~ or $HOME) is the shell's job, not ours.
func expandPath(path string) (string, error) {
	return filepath.Clean(path), nil
}

// isForbiddenPath checks if the given path is in the forbidden paths list
func (c *CLI) isForbiddenPath(path string) bool {
	path = filepath.Clean(path)

	for _, forbiddenPath := range c.config.Core.Trash.ForbiddenPaths {
		// Expand forbidden path with environment variables
		expandedForbiddenPath := os.ExpandEnv(forbiddenPath)
		expandedForbiddenPath = filepath.Clean(expandedForbiddenPath)

		// Check for exact match or sub-path
		if path == expandedForbiddenPath || strings.HasPrefix(path, expandedForbiddenPath+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// syncStringSlice is a thread-safe slice for storing strings
type syncStringSlice struct {
	mu    sync.Mutex
	items []string
}

// Append adds an item to the slice in a thread-safe manner
func (s *syncStringSlice) Append(item string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, item)
}

// Get returns a copy of the slice
func (s *syncStringSlice) Get() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.items...)
}
