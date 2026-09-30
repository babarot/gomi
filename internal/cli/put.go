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

	"github.com/babarot/gomi/internal/utils/fs"
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

	// Get absolute path. The forbidden paths check needs it: a relative
	// path such as "hosts" run in /etc must be checked as /etc/hosts.
	path, err := filepath.Abs(expandedPath)
	if err != nil {
		failed.Append(arg)
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check for forbidden paths
	if c.isForbiddenPath(path) {
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

	// Move to trash. Like rm -f, -f ignores only nonexistent files: a file
	// that exists but could not be moved is still an error.
	if err := c.trash.Put(path); err != nil {
		failed.Append(arg)
		return fmt.Errorf("failed to move to trash: %w", err)
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

// isForbiddenPath checks if the given absolute path is in the forbidden paths list
func (c *CLI) isForbiddenPath(path string) bool {
	return isForbidden(path, c.config.Core.Trash.ForbiddenPaths, os.TempDir())
}

// isForbidden reports whether path is one of forbiddenPaths or inside one.
//
// Paths are compared both as given and with symlinks resolved, so that on
// macOS, where /var, /etc and /tmp are symlinks into /private, a forbidden
// "/var" also covers "/private/var" and the other way around. Only the parent
// of path is resolved: when path itself is a symlink, the link is what gets
// removed, not what it points to.
//
// The contents of tempDir ($TMPDIR) are allowed even when a forbidden path
// contains tempDir. The per-user temporary directory on macOS lives under
// /var/folders, and forbidding "/var" is meant to protect the system, not
// the files that mktemp creates. tempDir itself stays forbidden, and so do
// forbidden paths inside it.
func isForbidden(path string, forbiddenPaths []string, tempDir string) bool {
	targets := targetForms(path)
	var temps []string
	if tempDir != "" {
		temps = dirForms(tempDir)
	}

	for _, forbiddenPath := range forbiddenPaths {
		// Expand forbidden path with environment variables
		forbidden := dirForms(os.ExpandEnv(forbiddenPath))

		if !anyWithin(targets, forbidden, true) {
			continue
		}
		if anyWithin(temps, forbidden, false) && anyWithin(targets, temps, false) {
			// Forbidden only because it is an ancestor of tempDir
			continue
		}
		return true
	}
	return false
}

// targetForms returns path cleaned, and path with its parent directory's
// symlinks resolved when that differs.
func targetForms(path string) []string {
	path = filepath.Clean(path)
	forms := []string{path}
	if dir, err := filepath.EvalSymlinks(filepath.Dir(path)); err == nil {
		if resolved := filepath.Join(dir, filepath.Base(path)); resolved != path {
			forms = append(forms, resolved)
		}
	}
	return forms
}

// dirForms returns dir cleaned, and dir with its symlinks resolved when that
// differs.
func dirForms(dir string) []string {
	dir = filepath.Clean(dir)
	forms := []string{dir}
	if resolved, err := filepath.EvalSymlinks(dir); err == nil && resolved != dir {
		forms = append(forms, resolved)
	}
	return forms
}

// anyWithin reports whether any of paths is inside any of dirs. With
// inclusive, a path equal to a dir counts too. Nothing is inside "/": it
// only matches exactly, as forbidding "/" has always meant.
func anyWithin(paths, dirs []string, inclusive bool) bool {
	for _, p := range paths {
		for _, d := range dirs {
			if inclusive && p == d {
				return true
			}
			if strings.HasPrefix(p, d+string(filepath.Separator)) {
				return true
			}
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
