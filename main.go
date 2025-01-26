package main

import (
	"fmt"
	"os"

	"github.com/babarot/gomi/internal/cli"
)

const appName = "gomi"

var (
	version  = "unset"
	revision = "unset"
	date     = "unknown"
)

func main() {
	err := cli.Run(cli.Version{
		AppName:  appName,
		Version:  version,
		Revision: revision,
		Date:     date,
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", appName, err)
		os.Exit(1)
	}
}
