//go:build !windows

package cli

import (
	"fmt"
	"log/slog"
	"os"
	"syscall"
)

// isSamePartition checks if the source and destination reside on the same filesystem partition.
func isSamePartition(src, dst string) (bool, error) {
	srcStat, err := os.Stat(src)
	if err != nil {
		return false, fmt.Errorf("failed to get source file stats: %w", err)
	}

	dstStat, err := os.Stat(dst)
	if err != nil {
		return false, fmt.Errorf("failed to get destination file stats: %w", err)
	}

	srcSys := srcStat.Sys().(*syscall.Stat_t)
	dstSys := dstStat.Sys().(*syscall.Stat_t)

	// Compare the device identifiers (st_dev) of the source and destination
	// If the device IDs are the same, the files are on the same partition.
	samePartition := srcSys.Dev == dstSys.Dev

	slog.Debug("check src/dst file info",
		"samePartition", samePartition,
		"src st_dev", srcSys.Dev,
		"dst st_dev", dstSys.Dev)

	return samePartition, nil
}
