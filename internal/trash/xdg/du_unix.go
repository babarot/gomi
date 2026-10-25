//go:build !windows

package xdg

import (
	"io/fs"
	"path/filepath"
	"syscall"
)

// diskUsage is the disk space path and what is under it use, in bytes, as
// du -B1 counts it and the trash spec asks for the directorysizes file: a
// file with several names counts once, and a symlink is not followed
func diskUsage(path string) (int64, error) {
	type fileID struct{ dev, ino uint64 }
	seen := map[fileID]bool{}
	var total int64
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			total += info.Size()
			return nil
		}
		id := fileID{uint64(st.Dev), st.Ino} //nolint:unconvert // Dev is int32 on darwin
		if seen[id] {
			return nil
		}
		seen[id] = true
		total += st.Blocks * 512
		return nil
	})
	return total, err
}
