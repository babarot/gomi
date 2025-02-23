package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/trash/xdg"
	"github.com/babarot/gomi/internal/ui"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
)

// OrphanedFile represents an orphaned metadata file with additional details
type OrphanedFile struct {
	Path          string
	DeletedAt     time.Time
	OriginalPath  string
	TrashInfoPath string
}

var (
	ErrInvalidArgument = errors.New("prune requires an argument (e.g., orphans)")
)

// Prune handles the pruning of trash contents
// It processes multiple subcommands for cleaning up the trash
func (c *CLI) Prune(args []string) error {
	slog.Debug("pruning trash contents started")
	defer slog.Debug("pruning trash contents finished")

	if len(args) == 0 {
		return ErrInvalidArgument
	}

	for _, arg := range args {
		// Check if orphans is present anywhere in the arguments
		if arg == "orphans" {
			if len(args) > 1 {
				return errors.New("orphans argument does not accept additional arguments")
			}
			return c.pruneOrphans()
		}
	}

	for _, arg := range args {
		switch arg {
		case "":
			return ErrInvalidArgument
		default:
			// Try to parse as durations
			var durations []time.Duration
			for _, arg := range args {
				duration, err := parseDuration(arg)
				if err != nil {
					return fmt.Errorf("invalid duration argument '%s': %w", arg, err)
				}
				durations = append(durations, duration)
			}
			return c.pruneDurationOverFiles(durations)
		}
	}

	return nil
}

// findOrphanedMetadata finds .trashinfo files without corresponding files
// Returns a list of paths to orphaned .trashinfo files
func findOrphanedMetadata(trashDir string) ([]OrphanedFile, error) {
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
			metadata, parseErr := parseTrashInfoFile(trashInfoPath)
			if parseErr != nil {
				continue
			}
			orphanedFiles = append(orphanedFiles, metadata)
		}
	}

	return orphanedFiles, nil
}

// pruneDurationOverFiles removes files that are older than the specified durations
func (c *CLI) pruneDurationOverFiles(durations []time.Duration) error {
	if len(durations) == 0 {
		return nil
	}

	// Find min and max durations
	minDuration := durations[0]
	maxDuration := durations[0]
	for _, d := range durations {
		if d < minDuration {
			minDuration = d
		}
		if d > maxDuration {
			maxDuration = d
		}
	}

	slog.Debug("Get all files from trash")
	files, err := c.manager.List()
	if err != nil {
		return fmt.Errorf("failed to list trash contents: %w", err)
	}

	// Filter files based on duration(s)
	var targetFiles []*trash.File
	for _, file := range files {
		age := time.Since(file.DeletedAt)
		if len(durations) == 1 {
			// Single duration: get files older than duration
			if age > maxDuration {
				targetFiles = append(targetFiles, file)
			}
		} else {
			// Multiple durations: get files between min and max duration
			if age >= minDuration && age <= maxDuration {
				targetFiles = append(targetFiles, file)
			}
		}
	}

	if len(targetFiles) == 0 {
		fmt.Println("No matching files found.")
		return nil
	}

	green := color.New(color.FgHiGreen).SprintfFunc()
	white := color.New(color.FgWhite).SprintfFunc()

	fmt.Printf("%s %s %s\n",
		green("%-20s", "Deleted At"),
		green("%-18s", ""),
		green("%-30s", "Path"))

	for _, file := range targetFiles {
		fmt.Printf("%s %s %s\n",
			white("%-20s", file.DeletedAt.Format("2006-01-02 15:04:05")),
			white("%-18s", "("+humanize.Time(file.DeletedAt)+")"),
			white("%-30s", file.Name))
	}
	fmt.Println()

	// Print files that will be removed
	if len(durations) == 1 {
		days := int(maxDuration.Hours() / 24)
		fmt.Printf("Found %d files that are older than %d days.\n\n", len(targetFiles), days)
	} else {
		minDays := int(minDuration.Hours() / 24)
		maxDays := int(maxDuration.Hours() / 24)
		fmt.Printf("Found %d files that were moved to trash between %d and %d days ago.\n\n",
			len(targetFiles), minDays, maxDays)
	}

	if !ui.Confirm(fmt.Sprintf("Are you sure you want to remove these %d files?", len(targetFiles))) {
		fmt.Println("Pruning canceled.")
		return nil
	}

	// Remove files
	var failedRemovals []string
	for _, file := range targetFiles {
		slog.Debug("removing trash file", "file", file.OriginalPath)
		if err := c.manager.Remove(file); err != nil {
			slog.Error("failed to remove file", "file", file.Name, "error", err)
			failedRemovals = append(failedRemovals, file.Name)
		}
	}

	if len(failedRemovals) > 0 {
		fmt.Printf("Failed to remove %d files:\n", len(failedRemovals))
		for _, file := range failedRemovals {
			fmt.Println("-", file)
		}
		return fmt.Errorf("some files could not be removed")
	}

	fmt.Printf("Successfully removed %d files.\n", len(targetFiles))
	return nil
}

