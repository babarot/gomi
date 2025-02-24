package table

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
)

const (
	timeFormat = "2006-01-02 15:04:05"
)

type FileEntry interface {
	GetName() string
	GetDeletedAt() time.Time
}

type SortOrder int

const (
	SortDesc SortOrder = iota
	SortAsc
)

type PrintOptions struct {
	ShowRelativeTime bool
	Order            SortOrder
}

func PrintFiles[T FileEntry](files []T, opts PrintOptions) {
	// Make a copy to avoid modifying the original slice
	sortedFiles := make([]T, len(files))
	copy(sortedFiles, files)

	// Sort the files
	sort.Slice(sortedFiles, func(i, j int) bool {
		switch opts.Order {
		case SortAsc:
			return sortedFiles[i].GetDeletedAt().Before(sortedFiles[j].GetDeletedAt())
		default: // SortDesc
			return sortedFiles[i].GetDeletedAt().After(sortedFiles[j].GetDeletedAt())
		}
	})

	// Initialize table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Deleted At", "Path"})

	// Configure table appearance
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetHeaderLine(false)
	table.SetRowLine(false)
	table.SetAutoWrapText(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	// Set column colors
	green := tablewriter.Colors{tablewriter.Bold, 92} // bright green (FgHiGreen)
	white := tablewriter.Colors{tablewriter.Bold, 37} // white (FgWhite)
	table.SetHeaderColor(green, green)
	table.SetColumnColor(white, white)

	// Add rows
	for _, file := range sortedFiles {
		deletedAt := file.GetDeletedAt().Format(timeFormat)
		if opts.ShowRelativeTime {
			deletedAt += fmt.Sprintf("  (%s)", humanize.Time(file.GetDeletedAt()))
		}

		row := []string{
			deletedAt,
			file.GetName(),
		}
		table.Append(row)
	}

	// Add padding between columns
	table.SetColumnSeparator(strings.Repeat(" ", 2))
	table.Render()
}
