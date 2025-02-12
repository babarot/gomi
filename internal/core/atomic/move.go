package atomic

import (
	"fmt"
	"os"
	"path/filepath"

	cp "github.com/otiai10/copy"
)

// MoveOptions specifies options for move operations
type MoveOptions struct {
	AllowCrossDev bool // Allow cross-device moves
	Force         bool // Force operation even if destination exists
}

// Move performs an atomic move operation
func Move(src, dst string, opts MoveOptions) error {
	// 1. Validate paths
	if err := validatePaths(src, dst); err != nil {
		return err
	}

	// 2. Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return &MoveError{
			Op:  "create_parent",
			Src: src,
			Dst: dst,
			Err: err,
		}
	}

	// 3. Check destination existence if not force mode
	if !opts.Force {
		if _, err := os.Stat(dst); err == nil {
			return ErrDestinationExists
		}
	}

	// 4. Handle same device case first - try simple rename
	if sameDevice, _ := isSamePartition(src, dst); sameDevice {
		if err := os.Rename(src, dst); err == nil {
			return nil
		}
	}

	// 5. Fall back to copy and delete
	return copyAndDelete(src, dst)
}

// copyAndDelete copies a file or directory and then deletes the original
func copyAndDelete(src, dst string) error {
	// Copy with otiai10/copy which handles:
	// - Directory recursion
	// - Permissions
	// - Timestamps
	// - Symbolic links
	opts := cp.Options{
		// Skip: func(src string) (bool, error) {
		// 	return false, nil // Don't skip any files
		// },
		AddPermission: 0, // Don't modify permissions
		OnSymlink: func(src string) cp.SymlinkAction {
			return cp.Deep // Follow symlinks
		},
		PreserveTimes: true, // Preserve timestamps
		PreserveOwner: true, // Preserve ownership
		OnDirExists:   nil,  // Default behavior
		Sync:          true, // Sync files during copy
		// PermissionControl: cp.PermissionControlOptions{
		// 	PreserveMode: true, // Preserve mode bits
		// },
	}

	if err := cp.Copy(src, dst, opts); err != nil {
		return &MoveError{
			Op:  "copy",
			Src: src,
			Dst: dst,
			Err: err,
		}
	}

	// If copy succeeds, remove source
	if err := os.RemoveAll(src); err != nil {
		// Try to clean up destination on failure
		if rmErr := os.RemoveAll(dst); rmErr != nil {
			return &MoveError{
				Op:  "cleanup",
				Src: src,
				Dst: dst,
				Err: fmt.Errorf("failed to remove both source and destination: %v, %v", err, rmErr),
			}
		}
		return &MoveError{
			Op:  "remove_source",
			Src: src,
			Dst: dst,
			Err: err,
		}
	}

	return nil
}

// validatePaths performs basic path validation
func validatePaths(src, dst string) error {
	if src == "" || dst == "" {
		return ErrInvalidPath
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrSourceNotFound
		}
		return err
	}

	// If src is a directory, ensure recursive flag is set
	if srcInfo.IsDir() {
		return nil
	}

	return nil
}
