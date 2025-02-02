package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	cp "github.com/otiai10/copy"
)

// copyAndDelete copies a file or directory (recursively) and then deletes the original.
func copyAndDelete(src, dst string) error {
	slog.Debug("starting copy and delete operation", "from", src, "to", dst)
	if err := cp.Copy(src, dst); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// If the copy is successful, remove the original file or directory
	if err := os.Remove(src); err != nil {
		// If removal of the source fails after copying, attempt to delete the copied file as well
		if rmErr := os.Remove(dst); rmErr != nil {
			return fmt.Errorf(
				"failed to remove both source and destination files: source error: %v, destination error: %v",
				err, rmErr)
		}
		return fmt.Errorf("failed to remove source file after successful copy: %w", err)
	}

	return nil
}

// move attempts to rename a file or directory. If it's on different partitions, it falls back to copying and deleting.
func move(src, dst string) error {
	dstDir := filepath.Dir(dst)

	// Ensure the destination directory exists before attempting to move
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		slog.Debug("mkdir", "dir", dstDir)
		if err := os.MkdirAll(dstDir, 0777); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}
	}

	// Check if source and destination are on the same partition
	samePartition, err := isSamePartition(src, dstDir)
	if err != nil {
		return err
	}
	defer slog.Debug("file moved", "from", src, "to", dst)

	// If they are on the same partition, use os.Rename; otherwise, fallback to copy-and-delete
	if samePartition {
		slog.Debug("moving file with os.Rename")
		return os.Rename(src, dst)
	}

	slog.Debug("different partitions detected, falling back to copy-and-delete operation")
	return copyAndDelete(src, dst)
}
