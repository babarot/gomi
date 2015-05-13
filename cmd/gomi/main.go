package main

import (
	"fmt"
	"github.com/b4b4r07/gomi"
	//"github.com/b4b4r07/gomi/mac"
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
	// Parse arguments
	args, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	// Restore Mode
	if opts.Restore {
		var path string
		if len(args) > 0 {
			path = args[0]
		}

		if err := gomi.Restore(path); err != nil {
			fmt.Fprintln(os.Stderr, "gomi: ", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Check arguments
	if len(args) == 0 {
		err = fmt.Errorf("too few arguments")
		fmt.Fprintln(os.Stderr, "gomi: ", err)
		os.Exit(1)
	}

	// Main
	var save string
	for _, arg := range args {
		if opts.System {
			//save, err = mac.Trash(arg)
			save, err = gomi.System(arg)
		} else {
			save, err = gomi.Remove(arg)
		}

		if err != nil {
			fmt.Fprintln(os.Stderr, "gomi: ", err)
			os.Exit(1)
		}

		arg, _ = filepath.Abs(arg)
		if err := gomi.Logging(arg, save); err != nil {
			fmt.Fprintln(os.Stderr, "gomi: ", err)
			os.Exit(1)
		}
	}
}
