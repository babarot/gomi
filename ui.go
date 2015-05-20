package gomi

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/b4b4r07/gomi/darwin"
)

func Restore(path string) error {
	selected, err := Interface()
	if err != nil {
		return err
	}

	for _, f := range selected {
		if path != "" {
			f.location = path
		}

		locationDir := filepath.Dir(f.location)
		if _, err := os.Stat(locationDir); err != nil {
			err = os.MkdirAll(locationDir, 0777)
			if err != nil {
				return fmt.Errorf("cannot create %s", locationDir)
			}
		}

		if info, err := os.Stat(f.location); err == nil {
			if info.IsDir() {
				// `gomi -r .` or `gomi -r ..` is ok
				f.location = filepath.Clean(filepath.Join(f.location, f.name))
			} else {
				var answer = "no"
				if runtime.GOOS != "windows" {
					fmt.Printf("WARNING: %s overwrite? (y/N): ", f.location)
					_, err := fmt.Scanf("%s", &answer)
					if err != nil {
						return err
					}
				}
				if answer != "y" {
					err = fmt.Errorf("%s: already exists", f.location)
					return err
				}
			}
		}
		if err := os.Rename(f.trashcan, f.location); err != nil {
			return fmt.Errorf("cannot restore `%s' to `%s'", f.trashcan, f.location)
		}
	}

	return nil
}

func Logging(location, trashcan string) error {
	location = filepath.Join(location)
	trashcan = filepath.Join(trashcan)

	lines, err := readLines(rm_log)
	if err != nil {
		return err
	}

	config := &Config{}
	err = config.ReadConfig()
	if err != nil {
		return err
	}
	for _, ignore := range config.Ignore {
		if m, _ := filepath.Match(ignore, filepath.Base(location)); m {
			return nil
		}
	}

	lines = append(lines, fmt.Sprintf("%s %s %s",
		time.Now().Format("2006-01-02 15:04:05"),
		location,
		trashcan,
	))
	if err := writeLines(lines, rm_log); err != nil {
		return err
	}
	return nil
}

func Remove(path string) (trashcan string, err error) {
	if info, sterr := os.Stat(rm_trash); sterr == nil {
		if info.IsDir() {
			// ok
		} else {
			err = fmt.Errorf("cannot create gomi directory")
			return
		}
	} else {
		err = os.MkdirAll(rm_trash, 0777)
		if err != nil {
			return
		}
	}

	if _, sterr := os.Stat(path); sterr == nil {
		path, _ = filepath.Abs(path)
		path = filepath.Join(path)
	} else {
		err = fmt.Errorf("%s: no such file or directory", path)
		return
	}

	trashcan = filepath.Join(
		rm_trash,
		time.Now().Format("2006"),
		time.Now().Format("01"),
		time.Now().Format("02"),
	)

	err = os.MkdirAll(trashcan, 0777)
	if err != nil {
		return
	}

	if filepath.Dir(path) == rm_trash {
		err = os.Remove(path)
		if err != nil {
			return
		}
		return
	} else {
		trashcan = filepath.Join(trashcan, filepath.Base(path)+"."+time.Now().Format("15_04_05"))
		err = os.Rename(path, trashcan)
		if err != nil {
			return
		}
	}

	return
}

func System(path string) (trashcan string, err error) {
	if _, sterr := os.Stat(path); sterr == nil {
		path, _ = filepath.Abs(path)
		path = filepath.Join(path)
	} else {
		err = fmt.Errorf("%s: no such file or directory", path)
		return
	}

	// Main
	switch runtime.GOOS {
	case "windows":
	case "darwin":
		trashcan, err = darwin.Trash(path)
	default:
		err = fmt.Errorf("not yet supported")
	}

	return
}

func (ls Lines) Dups() bool {
	if _, err := os.Stat(ls.location); err != nil {
		return false
	}
	return true
}
