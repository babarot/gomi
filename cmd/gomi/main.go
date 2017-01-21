package main

import (
	"os"
)

const (
	Name    = "gomi"
	Version = "0.1.6"
)

func main() {
	cli := &CLI{outStream: os.Stdout, errStream: os.Stderr}
	os.Exit(cli.Run(os.Args[1:]))
}
