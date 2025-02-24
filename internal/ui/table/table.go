package table

import (
	"fmt"
	"sort"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
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

	green := color.New(color.FgHiGreen).SprintfFunc()
	white := color.New(color.FgWhite).SprintfFunc()

	// Print header
	fmt.Printf("%s %s %s\n",
		green("%-20s", "Deleted At"),
		green("%-18s", ""),
		green("%-30s", "Path"),
	)

	// Print sorted files
	for _, file := range sortedFiles {
		var middleColumn string
		if opts.ShowRelativeTime {
			middleColumn = "(" + humanize.Time(file.GetDeletedAt()) + ")"
		}

		fmt.Printf("%s %s %s\n",
			white("%-20s", file.GetDeletedAt().Format(timeFormat)),
			white("%-18s", middleColumn),
			white("%-30s", file.GetName()),
		)
	}

	fmt.Println()
}
