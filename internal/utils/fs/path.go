package fs

import (
	"path/filepath"
	"strings"
)

// IsUnsafePath checks if the given path is unsafe to remove
func IsUnsafePath(path string) (bool, error) {
	// First check the original path before any normalization
	// This preserves the original input like "." or ".."
	originalBase := filepath.Base(path)
	if originalBase == "." || originalBase == ".." {
		return true, nil
	}

	// Clean the path to check for normalized root paths
	cleaned := filepath.Clean(path)

	// Check root path
	if cleaned == "/" {
		return true, nil
	}

	// Check double slashes and similar patterns
	if strings.HasPrefix(path, "//") {
		return true, nil
	}

	return false, nil
}
