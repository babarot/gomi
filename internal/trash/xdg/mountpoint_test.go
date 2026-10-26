//go:build !windows

package xdg

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
)

func TestIsOnSameDevice(t *testing.T) {
	dir := t.TempDir()

	fileA := filepath.Join(dir, "a.txt")
	fileB := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(fileA, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fileB, []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}

	same, err := isOnSameDevice(fileA, fileB)
	if err != nil {
		t.Fatalf("isOnSameDevice() error = %v", err)
	}
	if !same {
		t.Error("files in same tmpdir should be on same device")
	}
}

func TestIsOnSameDevice_NonExistent(t *testing.T) {
	_, err := isOnSameDevice("/nonexistent/path1", "/nonexistent/path2")
	if err == nil {
		t.Error("expected error for non-existent paths")
	}
}

func TestIsOnSameDevice_WithSymlink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.txt")
	link := filepath.Join(dir, "link.txt")

	if err := os.WriteFile(real, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}

	same, err := isOnSameDevice(real, link)
	if err != nil {
		t.Fatalf("isOnSameDevice() error = %v", err)
	}
	if !same {
		t.Error("symlink and target should be on same device")
	}
}

func TestIsValidExternalTrash_NotExists(t *testing.T) {
	if isValidExternalTrash("/nonexistent/path") {
		t.Error("non-existent path should not be valid")
	}
}

func TestIsValidExternalTrash_RegularFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "notadir")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	if isValidExternalTrash(file) {
		t.Error("regular file should not be valid external trash")
	}
}

func TestIsValidExternalTrash_ValidDir(t *testing.T) {
	dir := t.TempDir()
	trashDir := filepath.Join(dir, ".Trash-1000")
	if err := os.MkdirAll(filepath.Join(trashDir, "files"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(trashDir, "info"), 0700); err != nil {
		t.Fatal(err)
	}

	if !isValidExternalTrash(trashDir) {
		t.Error("properly structured trash dir should be valid")
	}
}

func TestIsValidExternalTrash_Symlink(t *testing.T) {
	dir := t.TempDir()
	realDir := filepath.Join(dir, "real")
	if err := os.MkdirAll(filepath.Join(realDir, "files"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(realDir, "info"), 0700); err != nil {
		t.Fatal(err)
	}

	linkDir := filepath.Join(dir, "link")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatal(err)
	}

	if isValidExternalTrash(linkDir) {
		t.Error("symlink should not be valid external trash")
	}
}

func TestCreateTrashDir(t *testing.T) {
	dir := t.TempDir()
	trashDir := filepath.Join(dir, "newtrash")

	if err := createTrashDir(trashDir); err != nil {
		t.Fatalf("createTrashDir() error = %v", err)
	}

	for _, sub := range []string{"files", "info"} {
		subPath := filepath.Join(trashDir, sub)
		fi, err := os.Stat(subPath)
		if err != nil {
			t.Errorf("subdirectory %q not created: %v", sub, err)
			continue
		}
		if !fi.IsDir() {
			t.Errorf("%q is not a directory", sub)
		}
		if fi.Mode().Perm() != 0700 {
			t.Errorf("%q permissions = %o, want 0700", sub, fi.Mode().Perm())
		}
	}
}

func TestGetMountPoints(t *testing.T) {
	points, err := getMountPoints()
	if err != nil {
		t.Fatalf("getMountPoints() error = %v", err)
	}
	if len(points) == 0 {
		t.Error("getMountPoints() returned empty list")
	}

	// Root should always be present
	var hasRoot bool
	if slices.Contains(points, "/") {
		hasRoot = true
	}
	if !hasRoot {
		t.Error("mount points should include /")
	}
}

// makeTrashDir makes dir with files/ and info/ in it.
func makeTrashDir(t *testing.T, dir string) {
	t.Helper()
	for _, sub := range []string{"files", "info"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0700); err != nil {
			t.Fatal(err)
		}
	}
}

