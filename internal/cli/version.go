package cli

import (
	"fmt"
	"runtime/debug"
	"strings"
)

const (
	appURL = "https://github.com/babarot/gomi"
)

type Version struct {
	AppName  string
	Version  string
	Revision string
	Date     string
}

func (v Version) Print() string {
	var s strings.Builder
	switch v.Version {
	case "unset", "unknown", "develop":
		if info, ok := debug.ReadBuildInfo(); ok {
			v.Version = info.Main.Version
		}
	}
	fmt.Fprintln(&s, v.AppName+" - trashcan in CLI")
	fmt.Fprintln(&s, appURL)
	fmt.Fprintln(&s, "")
	fmt.Fprintln(&s, "version: "+v.Version)
	fmt.Fprintln(&s, "revision: "+v.Revision)
	fmt.Fprintln(&s, "buildDate: "+v.Date)
	return s.String()
}
