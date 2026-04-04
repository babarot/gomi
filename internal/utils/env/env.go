package env

import (
	"os"
	"path/filepath"
	"sync"
)

const defaultXDGDataDirname = ".local/share"

var (
	// GOMI_LOG_PATH is the path to the log file, following XDG base directory spec.
	GOMI_LOG_PATH string

	once sync.Once
)

// Init initializes environment paths. It is safe to call multiple times;
// only the first call has effect. This replaces the previous init() function
// to avoid implicit side effects on package import.
func Init() {
	once.Do(func() {
		// https://github.com/charmbracelet/log/issues/35
		os.Setenv("CLICOLOR_FORCE", "1")

		// Follow https://specifications.freedesktop.org/basedir-spec/latest/
		if e := os.Getenv("GOMI_LOG_PATH"); e == "" {
			dataDir := os.Getenv("XDG_DATA_HOME")
			if dataDir == "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					panic(err)
				}
				dataDir = filepath.Join(homeDir, defaultXDGDataDirname)
			}
			GOMI_LOG_PATH = filepath.Join(dataDir, "gomi", "debug.log")
		} else {
			GOMI_LOG_PATH = e
		}
	})
}
