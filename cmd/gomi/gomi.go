package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/b4b4r07/gomi"
	"github.com/jessevdk/go-flags"
)

type Options struct {
	Restore bool `short:"r" long:"restore" description:"Restore removed files from the trash"`
	System  bool `short:"s" long:"system" description:"Use the trash of OSes instead of the trash of gomi"`
}

var opts Options

func main() {
	// Parse arguments
	args, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	err = gomi.Init()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gomi: ", err)
		os.Exit(1)
	}

	// Restore Mode
	if opts.Restore {
		var location string
		if len(args) > 0 {
			location = args[0]
		}

		if err := gomi.Restore(location); err != nil {
			fmt.Fprintln(os.Stderr, "gomi: ", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Check arguments
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "gomi: ", fmt.Errorf("too few arguments"))
		os.Exit(1)
	}

	var location, trashcan string

	for _, arg := range args {
		if opts.System {
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