// pruneOrphans removes metadata files without corresponding trashed files
func (c *CLI) pruneOrphans() error {
	slog.Debug("pruning orphaned trashinfo")

	trashDirs, err := xdg.FindAllTrashDirectories()
	if err != nil {
		return fmt.Errorf("failed to get trash dirs: %w", err)
	}

	var orphanedFiles []OrphanedFile
	for _, trashDir := range trashDirs {
		slog.Debug("pruning orphaned trashinfo", "trashDir", trashDir)
		files, err := findOrphanedMetadata(trashDir)
		if err != nil {
			slog.Error("failed to find orphaned metadata in trash dir", "dir", trashDir, "error", err)
			continue
		}
		orphanedFiles = append(orphanedFiles, files...)
	}

	if len(orphanedFiles) == 0 {
		fmt.Println("No orphaned metadata files found.")
		return nil
	}

	// Confirm deletion unless forced
	if !c.option.Rm.Force {
		slog.Debug("show orphaned trashinfo", "files", orphanedFiles)
		printOrphanedFilesTable(orphanedFiles)
		if !ui.Confirm(fmt.Sprintf("Are you sure you want to remove %d orphaned metadata files?", len(orphanedFiles))) {
			fmt.Println("Pruning canceled.")
			return nil
		}
	}

	// Remove orphaned files
	var failedRemovals []string
	for _, file := range orphanedFiles {
		slog.Debug("removing orphaned trashinfo", "file", file.TrashInfoPath)
		if err := os.Remove(file.TrashInfoPath); err != nil {
			slog.Error("failed to remove orphaned metadata file", "file", file.TrashInfoPath, "error", err)
			failedRemovals = append(failedRemovals, file.TrashInfoPath)
		}
	}

	if len(failedRemovals) > 0 {
		fmt.Printf("Failed to remove %d files:\n", len(failedRemovals))
		for _, file := range failedRemovals {
			fmt.Println("-", file)
		}
		return fmt.Errorf("some orphaned metadata files could not be removed")
	}

	fmt.Printf("Successfully removed %d orphaned metadata files.\n", len(orphanedFiles))
	return nil
}

// printOrphanedFilesTable prints a formatted table of orphaned files
func printOrphanedFilesTable(files []OrphanedFile) {
	green := color.New(color.FgHiGreen).SprintfFunc()
	white := color.New(color.FgWhite).SprintfFunc()

	fmt.Printf("%s %s\n",
		green("%-20s", "Deleted At"),
		green("%-30s", "Path"),
	)

	for _, file := range files {
		fmt.Printf("%s %s\n",
			white("%-20s", file.DeletedAt.Format("2006-01-02 15:04:05")),
			white("%-30s", file.TrashInfoPath),
		)
	}
	fmt.Println()
}

// parseTrashInfoFile parses a .trashinfo file and returns an OrphanedFile
func parseTrashInfoFile(path string) (OrphanedFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return OrphanedFile{}, err
	}

	lines := strings.Split(string(content), "\n")
	var deletedAt time.Time
	var originalPath string

	for _, line := range lines {
		if strings.HasPrefix(line, "Path=") {
			originalPath = strings.TrimPrefix(line, "Path=")
		}
		if strings.HasPrefix(line, "DeletionDate=") {
			deletedAtStr := strings.TrimPrefix(line, "DeletionDate=")
			deletedAt, err = time.Parse("2006-01-02T15:04:05", deletedAtStr)
			if err != nil {
				return OrphanedFile{}, err
			}
		}
	}

	return OrphanedFile{
		DeletedAt:     deletedAt,
		OriginalPath:  originalPath,
		TrashInfoPath: path,
	}, nil
}

var unitMap = map[string]string{
	"h":      "h",
	"hour":   "h",
	"hours":  "h",
	"d":      "d",
	"day":    "d",
	"days":   "d",
	"w":      "w",
	"week":   "w",
	"weeks":  "w",
	"m":      "m",
	"month":  "m",
	"months": "m",
	"y":      "y",
	"year":   "y",
	"years":  "y",
}

func splitNumberAndUnit(input string) (string, string, error) {
	input = strings.TrimSpace(input)
	numPart := strings.Builder{}
	unitPart := strings.Builder{}

	for _, r := range input {
		switch {
		case unicode.IsDigit(r):
			numPart.WriteRune(r)
		case unicode.IsLetter(r):
			unitPart.WriteRune(r)
		default:
			return "", "", errors.New("invalid char included")
		}
	}
	return numPart.String(), unitPart.String(), nil
}

func parseDuration(input string) (time.Duration, error) {
	numStr, unit, err := splitNumberAndUnit(strings.ToLower(input))
	if err != nil {
		return 0, fmt.Errorf("invalid chars in duration: %s", input)
	}

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid number in duration: %s", input)
	}

	mappedUnit, exists := unitMap[unit]
	if !exists {
		return 0, fmt.Errorf("unsupported duration unit: %s", unit)
	}

	unitDurations := map[string]time.Duration{
		"h": 1 * time.Hour,
		"d": 24 * time.Hour,
		"w": 7 * 24 * time.Hour,
		"m": 30 * 24 * time.Hour,
		"y": 365 * 24 * time.Hour,
	}

	return time.Duration(num) * unitDurations[mappedUnit], nil
}
