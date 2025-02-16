//go:build windows

package xdg

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
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
	drives, err := getLogicalDrives()
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

			// Get drive properties
			volumeName := make([]uint16, 261)
			serialNumber := uint32(0)
			maxComponentLength := uint32(0)
			fsFlags := uint32(0)
			fileSystemName := make([]uint16, 261)

			kernel32 := syscall.NewLazyDLL("kernel32.dll")
			proc := kernel32.NewProc("GetVolumeInformationW")
			r1, _, err := proc.Call(
				uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(drive))),
				uintptr(unsafe.Pointer(&volumeName[0])),
				uintptr(len(volumeName)),
				uintptr(unsafe.Pointer(&serialNumber)),
				uintptr(unsafe.Pointer(&maxComponentLength)),
				uintptr(unsafe.Pointer(&fsFlags)),
				uintptr(unsafe.Pointer(&fileSystemName[0])),
				uintptr(len(fileSystemName)),
			)

			if r1 == 0 {
				slog.Warn("could not get volume information", "drive", drive, "error", err)
				continue
			}

			points = append(points, drive)
			slog.Debug("found mount point",
				"mountpoint", drive,
				"fstype", syscall.UTF16ToString(fileSystemName),
				"volumeName", syscall.UTF16ToString(volumeName))
		}
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("no valid mount points found")
	}

	return points, nil
}

// getLogicalDrives retrieves the available logical drives
func getLogicalDrives() (uint32, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetLogicalDrives")
	r1, _, err := proc.Call()
	if r1 == 0 {
		return 0, fmt.Errorf("GetLogicalDrives failed: %w", err)
	}
	return uint32(r1), nil
}

// getFileSystemType retrieves the filesystem type for a given drive
func getFileSystemType(drive string) (string, error) {
	volumeName := make([]uint16, 261)
	fileSystemName := make([]uint16, 261)

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetVolumeInformationW")
	r1, _, err := proc.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(drive))),
		uintptr(unsafe.Pointer(&volumeName[0])),
		uintptr(len(volumeName)),
		0, // lpVolumeSerialNumber
		0, // lpMaximumComponentLength
		0, // lpFileSystemFlags
		uintptr(unsafe.Pointer(&fileSystemName[0])),
		uintptr(len(fileSystemName)),
	)

	if r1 == 0 {
		return "", fmt.Errorf("GetVolumeInformation failed: %w", err)
	}

	return syscall.UTF16ToString(fileSystemName), nil
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
		drive = filepath.VolumeName(os.Getenv("SYSTEMDRIVE"))
		if drive == "" {
			drive = "C:"
		}
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
	slog.Debug("resolved symlink", "from", path1, "to", real1)

	real2, err := filepath.EvalSymlinks(path2)
	if err != nil {
		return false, fmt.Errorf("failed to resolve path %s: %w", path2, err)
	}
	slog.Debug("resolved symlink", "from", path2, "to", real2)

	// On Windows, compare drive letters
	drive1 := filepath.VolumeName(real1)
	drive2 := filepath.VolumeName(real2)

	// Get volume information for comparison
	volumeInfo1, err := getVolumeInfo(drive1)
	if err != nil {
		return false, fmt.Errorf("failed to get volume info for %s: %w", drive1, err)
	}

	volumeInfo2, err := getVolumeInfo(drive2)
	if err != nil {
		return false, fmt.Errorf("failed to get volume info for %s: %w", drive2, err)
	}

	slog.Debug("device comparison",
		"path1", real1, "drive1", drive1, "volumeInfo1", volumeInfo1,
		"path2", real2, "drive2", drive2, "volumeInfo2", volumeInfo2)

	sameDevice := strings.EqualFold(volumeInfo1, volumeInfo2)
	slog.Debug("device comparison result", "sameDevice", sameDevice)
	return sameDevice, nil
}

// getVolumeInfo gets volume information for comparison
func getVolumeInfo(drive string) (string, error) {
	if drive == "" {
		return "", fmt.Errorf("empty drive letter")
	}

	// Ensure drive ends with backslash
	if !strings.HasSuffix(drive, "\\") {
		drive += "\\"
	}

	// Get volume name and serial number for unique identification
	volumeName := make([]uint16, 261)
	serialNumber := uint32(0)
	maxComponentLength := uint32(0)
	fsFlags := uint32(0)
	fileSystemName := make([]uint16, 261)

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetVolumeInformationW")
	r1, _, err := proc.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(drive))),
		uintptr(unsafe.Pointer(&volumeName[0])),
		uintptr(len(volumeName)),
		uintptr(unsafe.Pointer(&serialNumber)),
		uintptr(unsafe.Pointer(&maxComponentLength)),
		uintptr(unsafe.Pointer(&fsFlags)),
		uintptr(unsafe.Pointer(&fileSystemName[0])),
		uintptr(len(fileSystemName)),
	)

	if r1 == 0 {
		return "", fmt.Errorf("GetVolumeInformation failed: %w", err)
	}

	// Combine volume name and serial number for unique identification
	return fmt.Sprintf("%s_%d", syscall.UTF16ToString(volumeName), serialNumber), nil
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
	// Check for standard trash subdirectories
	for _, subdir := range []string{"files", "info"} {
		subdirPath := filepath.Join(path, subdir)
		if _, err := os.Stat(subdirPath); os.IsNotExist(err) {
			slog.Debug("missing subdirectory", "path", subdirPath)
			return false
		}
	}

	slog.Debug("external trash directory is valid", "path", path)
	return true
}

// createTrashDir creates a trash directory with proper permissions
func createTrashDir(path string) error {
	// Create the main trash directory
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create trash directory: %w", err)
	}

	// Create standard subdirectories
	for _, subdir := range []string{"files", "info"} {
		subdirPath := filepath.Join(path, subdir)
		if err := os.MkdirAll(subdirPath, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", subdir, err)
		}
	}

	slog.Debug("trash directory created successfully", "path", path)
	return nil
}
