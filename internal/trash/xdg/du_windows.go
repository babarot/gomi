//go:build windows

package xdg

import "github.com/babarot/gomi/internal/utils/fs"

// diskUsage is the size of what is under path: Windows does not tell the
// blocks a file uses, and no other program shares the trash there
func diskUsage(path string) (int64, error) {
	return fs.DirSize(path)
}
