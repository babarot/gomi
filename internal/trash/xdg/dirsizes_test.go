package xdg

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
)

func TestDirSizes_RoundTrip(t *testing.T) {
	root := t.TempDir()
	want := dirSizes{
		"plain":    {size: 4096, mtime: 1700000000},
		"a b":      {size: 1, mtime: 2},
		"a+b":      {size: 3, mtime: 4},
		"100%":     {size: 5, mtime: 6},
		"two\nrow": {size: 7, mtime: 8},
		"日本語":      {size: 9, mtime: 10},
	}
	if err := writeDirSizes(root, want); err != nil {
		t.Fatal(err)
	}
	got := readDirSizes(root)
	if len(got) != len(want) {
		t.Fatalf("read %d lines, want %d: %v", len(got), len(want), got)
	}
	for name, d := range want {
		if got[name] != d {
			t.Errorf("%q = %v, want %v", name, got[name], d)
		}
	}
	// no temporary file is left behind
	entries, _ := os.ReadDir(root)
	if len(entries) != 1 {
		t.Errorf("files in the trash root: %v, want only %s", entries, dirSizesFile)
	}
}

// Names other programs wrote, escaped more widely, read back; a + is not
// a space, and bad lines are left out
func TestReadDirSizes_OtherWriters(t *testing.T) {
	root := t.TempDir()
	content := strings.Join([]string{
		"16384 15803468 Documents",
		"8192 15803582 Another%20Folder",
		"10 20 a+b",
		"10 20 %E6%97%A5",
		"not a line",
		"1 2 bad%2Fname",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(root, dirSizesFile), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got := readDirSizes(root)
	for _, name := range []string{"Documents", "Another Folder", "a+b", "日"} {
		if _, ok := got[name]; !ok {
			t.Errorf("%q is missing from %v", name, got)
		}
	}
	if len(got) != 4 {
		t.Errorf("read %d lines, want 4: %v", len(got), got)
	}
}

// A file with two names counts once, as du counts it
func TestDiskUsage_HardLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("disk usage is the size of the contents on Windows")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a"), make([]byte, 64*1024), 0o600); err != nil {
		t.Fatal(err)
	}
	one, err := diskUsage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(dir, "a"), filepath.Join(dir, "b")); err != nil {
		t.Fatal(err)
	}
	two, err := diskUsage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if one < 64*1024 || two != one {
		t.Errorf("usage %d with one name, %d with two; want at least 64K and the same", one, two)
	}
}

// sizeTrash is an XDG trash in a temporary home whose list a size filter
// judges
type sizeTrash struct {
	t    *testing.T
	root string
	s    trash.Storage
}

func newSizeTrash(t *testing.T, history config.History) *sizeTrash {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("XDG trash is not used on Windows")
	}
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)
	s, err := NewStorage(trash.Config{
		Strategy:       trash.StrategyXDG,
		ForceHomeTrash: true,
		History:        history,
		RunID:          testRunID,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &sizeTrash{t: t, root: filepath.Join(dataDir, "Trash"), s: s}
}

// dir puts a directory of a file of n bytes in the trash, deleted ago
// before now
func (st *sizeTrash) dir(name string, n int, ago time.Duration) {
	st.t.Helper()
	path := filepath.Join(st.root, "files", name)
	if err := os.MkdirAll(path, 0o700); err != nil {
		st.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "data"), make([]byte, n), 0o600); err != nil {
		st.t.Fatal(err)
	}
	info := fmt.Sprintf("[Trash Info]\nPath=/home/user/%s\nDeletionDate=%s\n",
		name, time.Now().Add(-ago).Format("2006-01-02T15:04:05"))
	if err := os.WriteFile(filepath.Join(st.root, "info", name+".trashinfo"), []byte(info), 0o600); err != nil {
		st.t.Fatal(err)
	}
}

