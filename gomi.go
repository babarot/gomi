package gomi

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/b4b4r07/gomi/mac"
)

var timer *time.Timer

func Restore(path string) error {
	if err := cleanLog(); err != nil {
		return err
	}

	if line := pecoInterface(); line != "" && !ctx.quicklook {
		_, location, trashcan, err := logLineSplitter(line)
		if err != nil {
			return err
		}

		if path == "" {
			path = location
		}

		if _, err := os.Stat(filepath.Dir(path)); err != nil {
			if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
				return err
			}
		}
		if info, err := os.Stat(path); err == nil {
			if info.IsDir() {
				location = filepath.Join(path, filepath.Base(trashcan))
			} else {
				var answer = "no"
				if runtime.GOOS != "windows" {
					fmt.Printf("WARNING: %s overwrite? (y/N): ", location)
					_, err := fmt.Scanf("%s", &answer)
					if err != nil {
						return err
					}
				}
				if answer != "y" {
					err = fmt.Errorf("%s: already exists", location)
					return err
				}
			}
		}

		if err := os.Rename(trashcan, location); err != nil {
			return err
		}
	}

	return nil
}

func Logging(src, dest string) error {
	src = filepath.Join(src)
	dest = filepath.Join(dest)

	lines, err := readLines(rm_log)
	if err != nil {
		return err
	}

	// Read config.yaml
	c, err := readYaml()
	if err != nil {
		return err
	}

	// Ignore exclude file in the configuration file
	for _, ignore := range c.Ignore {
		if m, _ := filepath.Match(ignore, filepath.Base(src)); m {
			return nil
		}
	}
	lines = append(lines, fmt.Sprintf("%s %s %s",
		time.Now().Format("2006-01-02 15:04:05"),
		src,
		dest,
	))
	if err := writeLines(lines, rm_log); err != nil {
		return err
	}
	return nil
}

func Remove(src string) (dest string, err error) {
	// Check if rm_trash exists
	src = filepath.Join(src)
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
	dest = filepath.Join(rm_trash, time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"))
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
		return
	} else {
		dest = filepath.Join(dest, filepath.Base(src)+"."+time.Now().Format("15_04_05"))
		err = os.Rename(src, dest)
		if err != nil {
			return
		}
	}

	return
}

func System(src string) (dest string, err error) {
	// Check if src exists
	src = filepath.Join(src)
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
		dest, err = mac.Trash(src)
	default:
		err = fmt.Errorf("not yet supported")
	}

	return
}
