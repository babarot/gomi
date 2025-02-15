package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Put moves files to trash
func (c *CLI) Put(args []string) error {
	slog.Debug("cli.put started")
	defer slog.Debug("cli.put finished")

	if len(args) == 0 {
		return errors.New("too few arguments")
	}

	// Process each file concurrently
	var (
		eg     errgroup.Group
		mu     sync.Mutex // Mutex to synchronize storage operations
		failed []string
	)

	for _, arg := range args {
		arg := arg // Create new instance of arg for goroutine
		eg.Go(func() error {
			unsafe, err := isUnsafePath(arg)
			if err != nil {
				mu.Lock()
				failed = append(failed, arg)
				mu.Unlock()
				return fmt.Errorf("failed to check path safety: %w", err)
			}
			if unsafe {
				mu.Lock()
				failed = append(failed, arg)
				mu.Unlock()
				return fmt.Errorf("refusing to remove unsafe path: %q", arg)
			}

			// Get absolute path
			path, err := filepath.Abs(arg)
			if err != nil {
				mu.Lock()
				failed = append(failed, arg)
				mu.Unlock()
				return fmt.Errorf("failed to get absolute path: %w", err)
			}

			// Check if file exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				if !c.option.Rm.Force {
					mu.Lock()
					failed = append(failed, arg)
					mu.Unlock()
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
					mu.Lock()
					failed = append(failed, arg)
					mu.Unlock()
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
		})
	}

	// Wait for all goroutines to complete
	if err := eg.Wait(); err != nil {
		return err
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed to process files %v", failed)
	}

	return nil
}

// isUnsafePath checks if the given path is unsafe to remove
func isUnsafePath(path string) (bool, error) {
	// First check the original path before any normalization
	// This preserves the original input like "." or ".."
	originalBase := filepath.Base(path)
	if originalBase == "." || originalBase == ".." {
		return true, nil
	}

	// Get absolute path
	abs, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Clean the path to check for normalized root paths
	cleaned := filepath.Clean(abs)

	// Check root path
	if cleaned == "/" {
		return true, nil
	}

	// Check double slashes and similar patterns
	if strings.HasPrefix(path, "//") {
		return true, nil
	}

	slog.Debug("path safety check",
		"original", path,
		"originalBase", originalBase,
		"absolute", abs,
		"cleaned", cleaned)

	return false, nil
}
