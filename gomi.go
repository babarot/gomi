package main

import (
	//"bufio"
	//"errors"
	"fmt"
	"github.com/jessevdk/go-flags"
	"os"
	"path/filepath"
	"time"
)

var rm_trash string = os.Getenv("HOME") + "/.gomi"
var rm_log string = rm_trash + "/log"

type Options struct {
	Restore bool `short:"r" long:"restore" description:"Restore removed files from gomi box"`
}

var opts Options

func main() {
	args, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	if opts.Restore {
		if err := restore(); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(args) == 0 {
		fmt.Println("too few arguments")
		os.Exit(1)
	}
	for _, gomi := range args {
		if path, err := remove(gomi); err != nil {
			fmt.Println(err)
		} else {
			gomi, _ = filepath.Abs(gomi)
			if err := logging(gomi, path); err != nil {
				fmt.Println(err)
			}
		}
	}
}

func logging(src, dest string) (err error) {
	f, err := os.OpenFile(rm_log, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}

	defer f.Close()

	text := fmt.Sprintf("%s %s %s\n", time.Now().Format("2006-01-02 15:04:05"), src, dest)
	if _, err = f.WriteString(text); err != nil {
		return
	}

	return
}

func remove(src string) (dest string, err error) {
	// Check if rm_trash exists
	_, err = os.Stat(rm_trash)
	if os.IsNotExist(err) {
		err = os.MkdirAll(rm_trash, 0777)
		if err != nil {
			return
		}
	}
	//if !path.IsDir() {
	//	err = fmt.Errorf("%s: fatal error", rm_trash)
	//	return
	//}

	// Check if src exists
	_, err = os.Stat(src)
	if os.IsNotExist(err) {
		err = fmt.Errorf("%s: No such file or directory", src)
		return
	}

	// Make directory for trash
	dest = rm_trash + "/" + time.Now().Format("2006/01/02")
	err = os.MkdirAll(dest, 0777)
	if err != nil {
		return
	}

	// Remove
	if filepath.Dir(src) == rm_trash {
		err = os.Remove(src)
		if err != nil {
			return
		}
		err = fmt.Errorf("os.Remove %s, successfully", src)
		return
	} else {
		dest = dest + "/" + filepath.Base(src) + "." + time.Now().Format("15_04_05")
		err = os.Rename(src, dest)
		if err != nil {
			return
		}
	}

	return
}
