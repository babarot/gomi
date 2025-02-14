package xdg

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/babarot/gomi/internal/trash/core"
	"github.com/moby/sys/mountinfo"
)

// Skip file systems that can't have trash directories
var skipFSTypes = map[string]bool{
	"proc":        true,
	"sysfs":       true,
	"devtmpfs":    true,
	"devpts":      true,
	"tmpfs":       true,
	"cgroup":      true,
	"cgroup2":     true,
	"pstore":      true,
	"securityfs":  true,
	"debugfs":     true,
	"configfs":    true,
	"fusectl":     true,
	"bpf":         true,
	"nsfs":        true,
	"efivarfs":    true,
	"hugetlbfs":   true,
	"mqueue":      true,
	"binfmt_misc": true,
}

// getMountPoints returns a list of valid mount points that can contain trash directories
func getMountPoints() ([]string, error) {
	// Get all mount points
	mounts, err := mountinfo.GetMounts(func(info *mountinfo.Info) (skip, stop bool) {
		// Skip known unsuitable filesystems
		if skipFSTypes[info.FSType] {
			slog.Debug("skipping filesystem", "type", info.FSType, "mountpoint", info.Mountpoint)
			return true, false
		}

		// Skip read-only filesystems
		opts := strings.Split(info.Options, ",")
		for _, opt := range opts {
			if opt == "ro" {
				slog.Debug("skipping read-only filesystem", "mountpoint", info.Mountpoint)
				return true, false
			}
		}

		return false, false
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get mount info: %w", err)
	}

	// Convert to simple path list, ensuring uniqueness
	seen := make(map[string]bool)
	var points []string

	for _, m := range mounts {
		if !seen[m.Mountpoint] {
			points = append(points, m.Mountpoint)
			seen[m.Mountpoint] = true
			slog.Debug("found mount point", "mountpoint", m.Mountpoint, "fstype", m.FSType)
		}
	}

	// Always ensure root filesystem is included
	if !seen["/"] {
		points = append(points, "/")
		slog.Debug("added root filesystem")
	}

	return points, nil
}

// getMountPoint returns the mount point for the given path
func getMountPoint(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get all mount points
	mounts, err := mountinfo.GetMounts(nil)
	if err != nil {
		return "", fmt.Errorf("failed to get mount info: %w", err)
	}

	// Find the longest matching mount point
	var longest string
	for _, m := range mounts {
		if strings.HasPrefix(absPath, m.Mountpoint) {
			if len(m.Mountpoint) > len(longest) {
				longest = m.Mountpoint
			}
		}
	}

	if longest == "" {
		// If no mount point found, the path must be on the root filesystem
		return "/", nil
	}

	slog.Debug("found mount point", "path", absPath, "mountpoint", longest)
	return longest, nil
}

// isOnSameDevice checks if two paths are on the same device
func isOnSameDevice(path1, path2 string) (bool, error) {
	// Resolve any symlinks
	real1, err := filepath.EvalSymlinks(path1)
	if err != nil {
		return false, fmt.Errorf("failed to resolve path %s: %w", path1, err)
	}

	real2, err := filepath.EvalSymlinks(path2)
	if err != nil {
		return false, fmt.Errorf("failed to resolve path %s: %w", path2, err)
	}

	info1, err := os.Stat(real1)
	if err != nil {
		return false, fmt.Errorf("failed to stat %s: %w", real1, err)
	}

	info2, err := os.Stat(real2)
	if err != nil {
		return false, fmt.Errorf("failed to stat %s: %w", real2, err)
	}

	stat1, ok1 := info1.Sys().(*syscall.Stat_t)
	stat2, ok2 := info2.Sys().(*syscall.Stat_t)

	if !ok1 || !ok2 {
		return false, core.NewStorageError("check-device", "", fmt.Errorf("failed to get device information"))
	}

	slog.Debug("device comparison",
		"path1", real1, "dev1", stat1.Dev,
		"path2", real2, "dev2", stat2.Dev)

	return stat1.Dev == stat2.Dev, nil
}

// isValidExternalTrash checks if a directory is a valid trash directory according to the XDG spec
func isValidExternalTrash(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		// return fmt.Errorf("failed to stat trash directory: %w", err)
		slog.Debug("failed to stat trash directory", "path", path, "error", err)
		return false
	}

	// Must be a directory
	if !info.IsDir() {
		// return fmt.Errorf("%s is not a directory", path)
		slog.Debug("not a directory", "path", path)
		return false
	}

	// Must not be a symbolic link
	if info.Mode()&os.ModeSymlink != 0 {
		slog.Debug("is a symbolic link", "path", path)
		return false
	}

	// If it's a .Trash directory (not .Trash-$uid), check sticky bit
	if filepath.Base(path) == ".Trash" {
		if info.Mode()&os.ModeSticky == 0 {
			slog.Debug("missing sticky bit", "path", path)
			return false
		}
	}

	// All internal directories must be mode 0700
	for _, subdir := range []string{"files", "info"} {
		subdirPath := filepath.Join(path, subdir)
		info, err := os.Stat(subdirPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			slog.Debug("failed to stat subdirectory",
				"path", subdirPath,
				"error", err)
			return false
		}

		if info.Mode().Perm() != 0700 {
			slog.Warn("incorrect permissions on trash subdirectory",
				"path", subdirPath,
				"mode", info.Mode().Perm(),
				"expected", 0700)
		}
	}

	return true
}

// createTrashDir creates a trash directory with proper permissions
func createTrashDir(path string) error {
	// Create the main trash directory
	if err := os.MkdirAll(path, 0700); err != nil {
		return fmt.Errorf("failed to create trash directory: %w", err)
	}

	// Create standard subdirectories
	for _, subdir := range []string{"files", "info"} {
		subdirPath := filepath.Join(path, subdir)
		if err := os.MkdirAll(subdirPath, 0700); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", subdir, err)
		}
	}

	slog.Debug("created trash directory", "path", path)
	return nil
}
