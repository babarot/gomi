package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/jessevdk/go-flags"
)

var rm_trash string = os.Getenv("HOME") + "/.gomi"
var rm_log string = rm_trash + "/log"

type Options struct {
	Restore bool `short:"r" long:"restore" description:"Restore removed files from gomi box"`
	System  bool `short:"s" long:"system" description:"Use system recycle bin"`
}

var opts Options

func checkPath(cmd string) (ret string, err error) {
	ret, err = exec.LookPath(cmd)
	if err != nil {
		err = fmt.Errorf("%s: executable file not found in $PATH", cmd)
		return
	}

	return ret, nil
}
func main() {
	args, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	if opts.Restore {
		path := ""
		if len(args) != 0 {
			path = args[0]
		}

		if err := restore(path); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	var path string
	if len(args) == 0 {
		fmt.Println("too few arguments")
		os.Exit(1)
	}
	for _, gomi := range args {
		if _, err := os.Stat(gomi); err != nil {
			fmt.Fprintf(os.Stderr, "%s: no such file or directory\n", gomi)
			continue
		}
		if opts.System {
			cmd := ""
			if runtime.GOOS == "darwin" {
				if cmd, err = checkPath(cmd); err != nil {
					cmd = "./bin/osx-trash"
				}
			} else {
				fmt.Fprintf(os.Stderr, "Not yet\n")
				os.Exit(1)
			}
			_, err := exec.Command(cmd, gomi).Output()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: error: %v\n", cmd, err)
			}
			path = filepath.Clean(os.Getenv("HOME") + "/.Trash/" + filepath.Base(gomi))
		} else {
			path, err = remove(gomi)
			if err != nil {
				fmt.Fprintf(os.Stderr, "couldn't remove\n")
			}
		}

		gomi, _ = filepath.Abs(gomi)
		if err := logging(gomi, path); err != nil {
			fmt.Fprintf(os.Stderr, "couldn't logging to %s\n", rm_log)
		}
	}
}

func logging(src, dest string) (err error) {
	f, err := os.OpenFile(rm_log, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	for _, ignore := range config() {
		if m, _ := regexp.MatchString(ignore.(string), filepath.Base(src)); m {
			//if filepath.Base(src) == ignore {
			return nil
		}
	}

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

	// Check if src exists
	_, err = os.Stat(src)
	if os.IsNotExist(err) {
		err = fmt.Errorf("%s: no such file or directory", src)
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
