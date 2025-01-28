//go:build windows

package cli

import (
	"fmt"
	"log/slog"
	"os"
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
	err := windows.GetVolumeInformation(windows.StringToUTF16Ptr(srcVolume), nil, 0, &srcVolID, nil, nil, nil, 0)
	if err != nil {
		return false, fmt.Errorf("failed to get source volume information: %w", err)
	}

	err = windows.GetVolumeInformation(windows.StringToUTF16Ptr(dstVolume), nil, 0, &dstVolID, nil, nil, nil, 0)
	if err != nil {
		return false, fmt.Errorf("failed to get destination volume information: %w", err)
	}

	// Compare the volume IDs to determine if they are on the same partition
	samePartition := srcVolID == dstVolID

	slog.Debug("check src/dst volume info",
		"samePartition", samePartition,
		"src volume", srcVolID,
		"dst volume", dstVolID)

	return samePartition, nil
}