// makeSharedTrash makes $topdir/.Trash, sticky or not, and returns it.
func makeSharedTrash(t *testing.T, topdir string, sticky bool) string {
	t.Helper()
	dir := filepath.Join(topdir, ".Trash")
	if err := os.MkdirAll(dir, 0777); err != nil {
		t.Fatal(err)
	}
	mode := os.FileMode(0777)
	if sticky {
		mode |= os.ModeSticky
	}
	if err := os.Chmod(dir, mode); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode()&os.ModeSticky != 0; got != sticky {
		t.Skipf("cannot set the sticky bit to %v here", sticky)
	}
	return dir
}

// The spec: $topdir/.Trash must have the sticky bit set, or it is not used.
func TestIsValidExternalTrash_SharedTrashStickyBit(t *testing.T) {
	top := t.TempDir()
	if dir := makeSharedTrash(t, top, true); !isValidExternalTrash(dir) {
		t.Error(".Trash with the sticky bit should be valid")
	}
	top = t.TempDir()
	if dir := makeSharedTrash(t, top, false); isValidExternalTrash(dir) {
		t.Error(".Trash without the sticky bit should not be valid")
	}
}

// scanExternalTrashes asks isValidExternalTrash about $topdir/.Trash/$uid,
// and the spec says that one is not used unless $topdir/.Trash is a sticky,
// non-symlink directory (the user falls back to $topdir/.Trash-$uid).
func TestIsValidExternalTrash_UidDirUnderNonStickyTrash(t *testing.T) {
	top := t.TempDir()
	shared := makeSharedTrash(t, top, false)
	uidDir := filepath.Join(shared, strconv.Itoa(os.Getuid()))
	makeTrashDir(t, uidDir)

	if isValidExternalTrash(uidDir) {
		t.Error("$topdir/.Trash/$uid under a non-sticky .Trash should not be valid")
	}
}

func TestIsValidExternalTrash_UidDirUnderSymlinkedTrash(t *testing.T) {
	top := t.TempDir()
	elsewhere := makeSharedTrash(t, t.TempDir(), true)
	if err := os.Symlink(elsewhere, filepath.Join(top, ".Trash")); err != nil {
		t.Fatal(err)
	}
	uidDir := filepath.Join(top, ".Trash", strconv.Itoa(os.Getuid()))
	makeTrashDir(t, uidDir)

	if isValidExternalTrash(uidDir) {
		t.Error("$topdir/.Trash/$uid through a symlinked .Trash should not be valid")
	}
}

// The $uid directory under a proper sticky .Trash is valid.
func TestIsValidExternalTrash_UidDirUnderStickyTrash(t *testing.T) {
	shared := makeSharedTrash(t, t.TempDir(), true)
	uidDir := filepath.Join(shared, strconv.Itoa(os.Getuid()))
	makeTrashDir(t, uidDir)
	if !isValidExternalTrash(uidDir) {
		t.Error("$topdir/.Trash/$uid under a sticky .Trash should be valid")
	}
}

// .Trash-$uid needs no sticky bit; it belongs to one user.
func TestIsValidExternalTrash_UserTrashWithoutSticky(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".Trash-"+strconv.Itoa(os.Getuid()))
	makeTrashDir(t, dir)
	fi, err := os.Lstat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSticky != 0 {
		t.Fatal("test dir unexpectedly sticky")
	}
	if !isValidExternalTrash(dir) {
		t.Error(".Trash-$uid without the sticky bit should be valid")
	}
}

// A trash made by another tool with looser modes on files/ and info/ is
// still used, so what is in it stays listable and restorable.
func TestIsValidExternalTrash_LooseSubdirModes(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".Trash-"+strconv.Itoa(os.Getuid()))
	makeTrashDir(t, dir)
	for _, sub := range []string{"files", "info"} {
		if err := os.Chmod(filepath.Join(dir, sub), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if !isValidExternalTrash(dir) {
		t.Error("trash with 0755 subdirectories should still be valid")
	}
}

// A trash whose files/ and info/ cannot be looked at cannot take files.
func TestIsValidExternalTrash_UnsearchableTrash(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root is never denied permission")
	}
	dir := filepath.Join(t.TempDir(), ".Trash-"+strconv.Itoa(os.Getuid()))
	makeTrashDir(t, dir)
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) })

	if isValidExternalTrash(dir) {
		t.Error("trash whose subdirectories cannot be stat'ed should not be valid")
	}
}
