package fs

import (
	"os"
	"path/filepath"
	"sync"
)

func DirSize(path string) (int64, error) {
	var size int64
	var mu sync.Mutex

	// Function to calculate size for a given path
	var calculateSize func(string) error
	calculateSize = func(p string) error {
		fileInfo, err := os.Lstat(p)
		if err != nil {
			return err
		}

		// Skip symbolic links to avoid counting them multiple times
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		if fileInfo.IsDir() {
			entries, err := os.ReadDir(p)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if err := calculateSize(filepath.Join(p, entry.Name())); err != nil {
					return err
				}
			}
		} else {
			mu.Lock()
			size += fileInfo.Size()
			mu.Unlock()
		}
		return nil
	}

	// Start calculation from the root path
	if err := calculateSize(path); err != nil {
		return 0, err
	}

	return size, nil
}
