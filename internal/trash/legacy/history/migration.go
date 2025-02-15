package history

import (
	"log"
	"os"
	"path/filepath"
)

const OldFilename = "inventory.json"

// init is called when the application starts, to handle migration from inventory.json to history.json
func init() {
	// Before v1.2.2, the trash home was fixed, so these are hardcoded in the migration script.
	fixedHome := filepath.Join(os.Getenv("HOME"), ".gomi")

	oldPath := filepath.Join(fixedHome, OldFilename)
	newPath := filepath.Join(fixedHome, Filename)

	// Check if inventory.json exists and rename it to history.json
	if _, err := os.Stat(oldPath); err == nil {
		// File exists, so rename it
		err := os.Rename(oldPath, newPath)
		if err != nil {
			log.Fatalf("Failed to rename file %s to %s: %v", oldPath, newPath, err)
		}
	}
}
