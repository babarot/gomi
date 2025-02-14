package fs

import (
	"fmt"
	"os"
	"path/filepath"

	cp "github.com/otiai10/copy"
)

// CreateExclusive creates a new file with O_EXCL flag to ensure atomic creation.
// Returns error if the file already exists.
func CreateExclusive(path string, perm os.FileMode) (*os.File, error) {
	// O_EXCL ensures the file doesn't exist and creates it atomically
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
}

// MoveFile moves a file or directory from src to dst.
// If the move fails due to being on different devices and fallbackCopy is true,
// it will fall back to copy and delete.
func MoveFile(src, dst string, fallbackCopy bool) error {
	// Ensure the destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Try rename(2) first
	if err := os.Rename(src, dst); err != nil {
		if !fallbackCopy {
			return fmt.Errorf("failed to move file: %w", err)
		}

		// Fallback to copy and delete
		if err := cp.Copy(src, dst); err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}

		// If copy succeeds, remove the original
		if err := os.RemoveAll(src); err != nil {
			// If we can't remove the source, try to remove the copy
			_ = os.RemoveAll(dst)
			return fmt.Errorf("failed to remove source after copy: %w", err)
		}
	}

	return nil
}

// CreateWithBackup creates a new file while preserving the old one as a backup.
// The backup will have the same name as the original with ".backup" appended.
// Returns:
// - temporary file for writing
// - cleanup function to remove temporary file
// - commit function to save changes and create backup
// - error if any
func CreateWithBackup(path string) (*os.File, func(), func() error, error) {
	// Create temporary file
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create temporary file: %w", err)
	}

	// Cleanup removes the temporary file
	cleanup := func() {
		name := temp.Name()
		_ = temp.Close()
		_ = os.Remove(name)
	}

	// Commit saves changes and creates backup
	commit := func() error {
		name := temp.Name()
		if err := temp.Close(); err != nil {
			return fmt.Errorf("failed to close temporary file: %w", err)
		}

		// If original file exists, make a backup
		if _, err := os.Stat(path); err == nil {
			backupPath := path + ".backup"
			if err := os.Rename(path, backupPath); err != nil {
				return fmt.Errorf("failed to create backup: %w", err)
			}
		}

		// Move temporary file to final location
		if err := os.Rename(name, path); err != nil {
			return fmt.Errorf("failed to move temporary file: %w", err)
		}

		return nil
	}

	return temp, cleanup, commit, nil
}
