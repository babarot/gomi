package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/b4b4r07/gomi"
)

var re = regexp.MustCompile("^-.*")

const (
	ExitCodeOK    int = 0
	ExitCodeError int = 1 + iota
	ExitCodeRemoveError
	ExitCodeRestoreError
	ExitCodeLoggingError
	ExitCodeBadArgs
)

type CLI struct {
	outStream, errStream io.Writer
}

func (cli *CLI) Run(args []string) int {
	var system, restore bool
	// args := os.Args[1:]

L:
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			fmt.Fprintln(cli.errStream, helpText)
			return ExitCodeError
		case "--version":
			fmt.Fprintf(cli.errStream, "%s v%s\n", Name, Version)
			return ExitCodeOK
		case "-r", "--restore":
			restore = true
			args = args[1:]
		case "-s", "--system":
			system = true
			args = args[1:]
		case "--":
			args = args[1:]
			break L
		default:
			if re.Match([]byte(arg)) {
				fmt.Fprintf(cli.errStream, "gomi: %s: no such option\n", arg)
				return ExitCodeError
			}
		}
	}

	err := gomi.Init()
	if err != nil {
		fmt.Fprintln(cli.errStream, "gomi: ", err)
		return ExitCodeError
	}

	if restore {
		var location string
		if len(args) > 0 {
			location = args[0]
		}
		if err := gomi.Restore(location); err != nil {
			fmt.Fprintln(cli.errStream, "gomi: ", err)
			return ExitCodeRestoreError
		}
		return ExitCodeOK
	}

	if len(args) == 0 {
		fmt.Fprintln(cli.errStream, "gomi: ", fmt.Errorf("too few arguments"))
		return ExitCodeBadArgs
	}

	var location, trashcan string
	for _, arg := range args {
		if _, err := os.Stat(arg); err == nil {
		} else {
			fmt.Fprintf(cli.errStream, "gomi: %s not found\n", arg)
			continue
		}
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
