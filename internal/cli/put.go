package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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

			// Check if trying to remove . or ..
			base := filepath.Base(path)
			if base == "." || base == ".." {
				mu.Lock()
				failed = append(failed, arg)
				mu.Unlock()
				return fmt.Errorf("refusing to remove '.' or '..' directory: skipping %q", arg)
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
		if len(failed) > 0 {
			return fmt.Errorf("failed to process files %v: %w", failed, err)
		}
		return err
	}

	return nil
}
