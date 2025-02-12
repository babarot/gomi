//go:build !windows

package atomic

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// isSamePartition checks if the source and destination reside on the same filesystem partition.
func isSamePartition(src, dst string) (bool, error) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return false, fmt.Errorf("failed to get source file stats: %w", err)
	}

	dstInfo, err := os.Stat(filepath.Dir(dst))
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to get destination file stats: %w", err)
	}

	// If destination doesn't exist, check its parent directory
	if os.IsNotExist(err) {
		dstInfo, err = os.Stat(filepath.Dir(dst))
		if err != nil {
			return false, fmt.Errorf("failed to get destination parent directory stats: %w", err)
		}
	}

	srcSys, ok := srcInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return false, fmt.Errorf("failed to get source system info")
	}

	dstSys, ok := dstInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return false, fmt.Errorf("failed to get destination system info")
	}

	return srcSys.Dev == dstSys.Dev, nil
}
