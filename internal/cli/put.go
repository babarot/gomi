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

// List of forbidden paths that cannot be moved to trash
var forbiddenPaths = []string{
	// Default trash-related paths
	"$HOME/.local/share/Trash",
	"$HOME/.trash",
	"$XDG_DATA_HOME/Trash",
	"/tmp/Trash",
	"/var/tmp/Trash",

	// gomi dir
	"$HOME/.gomi",

	// Critical system directories
	"/",
	"/etc",
	"/usr",
	"/var",
	"/bin",
	"/sbin",
	"/lib",
	"/lib64",
}

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
	if isForbiddenPath(expandedPath) {
		failed.Append(arg)
		return fmt.Errorf("refusing to remove forbidden path: %q", arg)
	}

	// Check path safety
	unsafe, err := isUnsafePath(expandedPath)
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

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
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

// expandPath expands environment variables in the path
func expandPath(path string) (string, error) {
	// Expand environment variables
	if strings.HasPrefix(path, "$") {
		// Expand variables like $HOME
		parts := strings.SplitN(path, "/", 2)
		envVar := parts[0]

		// Get environment variable value
		expandedVar := os.ExpandEnv(envVar)
		if expandedVar == "" {
			return "", fmt.Errorf("environment variable %s not set", envVar)
		}

		// Add path part after environment variable
		if len(parts) > 1 {
			return filepath.Join(expandedVar, parts[1]), nil
		}
		return expandedVar, nil
	}

	// Expand environment variables in regular paths
	return os.ExpandEnv(path), nil
}

// isForbiddenPath checks if the given path is in the forbidden paths list
func isForbiddenPath(path string) bool {
	path = filepath.Clean(path)

	for _, forbiddenPath := range forbiddenPaths {
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

// isUnsafePath checks if the given path is unsafe to remove
func isUnsafePath(path string) (bool, error) {
	// First check the original path before any normalization
	// This preserves the original input like "." or ".."
	originalBase := filepath.Base(path)
	if originalBase == "." || originalBase == ".." {
		return true, nil
	}

	// Clean the path to check for normalized root paths
	cleaned := filepath.Clean(path)

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
		"cleaned", cleaned)

	return false, nil
}
