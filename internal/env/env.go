package env

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

const defaultXDGConfigDirname = ".config"
const gomiConfigDir = "gomi"
const gomiConfigFilename = "config.yaml"

var (
	// TODO: compatible with flag
	GOMI_CONFIG_PATH string

	GOMI_LOG_PATH string
)

func init() {
	if e, found := os.LookupEnv("GOMI_CONFIG_PATH"); !found || e == "" {
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				panic(err)
			}
			configDir = filepath.Join(homeDir, defaultXDGConfigDirname)
		}
		GOMI_CONFIG_PATH = filepath.Join(configDir, gomiConfigDir, gomiConfigFilename)
	}

	if e, found := os.LookupEnv("GOMI_LOG_PATH"); !found || e == "" {
		fp, err := xdg.CacheFile(fmt.Sprintf("%s/log", gomiConfigDir))
		if err != nil {
			fp = fmt.Sprintf("%s.log", gomiConfigDir)
		}
		GOMI_LOG_PATH = fp
	}
}
