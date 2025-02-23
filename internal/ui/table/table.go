package table

import (
	"fmt"
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

func PrintFiles[T FileEntry](files []T, showRelativeTime bool) {
	green := color.New(color.FgHiGreen).SprintfFunc()
	white := color.New(color.FgWhite).SprintfFunc()

	fmt.Printf("%s %s %s\n",
		green("%-20s", "Deleted At"),
		green("%-18s", ""),
		green("%-30s", "Path"),
	)

	for _, file := range files {
		var middleColumn string
		if showRelativeTime {
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
