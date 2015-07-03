package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/b4b4r07/gomi"
)

const (
	ExitCodeOK    int = 0
	ExitCodeError int = 1 + iota
	ExitCodeFlagParseError
	ExitCodeRemoveError
	ExitCodeRestoreError
	ExitCodeLoggingError
	ExitCodeBadArgs
)

type CLI struct {
	outStream, errStream io.Writer
}

func (cli *CLI) Run(args []string) int {
	var version, system, restore bool

	flags := flag.NewFlagSet("gomi", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.Usage = func() {
		fmt.Fprint(os.Stderr, helpText)
	}

	flags.BoolVar(&version, "version", false, "")
	flags.BoolVar(&restore, "restore", false, "")
	flags.BoolVar(&restore, "r", false, "")
	flags.BoolVar(&system, "system", false, "")
	flags.BoolVar(&system, "s", false, "")

	// Parse all the flags
	if err := flags.Parse(os.Args[1:]); err != nil {
		return ExitCodeFlagParseError
	}

	err := gomi.Init()
	if err != nil {
		fmt.Fprintln(cli.errStream, "gomi: ", err)
		return ExitCodeError
	}

	if restore {
		var location string
		if flags.NArg() > 0 {
			location = flags.Args()[0]
		}
		if err := gomi.Restore(location); err != nil {
			fmt.Fprintln(cli.errStream, "gomi: ", err)
			return ExitCodeRestoreError
		}
		return ExitCodeOK
	} else if version {
		fmt.Fprintf(cli.errStream, "%s v%s\n", Name, Version)
		return ExitCodeOK
	}

	if flags.NArg() < 1 {
		fmt.Fprintln(cli.errStream, "gomi: ", fmt.Errorf("too few arguments"))
		return ExitCodeBadArgs
	}

	var location, trashcan string

	for _, arg := range flags.Args() {
		if system {
			trashcan, err = gomi.System(arg)
		} else {
			trashcan, err = gomi.Remove(arg)
		}

		if err != nil {
			fmt.Fprintln(cli.errStream, "gomi: ", err)
			return ExitCodeRemoveError
		}

		location, _ = filepath.Abs(arg)
		if err := gomi.Logging(location, trashcan); err != nil {
			fmt.Fprintln(cli.errStream, "gomi: ", err)
			return ExitCodeLoggingError
		}
	}

	return ExitCodeOK
}

var helpText = `Usage: gomi [options] [path]
  gomi is a simple trash tool, easy to restore and preview

Options:
  --restore, -r      Restore the files that removed by gomi
                     with peco-like UI.
  --system, -s       Remove with system trash can
                     It supports only Macintosh.
  --version          Print the version of this application
`
