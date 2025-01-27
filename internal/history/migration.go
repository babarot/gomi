package history

import (
	"log"
	"os"
	"path/filepath"
)

const oldHistoryFile = "inventory.json"

// init is called when the application starts, to handle migration from inventory.json to history.json
func init() {
	oldPath := filepath.Join(gomiPath, oldHistoryFile)
	newPath := filepath.Join(gomiPath, historyFile)

	// Check if inventory.json exists and rename it to history.json
	if _, err := os.Stat(oldPath); err == nil {
		// File exists, so rename it
		err := os.Rename(oldPath, newPath)
		if err != nil {
			log.Fatalf("Failed to rename file %s to %s: %v", oldPath, newPath, err)
		}
	}
}
