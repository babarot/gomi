// +build windows

package gomi

import (
	"path/filepath"
)

var rm_trash = filepath.Join(`C:\ProgramData`, "gomi")
var rm_log = filepath.Join(rm_trash, "log")
