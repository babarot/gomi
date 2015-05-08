// +build !windows

package main

import "os"

var rm_trash string = os.Getenv("HOME") + "/.gomi"
var rm_log string = rm_trash + "/log"
