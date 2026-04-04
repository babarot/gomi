package xdg

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OrphanedFile represents an orphaned metadata file with additional details
type OrphanedFile struct {
	Path          string
	DeletedAt     time.Time
	OriginalPath  string
	TrashInfoPath string
	TrashInfoName string
}

func (o OrphanedFile) GetName() string         { return o.TrashInfoName }
func (o OrphanedFile) GetDeletedAt() time.Time { return o.DeletedAt }

// FindOrphanedTrashInfoFiles finds .trashinfo files without corresponding files
// Returns a list of paths to orphaned .trashinfo files
func FindOrphanedTrashInfoFiles(trashDir string) ([]OrphanedFile, error) {
	infoDir := filepath.Join(trashDir, "info")
	filesDir := filepath.Join(trashDir, "files")

	var orphanedFiles []OrphanedFile

	entries, err := os.ReadDir(infoDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read info directory: %w", err)
	}

	for _, entry := range entries {
		// Skip non-regular files and non-.trashinfo files
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".trashinfo") {
			continue
		}

		if strings.HasPrefix(entry.Name(), "._") {
			// exclude mac resource fork
			slog.Debug("skipped mac resource fork of .trashinfo", "path", entry.Name())
			continue
		}

		// Remove .trashinfo suffix to get the corresponding file name
		fileName := strings.TrimSuffix(entry.Name(), ".trashinfo")
		trashInfoPath := filepath.Join(infoDir, entry.Name())

		// Check if corresponding file exists in files directory
		_, err := os.Stat(filepath.Join(filesDir, fileName))
		if os.IsNotExist(err) {
			metadata, parseErr := ParseTrashInfoFile(trashInfoPath)
			if parseErr != nil {
				continue
			}
			orphanedFiles = append(orphanedFiles, metadata)
		}
	}

	return orphanedFiles, nil
}

// ParseTrashInfoFile parses a .trashinfo file and returns an OrphanedFile
func ParseTrashInfoFile(path string) (OrphanedFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return OrphanedFile{}, err
	}

	lines := strings.Split(string(content), "\n")
	var deletedAt time.Time
	var originalPath string

	for _, line := range lines {
		if after, ok := strings.CutPrefix(line, "Path="); ok {
			originalPath = after
		}
		if after, ok := strings.CutPrefix(line, "DeletionDate="); ok {
			deletedAt, err = time.Parse("2006-01-02T15:04:05", after)
			if err != nil {
				return OrphanedFile{}, err
			}
		}
	}

	return OrphanedFile{
		DeletedAt:     deletedAt,
		OriginalPath:  originalPath,
		TrashInfoPath: path,
		TrashInfoName: filepath.Base(path),
	}, nil
}
