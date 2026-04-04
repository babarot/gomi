package history

import (
	"log/slog"
	"os"
	"path/filepath"
)

const OldFilename = "inventory.json"

// migrateIfNeeded renames inventory.json to history.json if it exists.
// Before v1.2.2, the history file was called inventory.json.
func migrateIfNeeded(home string) {
	oldPath := filepath.Join(home, OldFilename)
	newPath := filepath.Join(home, Filename)

	if _, err := os.Stat(oldPath); err == nil {
		if err := os.Rename(oldPath, newPath); err != nil {
			slog.Error("failed to migrate history file", "from", oldPath, "to", newPath, "error", err)
		}
	}
}
