package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/jessevdk/go-flags"
	"os/exec"
)

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
		fmt.Fprintf(os.Stderr, "too few arguments\n")
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

func logging(src, dest string) error {
	// Open or create rm_log if rm_log doesn't exist
	f, err := os.OpenFile(rm_log, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	// Ignore exclude file in the configuration file
	for _, ignore := range config() {
		if m, _ := regexp.MatchString(ignore.(string), filepath.Base(src)); m {
			return nil
		}
	}

	// Main
	text := fmt.Sprintf("%s %s %s\n", time.Now().Format("2006-01-02 15:04:05"), src, dest)
	if _, err = f.WriteString(text); err != nil {
		err = fmt.Errorf("couldn't logging to %s", rm_log)
		return err
	}

	return nil
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

func removeTo(src string) (dest string, err error) {
	// Check if src exists
	_, err = os.Stat(src)
	if err != nil {
		err = fmt.Errorf("%s: no such file or directory", src)
		return
	}

	// Main
	switch runtime.GOOS {
	case "windows":
		cmd := "Recycle.exe"
		if cmd, err = checkPath(cmd); err != nil {
			cmd = "./bin/cmdutils/Recycle.exe"
		}
		_, cmderr := exec.Command(cmd, src).Output()
		if cmderr != nil {
			err = fmt.Errorf("error: %s: %v", cmd, cmderr)
			return
		}
		dest = filepath.Clean(`C:\$RECYCLER.BIN\` + filepath.Base(src))
	case "darwin":
		cmd := "osx-trash"
		if cmd, err = checkPath(cmd); err != nil {
			cmd = "./bin/osx-trash"
		}
		_, cmderr := exec.Command(cmd, src).Output()
		if cmderr != nil {
			err = fmt.Errorf("error: %s: %v", cmd, cmderr)
			return
		}
		dest = filepath.Clean(os.Getenv("HOME") + "/.Trash/" + filepath.Base(src))
	default:
		err = fmt.Errorf("not yet supported")
	}

	return
}
