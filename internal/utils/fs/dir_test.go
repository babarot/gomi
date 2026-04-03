package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirSize_EmptyDir(t *testing.T) {
	dir := createTempDir(t)

	size, err := DirSize(dir)
	if err != nil {
		t.Fatalf("DirSize() error = %v", err)
	}
	if size != 0 {
		t.Errorf("DirSize() = %d, want 0 for empty dir", size)
	}
}

func TestDirSize_SingleFile(t *testing.T) {
	dir := createTempDir(t)
	content := "hello world"
	createTestFile(t, filepath.Join(dir, "file.txt"), content)

	size, err := DirSize(dir)
	if err != nil {
		t.Fatalf("DirSize() error = %v", err)
	}
	if size != int64(len(content)) {
		t.Errorf("DirSize() = %d, want %d", size, len(content))
	}
}

func TestDirSize_MultipleFiles(t *testing.T) {
	dir := createTempDir(t)
	createTestFile(t, filepath.Join(dir, "a.txt"), "aaa")
	createTestFile(t, filepath.Join(dir, "b.txt"), "bbbbb")

	size, err := DirSize(dir)
	if err != nil {
		t.Fatalf("DirSize() error = %v", err)
	}
	if size != 8 {
		t.Errorf("DirSize() = %d, want 8", size)
	}
}

func TestDirSize_NestedDirs(t *testing.T) {
	dir := createTempDir(t)
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	createTestFile(t, filepath.Join(dir, "top.txt"), "top")
	createTestFile(t, filepath.Join(sub, "nested.txt"), "nested")

	size, err := DirSize(dir)
	if err != nil {
		t.Fatalf("DirSize() error = %v", err)
	}
	expected := int64(len("top") + len("nested"))
	if size != expected {
		t.Errorf("DirSize() = %d, want %d", size, expected)
	}
}

func TestDirSize_SymlinksSkipped(t *testing.T) {
	dir := createTempDir(t)
	content := "real file"
	realFile := filepath.Join(dir, "real.txt")
	createTestFile(t, realFile, content)

	linkPath := filepath.Join(dir, "link.txt")
	if err := os.Symlink(realFile, linkPath); err != nil {
		t.Fatal(err)
	}

	size, err := DirSize(dir)
	if err != nil {
		t.Fatalf("DirSize() error = %v", err)
	}
	// Symlink should be skipped, so only real file counted
	if size != int64(len(content)) {
		t.Errorf("DirSize() = %d, want %d (symlink should be skipped)", size, len(content))
	}
}

func TestDirSize_NonExistentPath(t *testing.T) {
	_, err := DirSize("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Fatal("DirSize() expected error for non-existent path, got nil")
	}
}

func TestDirSize_RegularFile(t *testing.T) {
	dir := createTempDir(t)
	file := filepath.Join(dir, "single.txt")
	content := "just a file"
	createTestFile(t, file, content)

	// DirSize should work on a single file too
	size, err := DirSize(file)
	if err != nil {
		t.Fatalf("DirSize() error = %v", err)
	}
	if size != int64(len(content)) {
		t.Errorf("DirSize() = %d, want %d", size, len(content))
	}
}
