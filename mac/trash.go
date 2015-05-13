// +build darwin

package mac

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	//"time"
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

func Trash(f string) (save string, err error) {
	bin, err := exec.LookPath("osascript")
	if err != nil {
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
	if _, err = os.Stat(dest); err != nil {
		save = dest
	} else {
		//save = filepath.Join(trash, fmt.Sprintf("%s %s%s", name, time.Now().Format("15.04.05"), ext))
		err = fmt.Errorf("already exists")
		return
	}

	params := append([]string{"-e", raw}, path)
	cmd := exec.Command(bin, params...)
	if err = cmd.Run(); err != nil {
		return
	}

	if _, err = os.Stat(save); err != nil {
		save = ""
	}

	return
}
