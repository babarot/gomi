//go:build windows

package atomic

import (
	"fmt"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// isSamePartition checks if the source and destination reside on the same filesystem partition on Windows.
func isSamePartition(src, dst string) (bool, error) {
	// Get the volume names (drive letters) for both source and destination paths
	srcVolume := filepath.VolumeName(src)
	dstVolume := filepath.VolumeName(dst)

	if srcVolume == "" || dstVolume == "" {
		return false, fmt.Errorf("failed to determine volume name from file paths")
	}

	// Get volume information for both source and destination volumes
	var srcVolID, dstVolID uint32
	err := windows.GetVolumeInformation(windows.StringToUTF16Ptr(srcVolume+"\\"), nil, 0, &srcVolID, nil, nil, nil, 0)
	if err != nil {
		return false, fmt.Errorf("failed to get source volume information: %w", err)
	}

	err = windows.GetVolumeInformation(windows.StringToUTF16Ptr(dstVolume+"\\"), nil, 0, &dstVolID, nil, nil, nil, 0)
	if err != nil {
		return false, fmt.Errorf("failed to get destination volume information: %w", err)
	}

	return srcVolID == dstVolID, nil
}
