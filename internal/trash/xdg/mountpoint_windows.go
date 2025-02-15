//go:build windows

package xdg

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/babarot/gomi/internal/trash"
)

// Windows-specific implementation
var skipFSTypes = map[string]bool{
	"NTFS":      false,
	"FAT32":     false,
	"exFAT":     false,
	"ReFS":      false,
	"Temporary": true,
	"Network":   false,
}

// getMountPoints returns a list of valid mount points that can contain trash directories
func getMountPoints() ([]string, error) {
	// On Windows, get logical drives
	drives, err := syscall.GetLogicalDrives()
	if err != nil {
		return nil, fmt.Errorf("failed to get logical drives: %w", err)
	}

	var points []string
	for i := 0; i < 26; i++ {
		if drives&(1<<uint(i)) != 0 {
			drive := string(rune('A'+i)) + ":\\"

			// Skip network drives and temporary drives
			fsType, err := getFileSystemType(drive)
			if err != nil {
				slog.Warn("could not get filesystem type", "drive", drive, "error", err)
				continue
			}

			if skipFSTypes[fsType] {
				slog.Debug("skipping filesystem", "type", fsType, "drive", drive)
				continue
			}

			points = append(points, drive)
			slog.Debug("found mount point", "mountpoint", drive, "fstype", fsType)
		}
	}

	return points, nil
}

// getFileSystemType retrieves the filesystem type for a given drive
func getFileSystemType(drive string) (string, error) {
	var volumeName [261]uint16
	var fileSystemName [261]uint16

	err := syscall.GetVolumeInformation(
		syscall.StringToUTF16Ptr(drive),
		&volumeName[0],
		uint32(len(volumeName)),
		nil,
		nil,
		nil,
		&fileSystemName[0],
		uint32(len(fileSystemName)),
	)
	if err != nil {
		return "", fmt.Errorf("failed to get volume information: %w", err)
	}

	return syscall.UTF16ToString(fileSystemName[:]), nil
}

// getMountPoint returns the mount point for the given path on Windows
func getMountPoint(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// On Windows, get the root of the drive
	drive := filepath.VolumeName(absPath)
	if drive == "" {
		return filepath.VolumeName(absPath) + "\\", nil
	}

	slog.Debug("found mount point", "path", absPath, "mountpoint", drive+"\\")
	return drive + "\\", nil
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

	// On Windows, compare drive letters
	drive1 := filepath.VolumeName(real1)
	drive2 := filepath.VolumeName(real2)

	slog.Debug("device comparison",
		"path1", real1, "drive1", drive1,
		"path2", real2, "drive2", drive2)

	return strings.EqualFold(drive1, drive2), nil
}

// isValidExternalTrash checks if a directory is a valid trash directory
func isValidExternalTrash(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		slog.Debug("no trash directory", "path", path)
		return false
	}

	// Must be a directory
	if !info.IsDir() {
		slog.Debug("not a directory", "path", path)
		return false
	}

	// Windows doesn't have sticky bit or exact permission matching
	// Additional validation might be needed based on specific requirements

	// Check for standard trash subdirectories
	for _, subdir := range []string{"files", "info"} {
		subdirPath := filepath.Join(path, subdir)
		if _, err := os.Stat(subdirPath); os.IsNotExist(err) {
			slog.Debug("missing subdirectory", "path", subdirPath)
			return false
		}
	}

	return true
}
