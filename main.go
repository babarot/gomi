package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"os"
	"path/filepath"
)

type Options struct {
	Restore bool `short:"r" long:"restore" description:"Restore removed files from the trash"`
	System  bool `short:"s" long:"system" description:"Use the trash of OSes instead of the trash of gomi"`
}

var opts Options

func main() {
	args, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	// Restore Mode
	if opts.Restore {
		path := ""
		if len(args) != 0 {
			path = args[0]
		}

		if err := restore(path); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Check arguments
	var path string
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "error: gomi: too few arguments\n")
		fmt.Fprintf(os.Stderr, "Try `gomi --help' for more information.\n")
		os.Exit(1)
	}

	// Main
	for _, gomi := range args {
		if opts.System {
			path, err = removeTo(gomi)
		} else {
			path, err = remove(gomi)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		if path == "" {
			fmt.Fprintf(os.Stderr, "no\n")
			os.Exit(1)
		}

		gomi, _ = filepath.Abs(gomi)
		if err := logging(gomi, path); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	}
}
