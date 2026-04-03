//go:build !windows

package xdg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsOnSameDevice(t *testing.T) {
	dir := t.TempDir()

	fileA := filepath.Join(dir, "a.txt")
	fileB := filepath.Join(dir, "b.txt")
	os.WriteFile(fileA, []byte("a"), 0644)
	os.WriteFile(fileB, []byte("b"), 0644)

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

	os.WriteFile(real, []byte("data"), 0644)
	os.Symlink(real, link)

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
	os.WriteFile(file, []byte("x"), 0644)

	if isValidExternalTrash(file) {
		t.Error("regular file should not be valid external trash")
	}
}

func TestIsValidExternalTrash_ValidDir(t *testing.T) {
	dir := t.TempDir()
	trashDir := filepath.Join(dir, ".Trash-1000")
	os.MkdirAll(filepath.Join(trashDir, "files"), 0700)
	os.MkdirAll(filepath.Join(trashDir, "info"), 0700)

	if !isValidExternalTrash(trashDir) {
		t.Error("properly structured trash dir should be valid")
	}
}

func TestIsValidExternalTrash_Symlink(t *testing.T) {
	dir := t.TempDir()
	realDir := filepath.Join(dir, "real")
	os.MkdirAll(filepath.Join(realDir, "files"), 0700)
	os.MkdirAll(filepath.Join(realDir, "info"), 0700)

	linkDir := filepath.Join(dir, "link")
	os.Symlink(realDir, linkDir)

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
	for _, p := range points {
		if p == "/" {
			hasRoot = true
			break
		}
	}
	if !hasRoot {
		t.Error("mount points should include /")
	}
}
