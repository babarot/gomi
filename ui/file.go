package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/babarot/gomi/inventory"
	"github.com/dustin/go-humanize"
)

type File struct {
	inventory.File
}

func (f File) isSelected() bool {
	return selectionManager.Contains(f)
}

func (f File) Description() string {
	_, err := os.Stat(f.File.To)
	if os.IsNotExist(err) {
		return "(already might have been deleted)"
	}

	return fmt.Sprintf("%s %s %s",
		humanize.Time(f.File.Timestamp),
		bullet,
		filepath.Dir(f.File.From),
	)
}

func (f File) Title() string {
	fi, err := os.Stat(f.File.To)
	if err != nil {
		return f.File.Name + "?"
	}
	if fi.IsDir() {
		return f.File.Name + "/"
	}
	return f.File.Name
}

func (f File) FilterValue() string {
	return f.File.Name
}
