package xdg

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/babarot/gomi/internal/fs"
	"github.com/babarot/gomi/internal/trash"
)

const (
	// According to XDG spec
	trashInfoHeader = "[Trash Info]"
	timeFormat      = "2006-01-02T15:04:05"
)

// TrashInfo represents the contents of a .trashinfo file
type TrashInfo struct {
	// Original absolute path of the file
	Path string

	// Original base name of the file
	OriginalName string

	// When the file was trashed
	DeletionDate time.Time
}

// loadTrashInfo loads and parses a .trashinfo file
func loadTrashInfo(path string) (*TrashInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open info file: %w", err)
	}
	defer f.Close()

	return parseTrashInfo(f)
}

// parseTrashInfo parses the content of a .trashinfo file
func parseTrashInfo(r io.Reader) (*TrashInfo, error) {
	scanner := bufio.NewScanner(r)
	info := &TrashInfo{}
	var headerFound bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for header
		if line == trashInfoHeader {
			headerFound = true
			continue
		}

		// Skip until header is found
		if !headerFound {
			continue
		}

		// Parse key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Path":
			path, err := url.QueryUnescape(value)
			if err != nil {
				return nil, fmt.Errorf("invalid Path encoding: %w", err)
			}
			/* TODO: support relative path? */
			// if !filepath.IsAbs(path) {
			// 	path, _ = filepath.Abs(path)
			// }
			info.Path = path
			info.OriginalName = filepath.Base(path)
			slog.Debug("parse trash info",
				"info.Path", info.Path,
				"info.OriginalName", info.OriginalName)

		case "DeletionDate":
			date, err := time.ParseInLocation(timeFormat, value, time.Local)
			if err != nil {
				return nil, fmt.Errorf("invalid DeletionDate format: %w", err)
			}
			info.DeletionDate = date
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading info file: %w", err)
	}

	// Validate required fields
	if !headerFound {
		return nil, trash.NewStorageError("parse", "", fmt.Errorf("missing [Trash Info] header"))
	}
	if info.Path == "" {
		return nil, trash.NewStorageError("parse", "", fmt.Errorf("missing Path field"))
	}
	if info.DeletionDate.IsZero() {
		return nil, trash.NewStorageError("parse", "", fmt.Errorf("missing DeletionDate field"))
	}

	return info, nil
}

// Save writes the trash info to a file atomically
func (i *TrashInfo) Save(path string) error {
	// Create content
	content := new(strings.Builder)
	fmt.Fprintln(content, trashInfoHeader)
	fmt.Fprintf(content, "Path=%s\n", encodeTrashPath(i.Path))
	fmt.Fprintf(content, "DeletionDate=%s\n", i.DeletionDate.Format(timeFormat))

	// Write atomically using O_EXCL flag to prevent overwriting existing files
	f, err := fs.Create(path, 0600)
	if err != nil {
		return fmt.Errorf("failed to create info file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(content.String()); err != nil {
		// Try to remove the file if write fails
		os.Remove(path)
		return fmt.Errorf("failed to write info file: %w", err)
	}

	return nil
}

// encodeTrashPath encodes a path according to the XDG specification:
// - Forward slashes are not encoded
// - Spaces are encoded as %20 (not +)
// - Other special characters are percent-encoded
func encodeTrashPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		// Split by space to handle spaces separately
		subparts := strings.Split(part, " ")
		for j, subpart := range subparts {
			subparts[j] = url.QueryEscape(subpart)
		}
		// Join with %20 instead of +
		parts[i] = strings.Join(subparts, "%20")
	}
	return strings.Join(parts, "/")
}

// isTrashInfo checks if a filename is a valid .trashinfo file
func isTrashInfo(name string) bool {
	return strings.HasSuffix(name, ".trashinfo")
}

// getOriginalPath returns the original path from a trashinfo file path
func getOriginalPath(infoPath string) (string, error) {
	info, err := loadTrashInfo(infoPath)
	if err != nil {
		return "", err
	}
	return info.Path, nil
}
