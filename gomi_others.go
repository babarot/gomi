// +build !windows

package gomi

import "os"

var rm_trash string = os.Getenv("HOME") + "/.gomi"
var rm_log string = rm_trash + "/log"
