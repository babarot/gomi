package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/mattn/go-isatty"

	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/trash/xdg"
	"github.com/babarot/gomi/internal/ui/table"
)

// Output formats of --dry-run
const (
	dryRunAuto  = "auto"
	dryRunTable = "table"
	dryRunText  = "text"
	dryRunJSON  = "json"
)

// dryRunFormat resolves the format of --dry-run output.
// "auto" prints a table for a terminal and one path per line otherwise,
// so that piping the output to other commands just works.
func dryRunFormat(format string, isTerminal bool) string {
	if format != dryRunAuto {
		return format
	}
	if isTerminal {
		return dryRunTable
	}
	return dryRunText
}

// stdoutIsTerminal reports whether stdout is a terminal
func stdoutIsTerminal() bool {
	fd := os.Stdout.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// dryRunFile is a trash file in the JSON output of --dry-run
type dryRunFile struct {
	Name         string    `json:"name"`
	OriginalPath string    `json:"original_path"`
	TrashPath    string    `json:"trash_path"`
	DeletedAt    time.Time `json:"deleted_at"`
	IsDir        bool      `json:"is_dir"`
}

// dryRunOrphan is an orphaned metadata file in the JSON output of --dry-run
type dryRunOrphan struct {
	OriginalPath  string    `json:"original_path"`
	TrashInfoPath string    `json:"trashinfo_path"`
	DeletedAt     time.Time `json:"deleted_at"`
}

// printDryRunFiles prints the trash files that would be removed in the
// text or JSON format, newest first as the table shows them
func printDryRunFiles(w io.Writer, files []*trash.File, format string) error {
	files = newestFirst(files)
	switch format {
	case dryRunJSON:
		out := make([]dryRunFile, len(files))
		for i, f := range files {
			out[i] = dryRunFile{
				Name:         f.Name,
				OriginalPath: f.OriginalPath,
				TrashPath:    f.TrashPath,
				DeletedAt:    f.DeletedAt,
				IsDir:        f.IsDir,
			}
		}
		return writeJSON(w, out)
	default:
		for _, f := range files {
			if _, err := fmt.Fprintln(w, f.TrashPath); err != nil {
				return err
			}
		}
		return nil
	}
}

// printDryRunOrphans prints the orphaned metadata files that would be
// removed in the text or JSON format, newest first as the table shows them
func printDryRunOrphans(w io.Writer, files []xdg.OrphanedFile, format string) error {
	files = newestFirst(files)
	switch format {
	case dryRunJSON:
		out := make([]dryRunOrphan, len(files))
		for i, f := range files {
			out[i] = dryRunOrphan{
				OriginalPath:  f.OriginalPath,
				TrashInfoPath: f.TrashInfoPath,
				DeletedAt:     f.DeletedAt,
			}
		}
		return writeJSON(w, out)
	default:
		for _, f := range files {
			if _, err := fmt.Fprintln(w, f.TrashInfoPath); err != nil {
				return err
			}
		}
		return nil
	}
}

// newestFirst returns a copy of files sorted by deletion time, newest first
func newestFirst[T table.FileEntry](files []T) []T {
	sorted := make([]T, len(files))
	copy(sorted, files)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].GetDeletedAt().After(sorted[j].GetDeletedAt())
	})
	return sorted
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// dryRunTimeRange shows the trash files that --prune would remove by age
func (c *CLI) dryRunTimeRange(files []*trash.File, newestAge, oldestAge time.Duration, isSingleDuration bool) error {
	format := dryRunFormat(c.option.Meta.DryRun, stdoutIsTerminal())
	if format != dryRunTable {
		return printDryRunFiles(os.Stdout, files, format)
	}

	if len(files) == 0 {
		fmt.Println("No matching files found.")
		return nil
	}
	table.PrintFiles(files, table.PrintOptions{
		ShowRelativeTime: true,
		Order:            table.SortDesc,
	})
	fmt.Println()
	printDeletionSummary(files, newestAge, oldestAge, isSingleDuration)
	fmt.Println("Dry run: no files were removed.")
	return nil
}

// dryRunOrphans shows the orphaned metadata files that --prune=orphans would remove
func (c *CLI) dryRunOrphans(files []xdg.OrphanedFile) error {
	format := dryRunFormat(c.option.Meta.DryRun, stdoutIsTerminal())
	if format != dryRunTable {
		return printDryRunOrphans(os.Stdout, files, format)
	}

	if len(files) == 0 {
		fmt.Println("No orphaned metadata files found.")
		return nil
	}
	table.PrintFiles(files, table.PrintOptions{
		ShowRelativeTime: false,
		Order:            table.SortDesc,
	})
	fmt.Println()
	fmt.Printf("Found %d orphaned metadata files.\n", len(files))
	fmt.Println("Dry run: no files were removed.")
	return nil
}