func (st *sizeTrash) list() map[string]bool {
	st.t.Helper()
	files, err := st.s.List()
	if err != nil {
		st.t.Fatal(err)
	}
	names := map[string]bool{}
	for _, f := range files {
		names[f.Name] = true
	}
	return names
}

func (st *sizeTrash) infoMtime(name string) int64 {
	st.t.Helper()
	fi, err := os.Stat(filepath.Join(st.root, "info", name+".trashinfo"))
	if err != nil {
		st.t.Fatal(err)
	}
	return fi.ModTime().Unix()
}

func TestList_SizeFilterOnDirectories(t *testing.T) {
	st := newSizeTrash(t, config.History{
		Include: config.IncludeConfig{Period: 30},
		Exclude: config.ExcludeConfig{Size: config.SizeConfig{Max: "100KB"}},
	})
	st.dir("small", 1024, time.Hour)
	st.dir("big", 512*1024, time.Hour)
	st.dir("old", 512*1024, 60*24*time.Hour)

	// the size filter now looks into directories
	if got := st.list(); !got["small"] || got["big"] || got["old"] {
		t.Fatalf("listed %v, want small only", got)
	}

	// the directories measured are cached, the one dropped for its size
	// too; the one the period dropped was not measured
	sizes := readDirSizes(st.root)
	if d, ok := sizes["big"]; !ok || d.size < 512*1024 || d.mtime != st.infoMtime("big") {
		t.Errorf("big cached as %v (ok %v), want at least 512K at its .trashinfo's mtime", d, ok)
	}
	if _, ok := sizes["small"]; !ok {
		t.Error("small is not cached")
	}
	if _, ok := sizes["old"]; ok {
		t.Error("old was measured, although the period dropped it")
	}

	// a cached size is used as it is: a size of 0 for big keeps it
	sizes["big"] = dirSize{size: 0, mtime: st.infoMtime("big")}
	// and a line whose mtime is off is measured again
	sizes["small"] = dirSize{size: 1 << 40, mtime: 1}
	// a line for what is no longer in files/ goes
	sizes["gone"] = dirSize{size: 1, mtime: 1}
	if err := writeDirSizes(st.root, sizes); err != nil {
		t.Fatal(err)
	}
	if got := st.list(); !got["small"] || !got["big"] {
		t.Errorf("listed %v, want small and big", got)
	}
	sizes = readDirSizes(st.root)
	if _, ok := sizes["gone"]; ok {
		t.Error("the line of a directory no longer in files/ stayed")
	}
	if d := sizes["small"]; d.size >= 1<<40 || d.mtime != st.infoMtime("small") {
		t.Errorf("small cached as %v, want it measured again", d)
	}
}

// Without a size filter, the cache is neither read nor written
func TestList_NoSizeFilterNoCache(t *testing.T) {
	st := newSizeTrash(t, config.History{})
	st.dir("d", 1024, time.Hour)
	if got := st.list(); !got["d"] {
		t.Fatalf("listed %v, want d", got)
	}
	if _, err := os.Stat(filepath.Join(st.root, dirSizesFile)); !os.IsNotExist(err) {
		t.Errorf("%s was written without a size filter", dirSizesFile)
	}
}

// A trash where the cache cannot be written still lists, judging the
// directories by measuring them
func TestList_CacheNotWritable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can write anywhere")
	}
	st := newSizeTrash(t, config.History{
		Exclude: config.ExcludeConfig{Size: config.SizeConfig{Max: "100KB"}},
	})
	st.dir("small", 1024, time.Hour)
	st.dir("big", 512*1024, time.Hour)
	if err := os.Chmod(st.root, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(st.root, 0o700) })

	if got := st.list(); !got["small"] || got["big"] {
		t.Errorf("listed %v, want small only", got)
	}
	if _, err := os.Stat(filepath.Join(st.root, dirSizesFile)); !os.IsNotExist(err) {
		t.Error("the cache was written into a read-only trash")
	}
}
