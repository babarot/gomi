package xdg

import (
	"bufio"
	"fmt"
	"log/slog"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/babarot/gomi/internal/trash"
)

// dirSizesFile is the cache of the sizes of the directories in a trash,
// which the trash spec puts next to files/ and info/
const dirSizesFile = "directorysizes"

// dirSize is a line of the directorysizes file: the disk usage of a
// trashed directory, valid while its .trashinfo has the modification time
// mtime
type dirSize struct {
	size  int64
	mtime int64
}

// dirSizes is the directorysizes file of one trash, by the name of the
// directory in files/
type dirSizes map[string]dirSize

// readDirSizes reads the directorysizes file of the trash at root. A file
// that is missing or cannot be read is an empty cache, and lines that
// cannot be parsed are left out.
func readDirSizes(root string) dirSizes {
	sizes := dirSizes{}
	f, err := os.Open(filepath.Join(root, dirSizesFile))
	if err != nil {
		return sizes
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.SplitN(scanner.Text(), " ", 3)
		if len(fields) != 3 {
			continue
		}
		size, err1 := strconv.ParseInt(fields[0], 10, 64)
		mtime, err2 := strconv.ParseInt(fields[1], 10, 64)
		// path style: a + stays a +, unlike in a query
		name, err3 := url.PathUnescape(fields[2])
		if err1 != nil || err2 != nil || err3 != nil || name == "" || strings.Contains(name, "/") {
			continue
		}
		sizes[name] = dirSize{size: size, mtime: mtime}
	}
	return sizes
}

// writeDirSizes replaces the directorysizes file of the trash at root, by
// writing a temporary file next to it and renaming it over, as the spec
// requires, so that another program writing it at the same time cannot
// leave it half written
func writeDirSizes(root string, sizes dirSizes) error {
	tmp, err := os.CreateTemp(root, dirSizesFile+".*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	w := bufio.NewWriter(tmp)
	for _, name := range slices.Sorted(maps.Keys(sizes)) {
		d := sizes[name]
		_, _ = fmt.Fprintf(w, "%d %d %s\n", d.size, d.mtime, escapeDirName(name))
	}
	if err := w.Flush(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(root, dirSizesFile))
}

// escapeDirName percent-encodes what would break a line: the spec asks
// for the newline and % only; the space is escaped too, as it separates
// the fields
func escapeDirName(name string) string {
	var b strings.Builder
	for i := 0; i < len(name); i++ {
		switch c := name[i]; c {
		case '%', '\n', '\r', ' ':
			fmt.Fprintf(&b, "%%%02X", c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// sizeCache is what one listing of a trash knows of its directorysizes:
// what the file held, and the directories listed, which the size filter
// may measure
type sizeCache struct {
	root   string
	sizes  dirSizes
	names  map[string]bool
	listed map[string]listedDir
}

// listedDir is a directory in files/ and the modification time of its
// .trashinfo
type listedDir struct {
	file  *trash.File
	mtime int64
}

func newSizeCache(root string) *sizeCache {
	return &sizeCache{
		root:   root,
		sizes:  readDirSizes(root),
		names:  map[string]bool{},
		listed: map[string]listedDir{},
	}
}

// track gives the directory file, named name in files/, the size the size
// filter judges it by: the cached one while its .trashinfo is unchanged,
// else a measure of its disk usage
func (c *sizeCache) track(name string, file *trash.File, infoPath string) {
	fi, err := os.Stat(infoPath)
	if err != nil {
		return
	}
	mtime := fi.ModTime().Unix()
	if d, ok := c.sizes[name]; ok && d.mtime == mtime {
		file.FilterSize, file.FilterSizeKnown = d.size, true
	} else {
		path := file.TrashPath
		file.Measure = func() (int64, error) { return diskUsage(path) }
	}
	c.listed[name] = listedDir{file: file, mtime: mtime}
}

// save writes back what the size filter measured and drops the lines of
// directories no longer in files/; it writes only when something changed.
// The lines of the directories the filters dropped before measuring them
// stay, as they do no harm.
func (c *sizeCache) save() {
	changed := false
	for name := range c.sizes {
		if !c.names[name] {
			delete(c.sizes, name)
			changed = true
		}
	}
	for name, l := range c.listed {
		if !l.file.FilterSizeKnown {
			continue
		}
		d := dirSize{size: l.file.FilterSize, mtime: l.mtime}
		if c.sizes[name] != d {
			c.sizes[name] = d
			changed = true
		}
	}
	if !changed {
		return
	}
	if err := writeDirSizes(c.root, c.sizes); err != nil {
		slog.Debug("cannot write directorysizes", "root", c.root, "error", err)
	}
}
