package cli

import (
	"fmt"
	"runtime/debug"
	"strings"
)

const (
	appURL = "https://gomi.dev"
)

type Version struct {
	AppName   string
	Version   string
	Revision  string
	BuildDate string
}

func (v Version) Print() string {
	var s strings.Builder
	switch v.Version {
	case "unset", "unknown", "develop":
		if info, ok := debug.ReadBuildInfo(); ok {
			v.Version = info.Main.Version
		}
	}
	fmt.Fprintln(&s, v.AppName+" - a CLI trash manager")
	fmt.Fprintln(&s, appURL)
	fmt.Fprintln(&s, "")
	fmt.Fprintln(&s, "version: "+v.Version)
	fmt.Fprintln(&s, "revision: "+v.Revision)
	fmt.Fprintln(&s, "buildDate: "+v.BuildDate)
	return s.String()
}
