//go:build !windows

package xdg

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// walkUsage is diskUsage as it was before it walked subdirectories at
// once: one WalkDir, in order. diskUsage is to agree with it.
func walkUsage(path string) (int64, error) {
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

func mustWrite(t *testing.T, path string, n int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, n), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDiskUsage_MatchesWalk(t *testing.T) {
	dir := t.TempDir()

	// a tree deeper and wider than dirReaders, so that walks wait for a
	// slot while others hold theirs
	var build func(d string, depth int)
	build = func(d string, depth int) {
		for i := range 5 {
			mustWrite(t, filepath.Join(d, fmt.Sprintf("f%d", i)), (i+1)*3000)
		}
		if depth == 0 {
			return
		}
		for i := range 5 {
			build(filepath.Join(d, fmt.Sprintf("d%d", i)), depth-1)
		}
	}
	build(filepath.Join(dir, "tree"), 3)

	// empty directories, a file with a name in two directories, and
	// symlinks to a directory and a file, which are not followed
	for _, d := range []string{"empty", "tree/d0/empty", "tree/d1/d2/empty"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(dir, "big"), 256*1024)
	if err := os.Link(filepath.Join(dir, "big"), filepath.Join(dir, "tree/d3/d1/big")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "tree"), filepath.Join(dir, "tree/d4/loop")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "big"), filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{dir, filepath.Join(dir, "tree"), filepath.Join(dir, "empty"), filepath.Join(dir, "big")} {
		want, err := walkUsage(p)
		if err != nil {
			t.Fatal(err)
		}
		got, err := diskUsage(p)
		if err != nil {
			t.Fatalf("diskUsage(%s): %v", p, err)
		}
		if got != want {
			t.Errorf("diskUsage(%s) = %d, want %d as one walk counts it", p, got, want)
		}
	}
}

func TestDiskUsage_UnreadableDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads any directory")
	}
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a/b/c/file"), 100)
	locked := filepath.Join(dir, "a/b")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })

	if _, err := diskUsage(dir); err == nil {
		t.Error("diskUsage of a tree with a directory it cannot read: no error, want one")
	}
}

func TestDiskUsage_Missing(t *testing.T) {
	if _, err := diskUsage(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("diskUsage of a missing path: no error, want one")
	}
}
