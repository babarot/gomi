// +build !windows

package gomi

import (
	"os"
	"path/filepath"
)

var (
	rm_trash = filepath.Join(os.Getenv("HOME"), ".gomi")
	rm_log   = filepath.Join(rm_trash, "log")
)
