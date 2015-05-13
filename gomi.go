package gomi

import (
	"bufio"
	"fmt"
	"github.com/b4b4r07/gomi/mac"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"time"
)

var timer *time.Timer

func cleanLog() error {
	var array []string
	for _, line := range fileToArray(rm_log) {
		s := logLineSplitter(filepath.Join(line))
		if len(s) < 2 {
			continue
		}
		if _, err := os.Stat(s[2]); err == nil {
			array = append(array, line)
		}
	}

	if err := func(lines []string, path string) error {
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()

		w := bufio.NewWriter(file)
		for _, line := range lines {
			fmt.Fprintln(w, line)
		}
		return w.Flush()
	}(array, rm_log); err != nil {
		return err
	}

	return nil
}

func Restore(path string) error {
	var answer string

	if err := cleanLog(); err != nil {
		return err
	}

	if d := pecoInterface(); d != "" && !ctx.quicklook {
		e := logLineSplitter(filepath.Join(d))
		src := e[2]
		dest := e[1]

		// gomi -r arg
		if path != "" {
			// case:
			// given `gomi -r dir/file` as arguments
			// --> if dir dose not exist
			if _, err := os.Stat(filepath.Dir(path)); err != nil {
				if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
					return err
				}
			}

			dest = path
			if info, err := os.Stat(path); err == nil {
				if info.IsDir() {
					dest = path + "/" + filepath.Base(src)
				}
			}
		}

		if _, err := os.Stat(dest); err == nil {
			fmt.Printf("WARNING: %s overwrite? (y/N): ", dest)
			_, err := fmt.Scanln(&answer)
			if err != nil {
				return err
			}
			if answer != "y" {
				err = fmt.Errorf("%s: already exists", dest)
				return err
			}
		}

		if err := os.Rename(src, dest); err != nil {
			return err
		}
		if err := cleanLog(); err != nil {
			return err
		}
	}

	return nil
}

func Logging(src, dest string) error {
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

func Remove(src string) (dest string, err error) {
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

func System(src string) (dest string, err error) {
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
		//cmd := "osx-trash"
		//if cmd, err = checkPath(cmd); err != nil {
		//	cmd = "./bin/osx-trash"
		//}
		//_, cmderr := exec.Command(cmd, src).Output()
		//if cmderr != nil {
		//	err = fmt.Errorf("error: %s: %v", cmd, cmderr)
		//	return
		//}
		//dest = filepath.Clean(os.Getenv("HOME") + "/.Trash/" + filepath.Base(src))
		dest, err = mac.Trash(src)
	default:
		err = fmt.Errorf("not yet supported")
	}

	return
}
