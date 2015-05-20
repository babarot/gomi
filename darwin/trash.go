package darwin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const raw = `
on run argv
  tell application "Finder"
    repeat with f in argv
      move (f as POSIX file) to trash
    end repeat
  end tell
end run
`

var trash = filepath.Join(os.Getenv("HOME"), ".Trash")

func Trash(f string) (trashcan string, err error) {
	bin, err := exec.LookPath("osascript")
	if err != nil {
		err = fmt.Errorf("not yet supported")
		return
	}

	if _, err = os.Stat(trash); err != nil {
		err = fmt.Errorf("not yet supported")
		return
	}

	path, err := filepath.Abs(f)
	if err != nil {
		return
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	_ = name

	dest := filepath.Join(trash, base)
	if _, err = os.Stat(dest); err == nil {
		err = fmt.Errorf("already exists")
		return
	} else {
		trashcan = dest
	}

	params := append([]string{"-e", raw}, path)
	cmd := exec.Command(bin, params...)
	if err = cmd.Run(); err != nil {
		return
	}

	if _, err = os.Stat(trashcan); err != nil {
		trashcan = ""
	}

	return
}
