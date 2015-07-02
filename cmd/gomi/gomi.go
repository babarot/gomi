package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/b4b4r07/gomi"
)

type Options struct {
	Restore bool `short:"r" long:"restore" description:"Restore removed files from the trash"`
	System  bool `short:"s" long:"system" description:"Use the trash of OSes instead of the trash of gomi"`
}

var opts Options

func main() {
	// Parse arguments
	var version, system, restore bool
	flags := flag.NewFlagSet("gch", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.Usage = func() {
		fmt.Fprint(os.Stderr, "help")
	}

	flags.BoolVar(&version, "version", false, "")
	flags.BoolVar(&restore, "restore", false, "")
	flags.BoolVar(&restore, "r", false, "")
	flags.BoolVar(&system, "system", false, "")
	flags.BoolVar(&system, "s", false, "")

	// Parse all the flags
	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	err := gomi.Init()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gomi: ", err)
		os.Exit(1)
	}

	if restore {
		var location string
		if flags.NArg() > 0 {
			location = flags.Args()[0]
		}
		if err := gomi.Restore(location); err != nil {
			fmt.Fprintln(os.Stderr, "gomi: ", err)
			os.Exit(1)
		}
		os.Exit(1)
	} else if version {
		fmt.Fprintln(os.Stderr, "version")
		os.Exit(0)
	}

	if flags.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "gomi: ", fmt.Errorf("too few arguments"))
		os.Exit(1)
	}

	var location, trashcan string

	for _, arg := range flags.Args() {
		if system {
			trashcan, err = gomi.System(arg)
		} else {
			trashcan, err = gomi.Remove(arg)
		}

		if err != nil {
			fmt.Fprintln(os.Stderr, "gomi: ", err)
			os.Exit(1)
		}

		location, _ = filepath.Abs(arg)
		if err := gomi.Logging(location, trashcan); err != nil {
			fmt.Fprintln(os.Stderr, "gomi: ", err)
			os.Exit(1)
		}
	}
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
