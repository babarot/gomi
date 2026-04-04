package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/fatih/color"

	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/trash/xdg"
	"github.com/babarot/gomi/internal/ui/table"
	"github.com/babarot/gomi/internal/utils/duration"
)

var (
	// Base errors
	ErrInvalidArgument = errors.New("prune requires an argument (e.g., orphans or duration like '30d')")

	// Orphans-related errors
	ErrOrphansCombination = errors.New("orphans argument cannot be combined with other arguments")

	// Duration-related errors
	ErrInvalidDuration     = errors.New("invalid duration format")
	ErrInvalidDurationNum  = errors.New("duration must be a positive number")
	ErrInvalidDurationUnit = errors.New("unsupported duration unit")
)

// Prune handles the pruning of trash contents
// It processes multiple subcommands for cleaning up the trash
func (c *CLI) Prune(args []string) error {
	slog.Debug("pruning trash contents started")
	defer slog.Debug("pruning trash contents finished")

	if len(args) == 0 {
		return fmt.Errorf("prune: %w", ErrInvalidArgument)
	}

	for _, arg := range args {
		if arg == "orphans" {
			if len(args) > 1 {
				return fmt.Errorf("prune: %w", ErrOrphansCombination)
			}
			return c.removeOrphanedMetadata()
		}
	}

	// Parse durations
	var durations []time.Duration
	for _, arg := range args {
		if arg == "" {
			return fmt.Errorf("prune: %w", ErrInvalidArgument)
		}
		duration, err := duration.Parse(arg)
		if err != nil {
			return fmt.Errorf("prune: %w", err)
		}
		durations = append(durations, duration)
	}
	return c.permanentlyDeleteByTimeRange(durations)
}

// permanentlyDeleteByTimeRange removes files from trash based on their age.
// For a single duration, it removes files older than the specified duration.
// For multiple durations, it removes files whose age falls between the shortest and longest durations.
// The operation requires user confirmation and cannot be undone.
func (c *CLI) permanentlyDeleteByTimeRange(durations []time.Duration) error {
	if len(durations) == 0 {
		return nil
	}

	// Find newest and oldest age boundaries
	newestAge := durations[0]
	oldestAge := durations[0]
	for _, d := range durations {
		if d < newestAge {
			newestAge = d
		}
		if d > oldestAge {
			oldestAge = d
		}
	}

	slog.Debug("Get all files from trash")
	files, err := c.trash.List()
	if err != nil {
		return fmt.Errorf("failed to list trash contents: %w", err)
	}

	// Filter files based on time range
	var filesToDelete []*trash.File
	for _, file := range files {
		age := time.Since(file.DeletedAt)
		if len(durations) == 1 {
			// Single duration: get files older than duration
			if age > oldestAge {
				filesToDelete = append(filesToDelete, file)
			}
		} else {
			// Multiple durations: get files between newest and oldest age
			if age >= newestAge && age <= oldestAge {
				filesToDelete = append(filesToDelete, file)
			}
		}
	}

	if len(filesToDelete) == 0 {
		fmt.Println("No matching files found.")
		return nil
	}

	table.PrintFiles(filesToDelete, table.PrintOptions{
		ShowRelativeTime: true,
		Order:            table.SortDesc,
	})
	fmt.Println()
	printDeletionSummary(filesToDelete, newestAge, oldestAge, len(durations) == 1)

	// First confirmation
	if !c.prompter.Confirm(fmt.Sprintf("Are you sure you want to remove these %d files?", len(filesToDelete))) {
		fmt.Println("Operation canceled.")
		return nil
	}

	// Second confirmation with warning
	fmt.Println()
	// WARNING: Files will be permanently deleted and CANNOT be recovered. Are you absolutely sure?
	fmt.Printf("%s\n", color.New(color.FgHiRed).Sprint("WARNING: This operation is permanent and cannot be undone!"))
	if !c.prompter.ConfirmYes("Do you really want to permanently delete these files?") {
		fmt.Println("Operation canceled.")
		return nil
	}

	// Remove files
	var failedDeletions []string
	for _, file := range filesToDelete {
		slog.Debug("removing trash file", "file", file.OriginalPath)
		if err := c.trash.Remove(file); err != nil {
			slog.Error("failed to remove file", "file", file.Name, "error", err)
			failedDeletions = append(failedDeletions, file.Name)
		}
	}

	if len(failedDeletions) > 0 {
		fmt.Printf("Failed to remove %d files:\n", len(failedDeletions))
		for _, file := range failedDeletions {
			fmt.Println("-", file)
		}
		return fmt.Errorf("some files could not be removed")
	}

	fmt.Printf("Successfully removed %d files.\n", len(filesToDelete))
	return nil
}

// printDeletionSummary prints a summary of the files to be deleted
func printDeletionSummary(files []*trash.File, newestAge, oldestAge time.Duration, isSingleDuration bool) {
	if isSingleDuration {
		days := int(oldestAge.Hours() / 24)
		fmt.Printf("Found %d files that are older than %d days.\n", len(files), days)
	} else {
		minDays := int(newestAge.Hours() / 24)
		maxDays := int(oldestAge.Hours() / 24)
		fmt.Printf("Found %d files that were moved to trash between %d and %d days ago.\n",
			len(files), minDays, maxDays)
	}
}

// removeOrphanedMetadata removes .trashinfo files that have lost their corresponding data files.
// These orphaned files can occur due to:
// - Manual deletion of files from the trash
// - System crashes during trash operations
// - File system corruption
// Returns an error if any orphaned files could not be removed.
func (c *CLI) removeOrphanedMetadata() error {
	slog.Debug("pruning orphaned trashinfo")

	trashDirs, err := xdg.FindAllTrashDirectories()
	if err != nil {
		return fmt.Errorf("failed to get trash dirs: %w", err)
	}

	var orphanedFiles []xdg.OrphanedFile
	for _, trashDir := range trashDirs {
		slog.Debug("pruning orphaned trashinfo", "trashDir", trashDir)
		files, err := xdg.FindOrphanedTrashInfoFiles(trashDir)
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
		table.PrintFiles(orphanedFiles, table.PrintOptions{
			ShowRelativeTime: false,
			Order:            table.SortDesc,
		})
		fmt.Println()
		if !c.prompter.Confirm(fmt.Sprintf("Are you sure you want to remove %d orphaned metadata files?", len(orphanedFiles))) {
			fmt.Println("Operation canceled.")
			return nil
		}
	}

	// Remove orphaned files
	var failedDeletions []string
	for _, file := range orphanedFiles {
		slog.Debug("removing orphaned trashinfo", "file", file.TrashInfoPath)
		if err := os.Remove(file.TrashInfoPath); err != nil {
			slog.Error("failed to remove orphaned metadata file", "file", file.TrashInfoPath, "error", err)
			failedDeletions = append(failedDeletions, file.TrashInfoPath)
		}
	}

	if len(failedDeletions) > 0 {
		fmt.Printf("Failed to remove %d files:\n", len(failedDeletions))
		for _, file := range failedDeletions {
			fmt.Println("-", file)
		}
		return fmt.Errorf("some orphaned metadata files could not be removed")
	}

	fmt.Printf("Successfully removed %d orphaned metadata files.\n", len(orphanedFiles))
	return nil
}
