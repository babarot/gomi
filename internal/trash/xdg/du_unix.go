//go:build !windows

package xdg

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

// dirReaders is how many directories diskUsage reads at once, across all
// the walks running at once. Most of a walk is spent waiting on the file
// system, and the waits of directories read at once overlap; the limit is
// shared so that the walks of several items do not multiply it.
var dirReaders = make(chan struct{}, 16)

// diskUsage is the disk space path and what is under it use, in bytes, as
// du -B1 counts it and the trash spec asks for the directorysizes file: a
// file with several names counts once, and a symlink is not followed. The
// subdirectories are walked at once, and the first error met is returned.
func diskUsage(path string) (int64, error) {
	type fileID struct{ dev, ino uint64 }
	var (
		mu    sync.Mutex
		seen  = map[fileID]bool{}
		total int64
		first error
		wg    sync.WaitGroup
	)
	add := func(info fs.FileInfo) {
		mu.Lock()
		defer mu.Unlock()
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			total += info.Size()
			return
		}
		id := fileID{uint64(st.Dev), st.Ino} //nolint:unconvert // Dev is int32 on darwin
		if seen[id] {
			return
		}
		seen[id] = true
		total += st.Blocks * 512
	}
	fail := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		if first == nil {
			first = err
		}
	}
	failed := func() bool {
		mu.Lock()
		defer mu.Unlock()
		return first != nil
	}

	// walk counts the entries of dir and walks its subdirectories, each in
	// a goroutine of its own. A slot of dirReaders is held only while dir
	// is read, never while its subdirectories are waited for, so walks
	// deeper or wider than the limit do not wait on each other.
	var walk func(dir string)
	walk = func(dir string) {
		defer wg.Done()
		if failed() {
			return
		}
		dirReaders <- struct{}{}
		subdirs, err := readDirUsage(dir, add)
		<-dirReaders
		if err != nil {
			fail(err)
			return
		}
		for _, sub := range subdirs {
			wg.Add(1)
			go walk(sub)
		}
	}

	info, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}
	add(info)
	if info.IsDir() {
		wg.Add(1)
		walk(path)
		wg.Wait()
	}
	if first != nil {
		return 0, first
	}
	return total, nil
}

// readDirUsage counts with add each entry of dir, as lstat reports it, and
// returns the subdirectories to walk; a symlink to a directory is not one
func readDirUsage(dir string, add func(fs.FileInfo)) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var subdirs []string
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			return nil, err
		}
		add(info)
		if e.IsDir() {
			subdirs = append(subdirs, filepath.Join(dir, e.Name()))
		}
	}
	return subdirs, nil
}
